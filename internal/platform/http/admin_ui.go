package httpserver

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
	"github.com/Onellan/tockrplatform/internal/store"
	"github.com/Onellan/tockrplatform/web/pages"
	"github.com/go-chi/chi/v5"
)

type organisationAdminContext struct {
	Organisation domain.Organisation
	Role         domain.OrganisationRole
	SystemAdmin  bool
	CanAdmin     bool
}

func (s *Server) adminLanding(w http.ResponseWriter, r *http.Request) {
	session, ok := s.session(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	organisations, err := s.store.ListUserOrganisations(r.Context(), session.User.ID)
	if err != nil {
		s.serverError(w, err)
		return
	}
	if organisationID := strings.TrimSpace(r.URL.Query().Get("organisation_id")); organisationID != "" {
		http.Redirect(w, r, "/organisations/"+url.PathEscape(organisationID)+"/admin", http.StatusSeeOther)
		return
	}
	if len(organisations) > 0 {
		http.Redirect(w, r, "/organisations/"+url.PathEscape(organisations[0].Organisation.ID)+"/admin", http.StatusSeeOther)
		return
	}
	isSystemAdmin, err := s.isSystemAdministrator(r, session.User.ID)
	if err != nil {
		s.serverError(w, err)
		return
	}
	if isSystemAdmin {
		http.Redirect(w, r, "/admin/system/products", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) organisationAdminPage(w http.ResponseWriter, r *http.Request) {
	session, csrf, ok := s.uiSession(w, r)
	if !ok {
		return
	}
	section := strings.TrimSpace(chi.URLParam(r, "section"))
	if section == "" {
		section = "general"
	}
	if section != "general" && section != "members" && section != "workspaces" && section != "products" {
		http.NotFound(w, r)
		return
	}
	organisation, err := s.store.GetOrganisation(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"))
	if err != nil {
		s.uiNotFoundOrError(w, err)
		return
	}
	access, err := s.resolveOrganisationAdminContext(r, session.User.ID, *organisation)
	if err != nil {
		s.uiNotFoundOrError(w, err)
		return
	}
	if (section == "members" || section == "products") && !access.CanAdmin {
		http.NotFound(w, r)
		return
	}
	props := pages.OrganisationAdminProps{
		User:             session.User,
		CSRF:             csrf,
		CurrentPath:      r.URL.Path,
		OrganisationID:   organisation.ID,
		OrganisationName: organisation.Name,
		Role:             organisationRoleLabel(access.Role, access.SystemAdmin),
		Section:          section,
		CanAdmin:         access.CanAdmin,
		SystemAdmin:      access.SystemAdmin,
		Navigation: []pages.AdminNavItem{
			{Href: "/organisations/" + organisation.ID + "/admin/general", Label: "General", Active: section == "general"},
			{Href: "/organisations/" + organisation.ID + "/admin/members", Label: "Members", Active: section == "members"},
			{Href: "/organisations/" + organisation.ID + "/admin/workspaces", Label: "Workspaces", Active: section == "workspaces"},
			{Href: "/organisations/" + organisation.ID + "/admin/products", Label: "Products", Active: section == "products"},
		},
		Flash: adminFlash(r.URL.Query().Get("status")),
	}
	if access.SystemAdmin {
		props.Navigation = append(props.Navigation, pages.AdminNavItem{Href: "/admin/system/products", Label: "System catalogue"})
	}
	if access.CanAdmin && section == "general" {
		audit, auditErr := s.store.ListOrganisationAudit(r.Context(), session.User.ID, organisation.ID, 50)
		if auditErr != nil {
			s.uiNotFoundOrError(w, auditErr)
			return
		}
		props.Audit = organisationAuditPages(audit)
	}
	if section == "members" {
		members, memberErr := s.store.ListOrganisationMembers(r.Context(), session.User.ID, organisation.ID)
		if memberErr != nil {
			s.uiNotFoundOrError(w, memberErr)
			return
		}
		props.Members = organisationMemberPages(members)
	}
	if section == "workspaces" {
		workspaces, workspaceErr := s.store.ListOrganisationWorkspaces(r.Context(), session.User.ID, organisation.ID)
		if workspaceErr != nil {
			s.uiNotFoundOrError(w, workspaceErr)
			return
		}
		for _, workspace := range workspaces {
			props.Workspaces = append(props.Workspaces, pages.AdminWorkspace{ID: workspace.ID, Name: workspace.Name, Status: string(workspace.Status), CreatedAt: adminTime(workspace.CreatedAt), AdminURL: "/workspaces/" + workspace.ID + "/admin"})
		}
	}
	if section == "products" {
		entitlements, entitlementErr := s.store.ListOrganisationProductEntitlements(r.Context(), session.User.ID, organisation.ID)
		if entitlementErr != nil {
			s.uiNotFoundOrError(w, entitlementErr)
			return
		}
		assignments, assignmentErr := s.store.ListUserProductAssignments(r.Context(), session.User.ID, organisation.ID)
		if assignmentErr != nil {
			s.uiNotFoundOrError(w, assignmentErr)
			return
		}
		for _, entitlement := range entitlements {
			props.Entitlements = append(props.Entitlements, pages.AdminEntitlement{ID: entitlement.ID, ProductKey: entitlement.ProductKey, Status: string(entitlement.Status), Active: entitlement.Active})
		}
		for _, assignment := range assignments {
			props.Assignments = append(props.Assignments, pages.AdminAssignment{ID: assignment.ID, UserID: assignment.UserID, ProductKey: assignment.ProductKey, Status: string(assignment.Status), Active: assignment.Active})
		}
	}
	if err := render(w, r, pages.OrganisationAdmin(props)); err != nil {
		http.Error(w, "service unavailable", http.StatusInternalServerError)
	}
}

func (s *Server) workspaceAdminPage(w http.ResponseWriter, r *http.Request) {
	session, csrf, ok := s.uiSession(w, r)
	if !ok {
		return
	}
	workspace, err := s.store.GetWorkspace(r.Context(), session.User.ID, chi.URLParam(r, "workspaceID"))
	if err != nil {
		s.uiNotFoundOrError(w, err)
		return
	}
	organisation, err := s.store.GetOrganisation(r.Context(), session.User.ID, workspace.OrganisationID)
	if err != nil {
		s.uiNotFoundOrError(w, err)
		return
	}
	scope, err := s.store.ProveWorkspaceScope(r.Context(), session.User.ID, workspace.ID, false)
	if err != nil {
		s.uiNotFoundOrError(w, err)
		return
	}
	canAdmin := scope.SystemAdmin || scope.OrganisationRole.CanAdminister() || scope.WorkspaceRole == domain.WorkspaceAdmin
	props := pages.WorkspaceAdminProps{
		User:             session.User,
		CSRF:             csrf,
		CurrentPath:      r.URL.Path,
		OrganisationID:   organisation.ID,
		OrganisationName: organisation.Name,
		WorkspaceID:      workspace.ID,
		WorkspaceName:    workspace.Name,
		OrganisationRole: organisationRoleLabel(scope.OrganisationRole, scope.SystemAdmin),
		WorkspaceRole:    workspaceRoleLabel(scope.WorkspaceRole, scope.OrganisationRole, scope.SystemAdmin),
		CanAdmin:         canAdmin,
		Flash:            adminFlash(r.URL.Query().Get("status")),
	}
	if canAdmin {
		members, memberErr := s.store.ListWorkspaceMembers(r.Context(), session.User.ID, workspace.ID)
		if memberErr != nil {
			s.uiNotFoundOrError(w, memberErr)
			return
		}
		audit, auditErr := s.store.ListWorkspaceAudit(r.Context(), session.User.ID, workspace.ID, 50)
		if auditErr != nil {
			s.uiNotFoundOrError(w, auditErr)
			return
		}
		props.Members = workspaceMemberPages(members)
		props.Audit = workspaceAuditPages(audit)
	}
	if err := render(w, r, pages.WorkspaceAdmin(props)); err != nil {
		http.Error(w, "service unavailable", http.StatusInternalServerError)
	}
}

func (s *Server) systemProductAdminPage(w http.ResponseWriter, r *http.Request) {
	session, csrf, ok := s.uiSession(w, r)
	if !ok {
		return
	}
	products, err := s.store.ListProducts(r.Context(), session.User.ID)
	if err != nil {
		s.uiNotFoundOrError(w, err)
		return
	}
	props := pages.SystemProductAdminProps{User: session.User, CSRF: csrf, CurrentPath: r.URL.Path, Flash: adminFlash(r.URL.Query().Get("status"))}
	for _, product := range products {
		props.Products = append(props.Products, pages.AdminProduct{Key: product.Key, DisplayName: product.DisplayName, Status: string(product.Status), Active: product.Status == domain.ProductActive})
	}
	if err := render(w, r, pages.SystemProductAdmin(props)); err != nil {
		http.Error(w, "service unavailable", http.StatusInternalServerError)
	}
}

func (s *Server) resolveOrganisationAdminContext(r *http.Request, userID string, organisation domain.Organisation) (organisationAdminContext, error) {
	organisations, err := s.store.ListUserOrganisations(r.Context(), userID)
	if err != nil {
		return organisationAdminContext{}, err
	}
	var role domain.OrganisationRole
	found := false
	for _, candidate := range organisations {
		if candidate.Organisation.ID == organisation.ID {
			role = candidate.Role
			found = true
			break
		}
	}
	if !found {
		return organisationAdminContext{}, store.ErrUnauthorisedOrganisationAction
	}
	systemAdmin, err := s.isSystemAdministrator(r, userID)
	if err != nil {
		return organisationAdminContext{}, err
	}
	return organisationAdminContext{Organisation: organisation, Role: role, SystemAdmin: systemAdmin, CanAdmin: systemAdmin || role.CanAdminister()}, nil
}

func (s *Server) isSystemAdministrator(r *http.Request, userID string) (bool, error) {
	_, err := s.store.ListProducts(r.Context(), userID)
	if errors.Is(err, store.ErrUnauthorisedProductAction) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Server) prepareUIForm(w http.ResponseWriter, r *http.Request) bool {
	if err := parseBoundedForm(w, r); err != nil {
		return false
	}
	return s.verifyMutationCSRF(w, r)
}

func (s *Server) renameOrganisationUI(w http.ResponseWriter, r *http.Request) {
	if !s.prepareUIForm(w, r) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	_, err := s.store.RenameOrganisation(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), r.FormValue("name"), r.FormValue("reason"), time.Now().UTC())
	if err != nil {
		s.uiMutationError(w, err)
		return
	}
	s.redirectAdmin(w, r, "/organisations/"+chi.URLParam(r, "organisationID")+"/admin/general", "saved")
}

func (s *Server) archiveOrganisationUI(w http.ResponseWriter, r *http.Request) {
	if !s.prepareUIForm(w, r) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if err := s.store.ArchiveOrganisation(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), r.FormValue("reason"), time.Now().UTC()); err != nil {
		s.uiMutationError(w, err)
		return
	}
	http.Redirect(w, r, "/organisations", http.StatusSeeOther)
}

func (s *Server) addOrganisationMemberUI(w http.ResponseWriter, r *http.Request) {
	if !s.prepareUIForm(w, r) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	_, err := s.store.AddOrganisationMember(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), r.FormValue("user_id"), domain.OrganisationRole(r.FormValue("role")), r.FormValue("reason"), time.Now().UTC())
	if err != nil {
		s.uiMutationError(w, err)
		return
	}
	s.redirectAdmin(w, r, "/organisations/"+chi.URLParam(r, "organisationID")+"/admin/members", "saved")
}

func (s *Server) changeOrganisationMemberRoleUI(w http.ResponseWriter, r *http.Request) {
	if !s.prepareUIForm(w, r) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	_, err := s.store.ChangeOrganisationMemberRole(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), chi.URLParam(r, "userID"), domain.OrganisationRole(r.FormValue("role")), r.FormValue("reason"), time.Now().UTC())
	if err != nil {
		s.uiMutationError(w, err)
		return
	}
	s.redirectAdmin(w, r, "/organisations/"+chi.URLParam(r, "organisationID")+"/admin/members", "saved")
}

func (s *Server) deactivateOrganisationMemberUI(w http.ResponseWriter, r *http.Request) {
	if !s.prepareUIForm(w, r) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if err := s.store.DeactivateOrganisationMember(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), chi.URLParam(r, "userID"), r.FormValue("reason"), time.Now().UTC()); err != nil {
		s.uiMutationError(w, err)
		return
	}
	s.redirectAdmin(w, r, "/organisations/"+chi.URLParam(r, "organisationID")+"/admin/members", "saved")
}

func (s *Server) createWorkspaceUI(w http.ResponseWriter, r *http.Request) {
	if !s.prepareUIForm(w, r) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	_, _, err := s.store.CreateWorkspace(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), domain.Workspace{Name: r.FormValue("name")}, r.FormValue("reason"), time.Now().UTC())
	if err != nil {
		s.uiMutationError(w, err)
		return
	}
	s.redirectAdmin(w, r, "/organisations/"+chi.URLParam(r, "organisationID")+"/admin/workspaces", "created")
}

func (s *Server) entitleOrganisationUI(w http.ResponseWriter, r *http.Request) {
	if !s.prepareUIForm(w, r) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	_, err := s.store.EntitleOrganisation(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), r.FormValue("product_key"), r.FormValue("reason"), time.Now().UTC())
	if err != nil {
		s.uiMutationError(w, err)
		return
	}
	s.redirectAdmin(w, r, "/organisations/"+chi.URLParam(r, "organisationID")+"/admin/products", "saved")
}

func (s *Server) revokeOrganisationEntitlementUI(w http.ResponseWriter, r *http.Request) {
	if !s.prepareUIForm(w, r) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if err := s.store.RevokeOrganisationEntitlement(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), chi.URLParam(r, "entitlementID"), r.FormValue("reason"), time.Now().UTC()); err != nil {
		s.uiMutationError(w, err)
		return
	}
	s.redirectAdmin(w, r, "/organisations/"+chi.URLParam(r, "organisationID")+"/admin/products", "saved")
}

func (s *Server) assignUserProductUI(w http.ResponseWriter, r *http.Request) {
	if !s.prepareUIForm(w, r) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	_, err := s.store.AssignUserProduct(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), r.FormValue("user_id"), r.FormValue("product_key"), r.FormValue("reason"), time.Now().UTC())
	if err != nil {
		s.uiMutationError(w, err)
		return
	}
	s.redirectAdmin(w, r, "/organisations/"+chi.URLParam(r, "organisationID")+"/admin/products", "saved")
}

func (s *Server) revokeUserProductUI(w http.ResponseWriter, r *http.Request) {
	if !s.prepareUIForm(w, r) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if err := s.store.RevokeUserProduct(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), chi.URLParam(r, "assignmentID"), r.FormValue("reason"), time.Now().UTC()); err != nil {
		s.uiMutationError(w, err)
		return
	}
	s.redirectAdmin(w, r, "/organisations/"+chi.URLParam(r, "organisationID")+"/admin/products", "saved")
}

func (s *Server) addWorkspaceMemberUI(w http.ResponseWriter, r *http.Request) {
	if !s.prepareUIForm(w, r) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	_, err := s.store.AddWorkspaceMember(r.Context(), session.User.ID, chi.URLParam(r, "workspaceID"), r.FormValue("user_id"), domain.WorkspaceRole(r.FormValue("role")), r.FormValue("reason"), time.Now().UTC())
	if err != nil {
		s.uiMutationError(w, err)
		return
	}
	s.redirectAdmin(w, r, "/workspaces/"+chi.URLParam(r, "workspaceID")+"/admin", "saved")
}

func (s *Server) changeWorkspaceMemberRoleUI(w http.ResponseWriter, r *http.Request) {
	if !s.prepareUIForm(w, r) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	_, err := s.store.ChangeWorkspaceMemberRole(r.Context(), session.User.ID, chi.URLParam(r, "workspaceID"), chi.URLParam(r, "userID"), domain.WorkspaceRole(r.FormValue("role")), r.FormValue("reason"), time.Now().UTC())
	if err != nil {
		s.uiMutationError(w, err)
		return
	}
	s.redirectAdmin(w, r, "/workspaces/"+chi.URLParam(r, "workspaceID")+"/admin", "saved")
}

func (s *Server) deactivateWorkspaceMemberUI(w http.ResponseWriter, r *http.Request) {
	if !s.prepareUIForm(w, r) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if err := s.store.DeactivateWorkspaceMember(r.Context(), session.User.ID, chi.URLParam(r, "workspaceID"), chi.URLParam(r, "userID"), r.FormValue("reason"), time.Now().UTC()); err != nil {
		s.uiMutationError(w, err)
		return
	}
	s.redirectAdmin(w, r, "/workspaces/"+chi.URLParam(r, "workspaceID")+"/admin", "saved")
}

func (s *Server) archiveWorkspaceUI(w http.ResponseWriter, r *http.Request) {
	if !s.prepareUIForm(w, r) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if err := s.store.ArchiveWorkspace(r.Context(), session.User.ID, chi.URLParam(r, "workspaceID"), r.FormValue("reason"), time.Now().UTC()); err != nil {
		s.uiMutationError(w, err)
		return
	}
	http.Redirect(w, r, "/organisations", http.StatusSeeOther)
}

func (s *Server) retireProductUI(w http.ResponseWriter, r *http.Request) {
	if !s.prepareUIForm(w, r) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if err := s.store.RetireProduct(r.Context(), session.User.ID, chi.URLParam(r, "productKey"), r.FormValue("reason"), time.Now().UTC()); err != nil {
		s.uiMutationError(w, err)
		return
	}
	s.redirectAdmin(w, r, "/admin/system/products", "saved")
}

func (s *Server) redirectAdmin(w http.ResponseWriter, r *http.Request, path, status string) {
	query := url.Values{"status": {status}}
	http.Redirect(w, r, path+"?"+query.Encode(), http.StatusSeeOther)
}

func (s *Server) uiMutationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrOrganisationNotFound), errors.Is(err, store.ErrOrganisationArchived), errors.Is(err, store.ErrMembershipNotFound), errors.Is(err, store.ErrUnauthorisedOrganisationAction), errors.Is(err, store.ErrWorkspaceNotFound), errors.Is(err, store.ErrWorkspaceArchived), errors.Is(err, store.ErrWorkspaceMembershipNotFound), errors.Is(err, store.ErrUnauthorisedWorkspaceAction), errors.Is(err, store.ErrProductNotFound), errors.Is(err, store.ErrEntitlementNotFound), errors.Is(err, store.ErrUnauthorisedProductAction), errors.Is(err, store.ErrProductAssignmentNotFound):
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	case errors.Is(err, store.ErrOwnerMutationNotAuthorised):
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	case errors.Is(err, store.ErrDuplicateOrganisationMember), errors.Is(err, store.ErrDuplicateWorkspaceMember), errors.Is(err, store.ErrDuplicateOrganisationEntitlement), errors.Is(err, store.ErrDuplicateUserProductAssignment), errors.Is(err, store.ErrProductRetired), errors.Is(err, store.ErrEntitlementInactive), errors.Is(err, store.ErrProductAssignmentInactive):
		http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
	case errors.Is(err, domain.ErrInvalidReason), errors.Is(err, domain.ErrInvalidOrganisationName), errors.Is(err, domain.ErrInvalidOrganisationRole), errors.Is(err, domain.ErrInvalidWorkspaceName), errors.Is(err, domain.ErrInvalidWorkspaceRole), errors.Is(err, domain.ErrInvalidProductKey):
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	default:
		s.serverError(w, err)
	}
}

func adminFlash(status string) string {
	switch status {
	case "saved":
		return "Platform administration change saved."
	case "created":
		return "Platform record created."
	default:
		return ""
	}
}

func organisationRoleLabel(role domain.OrganisationRole, systemAdmin bool) string {
	if systemAdmin {
		return "System administrator"
	}
	if role == "" {
		return "Organisation member"
	}
	return string(role)
}

func workspaceRoleLabel(workspaceRole domain.WorkspaceRole, organisationRole domain.OrganisationRole, systemAdmin bool) string {
	if systemAdmin {
		return "System administrator"
	}
	if organisationRole.CanAdminister() {
		return string(organisationRole)
	}
	if workspaceRole == "" {
		return "Workspace viewer"
	}
	return string(workspaceRole)
}

func adminTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format("2006-01-02 15:04 UTC")
}

func organisationMemberPages(members []store.OrganisationMemberRecord) []pages.AdminMember {
	result := make([]pages.AdminMember, 0, len(members))
	for _, member := range members {
		result = append(result, pages.AdminMember{MembershipID: member.MembershipID, UserID: member.UserID, Email: member.Email, DisplayName: member.DisplayName, Role: string(member.Role), RoleChange: organisationRoleChange(member.Role), AssignedAt: adminTime(member.AssignedAt)})
	}
	return result
}

func workspaceMemberPages(members []store.WorkspaceMemberRecord) []pages.AdminMember {
	result := make([]pages.AdminMember, 0, len(members))
	for _, member := range members {
		result = append(result, pages.AdminMember{MembershipID: member.MembershipID, UserID: member.UserID, Email: member.Email, DisplayName: member.DisplayName, Role: string(member.Role), RoleChange: workspaceRoleChange(member.Role), AssignedAt: adminTime(member.AssignedAt)})
	}
	return result
}

func organisationRoleChange(role domain.OrganisationRole) string {
	if role == domain.OrganisationMember {
		return string(domain.OrganisationAdmin)
	}
	return string(domain.OrganisationMember)
}

func workspaceRoleChange(role domain.WorkspaceRole) string {
	if role == domain.WorkspaceAdmin {
		return string(domain.WorkspaceMember)
	}
	return string(domain.WorkspaceAdmin)
}

func organisationAuditPages(events []store.OrganisationAuditRecord) []pages.AdminAudit {
	result := make([]pages.AdminAudit, 0, len(events))
	for _, event := range events {
		result = append(result, pages.AdminAudit{Event: event.Event, Details: event.Details, OccurredAt: adminTime(event.OccurredAt)})
	}
	return result
}

func workspaceAuditPages(events []store.WorkspaceAuditRecord) []pages.AdminAudit {
	result := make([]pages.AdminAudit, 0, len(events))
	for _, event := range events {
		result = append(result, pages.AdminAudit{Event: event.Event, Details: event.Details, OccurredAt: adminTime(event.OccurredAt)})
	}
	return result
}
