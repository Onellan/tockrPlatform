package pages

import (
	"github.com/Onellan/tockrplatform/internal/domain"
)

type AdminNavItem struct {
	Href   string
	Label  string
	Active bool
}

type AdminMember struct {
	MembershipID string
	UserID       string
	Email        string
	DisplayName  string
	Role         string
	RoleChange   string
	AssignedAt   string
}

type AdminWorkspace struct {
	ID        string
	Name      string
	Status    string
	CreatedAt string
	AdminURL  string
}

type AdminAudit struct {
	Event      string
	Details    string
	OccurredAt string
}

type AdminEntitlement struct {
	ID         string
	ProductKey string
	Status     string
	Active     bool
}

type AdminAssignment struct {
	ID         string
	UserID     string
	ProductKey string
	Status     string
	Active     bool
}

type AdminProduct struct {
	Key         string
	DisplayName string
	Status      string
	Active      bool
}

type OrganisationAdminProps struct {
	User             domain.User
	CSRF             string
	CurrentPath      string
	OrganisationID   string
	OrganisationName string
	Role             string
	Section          string
	CanAdmin         bool
	SystemAdmin      bool
	Navigation       []AdminNavItem
	Members          []AdminMember
	Workspaces       []AdminWorkspace
	Audit            []AdminAudit
	Entitlements     []AdminEntitlement
	Assignments      []AdminAssignment
	Flash            string
}

type WorkspaceAdminProps struct {
	User             domain.User
	CSRF             string
	CurrentPath      string
	OrganisationID   string
	OrganisationName string
	WorkspaceID      string
	WorkspaceName    string
	OrganisationRole string
	WorkspaceRole    string
	CanAdmin         bool
	Members          []AdminMember
	Audit            []AdminAudit
	Flash            string
}

type SystemProductAdminProps struct {
	User        domain.User
	CSRF        string
	CurrentPath string
	Products    []AdminProduct
	Flash       string
}
