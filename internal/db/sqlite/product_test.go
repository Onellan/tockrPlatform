package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
)

func TestProductCatalogueAndOrganisationEntitlementAreIndependentAndAudited(t *testing.T) {
	ctx := context.Background()
	store, users := newOrganisationStore(t, 5)
	now := time.Date(2026, 9, 16, 9, 0, 0, 0, time.UTC)
	organisationID, err := domain.NewOrganisationID()
	if err != nil {
		t.Fatal(err)
	}
	organisation, _, err := store.CreateOrganisation(ctx, users[0].ID, domain.Organisation{ID: organisationID, Name: "Product Organisation"}, "create product organisation", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[1].ID, domain.OrganisationAdmin, "appoint product administrator", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[2].ID, domain.OrganisationMember, "add product member", now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := seedSystemAdministrator(ctx, store, users[4].ID); err != nil {
		t.Fatal(err)
	}

	products, err := store.ListProducts(ctx, users[4].ID)
	if err != nil || len(products) != 2 || products[0].Key != "product.tockrctrl" || products[1].Key != "product.tockrims" {
		t.Fatalf("initial product catalogue = %#v, err=%v", products, err)
	}
	if _, err := store.ListProducts(ctx, users[0].ID); !errors.Is(err, ErrUnauthorisedProductAction) {
		t.Fatalf("non-system product catalogue read = %v, want unauthorised", err)
	}

	var membershipsBefore int
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM organisation_memberships WHERE organisation_id=(SELECT id FROM organisations WHERE public_id=?)`, organisation.ID).Scan(&membershipsBefore); err != nil {
		t.Fatal(err)
	}
	entitlement, err := store.EntitleOrganisation(ctx, users[0].ID, organisation.ID, "product.tockrctrl", "enable CTRL product", now.Add(3*time.Minute))
	if err != nil || entitlement.Status != domain.OrganisationProductEntitlementActive || !entitlement.Active {
		t.Fatalf("owner entitlement = %#v, err=%v", entitlement, err)
	}
	if _, err := store.EntitleOrganisation(ctx, users[0].ID, organisation.ID, "product.tockrctrl", "duplicate entitlement", now.Add(4*time.Minute)); !errors.Is(err, ErrDuplicateOrganisationEntitlement) {
		t.Fatalf("duplicate entitlement = %v, want duplicate", err)
	}
	if _, err := store.EntitleOrganisation(ctx, users[2].ID, organisation.ID, "product.tockrims", "member entitlement", now.Add(5*time.Minute)); !errors.Is(err, ErrUnauthorisedOrganisationAction) {
		t.Fatalf("member entitlement = %v, want unauthorised", err)
	}
	if _, err := store.EntitleOrganisation(ctx, users[3].ID, organisation.ID, "product.tockrims", "outsider entitlement", now.Add(6*time.Minute)); !errors.Is(err, ErrUnauthorisedOrganisationAction) {
		t.Fatalf("outsider entitlement = %v, want unauthorised", err)
	}
	var membershipsAfter int
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM organisation_memberships WHERE organisation_id=(SELECT id FROM organisations WHERE public_id=?)`, organisation.ID).Scan(&membershipsAfter); err != nil {
		t.Fatal(err)
	}
	if membershipsAfter != membershipsBefore {
		t.Fatalf("entitlement changed membership count from %d to %d", membershipsBefore, membershipsAfter)
	}
	var assignmentRows int
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM user_product_assignments`).Scan(&assignmentRows); err != nil {
		t.Fatal(err)
	}
	if assignmentRows != 0 {
		t.Fatalf("entitlement auto-created %d user product assignments", assignmentRows)
	}

	entitlements, err := store.ListOrganisationProductEntitlements(ctx, users[1].ID, organisation.ID)
	if err != nil || len(entitlements) != 1 || entitlements[0].Active != true {
		t.Fatalf("admin entitlement read = %#v, err=%v", entitlements, err)
	}
	if _, err := store.ListOrganisationProductEntitlements(ctx, users[2].ID, organisation.ID); !errors.Is(err, ErrUnauthorisedOrganisationAction) {
		t.Fatalf("member entitlement read = %v, want unauthorised", err)
	}
	if err := store.RevokeOrganisationEntitlement(ctx, users[1].ID, organisation.ID, entitlement.ID, "disable CTRL product", now.Add(7*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeOrganisationEntitlement(ctx, users[1].ID, organisation.ID, entitlement.ID, "repeat disable", now.Add(8*time.Minute)); !errors.Is(err, ErrEntitlementInactive) {
		t.Fatalf("repeat entitlement revocation = %v, want inactive", err)
	}
	second, err := store.EntitleOrganisation(ctx, users[4].ID, organisation.ID, "product.tockrctrl", "restore CTRL product", now.Add(9*time.Minute))
	if err != nil || second.ID == entitlement.ID {
		t.Fatalf("restored entitlement = %#v, err=%v", second, err)
	}
	entitlements, err = store.ListOrganisationProductEntitlements(ctx, users[4].ID, organisation.ID)
	if err != nil || len(entitlements) != 2 || entitlements[0].Status != domain.OrganisationProductEntitlementRevoked || entitlements[0].Active || entitlements[1].Status != domain.OrganisationProductEntitlementActive || !entitlements[1].Active {
		t.Fatalf("entitlement history = %#v, err=%v", entitlements, err)
	}
	for _, event := range []string{eventEntitlementGranted, eventEntitlementRevoked} {
		count, err := store.organisationAuditCount(ctx, organisation.ID, event)
		if err != nil || count == 0 {
			t.Fatalf("entitlement audit %q count = %d, err=%v", event, count, err)
		}
	}

	if err := store.RetireProduct(ctx, users[4].ID, "product.tockrims", "retire unused product", now.Add(10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.EntitleOrganisation(ctx, users[0].ID, organisation.ID, "product.tockrims", "retired product", now.Add(11*time.Minute)); !errors.Is(err, ErrProductRetired) {
		t.Fatalf("retired product entitlement = %v, want retired", err)
	}
	products, err = store.ListProducts(ctx, users[4].ID)
	if err != nil || products[1].Status != domain.ProductRetired || products[1].RetiredAt == nil {
		t.Fatalf("retired product catalogue = %#v, err=%v", products, err)
	}
}

func TestProductMigrationSeedsCatalogueOnFreshUpgradeAndReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "product.db")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	assertProductCatalogue(t, store)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	assertProductCatalogue(t, reopened)
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}

	legacyPath := filepath.Join(t.TempDir(), "legacy.db")
	legacy, err := sql.Open("sqlite", legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.ExecContext(ctx, `CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, checksum TEXT NOT NULL, applied_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	for _, migration := range supportedMigrations()[:5] {
		for _, statement := range migration.statements {
			if _, err := legacy.ExecContext(ctx, statement); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := legacy.ExecContext(ctx, `INSERT INTO schema_migrations(version,name,checksum,applied_at) VALUES(?,?,?,?)`, migration.version, migration.name, migrationChecksum(migration), formatTime(time.Now().UTC())); err != nil {
			t.Fatal(err)
		}
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	upgraded, err := Open(ctx, legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Close()
	assertProductCatalogue(t, upgraded)
}

func assertProductCatalogue(t *testing.T, store *Store) {
	t.Helper()
	var count int
	if err := store.DB().QueryRowContext(context.Background(), `SELECT COUNT(*) FROM products`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("product count = %d, want 2", count)
	}
	var ctrlStatus, imsStatus string
	if err := store.DB().QueryRowContext(context.Background(), `SELECT status FROM products WHERE product_key='product.tockrctrl'`).Scan(&ctrlStatus); err != nil {
		t.Fatal(err)
	}
	if err := store.DB().QueryRowContext(context.Background(), `SELECT status FROM products WHERE product_key='product.tockrims'`).Scan(&imsStatus); err != nil {
		t.Fatal(err)
	}
	if ctrlStatus != "active" || imsStatus != "active" {
		t.Fatalf("initial product statuses = %q/%q", ctrlStatus, imsStatus)
	}
	var assignmentTable int
	if err := store.DB().QueryRowContext(context.Background(), `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='user_product_assignments'`).Scan(&assignmentTable); err != nil {
		t.Fatal(err)
	}
	if assignmentTable != 1 {
		t.Fatalf("assignment table count = %d, want 1 after PF-B5-S02 migration", assignmentTable)
	}
}
