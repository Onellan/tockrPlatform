package domain

import "time"

// ReadAuthorityRecord is the persistence-neutral representation of one
// allow-listed Platform fact in a bootstrap snapshot. It intentionally has
// no credentials, product roles, billing data or SQLite fields.
type ReadAuthorityRecord struct {
	EntityKind          string `json:"entity_kind"`
	ID                  string `json:"id"`
	Status              string `json:"status"`
	UserID              string `json:"user_id,omitempty"`
	OrganisationID      string `json:"organisation_id,omitempty"`
	WorkspaceID         string `json:"workspace_id,omitempty"`
	ProductKey          string `json:"product_key,omitempty"`
	Role                string `json:"role,omitempty"`
	Name                string `json:"name,omitempty"`
	ProvenanceKind      string `json:"provenance_kind,omitempty"`
	SourceEventID       string `json:"source_event_id,omitempty"`
	SourceSequence      int64  `json:"source_sequence,omitempty"`
	SourceSchemaVersion int    `json:"source_schema_version,omitempty"`
	MigrationVersion    int    `json:"migration_version,omitempty"`
	MigrationName       string `json:"migration_name,omitempty"`
	MigrationChecksum   string `json:"migration_checksum,omitempty"`
}

// ReadAuthoritySnapshot is immutable metadata for a complete snapshot.
type ReadAuthoritySnapshot struct {
	Version           string
	SnapshotID        string
	Consumer          string
	ContractVersion   string
	SourceCursor      string
	ChecksumAlgorithm string
	Checksum          string
	CreatedAt         time.Time
	ExpiresAt         time.Time
	RecordCount       int
	PageSize          int
	Complete          bool
}

type ReadAuthoritySnapshotPage struct {
	Snapshot   ReadAuthoritySnapshot
	Records    []ReadAuthorityRecord
	NextCursor string
	Complete   bool
}
