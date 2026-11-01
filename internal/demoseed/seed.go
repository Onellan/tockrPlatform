// Package demoseed creates the versioned local fixture shared by TockrCTRL
// and TockrIMS. It writes only canonical Platform identities and access facts;
// product records remain in their owning applications.
package demoseed

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Onellan/tockrplatform/internal/db/sqlite"
	"github.com/Onellan/tockrplatform/internal/domain"
)

const (
	FixtureVersion = 1
	FixtureName    = "tockr-local-demo-v1"
	Password       = "Tockr-Demo-2026!"
	seedReason     = "create local demo fixture v1"
)

type Manifest struct {
	Version      int                `json:"version"`
	Fixture      string             `json:"fixture"`
	Organisation OrganisationRecord `json:"organisation"`
	Workspace    WorkspaceRecord    `json:"workspace"`
	Products     []string           `json:"products"`
	Users        []UserRecord       `json:"users"`
}

type OrganisationRecord struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type WorkspaceRecord struct {
	ID             string `json:"id"`
	OrganisationID string `json:"organisation_id"`
	Name           string `json:"name"`
}

type UserRecord struct {
	ID               string `json:"id"`
	Email            string `json:"email"`
	DisplayName      string `json:"display_name"`
	OrganisationRole string `json:"organisation_role"`
	WorkspaceRole    string `json:"workspace_role"`
}

var manifest = Manifest{
	Version: FixtureVersion,
	Fixture: FixtureName,
	Organisation: OrganisationRecord{
		ID:   "org_tockr_demo",
		Name: "Default Organisation",
	},
	Workspace: WorkspaceRecord{
		ID:             "wsp_tockr_demo",
		OrganisationID: "org_tockr_demo",
		Name:           "Default Workspace",
	},
	Products: []string{"product.tockrctrl", "product.tockrims"},
	Users: []UserRecord{
		{ID: "usr_tockr_demo_admin", Email: "admin@example.com", DisplayName: "Tockr Demo Administrator", OrganisationRole: "owner", WorkspaceRole: "admin"},
		{ID: "usr_tockr_demo_anika", Email: "anika.demo@tockrctrl.local", DisplayName: "Anika Maseko", OrganisationRole: "member", WorkspaceRole: "viewer"},
		{ID: "usr_tockr_demo_sipho", Email: "sipho.demo@tockrctrl.local", DisplayName: "Sipho Dlamini", OrganisationRole: "member", WorkspaceRole: "member"},
		{ID: "usr_tockr_demo_maya", Email: "maya.demo@tockrctrl.local", DisplayName: "Maya Naidoo", OrganisationRole: "member", WorkspaceRole: "member"},
		{ID: "usr_tockr_demo_daniel", Email: "daniel.demo@tockrctrl.local", DisplayName: "Daniel Jacobs", OrganisationRole: "member", WorkspaceRole: "member"},
	},
}

// Seed installs the same stable Platform identity graph on every fresh local
// demo database. Existing fixture rows are accepted only when their identity
// and meaning still match this version.
func Seed(ctx context.Context, persistence *sqlite.Store) (Manifest, error) {
	if persistence == nil {
		return Manifest{}, errors.New("Platform demo seed store is required")
	}
	now := time.Now().UTC()
	for _, person := range manifest.Users {
		user, err := persistence.FindUserByID(ctx, person.ID)
		if err != nil {
			return Manifest{}, fmt.Errorf("find demo User %s: %w", person.ID, err)
		}
		if user == nil {
			byEmail, err := persistence.FindUserByEmail(ctx, person.Email)
			if err != nil {
				return Manifest{}, fmt.Errorf("find demo User by email: %w", err)
			}
			if byEmail != nil {
				return Manifest{}, fmt.Errorf("demo email %s already belongs to Platform User %s", person.Email, byEmail.ID)
			}
			if _, err := persistence.CreateUser(ctx, domain.User{ID: person.ID, Email: person.Email, DisplayName: person.DisplayName, Active: true, CreatedAt: now}, Password); err != nil {
				return Manifest{}, fmt.Errorf("create demo User %s: %w", person.ID, err)
			}
			continue
		}
		if user.Email != person.Email || user.DisplayName != person.DisplayName || !user.Active {
			return Manifest{}, fmt.Errorf("Platform User %s conflicts with demo fixture v1", person.ID)
		}
	}

	admin := manifest.Users[0]
	organisation, err := persistence.GetOrganisation(ctx, admin.ID, manifest.Organisation.ID)
	if err != nil && !errors.Is(err, sqlite.ErrOrganisationNotFound) {
		return Manifest{}, fmt.Errorf("read demo Organisation: %w", err)
	}
	if errors.Is(err, sqlite.ErrOrganisationNotFound) {
		created, _, err := persistence.CreateOrganisation(ctx, admin.ID, domain.Organisation{ID: manifest.Organisation.ID, Name: manifest.Organisation.Name}, seedReason, now)
		if err != nil {
			return Manifest{}, fmt.Errorf("create demo Organisation: %w", err)
		}
		organisation = &created
	}
	if organisation.Name != manifest.Organisation.Name || organisation.Status != domain.OrganisationActive {
		return Manifest{}, fmt.Errorf("Platform Organisation %s conflicts with demo fixture v1", organisation.ID)
	}

	workspace, err := persistence.GetWorkspace(ctx, admin.ID, manifest.Workspace.ID)
	if err != nil && !errors.Is(err, sqlite.ErrWorkspaceNotFound) {
		return Manifest{}, fmt.Errorf("read demo Workspace: %w", err)
	}
	if errors.Is(err, sqlite.ErrWorkspaceNotFound) {
		created, _, err := persistence.CreateWorkspace(ctx, admin.ID, organisation.ID, domain.Workspace{ID: manifest.Workspace.ID, Name: manifest.Workspace.Name}, seedReason, now)
		if err != nil {
			return Manifest{}, fmt.Errorf("create demo Workspace: %w", err)
		}
		workspace = &created
	}
	if workspace.Name != manifest.Workspace.Name || workspace.OrganisationID != organisation.ID || workspace.Status != domain.WorkspaceActive {
		return Manifest{}, fmt.Errorf("Platform Workspace %s conflicts with demo fixture v1", workspace.ID)
	}

	for _, person := range manifest.Users[1:] {
		if err := ensureOrganisationMembership(ctx, persistence, admin.ID, organisation.ID, person, now); err != nil {
			return Manifest{}, err
		}
		if err := ensureWorkspaceMembership(ctx, persistence, admin.ID, workspace.ID, person, now); err != nil {
			return Manifest{}, err
		}
	}
	for _, productKey := range manifest.Products {
		if err := ensureEntitlement(ctx, persistence, admin.ID, organisation.ID, productKey, now); err != nil {
			return Manifest{}, err
		}
		for _, person := range manifest.Users {
			if err := ensureAssignment(ctx, persistence, admin.ID, organisation.ID, person.ID, productKey, now); err != nil {
				return Manifest{}, err
			}
		}
	}

	result := manifest
	result.Products = append([]string(nil), manifest.Products...)
	result.Users = append([]UserRecord(nil), manifest.Users...)
	return result, nil
}

func ensureOrganisationMembership(ctx context.Context, persistence *sqlite.Store, adminID, organisationID string, person UserRecord, now time.Time) error {
	role := domain.OrganisationRole(person.OrganisationRole)
	membership, err := persistence.GetOrganisationMembership(ctx, adminID, organisationID, person.ID)
	if err != nil && !errors.Is(err, sqlite.ErrMembershipNotFound) {
		return fmt.Errorf("read Organisation membership for %s: %w", person.ID, err)
	}
	if errors.Is(err, sqlite.ErrMembershipNotFound) {
		if _, err := persistence.AddOrganisationMember(ctx, adminID, organisationID, person.ID, role, seedReason, now); err != nil {
			return fmt.Errorf("add Organisation membership for %s: %w", person.ID, err)
		}
		return nil
	}
	if !membership.Active {
		return fmt.Errorf("Organisation membership for %s is inactive", person.ID)
	}
	if membership.Role != role {
		if _, err := persistence.ChangeOrganisationMemberRole(ctx, adminID, organisationID, person.ID, role, seedReason, now); err != nil {
			return fmt.Errorf("align Organisation role for %s: %w", person.ID, err)
		}
	}
	return nil
}

func ensureWorkspaceMembership(ctx context.Context, persistence *sqlite.Store, adminID, workspaceID string, person UserRecord, now time.Time) error {
	role := domain.WorkspaceRole(person.WorkspaceRole)
	membership, err := persistence.GetWorkspaceMembership(ctx, adminID, workspaceID, person.ID)
	if err != nil && !errors.Is(err, sqlite.ErrWorkspaceMembershipNotFound) {
		return fmt.Errorf("read Workspace membership for %s: %w", person.ID, err)
	}
	if errors.Is(err, sqlite.ErrWorkspaceMembershipNotFound) {
		if _, err := persistence.AddWorkspaceMember(ctx, adminID, workspaceID, person.ID, role, seedReason, now); err != nil {
			return fmt.Errorf("add Workspace membership for %s: %w", person.ID, err)
		}
		return nil
	}
	if !membership.Active {
		return fmt.Errorf("Workspace membership for %s is inactive", person.ID)
	}
	if membership.Role != role {
		if _, err := persistence.ChangeWorkspaceMemberRole(ctx, adminID, workspaceID, person.ID, role, seedReason, now); err != nil {
			return fmt.Errorf("align Workspace role for %s: %w", person.ID, err)
		}
	}
	return nil
}

func ensureEntitlement(ctx context.Context, persistence *sqlite.Store, adminID, organisationID, productKey string, now time.Time) error {
	items, err := persistence.ListOrganisationProductEntitlements(ctx, adminID, organisationID)
	if err != nil {
		return fmt.Errorf("read demo product entitlements: %w", err)
	}
	for _, item := range items {
		if item.ProductKey == productKey && item.Active {
			return nil
		}
	}
	if _, err := persistence.EntitleOrganisation(ctx, adminID, organisationID, productKey, seedReason, now); err != nil {
		return fmt.Errorf("entitle demo product %s: %w", productKey, err)
	}
	return nil
}

func ensureAssignment(ctx context.Context, persistence *sqlite.Store, adminID, organisationID, userID, productKey string, now time.Time) error {
	items, err := persistence.ListUserProductAssignments(ctx, adminID, organisationID)
	if err != nil {
		return fmt.Errorf("read demo product assignments: %w", err)
	}
	for _, item := range items {
		if item.UserID == userID && item.ProductKey == productKey && item.Active {
			return nil
		}
	}
	if _, err := persistence.AssignUserProduct(ctx, adminID, organisationID, userID, productKey, seedReason, now); err != nil {
		return fmt.Errorf("assign demo product %s to %s: %w", productKey, userID, err)
	}
	return nil
}
