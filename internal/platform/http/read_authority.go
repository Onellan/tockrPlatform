package httpserver

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
	"github.com/Onellan/tockrplatform/internal/platform/readauthority"
	"github.com/Onellan/tockrplatform/internal/store"
	"github.com/go-chi/chi/v5"
)

type readAuthorityConsumerContextKey struct{}

const readAuthorityRateWindow = time.Minute

type readAuthorityRateEntry struct {
	started time.Time
	count   int
}

type readAuthorityRateLimiter struct {
	mu      sync.Mutex
	entries map[string]readAuthorityRateEntry
}

func newReadAuthorityRateLimiter() *readAuthorityRateLimiter {
	return &readAuthorityRateLimiter{entries: make(map[string]readAuthorityRateEntry)}
}

func (l *readAuthorityRateLimiter) Allow(consumer string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for key, entry := range l.entries {
		if !now.Before(entry.started.Add(readAuthorityRateWindow)) {
			delete(l.entries, key)
		}
	}
	entry, ok := l.entries[consumer]
	if !ok || !now.Before(entry.started.Add(readAuthorityRateWindow)) {
		if len(l.entries) >= 4096 {
			for key := range l.entries {
				delete(l.entries, key)
				break
			}
		}
		l.entries[consumer] = readAuthorityRateEntry{started: now, count: 1}
		return true
	}
	if entry.count >= readauthority.RequestsPerMinute {
		return false
	}
	entry.count++
	l.entries[consumer] = entry
	return true
}

func cloneReadAuthorityKeys(source readauthority.PublicKeySet) readauthority.PublicKeySet {
	cloned := make(readauthority.PublicKeySet, len(source))
	for consumer, keys := range source {
		cloned[consumer] = make(map[string]ed25519.PublicKey, len(keys))
		for keyID, publicKey := range keys {
			cloned[consumer][keyID] = append(ed25519.PublicKey(nil), publicKey...)
		}
	}
	return cloned
}

func (s *Server) requireReadAuthority(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(readauthority.VersionHeader, readauthority.VersionV2)
		if strings.TrimSpace(r.Header.Get(readauthority.VersionHeader)) != readauthority.VersionV2 {
			writeReadAuthorityError(w, readauthority.ErrorVersionMismatch, readauthority.StateUnavailable, false, false)
			return
		}
		consumer := strings.TrimSpace(r.Header.Get(readauthority.ConsumerHeader))
		keyID := strings.TrimSpace(r.Header.Get(readauthority.KeyIDHeader))
		timestamp := strings.TrimSpace(r.Header.Get(readauthority.TimestampHeader))
		nonce := strings.TrimSpace(r.Header.Get(readauthority.NonceHeader))
		signatureText := strings.TrimSpace(r.Header.Get(readauthority.SignatureHeader))
		if err := readauthority.ValidateConsumer(consumer); err != nil || !readauthority.ValidateKeyID(keyID) || !validReadAuthorityTimestamp(timestamp) || !readauthority.ValidateKeyID(nonce) || signatureText == "" {
			writeReadAuthorityError(w, readauthority.ErrorUnauthenticated, readauthority.StateUnavailable, false, false)
			return
		}
		publicKey, ok := s.readAuthorityKeys[consumer][keyID]
		if !ok || len(publicKey) != ed25519.PublicKeySize {
			writeReadAuthorityError(w, readauthority.ErrorUnauthenticated, readauthority.StateUnavailable, false, false)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, readauthority.RequestMaxBodyBytes))
		if err != nil {
			writeReadAuthorityError(w, readauthority.ErrorLimitExceeded, readauthority.StateUnavailable, false, false)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		path, err := canonicalReadAuthorityPath(r)
		if err != nil {
			writeReadAuthorityError(w, readauthority.ErrorUnauthenticated, readauthority.StateUnavailable, false, false)
			return
		}
		canonical, err := readauthority.CanonicalRequest(r.Method, path, readauthority.BodyDigest(body), consumer, keyID, timestamp, nonce)
		if err != nil {
			writeReadAuthorityError(w, readauthority.ErrorUnauthenticated, readauthority.StateUnavailable, false, false)
			return
		}
		signature, err := base64.RawURLEncoding.DecodeString(signatureText)
		if err != nil || len(signature) != ed25519.SignatureSize || !ed25519.Verify(publicKey, []byte(canonical), signature) {
			writeReadAuthorityError(w, readauthority.ErrorUnauthenticated, readauthority.StateUnavailable, false, false)
			return
		}
		if s.store == nil {
			writeReadAuthorityError(w, readauthority.ErrorUnavailable, readauthority.StateUnavailable, true, false)
			return
		}
		if s.readAuthorityRate == nil || !s.readAuthorityRate.Allow(consumer, time.Now().UTC()) {
			writeReadAuthorityError(w, readauthority.ErrorLimitExceeded, readauthority.StateUnavailable, true, false)
			return
		}
		reservedAt := time.Now().UTC()
		if err := s.store.ReserveReadAuthorityNonce(r.Context(), consumer, nonce, reservedAt, reservedAt.Add(24*time.Hour)); err != nil {
			if errors.Is(err, store.ErrReadAuthorityNonceReplay) || errors.Is(err, store.ErrInvalidReadAuthorityNonce) {
				writeReadAuthorityError(w, readauthority.ErrorUnauthenticated, readauthority.StateUnavailable, false, false)
				return
			}
			writeReadAuthorityError(w, readauthority.ErrorUnavailable, readauthority.StateUnavailable, true, false)
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), readAuthorityConsumerContextKey{}, consumer))
		next.ServeHTTP(w, r)
	})
}

func validReadAuthorityTimestamp(value string) bool {
	if value == "" || strings.Trim(value, "0123456789") != "" {
		return false
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return false
	}
	difference := time.Now().UTC().Unix() - parsed
	if difference < 0 {
		difference = -difference
	}
	return difference <= readauthority.SignatureMaxClockSkewSecs
}

func canonicalReadAuthorityPath(r *http.Request) (string, error) {
	path := r.URL.EscapedPath()
	if path == "" {
		path = "/"
	}
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return "", err
	}
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	return path, nil
}

func readAuthorityConsumer(r *http.Request) string {
	consumer, _ := r.Context().Value(readAuthorityConsumerContextKey{}).(string)
	return consumer
}

func (s *Server) createReadAuthoritySnapshot(w http.ResponseWriter, r *http.Request) {
	var request readauthority.SnapshotRequest
	if !decodeReadAuthorityJSON(w, r, &request) {
		writeReadAuthorityError(w, readauthority.ErrorInvalidRequest, readauthority.StateUnavailable, false, false)
		return
	}
	if err := readauthority.ValidateSnapshotRequest(request); err != nil {
		writeReadAuthorityError(w, readauthority.ErrorInvalidRequest, readauthority.StateUnavailable, false, false)
		return
	}
	kinds := make([]string, 0, len(request.EntityKinds))
	for _, kind := range request.EntityKinds {
		kinds = append(kinds, string(kind))
	}
	snapshot, err := s.store.CreateReadAuthoritySnapshot(r.Context(), readAuthorityConsumer(r), kinds, time.Duration(request.ExpiresInSeconds)*time.Second, readauthority.SnapshotMaxPageSize, time.Now().UTC())
	if err != nil {
		writeReadAuthorityStoreError(w, err)
		return
	}
	writeReadAuthorityResponse(w, http.StatusCreated, readAuthoritySnapshotMetadata(snapshot))
}

func (s *Server) listReadAuthoritySnapshotRecords(w http.ResponseWriter, r *http.Request) {
	limit, ok := readAuthorityLimit(r.URL.Query().Get("limit"), readauthority.SnapshotMaxPageSize)
	if !ok {
		writeReadAuthorityError(w, readauthority.ErrorLimitExceeded, readauthority.StateUnavailable, false, false)
		return
	}
	page, err := s.store.ListReadAuthoritySnapshotRecords(r.Context(), readAuthorityConsumer(r), chi.URLParam(r, "snapshotID"), r.URL.Query().Get("entity_kind"), r.URL.Query().Get("after"), limit, time.Now().UTC())
	if err != nil {
		writeReadAuthorityStoreError(w, err)
		return
	}
	records := make([]readauthority.Record, 0, len(page.Records))
	for _, record := range page.Records {
		wire := readAuthorityDomainRecord(record)
		if err := readauthority.ValidateRecordVersion(readauthority.VersionV2, wire); err != nil {
			writeReadAuthorityError(w, readauthority.ErrorChecksumMismatch, readauthority.StateBlocked, false, false)
			return
		}
		records = append(records, wire)
	}
	var nextCursor *string
	if page.NextCursor != "" {
		nextCursor = &page.NextCursor
	}
	writeReadAuthorityResponse(w, http.StatusOK, struct {
		Version      string                 `json:"version"`
		SnapshotID   string                 `json:"snapshot_id"`
		SourceCursor string                 `json:"source_cursor"`
		Checksum     string                 `json:"checksum"`
		Records      []readauthority.Record `json:"records"`
		NextCursor   *string                `json:"next_cursor"`
		Complete     bool                   `json:"complete"`
	}{readauthority.VersionV2, page.Snapshot.SnapshotID, page.Snapshot.SourceCursor, page.Snapshot.Checksum, records, nextCursor, page.Complete})
}

func (s *Server) listReadAuthorityChanges(w http.ResponseWriter, r *http.Request) {
	after := int64(0)
	if value := strings.TrimSpace(r.URL.Query().Get("after")); value != "" {
		var err error
		after, err = readauthority.ParseSourceCursor(value)
		if err != nil {
			writeReadAuthorityError(w, readauthority.ErrorInvalidRequest, readauthority.StateUnavailable, false, false)
			return
		}
	}
	limit, ok := readAuthorityLimit(r.URL.Query().Get("limit"), readauthority.ChangeMaxPageSize)
	if !ok {
		writeReadAuthorityError(w, readauthority.ErrorLimitExceeded, readauthority.StateUnavailable, false, false)
		return
	}
	feed, err := s.store.ListPlatformEventsAfter(r.Context(), after, limit)
	if err != nil {
		writeReadAuthorityStoreError(w, err)
		return
	}
	changes := make([]readauthority.Change, 0, len(feed.Events))
	for _, event := range feed.Events {
		eventPayload := struct {
			EventID       string          `json:"event_id"`
			EventType     string          `json:"event_type"`
			AggregateType string          `json:"aggregate_type"`
			AggregateID   string          `json:"aggregate_id"`
			Sequence      int64           `json:"sequence"`
			SchemaVersion int             `json:"schema_version"`
			OccurredAt    string          `json:"occurred_at"`
			Payload       json.RawMessage `json:"payload"`
		}{event.EventID, event.EventType, event.AggregateType, event.AggregateID, event.Sequence, event.SchemaVersion, event.OccurredAt.UTC().Format(time.RFC3339Nano), event.Payload}
		encoded, err := json.Marshal(eventPayload)
		if err != nil {
			writeReadAuthorityError(w, readauthority.ErrorBlocked, readauthority.StateBlocked, false, false)
			return
		}
		changes = append(changes, readauthority.Change{Cursor: readauthority.SourceCursor(event.GlobalCursor), Event: encoded})
	}
	var nextCursor *string
	if feed.HasMore {
		value := readauthority.SourceCursor(feed.NextCursor)
		nextCursor = &value
	}
	writeReadAuthorityResponse(w, http.StatusOK, struct {
		Version      string                 `json:"version"`
		State        readauthority.State    `json:"state"`
		SourceCursor string                 `json:"source_cursor"`
		Changes      []readauthority.Change `json:"changes"`
		NextCursor   *string                `json:"next_cursor"`
		HasMore      bool                   `json:"has_more"`
	}{readauthority.VersionV2, readauthority.StateCurrent, readauthority.SourceCursor(feed.SourceCursor), changes, nextCursor, feed.HasMore})
}

func (s *Server) readAuthorityStatus(w http.ResponseWriter, r *http.Request) {
	current, horizon, err := s.store.PlatformEventCursorBounds(r.Context())
	if err != nil {
		writeReadAuthorityError(w, readauthority.ErrorUnavailable, readauthority.StateUnavailable, true, false)
		return
	}
	writeReadAuthorityResponse(w, http.StatusOK, struct {
		Version          string              `json:"version"`
		Consumer         string              `json:"consumer"`
		ContractVersion  string              `json:"contract_version"`
		State            readauthority.State `json:"state"`
		Ready            bool                `json:"ready"`
		SourceCursor     string              `json:"source_cursor"`
		RetentionHorizon string              `json:"retention_horizon"`
	}{readauthority.VersionV2, readAuthorityConsumer(r), readauthority.VersionV2, readauthority.StateCurrent, true, readauthority.SourceCursor(current), readauthority.SourceCursor(horizon)})
}

func readAuthorityLimit(value string, maximum int) (int, bool) {
	if strings.TrimSpace(value) == "" {
		return maximum, true
	}
	parsed, err := strconv.Atoi(value)
	return parsed, err == nil && parsed > 0 && parsed <= maximum
}

func readAuthoritySnapshotMetadata(snapshot domain.ReadAuthoritySnapshot) readauthority.SnapshotMetadata {
	return readauthority.SnapshotMetadata{
		Version: readauthority.VersionV2, SnapshotID: snapshot.SnapshotID, Consumer: snapshot.Consumer,
		ContractVersion: readauthority.VersionV2, SourceCursor: snapshot.SourceCursor,
		ChecksumAlgorithm: snapshot.ChecksumAlgorithm, Checksum: snapshot.Checksum,
		CreatedAt: snapshot.CreatedAt.UTC().Format(time.RFC3339Nano), ExpiresAt: snapshot.ExpiresAt.UTC().Format(time.RFC3339Nano),
		RecordCount: snapshot.RecordCount, PageSize: snapshot.PageSize, Complete: snapshot.Complete,
	}
}

func readAuthorityDomainRecord(record domain.ReadAuthorityRecord) readauthority.Record {
	return readauthority.Record{EntityKind: readauthority.EntityKind(record.EntityKind), ID: record.ID, Status: readauthority.RecordStatus(record.Status), UserID: record.UserID, OrganisationID: record.OrganisationID, WorkspaceID: record.WorkspaceID, ProductKey: record.ProductKey, Role: record.Role, Name: record.Name, ProvenanceKind: readauthority.ProvenanceKind(record.ProvenanceKind), SourceEventID: record.SourceEventID, SourceSequence: record.SourceSequence, SourceSchemaVersion: record.SourceSchemaVersion, MigrationVersion: record.MigrationVersion, MigrationName: record.MigrationName, MigrationChecksum: record.MigrationChecksum}
}

func decodeReadAuthorityJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, readauthority.RequestMaxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return false
	}
	var extra any
	return errors.Is(decoder.Decode(&extra), io.EOF)
}

func writeReadAuthorityStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrInvalidReadAuthorityRequest), errors.Is(err, store.ErrInvalidOutboxLimit), errors.Is(err, store.ErrReadAuthorityCursorInvalid):
		writeReadAuthorityError(w, readauthority.ErrorInvalidRequest, readauthority.StateUnavailable, false, false)
	case errors.Is(err, store.ErrReadAuthoritySnapshotExpired):
		writeReadAuthorityError(w, readauthority.ErrorSnapshotExpired, readauthority.StateStale, false, true)
	case errors.Is(err, store.ErrReadAuthoritySnapshotNotFound), errors.Is(err, store.ErrReadAuthoritySnapshotIncomplete):
		writeReadAuthorityError(w, readauthority.ErrorSnapshotPartial, readauthority.StateBlocked, false, true)
	case errors.Is(err, store.ErrReadAuthorityChecksumMismatch):
		writeReadAuthorityError(w, readauthority.ErrorChecksumMismatch, readauthority.StateBlocked, false, true)
	case errors.Is(err, store.ErrReadAuthoritySourceGap):
		writeReadAuthorityError(w, readauthority.ErrorResyncRequired, readauthority.StateResyncRequired, false, true)
	case errors.Is(err, store.ErrReadAuthoritySourceInvalid):
		writeReadAuthorityError(w, readauthority.ErrorBlocked, readauthority.StateBlocked, false, false)
	default:
		writeReadAuthorityError(w, readauthority.ErrorUnavailable, readauthority.StateUnavailable, true, false)
	}
}

func writeReadAuthorityError(w http.ResponseWriter, code readauthority.ErrorCode, state readauthority.State, retryable, resync bool) {
	status := readauthority.HTTPStatus(code)
	writeReadAuthorityResponse(w, status, readauthority.ErrorResponse{Version: readauthority.VersionV2, Code: code, State: state, Retryable: retryable, ResyncRequired: resync, Message: readAuthorityMessage(code)})
}

func readAuthorityMessage(code readauthority.ErrorCode) string {
	switch code {
	case readauthority.ErrorVersionMismatch:
		return "read authority contract version is unsupported"
	case readauthority.ErrorUnauthenticated:
		return "read authority machine authentication failed"
	case readauthority.ErrorInvalidRequest:
		return "read authority request is invalid"
	case readauthority.ErrorLimitExceeded:
		return "read authority request exceeds its limit"
	case readauthority.ErrorSnapshotExpired:
		return "read authority snapshot has expired"
	case readauthority.ErrorSnapshotPartial:
		return "read authority snapshot is incomplete"
	case readauthority.ErrorChecksumMismatch:
		return "read authority snapshot integrity could not be verified"
	case readauthority.ErrorResyncRequired:
		return "read authority requires resynchronisation"
	case readauthority.ErrorBlocked:
		return "read authority source is blocked"
	default:
		return "read authority is unavailable"
	}
}

func writeReadAuthorityResponse(w http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil || len(body) > readauthority.ResponseMaxBytes {
		writeReadAuthorityError(w, readauthority.ErrorLimitExceeded, readauthority.StateUnavailable, false, false)
		return
	}
	w.Header().Set(readauthority.VersionHeader, readauthority.VersionV2)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}
