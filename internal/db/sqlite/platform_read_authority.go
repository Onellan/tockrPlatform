package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
	"github.com/Onellan/tockrplatform/internal/events"
	"github.com/Onellan/tockrplatform/internal/platform/readauthority"
	"github.com/Onellan/tockrplatform/internal/store"
)

const (
	readAuthorityChecksumAlgorithm = "sha256"
	readAuthoritySnapshotPageSize  = readauthority.SnapshotMaxPageSize
	readAuthoritySnapshotMaxRows   = readauthority.SnapshotMaxRecords
)

type readAuthorityMigrationSeed struct {
	Version  int
	Name     string
	Checksum string
}

type readAuthoritySourceEvent struct {
	EventID       string
	EventType     string
	Sequence      int64
	SchemaVersion int
}

func (s *Store) CreateReadAuthoritySnapshot(ctx context.Context, consumer string, entityKinds []string, ttl time.Duration, pageSize int, createdAt time.Time) (domain.ReadAuthoritySnapshot, error) {
	consumer = strings.TrimSpace(consumer)
	if err := readauthority.ValidateConsumer(consumer); err != nil {
		return domain.ReadAuthoritySnapshot{}, store.ErrInvalidReadAuthorityConsumer
	}
	if pageSize == 0 {
		pageSize = readAuthoritySnapshotPageSize
	}
	if pageSize < 1 || pageSize > readAuthoritySnapshotPageSize || ttl <= 0 || ttl > store.MaxReadAuthoritySnapshotTTL || ttl%time.Second != 0 || createdAt.IsZero() {
		return domain.ReadAuthoritySnapshot{}, store.ErrInvalidReadAuthorityRequest
	}
	kinds, err := validateReadAuthorityKinds(entityKinds)
	if err != nil {
		return domain.ReadAuthoritySnapshot{}, err
	}

	createdAt = createdAt.UTC()
	expiresAt := createdAt.Add(ttl)
	snapshotID, err := newReadAuthorityID()
	if err != nil {
		return domain.ReadAuthoritySnapshot{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ReadAuthoritySnapshot{}, fmt.Errorf("begin read-authority snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	sourceCursor, err := readAuthoritySourceCursorTx(ctx, tx)
	if err != nil {
		return domain.ReadAuthoritySnapshot{}, err
	}
	seed, err := readAuthorityMigrationSeedTx(ctx, tx)
	if err != nil {
		return domain.ReadAuthoritySnapshot{}, err
	}
	records, err := materializeReadAuthorityRecordsTx(ctx, tx, kinds, seed)
	if err != nil {
		return domain.ReadAuthoritySnapshot{}, err
	}
	if len(records) > readAuthoritySnapshotMaxRows {
		return domain.ReadAuthoritySnapshot{}, store.ErrInvalidReadAuthorityRequest
	}
	checksum, err := readAuthorityRecordsChecksum(records)
	if err != nil {
		return domain.ReadAuthoritySnapshot{}, err
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO platform_read_authority_snapshots(
		snapshot_id,consumer_key,contract_version,source_cursor,checksum_algorithm,
		checksum,created_at,expires_at,record_count,page_size,complete,finalized_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,0,NULL)`, snapshotID, consumer,
		readauthority.VersionV2, sourceCursor, readAuthorityChecksumAlgorithm,
		checksum, formatTime(createdAt), formatTime(expiresAt), len(records), pageSize); err != nil {
		return domain.ReadAuthoritySnapshot{}, fmt.Errorf("create read-authority snapshot: %w", err)
	}
	for ordinal, record := range records {
		recordJSON, err := json.Marshal(record)
		if err != nil {
			return domain.ReadAuthoritySnapshot{}, fmt.Errorf("encode read-authority record: %w", err)
		}
		recordHash := readauthority.BodyDigest(recordJSON)
		if _, err := tx.ExecContext(ctx, `INSERT INTO platform_read_authority_snapshot_records(
			snapshot_id,ordinal,entity_kind,record_id,record_hash,record_json)
			VALUES(?,?,?,?,?,?)`, snapshotID, ordinal, string(record.EntityKind), record.ID,
			recordHash, string(recordJSON)); err != nil {
			return domain.ReadAuthoritySnapshot{}, fmt.Errorf("store read-authority record: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE platform_read_authority_snapshots
		SET complete=1,finalized_at=? WHERE snapshot_id=? AND complete=0`, formatTime(createdAt), snapshotID); err != nil {
		return domain.ReadAuthoritySnapshot{}, fmt.Errorf("finalize read-authority snapshot: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.ReadAuthoritySnapshot{}, fmt.Errorf("commit read-authority snapshot: %w", err)
	}
	return domain.ReadAuthoritySnapshot{
		Version:           readauthority.VersionV2,
		SnapshotID:        snapshotID,
		Consumer:          consumer,
		ContractVersion:   readauthority.VersionV2,
		SourceCursor:      sourceCursor,
		ChecksumAlgorithm: readAuthorityChecksumAlgorithm,
		Checksum:          checksum,
		CreatedAt:         createdAt,
		ExpiresAt:         expiresAt,
		RecordCount:       len(records),
		PageSize:          pageSize,
		Complete:          true,
	}, nil
}

func (s *Store) GetReadAuthoritySnapshot(ctx context.Context, consumer, snapshotID string, at time.Time) (domain.ReadAuthoritySnapshot, error) {
	consumer = strings.TrimSpace(consumer)
	if err := readauthority.ValidateConsumer(consumer); err != nil {
		return domain.ReadAuthoritySnapshot{}, store.ErrInvalidReadAuthorityConsumer
	}
	if strings.TrimSpace(snapshotID) == "" || at.IsZero() {
		return domain.ReadAuthoritySnapshot{}, store.ErrInvalidReadAuthorityRequest
	}
	snapshot, records, err := s.readAndVerifyReadAuthoritySnapshot(ctx, consumer, snapshotID, at.UTC())
	if err != nil {
		return domain.ReadAuthoritySnapshot{}, err
	}
	if len(records) != snapshot.RecordCount {
		return domain.ReadAuthoritySnapshot{}, store.ErrReadAuthorityChecksumMismatch
	}
	return snapshot, nil
}

func (s *Store) ListReadAuthoritySnapshotRecords(ctx context.Context, consumer, snapshotID, entityKind, after string, limit int, at time.Time) (domain.ReadAuthoritySnapshotPage, error) {
	consumer = strings.TrimSpace(consumer)
	if err := readauthority.ValidateConsumer(consumer); err != nil {
		return domain.ReadAuthoritySnapshotPage{}, store.ErrInvalidReadAuthorityConsumer
	}
	if limit < 1 || limit > readAuthoritySnapshotPageSize || strings.TrimSpace(snapshotID) == "" || at.IsZero() {
		return domain.ReadAuthoritySnapshotPage{}, store.ErrInvalidReadAuthorityRequest
	}
	if entityKind != "" {
		if err := readauthority.ValidateEntityKind(readauthority.EntityKind(entityKind)); err != nil {
			return domain.ReadAuthoritySnapshotPage{}, store.ErrInvalidReadAuthorityRequest
		}
	}
	snapshot, records, err := s.readAndVerifyReadAuthoritySnapshot(ctx, consumer, snapshotID, at.UTC())
	if err != nil {
		return domain.ReadAuthoritySnapshotPage{}, err
	}
	startOrdinal, err := decodeReadAuthorityPageCursor(after, snapshot, entityKind)
	if err != nil {
		return domain.ReadAuthoritySnapshotPage{}, err
	}

	selected := make([]struct {
		ordinal int
		record  readauthority.Record
	}, 0, limit)
	for ordinal, record := range records {
		if ordinal <= startOrdinal || (entityKind != "" && string(record.EntityKind) != entityKind) {
			continue
		}
		selected = append(selected, struct {
			ordinal int
			record  readauthority.Record
		}{ordinal: ordinal, record: record})
		if len(selected) == limit {
			break
		}
	}
	page := domain.ReadAuthoritySnapshotPage{Snapshot: snapshot, Complete: true, Records: make([]domain.ReadAuthorityRecord, 0, len(selected))}
	for _, item := range selected {
		page.Records = append(page.Records, readAuthorityRecordToDomain(item.record))
	}
	if len(selected) == limit {
		for ordinal := selected[len(selected)-1].ordinal + 1; ordinal < len(records); ordinal++ {
			if entityKind == "" || string(records[ordinal].EntityKind) == entityKind {
				page.Complete = false
				page.NextCursor = encodeReadAuthorityPageCursor(snapshot, entityKind, selected[len(selected)-1].ordinal)
				break
			}
		}
	}
	return page, nil
}

func (s *Store) CleanupReadAuthoritySnapshots(ctx context.Context, consumer string, at time.Time, limit int) (int64, error) {
	consumer = strings.TrimSpace(consumer)
	if err := readauthority.ValidateConsumer(consumer); err != nil {
		return 0, store.ErrInvalidReadAuthorityConsumer
	}
	if at.IsZero() || limit < 1 || limit > 1000 {
		return 0, store.ErrInvalidReadAuthorityRequest
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM platform_read_authority_snapshots
		WHERE snapshot_id IN (SELECT snapshot_id FROM platform_read_authority_snapshots
		WHERE consumer_key=? AND expires_at<=? ORDER BY expires_at,snapshot_id LIMIT ?)`, consumer, formatTime(at.UTC()), limit)
	if err != nil {
		return 0, fmt.Errorf("cleanup read-authority snapshots: %w", err)
	}
	removed, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count cleaned read-authority snapshots: %w", err)
	}
	return removed, nil
}

func validateReadAuthorityKinds(values []string) ([]readauthority.EntityKind, error) {
	if len(values) == 0 || len(values) > len(readauthority.CanonicalEntityKinds()) {
		return nil, store.ErrInvalidReadAuthorityRequest
	}
	seen := make(map[readauthority.EntityKind]struct{}, len(values))
	for _, value := range values {
		kind := readauthority.EntityKind(strings.TrimSpace(value))
		if err := readauthority.ValidateEntityKind(kind); err != nil {
			return nil, store.ErrInvalidReadAuthorityRequest
		}
		if _, exists := seen[kind]; exists {
			return nil, store.ErrInvalidReadAuthorityRequest
		}
		seen[kind] = struct{}{}
	}
	ordered := make([]readauthority.EntityKind, 0, len(seen))
	for _, kind := range readauthority.CanonicalEntityKinds() {
		if _, exists := seen[kind]; exists {
			ordered = append(ordered, kind)
		}
	}
	return ordered, nil
}

func newReadAuthorityID() (string, error) {
	var raw [18]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate read-authority snapshot id: %w", err)
	}
	return "snap_" + base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func readAuthoritySourceCursorTx(ctx context.Context, tx *sql.Tx) (string, error) {
	var boundary int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(id),0) FROM platform_outbox`).Scan(&boundary); err != nil {
		return "", fmt.Errorf("read read-authority source cursor: %w", err)
	}
	return readauthority.SourceCursor(boundary), nil
}

func readAuthorityMigrationSeedTx(ctx context.Context, tx *sql.Tx) (*readAuthorityMigrationSeed, error) {
	var seed readAuthorityMigrationSeed
	if err := tx.QueryRowContext(ctx, `SELECT version,name,checksum FROM schema_migrations WHERE version=6`).Scan(&seed.Version, &seed.Name, &seed.Checksum); err != nil {
		return nil, fmt.Errorf("read product migration provenance: %w", err)
	}
	return &seed, nil
}

func materializeReadAuthorityRecordsTx(ctx context.Context, tx *sql.Tx, kinds []readauthority.EntityKind, seed *readAuthorityMigrationSeed) ([]readauthority.Record, error) {
	records := make([]readauthority.Record, 0)
	for _, kind := range kinds {
		var values []readauthority.Record
		var err error
		switch kind {
		case readauthority.EntityUser:
			values, err = materializeUsersTx(ctx, tx)
		case readauthority.EntityOrganisation:
			values, err = materializeOrganisationsTx(ctx, tx)
		case readauthority.EntityOrganisationMembership:
			values, err = materializeOrganisationMembershipsTx(ctx, tx)
		case readauthority.EntityWorkspace:
			values, err = materializeWorkspacesTx(ctx, tx)
		case readauthority.EntityWorkspaceMembership:
			values, err = materializeWorkspaceMembershipsTx(ctx, tx)
		case readauthority.EntityProduct:
			values, err = materializeProductsTx(ctx, tx, seed)
		case readauthority.EntityOrganisationEntitlement:
			values, err = materializeEntitlementsTx(ctx, tx)
		case readauthority.EntityUserProductAssignment:
			values, err = materializeAssignmentsTx(ctx, tx)
		}
		if err != nil {
			return nil, err
		}
		records = append(records, values...)
		if len(records) > readAuthoritySnapshotMaxRows {
			return nil, store.ErrInvalidReadAuthorityRequest
		}
	}
	sort.SliceStable(records, func(i, j int) bool {
		left, right := records[i], records[j]
		leftRank, rightRank := readAuthorityKindRank(left.EntityKind), readAuthorityKindRank(right.EntityKind)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		for _, pair := range [][2]string{{left.ID, right.ID}, {left.UserID, right.UserID}, {left.OrganisationID, right.OrganisationID}, {left.WorkspaceID, right.WorkspaceID}, {left.ProductKey, right.ProductKey}, {left.Role, right.Role}} {
			if pair[0] != pair[1] {
				return pair[0] < pair[1]
			}
		}
		return false
	})
	return records, nil
}

func readAuthorityKindRank(kind readauthority.EntityKind) int {
	for index, candidate := range readauthority.CanonicalEntityKinds() {
		if kind == candidate {
			return index
		}
	}
	return len(readauthority.CanonicalEntityKinds())
}

func materializeUsersTx(ctx context.Context, tx *sql.Tx) ([]readauthority.Record, error) {
	rows, err := tx.QueryContext(ctx, `SELECT public_id,active FROM users ORDER BY public_id LIMIT 10001`)
	if err != nil {
		return nil, fmt.Errorf("read snapshot users: %w", err)
	}
	type rowValue struct {
		id     string
		active int
	}
	values := make([]rowValue, 0)
	for rows.Next() {
		var value rowValue
		if err := rows.Scan(&value.id, &value.active); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan snapshot user: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate snapshot users: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close snapshot users: %w", err)
	}
	if len(values) > readAuthoritySnapshotMaxRows {
		return nil, store.ErrInvalidReadAuthorityRequest
	}
	records := make([]readauthority.Record, 0, len(values))
	for _, value := range values {
		record := readauthority.Record{EntityKind: readauthority.EntityUser, ID: value.id, Status: snapshotActiveStatus(value.active)}
		if err := addEventProvenance(ctx, tx, &record, string(events.AggregateUser), value.id, "", ""); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func materializeOrganisationsTx(ctx context.Context, tx *sql.Tx) ([]readauthority.Record, error) {
	rows, err := tx.QueryContext(ctx, `SELECT public_id,name,status FROM organisations ORDER BY public_id LIMIT 10001`)
	if err != nil {
		return nil, fmt.Errorf("read snapshot organisations: %w", err)
	}
	type rowValue struct{ id, name, status string }
	values := make([]rowValue, 0)
	for rows.Next() {
		var value rowValue
		if err := rows.Scan(&value.id, &value.name, &value.status); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan snapshot organisation: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate snapshot organisations: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close snapshot organisations: %w", err)
	}
	if len(values) > readAuthoritySnapshotMaxRows {
		return nil, store.ErrInvalidReadAuthorityRequest
	}
	records := make([]readauthority.Record, 0, len(values))
	for _, value := range values {
		record := readauthority.Record{EntityKind: readauthority.EntityOrganisation, ID: value.id, Status: readauthority.RecordStatus(value.status), Name: value.name}
		if err := addEventProvenance(ctx, tx, &record, string(events.AggregateOrganisation), value.id, "", ""); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func materializeOrganisationMembershipsTx(ctx context.Context, tx *sql.Tx) ([]readauthority.Record, error) {
	rows, err := tx.QueryContext(ctx, `SELECT m.public_id,u.public_id,o.public_id,m.role,m.active
		FROM organisation_memberships m JOIN users u ON u.id=m.user_id JOIN organisations o ON o.id=m.organisation_id
		ORDER BY m.public_id LIMIT 10001`)
	if err != nil {
		return nil, fmt.Errorf("read snapshot organisation memberships: %w", err)
	}
	type rowValue struct {
		id, userID, organisationID, role string
		active                           int
	}
	values := make([]rowValue, 0)
	for rows.Next() {
		var value rowValue
		if err := rows.Scan(&value.id, &value.userID, &value.organisationID, &value.role, &value.active); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan snapshot organisation membership: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate snapshot organisation memberships: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close snapshot organisation memberships: %w", err)
	}
	if len(values) > readAuthoritySnapshotMaxRows {
		return nil, store.ErrInvalidReadAuthorityRequest
	}
	records := make([]readauthority.Record, 0, len(values))
	for _, value := range values {
		record := readauthority.Record{EntityKind: readauthority.EntityOrganisationMembership, ID: value.id, Status: snapshotMembershipStatus(value.active), UserID: value.userID, OrganisationID: value.organisationID, Role: value.role}
		if err := addEventProvenance(ctx, tx, &record, string(events.AggregateOrganisation), value.organisationID, "membership_id", value.id); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func materializeWorkspacesTx(ctx context.Context, tx *sql.Tx) ([]readauthority.Record, error) {
	rows, err := tx.QueryContext(ctx, `SELECT w.public_id,o.public_id,w.name,w.status FROM workspaces w JOIN organisations o ON o.id=w.organisation_id ORDER BY w.public_id LIMIT 10001`)
	if err != nil {
		return nil, fmt.Errorf("read snapshot workspaces: %w", err)
	}
	type rowValue struct{ id, organisationID, name, status string }
	values := make([]rowValue, 0)
	for rows.Next() {
		var value rowValue
		if err := rows.Scan(&value.id, &value.organisationID, &value.name, &value.status); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan snapshot workspace: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate snapshot workspaces: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close snapshot workspaces: %w", err)
	}
	if len(values) > readAuthoritySnapshotMaxRows {
		return nil, store.ErrInvalidReadAuthorityRequest
	}
	records := make([]readauthority.Record, 0, len(values))
	for _, value := range values {
		record := readauthority.Record{EntityKind: readauthority.EntityWorkspace, ID: value.id, Status: readauthority.RecordStatus(value.status), OrganisationID: value.organisationID, Name: value.name}
		if err := addEventProvenance(ctx, tx, &record, string(events.AggregateWorkspace), value.id, "", ""); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func materializeWorkspaceMembershipsTx(ctx context.Context, tx *sql.Tx) ([]readauthority.Record, error) {
	rows, err := tx.QueryContext(ctx, `SELECT m.public_id,u.public_id,w.public_id,m.role,m.active
		FROM workspace_memberships m JOIN users u ON u.id=m.user_id JOIN workspaces w ON w.id=m.workspace_id
		ORDER BY m.public_id LIMIT 10001`)
	if err != nil {
		return nil, fmt.Errorf("read snapshot workspace memberships: %w", err)
	}
	type rowValue struct {
		id, userID, workspaceID, role string
		active                        int
	}
	values := make([]rowValue, 0)
	for rows.Next() {
		var value rowValue
		if err := rows.Scan(&value.id, &value.userID, &value.workspaceID, &value.role, &value.active); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan snapshot workspace membership: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate snapshot workspace memberships: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close snapshot workspace memberships: %w", err)
	}
	if len(values) > readAuthoritySnapshotMaxRows {
		return nil, store.ErrInvalidReadAuthorityRequest
	}
	records := make([]readauthority.Record, 0, len(values))
	for _, value := range values {
		record := readauthority.Record{EntityKind: readauthority.EntityWorkspaceMembership, ID: value.id, Status: snapshotMembershipStatus(value.active), UserID: value.userID, WorkspaceID: value.workspaceID, Role: value.role}
		if err := addEventProvenance(ctx, tx, &record, string(events.AggregateWorkspace), value.workspaceID, "membership_id", value.id); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func materializeProductsTx(ctx context.Context, tx *sql.Tx, seed *readAuthorityMigrationSeed) ([]readauthority.Record, error) {
	rows, err := tx.QueryContext(ctx, `SELECT product_key,status FROM products ORDER BY product_key LIMIT 10001`)
	if err != nil {
		return nil, fmt.Errorf("read snapshot products: %w", err)
	}
	type rowValue struct{ key, status string }
	values := make([]rowValue, 0)
	for rows.Next() {
		var value rowValue
		if err := rows.Scan(&value.key, &value.status); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan snapshot product: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate snapshot products: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close snapshot products: %w", err)
	}
	if len(values) > readAuthoritySnapshotMaxRows {
		return nil, store.ErrInvalidReadAuthorityRequest
	}
	records := make([]readauthority.Record, 0, len(values))
	for _, value := range values {
		record := readauthority.Record{EntityKind: readauthority.EntityProduct, ID: value.key, Status: readauthority.RecordStatus(value.status)}
		event, found, err := latestReadAuthoritySourceEventTx(ctx, tx, string(events.AggregateProduct), value.key, "", "")
		if err != nil {
			return nil, err
		}
		if found {
			setReadAuthorityEventProvenance(&record, event)
		} else if value.status == string(domain.ProductActive) && (value.key == readauthority.ProductCTRL || value.key == readauthority.ProductIMS) && seed != nil {
			record.ProvenanceKind = readauthority.ProvenanceMigrationSeed
			record.MigrationVersion = seed.Version
			record.MigrationName = seed.Name
			record.MigrationChecksum = seed.Checksum
		} else {
			return nil, fmt.Errorf("missing source provenance for product %q", value.key)
		}
		if err := readauthority.ValidateRecordVersion(readauthority.VersionV2, record); err != nil {
			return nil, fmt.Errorf("validate snapshot product %q: %w", value.key, err)
		}
		records = append(records, record)
	}
	return records, nil
}

func materializeEntitlementsTx(ctx context.Context, tx *sql.Tx) ([]readauthority.Record, error) {
	rows, err := tx.QueryContext(ctx, `SELECT e.public_id,o.public_id,e.product_key,e.status FROM organisation_product_entitlements e JOIN organisations o ON o.id=e.organisation_id ORDER BY e.public_id LIMIT 10001`)
	if err != nil {
		return nil, fmt.Errorf("read snapshot entitlements: %w", err)
	}
	type rowValue struct{ id, organisationID, productKey, status string }
	values := make([]rowValue, 0)
	for rows.Next() {
		var value rowValue
		if err := rows.Scan(&value.id, &value.organisationID, &value.productKey, &value.status); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan snapshot entitlement: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate snapshot entitlements: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close snapshot entitlements: %w", err)
	}
	if len(values) > readAuthoritySnapshotMaxRows {
		return nil, store.ErrInvalidReadAuthorityRequest
	}
	records := make([]readauthority.Record, 0, len(values))
	for _, value := range values {
		record := readauthority.Record{EntityKind: readauthority.EntityOrganisationEntitlement, ID: value.id, Status: readauthority.RecordStatus(value.status), OrganisationID: value.organisationID, ProductKey: value.productKey}
		if err := addEventProvenance(ctx, tx, &record, string(events.AggregateAccess), value.id, "entitlement_id", value.id); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func materializeAssignmentsTx(ctx context.Context, tx *sql.Tx) ([]readauthority.Record, error) {
	rows, err := tx.QueryContext(ctx, `SELECT a.public_id,u.public_id,o.public_id,a.product_key,a.status FROM user_product_assignments a JOIN organisations o ON o.id=a.organisation_id JOIN users u ON u.id=a.user_id ORDER BY a.public_id LIMIT 10001`)
	if err != nil {
		return nil, fmt.Errorf("read snapshot assignments: %w", err)
	}
	type rowValue struct{ id, userID, organisationID, productKey, status string }
	values := make([]rowValue, 0)
	for rows.Next() {
		var value rowValue
		if err := rows.Scan(&value.id, &value.userID, &value.organisationID, &value.productKey, &value.status); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan snapshot assignment: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate snapshot assignments: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close snapshot assignments: %w", err)
	}
	if len(values) > readAuthoritySnapshotMaxRows {
		return nil, store.ErrInvalidReadAuthorityRequest
	}
	records := make([]readauthority.Record, 0, len(values))
	for _, value := range values {
		record := readauthority.Record{EntityKind: readauthority.EntityUserProductAssignment, ID: value.id, Status: readauthority.RecordStatus(value.status), UserID: value.userID, OrganisationID: value.organisationID, ProductKey: value.productKey}
		if err := addEventProvenance(ctx, tx, &record, string(events.AggregateAccess), value.id, "assignment_id", value.id); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func snapshotActiveStatus(active int) readauthority.RecordStatus {
	if active == 1 {
		return readauthority.StatusActive
	}
	return readauthority.StatusArchived
}

func snapshotMembershipStatus(active int) readauthority.RecordStatus {
	if active == 1 {
		return readauthority.StatusActive
	}
	return readauthority.StatusRevoked
}

func addEventProvenance(ctx context.Context, tx *sql.Tx, record *readauthority.Record, aggregateType, aggregateID, payloadField, payloadValue string) error {
	event, found, err := latestReadAuthoritySourceEventTx(ctx, tx, aggregateType, aggregateID, payloadField, payloadValue)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("missing source provenance for %s %q", record.EntityKind, record.ID)
	}
	setReadAuthorityEventProvenance(record, event)
	if err := readauthority.ValidateRecordVersion(readauthority.VersionV2, *record); err != nil {
		return fmt.Errorf("validate snapshot %s %q: %w", record.EntityKind, record.ID, err)
	}
	return nil
}

func latestReadAuthoritySourceEventTx(ctx context.Context, tx *sql.Tx, aggregateType, aggregateID, payloadField, payloadValue string) (readAuthoritySourceEvent, bool, error) {
	query := `SELECT event_id,event_type,sequence,schema_version
		FROM platform_outbox WHERE aggregate_type=? AND aggregate_id=?`
	args := []any{aggregateType, aggregateID}
	if payloadField != "" {
		query += ` AND payload LIKE ?`
		args = append(args, `%"`+payloadField+`":"`+payloadValue+`"%`)
	}
	query += ` ORDER BY id DESC LIMIT 1`
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return readAuthoritySourceEvent{}, false, fmt.Errorf("read source provenance: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var event readAuthoritySourceEvent
		if err := rows.Scan(&event.EventID, &event.EventType, &event.Sequence, &event.SchemaVersion); err != nil {
			return readAuthoritySourceEvent{}, false, fmt.Errorf("scan source provenance: %w", err)
		}
		return event, true, nil
	}
	if err := rows.Err(); err != nil {
		return readAuthoritySourceEvent{}, false, fmt.Errorf("iterate source provenance: %w", err)
	}
	return readAuthoritySourceEvent{}, false, nil
}

func setReadAuthorityEventProvenance(record *readauthority.Record, event readAuthoritySourceEvent) {
	record.ProvenanceKind = readauthority.ProvenanceEvent
	record.SourceEventID = event.EventID
	record.SourceSequence = event.Sequence
	record.SourceSchemaVersion = event.SchemaVersion
}

func readAuthorityRecordsChecksum(records []readauthority.Record) (string, error) {
	return readauthority.RecordsChecksum(records)
}

func (s *Store) readAndVerifyReadAuthoritySnapshot(ctx context.Context, consumer, snapshotID string, at time.Time) (domain.ReadAuthoritySnapshot, []readauthority.Record, error) {
	snapshot, err := s.readReadAuthoritySnapshotMetadata(ctx, consumer, snapshotID)
	if err != nil {
		return domain.ReadAuthoritySnapshot{}, nil, err
	}
	if !snapshot.Complete {
		return domain.ReadAuthoritySnapshot{}, nil, store.ErrReadAuthoritySnapshotIncomplete
	}
	if !at.Before(snapshot.ExpiresAt) {
		return domain.ReadAuthoritySnapshot{}, nil, store.ErrReadAuthoritySnapshotExpired
	}
	records, err := s.readReadAuthoritySnapshotRecords(ctx, snapshotID)
	if err != nil {
		return domain.ReadAuthoritySnapshot{}, nil, err
	}
	if len(records) != snapshot.RecordCount {
		return domain.ReadAuthoritySnapshot{}, nil, store.ErrReadAuthorityChecksumMismatch
	}
	checksum, err := readAuthorityRecordsChecksum(records)
	if err != nil {
		return domain.ReadAuthoritySnapshot{}, nil, err
	}
	if checksum != snapshot.Checksum {
		return domain.ReadAuthoritySnapshot{}, nil, store.ErrReadAuthorityChecksumMismatch
	}
	return snapshot, records, nil
}

func (s *Store) readReadAuthoritySnapshotMetadata(ctx context.Context, consumer, snapshotID string) (domain.ReadAuthoritySnapshot, error) {
	var snapshot domain.ReadAuthoritySnapshot
	var createdAt, expiresAt, finalizedAt string
	var complete int
	err := s.db.QueryRowContext(ctx, `SELECT snapshot_id,consumer_key,contract_version,source_cursor,checksum_algorithm,
		checksum,created_at,expires_at,record_count,page_size,complete,finalized_at
		FROM platform_read_authority_snapshots WHERE consumer_key=? AND snapshot_id=?`, consumer, snapshotID).
		Scan(&snapshot.SnapshotID, &snapshot.Consumer, &snapshot.ContractVersion, &snapshot.SourceCursor, &snapshot.ChecksumAlgorithm,
			&snapshot.Checksum, &createdAt, &expiresAt, &snapshot.RecordCount, &snapshot.PageSize, &complete, &finalizedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ReadAuthoritySnapshot{}, store.ErrReadAuthoritySnapshotNotFound
	}
	if err != nil {
		return domain.ReadAuthoritySnapshot{}, fmt.Errorf("read read-authority snapshot: %w", err)
	}
	snapshot.Version = snapshot.ContractVersion
	snapshot.CreatedAt = parseTime(createdAt)
	snapshot.ExpiresAt = parseTime(expiresAt)
	snapshot.Complete = complete == 1 && finalizedAt != ""
	if snapshot.Version != readauthority.VersionV2 || snapshot.ChecksumAlgorithm != readAuthorityChecksumAlgorithm || snapshot.RecordCount < 0 || snapshot.RecordCount > readAuthoritySnapshotMaxRows || snapshot.PageSize < 1 || snapshot.PageSize > readAuthoritySnapshotPageSize || snapshot.CreatedAt.IsZero() || snapshot.ExpiresAt.IsZero() || !snapshot.ExpiresAt.After(snapshot.CreatedAt) {
		return domain.ReadAuthoritySnapshot{}, store.ErrReadAuthorityChecksumMismatch
	}
	if _, err := readauthority.ParseSourceCursor(snapshot.SourceCursor); err != nil {
		return domain.ReadAuthoritySnapshot{}, store.ErrReadAuthorityChecksumMismatch
	}
	return snapshot, nil
}

func (s *Store) readReadAuthoritySnapshotRecords(ctx context.Context, snapshotID string) ([]readauthority.Record, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT ordinal,record_json FROM platform_read_authority_snapshot_records WHERE snapshot_id=? ORDER BY ordinal`, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("read read-authority snapshot records: %w", err)
	}
	defer rows.Close()
	records := make([]readauthority.Record, 0)
	wantOrdinal := 0
	for rows.Next() {
		var ordinal int
		var recordJSON string
		if err := rows.Scan(&ordinal, &recordJSON); err != nil {
			return nil, fmt.Errorf("%w: scan read-authority snapshot record: %v", store.ErrReadAuthorityChecksumMismatch, err)
		}
		if ordinal != wantOrdinal {
			return nil, store.ErrReadAuthorityChecksumMismatch
		}
		var record readauthority.Record
		decoder := json.NewDecoder(strings.NewReader(recordJSON))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&record); err != nil {
			return nil, fmt.Errorf("%w: decode read-authority snapshot record: %v", store.ErrReadAuthorityChecksumMismatch, err)
		}
		if err := readauthority.ValidateRecordVersion(readauthority.VersionV2, record); err != nil {
			return nil, fmt.Errorf("%w: validate read-authority snapshot record: %v", store.ErrReadAuthorityChecksumMismatch, err)
		}
		records = append(records, record)
		wantOrdinal++
		if wantOrdinal > readAuthoritySnapshotMaxRows {
			return nil, store.ErrReadAuthorityChecksumMismatch
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate read-authority snapshot records: %w", err)
	}
	return records, nil
}

type readAuthorityPageCursor struct {
	SnapshotID string `json:"snapshot_id"`
	Consumer   string `json:"consumer"`
	EntityKind string `json:"entity_kind"`
	Ordinal    int    `json:"ordinal"`
}

func encodeReadAuthorityPageCursor(snapshot domain.ReadAuthoritySnapshot, entityKind string, ordinal int) string {
	payload, _ := json.Marshal(readAuthorityPageCursor{SnapshotID: snapshot.SnapshotID, Consumer: snapshot.Consumer, EntityKind: entityKind, Ordinal: ordinal})
	return "page_" + base64.RawURLEncoding.EncodeToString(payload)
}

func decodeReadAuthorityPageCursor(value string, snapshot domain.ReadAuthoritySnapshot, entityKind string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return -1, nil
	}
	if !strings.HasPrefix(value, "page_") {
		return 0, store.ErrReadAuthorityCursorInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, "page_"))
	if err != nil {
		return 0, store.ErrReadAuthorityCursorInvalid
	}
	var cursor readAuthorityPageCursor
	if err := json.Unmarshal(payload, &cursor); err != nil || cursor.SnapshotID != snapshot.SnapshotID || cursor.Consumer != snapshot.Consumer || cursor.EntityKind != entityKind || cursor.Ordinal < 0 || cursor.Ordinal >= snapshot.RecordCount {
		return 0, store.ErrReadAuthorityCursorInvalid
	}
	return cursor.Ordinal, nil
}

func readAuthorityRecordToDomain(record readauthority.Record) domain.ReadAuthorityRecord {
	return domain.ReadAuthorityRecord{
		EntityKind: string(record.EntityKind), ID: record.ID, Status: string(record.Status), UserID: record.UserID,
		OrganisationID: record.OrganisationID, WorkspaceID: record.WorkspaceID, ProductKey: record.ProductKey,
		Role: record.Role, Name: record.Name, ProvenanceKind: string(record.ProvenanceKind), SourceEventID: record.SourceEventID,
		SourceSequence: record.SourceSequence, SourceSchemaVersion: record.SourceSchemaVersion, MigrationVersion: record.MigrationVersion,
		MigrationName: record.MigrationName, MigrationChecksum: record.MigrationChecksum,
	}
}
