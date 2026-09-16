package pages

import "github.com/Onellan/tockrplatform/internal/domain"

type OrganisationOption struct {
	ID       string
	Name     string
	Role     string
	Selected bool
}

type WorkspaceOption struct {
	ID       string
	Name     string
	Selected bool
}

type ProductOption struct {
	Key         string
	DisplayName string
	AccessURL   string
}

type LauncherProps struct {
	User             domain.User
	CSRF             string
	CurrentPath      string
	Organisations    []OrganisationOption
	Workspaces       []WorkspaceOption
	Products         []ProductOption
	OrganisationID   string
	OrganisationName string
	WorkspaceID      string
	WorkspaceName    string
}

type WorkspaceSelectorProps struct {
	User             domain.User
	CSRF             string
	CurrentPath      string
	OrganisationID   string
	OrganisationName string
	Workspaces       []WorkspaceOption
}

type ProductAccessProps struct {
	User             domain.User
	CSRF             string
	CurrentPath      string
	Product          ProductOption
	OrganisationID   string
	OrganisationName string
	WorkspaceID      string
	WorkspaceName    string
}
