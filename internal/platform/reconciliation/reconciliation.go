// Package reconciliation contains read-only, deterministic identity mapping
// proposals for Platform migration preparation. It deliberately has no
// database or product-repository dependency: source adapters supply an
// explicitly canonical match key and this package reports what still needs
// human review.
package reconciliation

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

const (
	SchemaVersion = 1

	SourceCTRL SourceSystem = "ctrl"
	SourceIMS  SourceSystem = "ims"

	EntityUser                   EntityKind = "user"
	EntityOrganisation           EntityKind = "organisation"
	EntityWorkspace              EntityKind = "workspace"
	EntityOrganisationMembership EntityKind = "organisation_membership"
	EntityWorkspaceMembership    EntityKind = "workspace_membership"

	StatusProposed  ProposalStatus = "proposed"
	StatusBlocked   ProposalStatus = "blocked"
	StatusAmbiguous ProposalStatus = "ambiguous"
	StatusCollision ProposalStatus = "collision"
)

// SourceSystem identifies the source repository represented by an adapter.
// Source records are evidence only; this package never opens either source.
type SourceSystem string

// EntityKind identifies one Platform-owned identity or relationship shape.
type EntityKind string

// ProposalStatus is intentionally not an acceptance or import decision.
type ProposalStatus string

// Inventory is the adapter boundary. MatchKey must be supplied by the
// adapter from an explicitly authorized canonical identity rule. The engine
// never derives one from SourceID.
type Inventory struct {
	Records []SourceRecord `json:"records"`
}

// SourceRecord is a normalized, non-mutating source observation. MatchKey is
// retained only in memory and represented by a digest in the report so a
// dry-run report does not unnecessarily repeat identity material.
type SourceRecord struct {
	Source         SourceSystem `json:"source"`
	SourceVersion  string       `json:"source_version"`
	Entity         EntityKind   `json:"entity"`
	SourceID       string       `json:"source_id"`
	MatchKey       string       `json:"match_key"`
	ParentSourceID string       `json:"parent_source_id,omitempty"`
	ParentMatchKey string       `json:"parent_match_key,omitempty"`
	UserSourceID   string       `json:"user_source_id,omitempty"`
	UserMatchKey   string       `json:"user_match_key,omitempty"`
	Role           string       `json:"role,omitempty"`
	Active         bool         `json:"active"`
}

// SourceRef preserves exact source/version/identity provenance for a proposal.
type SourceRef struct {
	Source        SourceSystem `json:"source"`
	SourceVersion string       `json:"source_version"`
	SourceID      string       `json:"source_id"`
}

// Proposal is a deterministic candidate, never an accepted mapping. A
// consumer must explicitly review and approve a proposed result in a later
// authorized workflow.
type Proposal struct {
	Entity         EntityKind     `json:"entity"`
	PlatformID     string         `json:"platform_id,omitempty"`
	Status         ProposalStatus `json:"status"`
	Confidence     string         `json:"confidence,omitempty"`
	MatchKeySHA256 string         `json:"match_key_sha256,omitempty"`
	Reason         string         `json:"reason"`
	ReviewRequired bool           `json:"review_required"`
	SourceRefs     []SourceRef    `json:"source_refs"`
}

// Summary counts proposal groups, not raw source records.
type Summary struct {
	RecordCount    int `json:"record_count"`
	ProposalCount  int `json:"proposal_count"`
	BlockedCount   int `json:"blocked_count"`
	AmbiguousCount int `json:"ambiguous_count"`
	CollisionCount int `json:"collision_count"`
}

// Report is intentionally timestamp-free so equal inputs produce equal bytes.
type Report struct {
	SchemaVersion int        `json:"schema_version"`
	ReportType    string     `json:"report_type"`
	Summary       Summary    `json:"summary"`
	Proposals     []Proposal `json:"proposals"`
}

type normalizedRecord struct {
	record SourceRecord
	index  int
	valid  bool
	reason string
}

type proposalGroup struct {
	key        string
	entity     EntityKind
	matchKey   string
	records    []normalizedRecord
	base       ProposalStatus
	reason     string
	platformID string
}

// Build creates a deterministic, read-only proposal report. Invalid records,
// source conflicts and unresolved relationship dependencies remain visible as
// blocked/ambiguous/collision proposals; none is silently dropped.
func Build(inventory Inventory) Report {
	groups := make(map[string]*proposalGroup)
	sourceIdentityGroups := make(map[string]map[string]struct{})

	for index, raw := range inventory.Records {
		record := normalizeRecord(raw)
		if !record.valid {
			key := "invalid\x00" + string(rune(index))
			groups[key] = &proposalGroup{key: key, entity: record.record.Entity, records: []normalizedRecord{record}, base: StatusBlocked, reason: record.reason}
			continue
		}

		key := groupKey(record.record.Entity, record.record.MatchKey)
		group := groups[key]
		if group == nil {
			group = &proposalGroup{key: key, entity: record.record.Entity, matchKey: record.record.MatchKey, base: StatusProposed}
			groups[key] = group
		}
		group.records = append(group.records, record)

		sourceKey := sourceIdentityKey(record.record)
		if sourceIdentityGroups[sourceKey] == nil {
			sourceIdentityGroups[sourceKey] = make(map[string]struct{})
		}
		sourceIdentityGroups[sourceKey][key] = struct{}{}
	}

	for _, group := range groups {
		if group.base == StatusBlocked {
			continue
		}
		fingerprints := make(map[string]struct{})
		sources := make(map[string]struct{})
		for _, record := range group.records {
			fingerprints[fingerprint(record.record)] = struct{}{}
			sources[string(record.record.Source)] = struct{}{}
		}
		if len(fingerprints) > 1 {
			group.base = StatusAmbiguous
			group.reason = "canonical_identity_conflict_requires_review"
		} else if len(sources) < len(group.records) {
			group.base = StatusCollision
			group.reason = "multiple_records_from_one_source_require_review"
		}
		if group.base == StatusProposed {
			for sourceKey, groupKeys := range sourceIdentityGroups {
				if !strings.HasPrefix(sourceKey, string(group.records[0].record.Source)+"\x00"+string(group.entity)+"\x00") {
					continue
				}
				if _, present := groupKeys[group.key]; present && len(groupKeys) > 1 {
					group.base = StatusCollision
					group.reason = "source_identity_maps_to_multiple_canonical_keys"
					break
				}
			}
		}
		if group.base == StatusProposed {
			group.platformID = stablePlatformID(group.entity, group.matchKey)
		}
	}

	ordered := make([]*proposalGroup, 0, len(groups))
	for _, group := range groups {
		ordered = append(ordered, group)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].key < ordered[j].key })

	report := Report{
		SchemaVersion: SchemaVersion,
		ReportType:    "platform.reconciliation.inventory.v1",
		Summary:       Summary{RecordCount: len(inventory.Records)},
		Proposals:     make([]Proposal, 0, len(ordered)),
	}
	for _, group := range ordered {
		status, reason := group.base, group.reason
		if status == StatusProposed && !dependenciesResolved(group, groups) {
			status = StatusBlocked
			reason = "related_identity_mapping_unresolved"
		}
		proposal := Proposal{
			Entity:         group.entity,
			Status:         status,
			Reason:         reason,
			ReviewRequired: true,
			SourceRefs:     sourceRefs(group.records),
		}
		if status == StatusProposed {
			proposal.PlatformID = group.platformID
		}
		if group.matchKey != "" {
			proposal.MatchKeySHA256 = digest(group.matchKey)
		}
		if status == StatusProposed {
			proposal.Confidence = "exact-canonical-key"
			report.Summary.ProposalCount++
		} else {
			switch status {
			case StatusBlocked:
				report.Summary.BlockedCount++
			case StatusAmbiguous:
				report.Summary.AmbiguousCount++
			case StatusCollision:
				report.Summary.CollisionCount++
			}
		}
		report.Proposals = append(report.Proposals, proposal)
	}
	return report
}

// MarshalReport emits stable, review-friendly JSON. It contains no raw
// canonical match keys and no generated-at field, making repeatability
// comparisons meaningful.
func MarshalReport(report Report) ([]byte, error) {
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

func normalizeRecord(raw SourceRecord) normalizedRecord {
	record := raw
	record.Source = SourceSystem(strings.ToLower(strings.TrimSpace(string(raw.Source))))
	record.SourceVersion = strings.ToLower(strings.TrimSpace(raw.SourceVersion))
	record.Entity = EntityKind(strings.ToLower(strings.TrimSpace(string(raw.Entity))))
	record.SourceID = strings.TrimSpace(raw.SourceID)
	record.MatchKey = strings.TrimSpace(raw.MatchKey)
	record.ParentSourceID = strings.TrimSpace(raw.ParentSourceID)
	record.ParentMatchKey = strings.TrimSpace(raw.ParentMatchKey)
	record.UserSourceID = strings.TrimSpace(raw.UserSourceID)
	record.UserMatchKey = strings.TrimSpace(raw.UserMatchKey)
	record.Role = strings.TrimSpace(raw.Role)

	reason := ""
	switch {
	case record.Source != SourceCTRL && record.Source != SourceIMS:
		reason = "unsupported_source"
	case !isSourceSHA(record.SourceVersion):
		reason = "source_version_provenance_required"
	case !validEntity(record.Entity):
		reason = "unsupported_entity"
	case record.SourceID == "":
		reason = "source_identity_required"
	case record.MatchKey == "":
		reason = "canonical_match_key_required"
	case requiresParent(record.Entity) && (record.ParentSourceID == "" || record.ParentMatchKey == ""):
		reason = "parent_identity_reference_required"
	case requiresUser(record.Entity) && (record.UserSourceID == "" || record.UserMatchKey == ""):
		reason = "user_identity_reference_required"
	case requiresRole(record.Entity) && record.Role == "":
		reason = "membership_role_required"
	}
	return normalizedRecord{record: record, valid: reason == "", reason: reason}
}

func validEntity(entity EntityKind) bool {
	switch entity {
	case EntityUser, EntityOrganisation, EntityWorkspace, EntityOrganisationMembership, EntityWorkspaceMembership:
		return true
	default:
		return false
	}
}

func requiresParent(entity EntityKind) bool {
	return entity == EntityWorkspace || entity == EntityOrganisationMembership || entity == EntityWorkspaceMembership
}

func requiresUser(entity EntityKind) bool {
	return entity == EntityOrganisationMembership || entity == EntityWorkspaceMembership
}

func requiresRole(entity EntityKind) bool {
	return entity == EntityOrganisationMembership || entity == EntityWorkspaceMembership
}

func isSourceSHA(value string) bool {
	if len(value) != 40 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func groupKey(entity EntityKind, matchKey string) string {
	return string(entity) + "\x00" + matchKey
}

func sourceIdentityKey(record SourceRecord) string {
	return string(record.Source) + "\x00" + string(record.Entity) + "\x00" + record.SourceID
}

func fingerprint(record SourceRecord) string {
	value := strings.Join([]string{string(record.Entity), record.MatchKey, record.ParentMatchKey, record.UserMatchKey, record.Role}, "\x00")
	return digest(value)
}

func stablePlatformID(entity EntityKind, matchKey string) string {
	sum := sha256.Sum256([]byte("tockrplatform/pf-b9-s01/platform-id/v1\x00" + string(entity) + "\x00" + matchKey))
	encoded := base64.RawURLEncoding.EncodeToString(sum[:18])
	prefix := map[EntityKind]string{
		EntityUser:                   "usr_",
		EntityOrganisation:           "org_",
		EntityWorkspace:              "wsp_",
		EntityOrganisationMembership: "omem_",
		EntityWorkspaceMembership:    "wmem_",
	}[entity]
	return prefix + encoded
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func dependenciesResolved(group *proposalGroup, groups map[string]*proposalGroup) bool {
	for _, record := range group.records {
		if requiresParent(record.record.Entity) {
			parentEntity := EntityOrganisation
			if record.record.Entity == EntityWorkspaceMembership {
				parentEntity = EntityWorkspace
			}
			parent, ok := groups[groupKey(parentEntity, record.record.ParentMatchKey)]
			if !ok || parent.base != StatusProposed {
				return false
			}
		}
		if requiresUser(record.record.Entity) {
			user, ok := groups[groupKey(EntityUser, record.record.UserMatchKey)]
			if !ok || user.base != StatusProposed {
				return false
			}
		}
	}
	return true
}

func sourceRefs(records []normalizedRecord) []SourceRef {
	refs := make([]SourceRef, 0, len(records))
	for _, record := range records {
		refs = append(refs, SourceRef{Source: record.record.Source, SourceVersion: record.record.SourceVersion, SourceID: record.record.SourceID})
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Source != refs[j].Source {
			return refs[i].Source < refs[j].Source
		}
		if refs[i].SourceVersion != refs[j].SourceVersion {
			return refs[i].SourceVersion < refs[j].SourceVersion
		}
		return refs[i].SourceID < refs[j].SourceID
	})
	return refs
}
