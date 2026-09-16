package httpserver

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/Onellan/tockrplatform/internal/domain"
	"github.com/Onellan/tockrplatform/internal/store"
	"github.com/Onellan/tockrplatform/web/pages"
	"github.com/go-chi/chi/v5"
)

func (s *Server) platformLauncher(w http.ResponseWriter, r *http.Request) {
	session, csrf, ok := s.uiSession(w, r)
	if !ok {
		return
	}
	props, err := s.launcherProps(r, session.User.ID, csrf)
	if err != nil {
		s.uiNotFoundOrError(w, err)
		return
	}
	if err := render(w, r, pages.Launcher(props)); err != nil {
		http.Error(w, "service unavailable", http.StatusInternalServerError)
	}
}

func (s *Server) workspaceSelector(w http.ResponseWriter, r *http.Request) {
	session, csrf, ok := s.uiSession(w, r)
	if !ok {
		return
	}
	organisationID := strings.TrimSpace(chi.URLParam(r, "organisationID"))
	organisation, err := s.store.GetOrganisation(r.Context(), session.User.ID, organisationID)
	if err != nil {
		s.uiNotFoundOrError(w, err)
		return
	}
	workspaces, err := s.store.ListOrganisationWorkspaces(r.Context(), session.User.ID, organisationID)
	if err != nil {
		s.uiNotFoundOrError(w, err)
		return
	}
	options := make([]pages.WorkspaceOption, 0, len(workspaces))
	for _, workspace := range workspaces {
		options = append(options, pages.WorkspaceOption{ID: workspace.ID, Name: workspace.Name})
	}
	props := pages.WorkspaceSelectorProps{
		User:             session.User,
		CSRF:             csrf,
		CurrentPath:      r.URL.Path,
		OrganisationID:   organisation.ID,
		OrganisationName: organisation.Name,
		Workspaces:       options,
	}
	if err := render(w, r, pages.WorkspaceSelector(props)); err != nil {
		http.Error(w, "service unavailable", http.StatusInternalServerError)
	}
}

func (s *Server) productAccessPage(w http.ResponseWriter, r *http.Request) {
	session, csrf, ok := s.uiSession(w, r)
	if !ok {
		return
	}
	query := r.URL.Query()
	organisationID := strings.TrimSpace(query.Get("organisation_id"))
	workspaceID := strings.TrimSpace(query.Get("workspace_id"))
	productKey := strings.TrimSpace(chi.URLParam(r, "productKey"))
	if organisationID == "" || workspaceID == "" || productKey == "" {
		http.NotFound(w, r)
		return
	}
	organisation, err := s.store.GetOrganisation(r.Context(), session.User.ID, organisationID)
	if err != nil {
		s.uiNotFoundOrError(w, err)
		return
	}
	workspace, err := s.store.GetWorkspace(r.Context(), session.User.ID, workspaceID)
	if err != nil || workspace.OrganisationID != organisation.ID {
		if err != nil {
			s.uiNotFoundOrError(w, err)
		} else {
			http.NotFound(w, r)
		}
		return
	}
	if _, err := s.store.ProveProductAccess(r.Context(), session.User.ID, organisationID, productKey, workspaceID); err != nil {
		s.uiNotFoundOrError(w, err)
		return
	}
	products, err := s.store.ListLaunchableProducts(r.Context(), session.User.ID, organisationID, workspaceID)
	if err != nil {
		s.uiNotFoundOrError(w, err)
		return
	}
	var product pages.ProductOption
	for _, candidate := range products {
		if candidate.Key != productKey {
			continue
		}
		product = pages.ProductOption{Key: candidate.Key, DisplayName: candidate.DisplayName}
		break
	}
	if product.Key == "" {
		http.NotFound(w, r)
		return
	}
	props := pages.ProductAccessProps{
		User:             session.User,
		CSRF:             csrf,
		CurrentPath:      r.URL.Path,
		Product:          product,
		OrganisationID:   organisation.ID,
		OrganisationName: organisation.Name,
		WorkspaceID:      workspace.ID,
		WorkspaceName:    workspace.Name,
	}
	if err := render(w, r, pages.ProductAccess(props)); err != nil {
		http.Error(w, "service unavailable", http.StatusInternalServerError)
	}
}

func (s *Server) uiSession(w http.ResponseWriter, r *http.Request) (store.AuthenticatedSession, string, bool) {
	session, ok := s.session(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return store.AuthenticatedSession{}, "", false
	}
	csrfCookie, err := r.Cookie("platform_csrf")
	if err != nil || strings.TrimSpace(csrfCookie.Value) == "" {
		http.Error(w, "session requires renewal", http.StatusUnauthorized)
		return store.AuthenticatedSession{}, "", false
	}
	return session, csrfCookie.Value, true
}

func (s *Server) launcherProps(r *http.Request, userID, csrf string) (pages.LauncherProps, error) {
	organisations, err := s.store.ListUserOrganisations(r.Context(), userID)
	if err != nil {
		return pages.LauncherProps{}, err
	}
	organisationID := strings.TrimSpace(r.URL.Query().Get("organisation_id"))
	if organisationID == "" && len(organisations) > 0 {
		organisationID = organisations[0].Organisation.ID
	}
	var organisationName string
	organisationOptions := make([]pages.OrganisationOption, 0, len(organisations))
	organisationKnown := organisationID == ""
	for _, organisation := range organisations {
		selected := organisation.Organisation.ID == organisationID
		if selected {
			organisationKnown = true
			organisationName = organisation.Organisation.Name
		}
		organisationOptions = append(organisationOptions, pages.OrganisationOption{ID: organisation.Organisation.ID, Name: organisation.Organisation.Name, Role: string(organisation.Role), Selected: selected})
	}
	if !organisationKnown {
		return pages.LauncherProps{}, store.ErrUnauthorisedOrganisationAction
	}

	workspaceID := ""
	workspaceName := ""
	workspaceOptions := make([]pages.WorkspaceOption, 0)
	products := make([]pages.ProductOption, 0)
	if organisationID != "" {
		workspaces, listErr := s.store.ListOrganisationWorkspaces(r.Context(), userID, organisationID)
		if listErr != nil {
			return pages.LauncherProps{}, listErr
		}
		requestedWorkspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
		if requestedWorkspaceID == "" && len(workspaces) > 0 {
			requestedWorkspaceID = workspaces[0].ID
		}
		workspaceKnown := requestedWorkspaceID == ""
		for _, workspace := range workspaces {
			selected := workspace.ID == requestedWorkspaceID
			if selected {
				workspaceKnown = true
				workspaceID = workspace.ID
				workspaceName = workspace.Name
			}
			workspaceOptions = append(workspaceOptions, pages.WorkspaceOption{ID: workspace.ID, Name: workspace.Name, Selected: selected})
		}
		if !workspaceKnown {
			return pages.LauncherProps{}, store.ErrAccessScopeDenied
		}
		if workspaceID != "" {
			launchable, launchErr := s.store.ListLaunchableProducts(r.Context(), userID, organisationID, workspaceID)
			if launchErr != nil {
				return pages.LauncherProps{}, launchErr
			}
			for _, product := range launchable {
				values := url.Values{"organisation_id": {organisationID}, "workspace_id": {workspaceID}}
				products = append(products, pages.ProductOption{Key: product.Key, DisplayName: product.DisplayName, AccessURL: "/launch/" + url.PathEscape(product.Key) + "?" + values.Encode()})
			}
		}
	}
	return pages.LauncherProps{
		User:             s.mustSessionUser(r, userID),
		CSRF:             csrf,
		CurrentPath:      r.URL.Path,
		Organisations:    organisationOptions,
		Workspaces:       workspaceOptions,
		Products:         products,
		OrganisationID:   organisationID,
		OrganisationName: organisationName,
		WorkspaceID:      workspaceID,
		WorkspaceName:    workspaceName,
	}, nil
}

func (s *Server) mustSessionUser(r *http.Request, userID string) domain.User {
	if session, ok := s.session(r); ok && session.User.ID == userID {
		return session.User
	}
	return domain.User{ID: userID}
}

func (s *Server) uiNotFoundOrError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrOrganisationNotFound) || errors.Is(err, store.ErrOrganisationArchived) ||
		errors.Is(err, store.ErrUnauthorisedOrganisationAction) || errors.Is(err, store.ErrAccessScopeDenied) ||
		errors.Is(err, store.ErrWorkspaceNotFound) || errors.Is(err, store.ErrWorkspaceArchived) ||
		errors.Is(err, store.ErrUnauthorisedWorkspaceAction) || errors.Is(err, store.ErrProductAccessDenied) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	s.serverError(w, err)
}
