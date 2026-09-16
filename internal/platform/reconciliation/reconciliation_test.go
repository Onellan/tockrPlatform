package reconciliation

import (
	"bytes"
	"strings"
	"testing"
)

const testVersion = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestBuildIsRepeatableAndUsesCanonicalKeysNotSourceIDs(t *testing.T) {
	first := Inventory{Records: []SourceRecord{{
		Source: SourceCTRL, SourceVersion: testVersion, Entity: EntityUser,
		SourceID: "ctrl-user-1", MatchKey: "user-canonical-1", Active: true,
	}}}
	second := Inventory{Records: append([]SourceRecord(nil), first.Records...)}
	second.Records[0].SourceID = "ctrl-user-renumbered"

	firstBytes, err := MarshalReport(Build(first))
	if err != nil {
		t.Fatal(err)
	}
	secondBytes, err := MarshalReport(Build(second))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(firstBytes, secondBytes) {
		t.Fatal("report must retain changed source provenance")
	}
	firstReport := Build(first)
	secondReport := Build(second)
	if firstReport.Proposals[0].PlatformID != secondReport.Proposals[0].PlatformID {
		t.Fatalf("source ID changed candidate: %q != %q", firstReport.Proposals[0].PlatformID, secondReport.Proposals[0].PlatformID)
	}
	if firstReport.Proposals[0].Status != StatusProposed {
		t.Fatalf("status = %q, want proposed", firstReport.Proposals[0].Status)
	}
	if strings.Contains(string(firstBytes), "user-canonical-1") {
		t.Fatal("report leaked the raw canonical match key")
	}
}

func TestBuildMergesExactCrossSourceIdentityAndPreservesProvenance(t *testing.T) {
	report := Build(Inventory{Records: []SourceRecord{
		{Source: SourceCTRL, SourceVersion: testVersion, Entity: EntityUser, SourceID: "ctrl-1", MatchKey: "user-1"},
		{Source: SourceIMS, SourceVersion: testVersion, Entity: EntityUser, SourceID: "ims-9", MatchKey: "user-1"},
	}})
	if len(report.Proposals) != 1 || report.Summary.ProposalCount != 1 {
		t.Fatalf("proposals = %#v, summary = %#v", report.Proposals, report.Summary)
	}
	proposal := report.Proposals[0]
	if len(proposal.SourceRefs) != 2 {
		t.Fatalf("source refs = %#v", proposal.SourceRefs)
	}
	if proposal.Status != StatusProposed || !proposal.ReviewRequired {
		t.Fatalf("proposal = %#v", proposal)
	}
}

func TestBuildBlocksAmbiguousAndCollidingIdentity(t *testing.T) {
	report := Build(Inventory{Records: []SourceRecord{
		{Source: SourceCTRL, SourceVersion: testVersion, Entity: EntityUser, SourceID: "ctrl-1", MatchKey: "user-1"},
		{Source: SourceIMS, SourceVersion: testVersion, Entity: EntityUser, SourceID: "ims-1", MatchKey: "user-1"},
		{Source: SourceCTRL, SourceVersion: testVersion, Entity: EntityUser, SourceID: "ctrl-1", MatchKey: "user-2"},
	}})
	if report.Summary.CollisionCount != 2 {
		t.Fatalf("collision count = %d, want 2", report.Summary.CollisionCount)
	}
	for _, proposal := range report.Proposals {
		if proposal.Status != StatusCollision {
			t.Fatalf("proposal status = %q, want collision: %#v", proposal.Status, proposal)
		}
	}

	ambiguous := Build(Inventory{Records: []SourceRecord{
		{Source: SourceCTRL, SourceVersion: testVersion, Entity: EntityOrganisationMembership, SourceID: "ctrl-m", MatchKey: "membership-1", ParentSourceID: "ctrl-o", ParentMatchKey: "org-1", UserSourceID: "ctrl-u", UserMatchKey: "user-1", Role: "admin"},
		{Source: SourceIMS, SourceVersion: testVersion, Entity: EntityOrganisationMembership, SourceID: "ims-m", MatchKey: "membership-1", ParentSourceID: "ims-o", ParentMatchKey: "org-1", UserSourceID: "ims-u", UserMatchKey: "user-1", Role: "member"},
	}})
	if ambiguous.Summary.AmbiguousCount != 1 || ambiguous.Proposals[0].Status != StatusAmbiguous {
		t.Fatalf("ambiguous report = %#v", ambiguous)
	}
}

func TestBuildPropagatesSourceIdentityCollisionAcrossMergedSources(t *testing.T) {
	report := Build(Inventory{Records: []SourceRecord{
		{Source: SourceIMS, SourceVersion: testVersion, Entity: EntityUser, SourceID: "ims-1", MatchKey: "user-1"},
		{Source: SourceCTRL, SourceVersion: testVersion, Entity: EntityUser, SourceID: "ctrl-1", MatchKey: "user-1"},
		{Source: SourceCTRL, SourceVersion: testVersion, Entity: EntityUser, SourceID: "ctrl-1", MatchKey: "user-2"},
	}})
	if len(report.Proposals) != 2 || report.Summary.CollisionCount != 2 {
		t.Fatalf("report = %#v", report)
	}
	for _, proposal := range report.Proposals {
		if proposal.Status != StatusCollision {
			t.Fatalf("proposal status = %q, want collision: %#v", proposal.Status, proposal)
		}
	}
}

func TestBuildBlocksMissingProvenanceAndUnresolvedDependencies(t *testing.T) {
	report := Build(Inventory{Records: []SourceRecord{
		{Source: SourceCTRL, SourceVersion: "not-a-sha", Entity: EntityUser, SourceID: "ctrl-u", MatchKey: "user-1"},
		{Source: SourceCTRL, SourceVersion: testVersion, Entity: EntityWorkspace, SourceID: "ctrl-w", MatchKey: "workspace-1", ParentSourceID: "ctrl-o", ParentMatchKey: "org-missing"},
		{Source: SourceCTRL, SourceVersion: testVersion, Entity: EntityOrganisationMembership, SourceID: "ctrl-m", MatchKey: "membership-1", ParentSourceID: "ctrl-o", ParentMatchKey: "org-missing", UserSourceID: "ctrl-u", UserMatchKey: "user-missing", Role: "member"},
	}})
	if report.Summary.BlockedCount != 3 {
		t.Fatalf("blocked count = %d, want 3: %#v", report.Summary.BlockedCount, report.Proposals)
	}
	for _, proposal := range report.Proposals {
		if proposal.Status != StatusBlocked || proposal.PlatformID != "" {
			t.Fatalf("blocked proposal = %#v", proposal)
		}
	}
}

func TestBuildReportsMissingRelationshipFieldsWithoutGuessing(t *testing.T) {
	report := Build(Inventory{Records: []SourceRecord{{
		Source: SourceCTRL, SourceVersion: testVersion, Entity: EntityWorkspace,
		SourceID: "ctrl-w", MatchKey: "workspace-1",
	}}})
	if len(report.Proposals) != 1 || report.Proposals[0].Reason != "parent_identity_reference_required" {
		t.Fatalf("report = %#v", report)
	}
}

func TestBuildBlocksRelationshipWhenSameSourceProvenanceIsMissing(t *testing.T) {
	report := Build(Inventory{Records: []SourceRecord{
		{Source: SourceCTRL, SourceVersion: testVersion, Entity: EntityUser, SourceID: "ctrl-user-1", MatchKey: "user-1"},
		{Source: SourceCTRL, SourceVersion: testVersion, Entity: EntityOrganisation, SourceID: "ctrl-org-1", MatchKey: "org-1"},
		{Source: SourceCTRL, SourceVersion: testVersion, Entity: EntityOrganisationMembership, SourceID: "ctrl-membership-1", MatchKey: "membership-1", ParentSourceID: "ctrl-org-missing", ParentMatchKey: "org-1", UserSourceID: "ctrl-user-1", UserMatchKey: "user-1", Role: "member"},
	}})
	if report.Summary.ProposalCount != 2 || report.Summary.BlockedCount != 1 {
		t.Fatalf("report summary = %#v, proposals = %#v", report.Summary, report.Proposals)
	}
	for _, proposal := range report.Proposals {
		if proposal.Entity == EntityOrganisationMembership && (proposal.Status != StatusBlocked || proposal.PlatformID != "") {
			t.Fatalf("membership proposal = %#v", proposal)
		}
	}
}
