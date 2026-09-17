package httpserver

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Onellan/tockrplatform/internal/auth"
	"github.com/Onellan/tockrplatform/internal/domain"
	"github.com/Onellan/tockrplatform/internal/platform/assertion"
	"github.com/Onellan/tockrplatform/internal/platform/readauthority"
	"github.com/Onellan/tockrplatform/internal/store"
	"github.com/Onellan/tockrplatform/web/templates"
	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
)

const (
	sessionCookieName             = "platform_session"
	loginCSRFCookie               = "platform_login_csrf"
	defaultRequestBodyLimit int64 = 1 << 20
)

type Config struct {
	CookieSecure         bool
	AllowInsecureCookies bool
	SessionTTL           time.Duration
	RateLimitEnabled     bool
	StaticDir            string
	LoginLimiter         auth.LoginLimiterConfig
	AssertionIssuer      *assertion.Issuer
	ReadinessCheck       func(context.Context) error
	MaxRequestBodyBytes  int64
	ReadAuthorityKeys    readauthority.PublicKeySet
}

type Server struct {
	store             store.PlatformStore
	cfg               Config
	limiter           *auth.LoginLimiter
	readAuthorityKeys readauthority.PublicKeySet
	readAuthorityRate *readAuthorityRateLimiter
}

type sessionContextKey struct{}
type workspaceScopeContextKey struct{}

func NewServer(persistence store.PlatformStore, cfg Config) *Server {
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = 12 * time.Hour
	}
	if cfg.StaticDir == "" {
		cfg.StaticDir = "web/static"
	}
	if !cfg.AllowInsecureCookies {
		cfg.CookieSecure = true
	}
	if cfg.MaxRequestBodyBytes <= 0 {
		cfg.MaxRequestBodyBytes = defaultRequestBodyLimit
	}
	return &Server{store: persistence, cfg: cfg, limiter: auth.NewLoginLimiter(cfg.LoginLimiter), readAuthorityKeys: cloneReadAuthorityKeys(cfg.ReadAuthorityKeys), readAuthorityRate: newReadAuthorityRateLimiter()}
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(s.securityHeaders)
	r.Use(s.requestBodyLimit)
	r.Get("/healthz", s.health)
	r.Get("/readyz", s.ready)
	r.Get("/favicon.ico", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	r.Get("/.well-known/tockr-platform-assertion-keys", s.assertionKeys)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(s.cfg.StaticDir))))
	readAuthority := r.With(s.requireReadAuthority)
	readAuthority.Post("/api/v1/read-authority/snapshots", s.createReadAuthoritySnapshot)
	readAuthority.Get("/api/v1/read-authority/snapshots/{snapshotID}/records", s.listReadAuthoritySnapshotRecords)
	readAuthority.Get("/api/v1/read-authority/changes", s.listReadAuthorityChanges)
	readAuthority.Get("/api/v1/read-authority/status", s.readAuthorityStatus)
	r.Get("/login", s.loginPage)
	r.Post("/login", s.login)
	r.Group(func(protected chi.Router) {
		protected.Use(s.requireSession)
		protected.Get("/", s.platformLauncher)
		protected.Get("/organisations", s.platformLauncher)
		protected.Get("/organisations/{organisationID}/workspaces", s.workspaceSelector)
		protected.Get("/launch/{productKey}", s.productAccessPage)
		protected.Get("/admin", s.adminLanding)
		protected.Get("/admin/system/products", s.systemProductAdminPage)
		protected.Post("/admin/system/products/{productKey}/retire", s.retireProductUI)
		protected.Get("/organisations/{organisationID}/admin", s.organisationAdminPage)
		protected.Get("/organisations/{organisationID}/admin/{section}", s.organisationAdminPage)
		protected.Get("/workspaces/{workspaceID}/admin", s.workspaceAdminPage)
		protected.Post("/organisations/{organisationID}/admin/general/rename", s.renameOrganisationUI)
		protected.Post("/organisations/{organisationID}/admin/general/archive", s.archiveOrganisationUI)
		protected.Post("/organisations/{organisationID}/admin/members/add", s.addOrganisationMemberUI)
		protected.Post("/organisations/{organisationID}/admin/members/{userID}/role", s.changeOrganisationMemberRoleUI)
		protected.Post("/organisations/{organisationID}/admin/members/{userID}/remove", s.deactivateOrganisationMemberUI)
		protected.Post("/organisations/{organisationID}/admin/workspaces/create", s.createWorkspaceUI)
		protected.Post("/organisations/{organisationID}/admin/products/entitlements/add", s.entitleOrganisationUI)
		protected.Post("/organisations/{organisationID}/admin/products/entitlements/{entitlementID}/revoke", s.revokeOrganisationEntitlementUI)
		protected.Post("/organisations/{organisationID}/admin/products/assignments/add", s.assignUserProductUI)
		protected.Post("/organisations/{organisationID}/admin/products/assignments/{assignmentID}/revoke", s.revokeUserProductUI)
		protected.Post("/workspaces/{workspaceID}/admin/members/add", s.addWorkspaceMemberUI)
		protected.Post("/workspaces/{workspaceID}/admin/members/{userID}/role", s.changeWorkspaceMemberRoleUI)
		protected.Post("/workspaces/{workspaceID}/admin/members/{userID}/remove", s.deactivateWorkspaceMemberUI)
		protected.Post("/workspaces/{workspaceID}/admin/archive", s.archiveWorkspaceUI)
		protected.Get("/account", s.accountPage)
		protected.Post("/account/mfa/setup", s.mfaSetup)
		protected.Post("/account/mfa/confirm", s.mfaConfirm)
		protected.Post("/logout", s.logout)
		protected.Post("/api/organisations", s.createOrganisation)
		protected.Get("/api/organisations/{organisationID}", s.getOrganisation)
		protected.Get("/api/organisations/{organisationID}/workspace-entry", s.getOrganisationWorkspaceEntry)
		protected.Post("/api/organisations/{organisationID}/workspaces", s.createWorkspace)
		protected.Get("/api/organisations/{organisationID}/workspaces", s.listOrganisationWorkspaces)
		protected.Get("/api/organisations/{organisationID}/members", s.listOrganisationMembers)
		protected.Get("/api/organisations/{organisationID}/audit", s.listOrganisationAudit)
		protected.Patch("/api/organisations/{organisationID}", s.renameOrganisation)
		protected.Post("/api/organisations/{organisationID}/members", s.addOrganisationMember)
		protected.Patch("/api/organisations/{organisationID}/members/{userID}", s.changeOrganisationMemberRole)
		protected.Delete("/api/organisations/{organisationID}/members/{userID}", s.deactivateOrganisationMember)
		protected.Post("/api/organisations/{organisationID}/archive", s.archiveOrganisation)
		protected.Get("/api/platform/products", s.listProducts)
		protected.Post("/api/platform/products/{productKey}/retire", s.retireProduct)
		protected.Get("/api/organisations/{organisationID}/product-entitlements", s.listOrganisationProductEntitlements)
		protected.Post("/api/organisations/{organisationID}/product-entitlements", s.entitleOrganisation)
		protected.Delete("/api/organisations/{organisationID}/product-entitlements/{entitlementID}", s.revokeOrganisationEntitlement)
		protected.Get("/api/organisations/{organisationID}/product-assignments", s.listUserProductAssignments)
		protected.Post("/api/organisations/{organisationID}/product-assignments", s.assignUserProduct)
		protected.Delete("/api/organisations/{organisationID}/product-assignments/{assignmentID}", s.revokeUserProduct)
		protected.Get("/api/organisations/{organisationID}/product-access/{productKey}/workspaces/{workspaceID}", s.proveProductAccess)
		protected.Post("/api/organisations/{organisationID}/product-assertions/{productKey}/workspaces/{workspaceID}", s.issueProductAssertion)
		workspaceRead := protected.With(s.requireWorkspaceScope(false))
		workspaceRead.Get("/api/workspaces/{workspaceID}", s.getWorkspace)
		workspaceAdmin := protected.With(s.requireWorkspaceScope(true))
		workspaceAdmin.Get("/api/workspaces/{workspaceID}/members", s.listWorkspaceMembers)
		workspaceAdmin.Get("/api/workspaces/{workspaceID}/audit", s.listWorkspaceAudit)
		workspaceAdmin.Post("/api/workspaces/{workspaceID}/members", s.addWorkspaceMember)
		workspaceAdmin.Patch("/api/workspaces/{workspaceID}/members/{userID}", s.changeWorkspaceMemberRole)
		workspaceAdmin.Delete("/api/workspaces/{workspaceID}/members/{userID}", s.deactivateWorkspaceMember)
		workspaceAdmin.Post("/api/workspaces/{workspaceID}/archive", s.archiveWorkspace)
	})
	return r
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), geolocation=(), microphone=(), payment=(), usb=()")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if s.cfg.ReadinessCheck != nil {
		if err := s.cfg.ReadinessCheck(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) requestBodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxRequestBodyBytes)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.session(r); ok {
		http.Redirect(w, r, "/account", http.StatusSeeOther)
		return
	}
	token, err := auth.NewOpaqueToken(32)
	if err != nil {
		http.Error(w, "service unavailable", http.StatusInternalServerError)
		return
	}
	setCookie(w, &http.Cookie{Name: loginCSRFCookie, Value: token, Path: "/login", MaxAge: 900, HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode})
	message := ""
	if r.URL.Query().Get("error") == "1" {
		message = "Invalid credentials"
	}
	if err := render(w, r, templates.Login(token, message)); err != nil {
		http.Error(w, "service unavailable", http.StatusInternalServerError)
	}
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if err := parseBoundedForm(w, r); err != nil {
		return
	}
	csrfCookie, err := r.Cookie(loginCSRFCookie)
	if err != nil || !auth.VerifyTokenHash(auth.HashToken(csrfCookie.Value), r.FormValue("csrf")) {
		http.Error(w, "request could not be validated", http.StatusForbidden)
		return
	}
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	key := auth.LoginAbuseKey(email, r.RemoteAddr)
	now := time.Now().UTC()
	if s.cfg.RateLimitEnabled {
		if retryAfter, blocked := s.limiter.RetryAfter(key, now); blocked {
			s.renderLoginThrottle(w, retryAfter)
			return
		}
	}
	credential, err := s.store.FindLoginCredential(r.Context(), email)
	if err != nil {
		s.serverError(w, err)
		return
	}
	hash := auth.DummyPasswordHash
	if credential != nil && credential.PasswordHash != "" {
		hash = credential.PasswordHash
	}
	passwordOK := auth.CheckPassword(hash, r.FormValue("password"))
	if credential == nil || !passwordOK || !credential.User.Active {
		var userID *string
		if credential != nil {
			userID = &credential.User.ID
		}
		if err := s.store.RecordSecurityEvent(r.Context(), userID, "failed_login", "credentials rejected", now); err != nil {
			s.serverError(w, err)
			return
		}
		if s.cfg.RateLimitEnabled {
			if retryAfter, blocked := s.limiter.RecordFailure(key, now); blocked {
				s.renderLoginThrottle(w, retryAfter)
				return
			}
		}
		http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
		return
	}
	user := &credential.User
	if user.MFAEnabled {
		mfaValid := false
		if code := strings.TrimSpace(r.FormValue("mfa_code")); code != "" {
			mfaValid, err = s.store.VerifyMFA(r.Context(), user.ID, code)
		}
		if !mfaValid && strings.TrimSpace(r.FormValue("recovery_code")) != "" {
			mfaValid, err = s.store.UseRecoveryCode(r.Context(), user.ID, r.FormValue("recovery_code"), now)
		}
		if err != nil {
			s.serverError(w, err)
			return
		}
		if !mfaValid {
			if auditErr := s.store.RecordSecurityEvent(r.Context(), &user.ID, "failed_mfa", "second factor rejected", now); auditErr != nil {
				s.serverError(w, auditErr)
				return
			}
			if s.cfg.RateLimitEnabled {
				if retryAfter, blocked := s.limiter.RecordFailure(key, now); blocked {
					s.renderLoginThrottle(w, retryAfter)
					return
				}
			}
			http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
			return
		}
	}
	if s.cfg.RateLimitEnabled {
		s.limiter.RecordSuccess(key)
	}
	token, err := auth.NewOpaqueToken(32)
	if err != nil {
		s.serverError(w, err)
		return
	}
	sessionCSRF, err := auth.NewOpaqueToken(32)
	if err != nil {
		s.serverError(w, err)
		return
	}
	if err := s.store.EstablishSession(r.Context(), user.ID, token, sessionCSRF, now.Add(s.cfg.SessionTTL), now); err != nil {
		s.serverError(w, err)
		return
	}
	setCookie(w, &http.Cookie{Name: sessionCookieName, Value: token, Path: "/", MaxAge: int(s.cfg.SessionTTL.Seconds()), HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode})
	setCookie(w, &http.Cookie{Name: "platform_csrf", Value: sessionCSRF, Path: "/", MaxAge: int(s.cfg.SessionTTL.Seconds()), HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode})
	setCookie(w, &http.Cookie{Name: loginCSRFCookie, Value: "", Path: "/login", MaxAge: -1, HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, "/account", http.StatusSeeOther)
}

func (s *Server) renderLoginThrottle(w http.ResponseWriter, retryAfter time.Duration) {
	seconds := int((retryAfter + time.Second - 1) / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", fmt.Sprint(seconds))
	http.Error(w, "too many login attempts", http.StatusTooManyRequests)
}

func (s *Server) accountPage(w http.ResponseWriter, r *http.Request) {
	session, ok := s.session(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	csrfCookie, err := r.Cookie("platform_csrf")
	if err != nil || strings.TrimSpace(csrfCookie.Value) == "" {
		http.Error(w, "session requires renewal", http.StatusUnauthorized)
		return
	}
	if err := render(w, r, templates.Account(session.User, csrfCookie.Value)); err != nil {
		http.Error(w, "service unavailable", http.StatusInternalServerError)
	}
}

type organisationCreateRequest struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type workspaceCreateRequest struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type organisationRenameRequest struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type organisationMemberMutationRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Reason string `json:"reason"`
}

type organisationReasonRequest struct {
	Reason string `json:"reason"`
}

type productEntitlementRequest struct {
	ProductKey string `json:"product_key"`
	Reason     string `json:"reason"`
}

type productAssignmentRequest struct {
	UserID     string `json:"user_id"`
	ProductKey string `json:"product_key"`
	Reason     string `json:"reason"`
}

type organisationResponse struct {
	ID         string                    `json:"id"`
	Name       string                    `json:"name"`
	Status     domain.OrganisationStatus `json:"status"`
	CreatedAt  time.Time                 `json:"created_at"`
	ArchivedAt *time.Time                `json:"archived_at,omitempty"`
}

type organisationMemberResponse struct {
	MembershipID   string                  `json:"membership_id"`
	OrganisationID string                  `json:"organisation_id"`
	UserID         string                  `json:"user_id"`
	Email          string                  `json:"email"`
	DisplayName    string                  `json:"display_name"`
	Role           domain.OrganisationRole `json:"role"`
	AssignedAt     time.Time               `json:"assigned_at"`
}

type organisationAuditResponse struct {
	ID             int64     `json:"id"`
	OrganisationID string    `json:"organisation_id"`
	ActorUserID    string    `json:"actor_user_id"`
	Event          string    `json:"event"`
	Details        string    `json:"details"`
	OccurredAt     time.Time `json:"occurred_at"`
}

type productResponse struct {
	Key         string               `json:"key"`
	DisplayName string               `json:"display_name"`
	Status      domain.ProductStatus `json:"status"`
	CreatedAt   time.Time            `json:"created_at"`
	RetiredAt   *time.Time           `json:"retired_at,omitempty"`
}

type organisationProductEntitlementResponse struct {
	ID               string                                      `json:"id"`
	OrganisationID   string                                      `json:"organisation_id"`
	ProductKey       string                                      `json:"product_key"`
	Status           domain.OrganisationProductEntitlementStatus `json:"status"`
	Active           bool                                        `json:"active"`
	GrantedBy        string                                      `json:"granted_by"`
	GrantedAt        time.Time                                   `json:"granted_at"`
	RevokedBy        string                                      `json:"revoked_by,omitempty"`
	RevokedAt        *time.Time                                  `json:"revoked_at,omitempty"`
	RevocationReason string                                      `json:"revocation_reason,omitempty"`
}

type userProductAssignmentResponse struct {
	ID               string                             `json:"id"`
	UserID           string                             `json:"user_id"`
	OrganisationID   string                             `json:"organisation_id"`
	ProductKey       string                             `json:"product_key"`
	Status           domain.UserProductAssignmentStatus `json:"status"`
	Active           bool                               `json:"active"`
	AssignedBy       string                             `json:"assigned_by"`
	AssignedAt       time.Time                          `json:"assigned_at"`
	RevokedBy        string                             `json:"revoked_by,omitempty"`
	RevokedAt        *time.Time                         `json:"revoked_at,omitempty"`
	RevocationReason string                             `json:"revocation_reason,omitempty"`
}

type productAccessResponse struct {
	Allowed          bool                    `json:"allowed"`
	UserID           string                  `json:"user_id"`
	OrganisationID   string                  `json:"organisation_id"`
	ProductKey       string                  `json:"product_key"`
	WorkspaceID      string                  `json:"workspace_id"`
	OrganisationRole domain.OrganisationRole `json:"organisation_role"`
	WorkspaceRole    domain.WorkspaceRole    `json:"workspace_role"`
}

type productAssertionRequest struct {
	Audience string `json:"audience"`
}

type productAssertionResponse struct {
	Assertion        string    `json:"assertion"`
	AssertionVersion int       `json:"assertion_version"`
	IssuedAt         time.Time `json:"issued_at"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type assertionKeyResponse struct {
	KeyID     string `json:"key_id"`
	Algorithm string `json:"algorithm"`
	PublicKey string `json:"public_key"`
}

type organisationWorkspaceEntryResponse struct {
	OrganisationID     string              `json:"organisation_id"`
	Resource           string              `json:"resource"`
	Available          bool                `json:"available"`
	DefaultWorkspaceID string              `json:"default_workspace_id,omitempty"`
	Workspaces         []workspaceResponse `json:"workspaces"`
}

type workspaceResponse struct {
	ID             string                 `json:"id"`
	OrganisationID string                 `json:"organisation_id"`
	Name           string                 `json:"name"`
	Status         domain.WorkspaceStatus `json:"status"`
	CreatedAt      time.Time              `json:"created_at"`
	ArchivedAt     *time.Time             `json:"archived_at,omitempty"`
}

type workspaceMemberResponse struct {
	MembershipID string               `json:"membership_id"`
	WorkspaceID  string               `json:"workspace_id"`
	UserID       string               `json:"user_id"`
	Email        string               `json:"email,omitempty"`
	DisplayName  string               `json:"display_name,omitempty"`
	Role         domain.WorkspaceRole `json:"role"`
	AssignedAt   time.Time            `json:"assigned_at"`
}

type workspaceAuditResponse struct {
	ID          int64     `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	ActorUserID string    `json:"actor_user_id"`
	Event       string    `json:"event"`
	Details     string    `json:"details"`
	OccurredAt  time.Time `json:"occurred_at"`
}

func (s *Server) createOrganisation(w http.ResponseWriter, r *http.Request) {
	if !s.verifyMutationCSRF(w, r) {
		return
	}
	var request organisationCreateRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	organisation, _, err := s.store.CreateOrganisation(r.Context(), session.User.ID, domain.Organisation{Name: request.Name}, request.Reason, time.Now().UTC())
	if err != nil {
		writeOrganisationError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, organisationResponseFromDomain(organisation))
}

func (s *Server) getOrganisation(w http.ResponseWriter, r *http.Request) {
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	organisation, err := s.store.GetOrganisation(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"))
	if err != nil {
		writeOrganisationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, organisationResponseFromDomain(*organisation))
}

func (s *Server) listOrganisationMembers(w http.ResponseWriter, r *http.Request) {
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	members, err := s.store.ListOrganisationMembers(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"))
	if err != nil {
		writeOrganisationError(w, err)
		return
	}
	response := make([]organisationMemberResponse, 0, len(members))
	for _, member := range members {
		response = append(response, organisationMemberResponse{MembershipID: member.MembershipID, OrganisationID: member.OrganisationID, UserID: member.UserID, Email: member.Email, DisplayName: member.DisplayName, Role: member.Role, AssignedAt: member.AssignedAt})
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) listOrganisationAudit(w http.ResponseWriter, r *http.Request) {
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	limit := 50
	if value := strings.TrimSpace(r.URL.Query().Get("limit")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	if limit < 1 || limit > 100 {
		http.Error(w, "invalid limit", http.StatusBadRequest)
		return
	}
	events, err := s.store.ListOrganisationAudit(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), limit)
	if err != nil {
		writeOrganisationError(w, err)
		return
	}
	response := make([]organisationAuditResponse, 0, len(events))
	for _, event := range events {
		response = append(response, organisationAuditResponse{ID: event.ID, OrganisationID: event.OrganisationID, ActorUserID: event.ActorUserID, Event: event.Event, Details: event.Details, OccurredAt: event.OccurredAt})
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) getOrganisationWorkspaceEntry(w http.ResponseWriter, r *http.Request) {
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	entry, err := s.store.GetOrganisationWorkspaceEntry(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"))
	if err != nil {
		writeOrganisationError(w, err)
		return
	}
	workspaces := make([]workspaceResponse, 0, len(entry.Workspaces))
	for _, workspace := range entry.Workspaces {
		workspaces = append(workspaces, workspaceResponseFromDomain(workspace))
	}
	writeJSON(w, http.StatusOK, organisationWorkspaceEntryResponse{OrganisationID: entry.OrganisationID, Resource: entry.Resource, Available: entry.Available, DefaultWorkspaceID: entry.DefaultWorkspaceID, Workspaces: workspaces})
}

func (s *Server) createWorkspace(w http.ResponseWriter, r *http.Request) {
	if !s.verifyMutationCSRF(w, r) {
		return
	}
	var request workspaceCreateRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	workspace, _, err := s.store.CreateWorkspace(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), domain.Workspace{Name: request.Name}, request.Reason, time.Now().UTC())
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, workspaceResponseFromDomain(workspace))
}

func (s *Server) listOrganisationWorkspaces(w http.ResponseWriter, r *http.Request) {
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	workspaces, err := s.store.ListOrganisationWorkspaces(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"))
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}
	response := make([]workspaceResponse, 0, len(workspaces))
	for _, workspace := range workspaces {
		response = append(response, workspaceResponseFromDomain(workspace))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) getWorkspace(w http.ResponseWriter, r *http.Request) {
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	workspace, err := s.store.GetWorkspace(r.Context(), session.User.ID, chi.URLParam(r, "workspaceID"))
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, workspaceResponseFromDomain(*workspace))
}

func (s *Server) listWorkspaceMembers(w http.ResponseWriter, r *http.Request) {
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	members, err := s.store.ListWorkspaceMembers(r.Context(), session.User.ID, chi.URLParam(r, "workspaceID"))
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}
	response := make([]workspaceMemberResponse, 0, len(members))
	for _, member := range members {
		response = append(response, workspaceMemberResponse{MembershipID: member.MembershipID, WorkspaceID: member.WorkspaceID, UserID: member.UserID, Email: member.Email, DisplayName: member.DisplayName, Role: member.Role, AssignedAt: member.AssignedAt})
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) listWorkspaceAudit(w http.ResponseWriter, r *http.Request) {
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	limit := 50
	if value := strings.TrimSpace(r.URL.Query().Get("limit")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	if limit < 1 || limit > 100 {
		http.Error(w, "invalid limit", http.StatusBadRequest)
		return
	}
	events, err := s.store.ListWorkspaceAudit(r.Context(), session.User.ID, chi.URLParam(r, "workspaceID"), limit)
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}
	response := make([]workspaceAuditResponse, 0, len(events))
	for _, event := range events {
		response = append(response, workspaceAuditResponse{ID: event.ID, WorkspaceID: event.WorkspaceID, ActorUserID: event.ActorUserID, Event: event.Event, Details: event.Details, OccurredAt: event.OccurredAt})
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) addWorkspaceMember(w http.ResponseWriter, r *http.Request) {
	if !s.verifyMutationCSRF(w, r) {
		return
	}
	var request organisationMemberMutationRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	membership, err := s.store.AddWorkspaceMember(r.Context(), session.User.ID, chi.URLParam(r, "workspaceID"), strings.TrimSpace(request.UserID), domain.WorkspaceRole(strings.TrimSpace(request.Role)), request.Reason, time.Now().UTC())
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, workspaceMemberResponse{MembershipID: membership.ID, WorkspaceID: membership.WorkspaceID, UserID: membership.UserID, Role: membership.Role, AssignedAt: membership.AssignedAt})
}

func (s *Server) changeWorkspaceMemberRole(w http.ResponseWriter, r *http.Request) {
	if !s.verifyMutationCSRF(w, r) {
		return
	}
	var request organisationMemberMutationRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	membership, err := s.store.ChangeWorkspaceMemberRole(r.Context(), session.User.ID, chi.URLParam(r, "workspaceID"), chi.URLParam(r, "userID"), domain.WorkspaceRole(strings.TrimSpace(request.Role)), request.Reason, time.Now().UTC())
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, workspaceMemberResponse{MembershipID: membership.ID, WorkspaceID: membership.WorkspaceID, UserID: membership.UserID, Role: membership.Role, AssignedAt: membership.AssignedAt})
}

func (s *Server) deactivateWorkspaceMember(w http.ResponseWriter, r *http.Request) {
	if !s.verifyMutationCSRF(w, r) {
		return
	}
	var request organisationReasonRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if err := s.store.DeactivateWorkspaceMember(r.Context(), session.User.ID, chi.URLParam(r, "workspaceID"), chi.URLParam(r, "userID"), request.Reason, time.Now().UTC()); err != nil {
		writeWorkspaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) archiveWorkspace(w http.ResponseWriter, r *http.Request) {
	if !s.verifyMutationCSRF(w, r) {
		return
	}
	var request organisationReasonRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if err := s.store.ArchiveWorkspace(r.Context(), session.User.ID, chi.URLParam(r, "workspaceID"), request.Reason, time.Now().UTC()); err != nil {
		writeWorkspaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) renameOrganisation(w http.ResponseWriter, r *http.Request) {
	if !s.verifyMutationCSRF(w, r) {
		return
	}
	var request organisationRenameRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	organisation, err := s.store.RenameOrganisation(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), request.Name, request.Reason, time.Now().UTC())
	if err != nil {
		writeOrganisationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, organisationResponseFromDomain(organisation))
}

func (s *Server) addOrganisationMember(w http.ResponseWriter, r *http.Request) {
	if !s.verifyMutationCSRF(w, r) {
		return
	}
	var request organisationMemberMutationRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	role := domain.OrganisationRole(strings.TrimSpace(request.Role))
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	membership, err := s.store.AddOrganisationMember(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), strings.TrimSpace(request.UserID), role, request.Reason, time.Now().UTC())
	if err != nil {
		writeOrganisationError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, organisationMemberResponse{MembershipID: membership.ID, OrganisationID: membership.OrganisationID, UserID: membership.UserID, Role: membership.Role, AssignedAt: membership.AssignedAt})
}

func (s *Server) changeOrganisationMemberRole(w http.ResponseWriter, r *http.Request) {
	if !s.verifyMutationCSRF(w, r) {
		return
	}
	var request organisationMemberMutationRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	role := domain.OrganisationRole(strings.TrimSpace(request.Role))
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	membership, err := s.store.ChangeOrganisationMemberRole(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), chi.URLParam(r, "userID"), role, request.Reason, time.Now().UTC())
	if err != nil {
		writeOrganisationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, organisationMemberResponse{MembershipID: membership.ID, OrganisationID: membership.OrganisationID, UserID: membership.UserID, Role: membership.Role, AssignedAt: membership.AssignedAt})
}

func (s *Server) deactivateOrganisationMember(w http.ResponseWriter, r *http.Request) {
	if !s.verifyMutationCSRF(w, r) {
		return
	}
	var request organisationReasonRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if err := s.store.DeactivateOrganisationMember(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), chi.URLParam(r, "userID"), request.Reason, time.Now().UTC()); err != nil {
		writeOrganisationError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) archiveOrganisation(w http.ResponseWriter, r *http.Request) {
	if !s.verifyMutationCSRF(w, r) {
		return
	}
	var request organisationReasonRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if err := s.store.ArchiveOrganisation(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), request.Reason, time.Now().UTC()); err != nil {
		writeOrganisationError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listProducts(w http.ResponseWriter, r *http.Request) {
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	products, err := s.store.ListProducts(r.Context(), session.User.ID)
	if err != nil {
		writeProductError(w, err)
		return
	}
	response := make([]productResponse, 0, len(products))
	for _, product := range products {
		response = append(response, productResponseFromDomain(product))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) retireProduct(w http.ResponseWriter, r *http.Request) {
	if !s.verifyMutationCSRF(w, r) {
		return
	}
	var request organisationReasonRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if err := s.store.RetireProduct(r.Context(), session.User.ID, chi.URLParam(r, "productKey"), request.Reason, time.Now().UTC()); err != nil {
		writeProductError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listOrganisationProductEntitlements(w http.ResponseWriter, r *http.Request) {
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	entitlements, err := s.store.ListOrganisationProductEntitlements(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"))
	if err != nil {
		writeProductError(w, err)
		return
	}
	response := make([]organisationProductEntitlementResponse, 0, len(entitlements))
	for _, entitlement := range entitlements {
		response = append(response, organisationProductEntitlementResponseFromDomain(entitlement))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) entitleOrganisation(w http.ResponseWriter, r *http.Request) {
	if !s.verifyMutationCSRF(w, r) {
		return
	}
	var request productEntitlementRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	entitlement, err := s.store.EntitleOrganisation(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), request.ProductKey, request.Reason, time.Now().UTC())
	if err != nil {
		writeProductError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, organisationProductEntitlementResponseFromDomain(entitlement))
}

func (s *Server) revokeOrganisationEntitlement(w http.ResponseWriter, r *http.Request) {
	if !s.verifyMutationCSRF(w, r) {
		return
	}
	var request organisationReasonRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if err := s.store.RevokeOrganisationEntitlement(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), chi.URLParam(r, "entitlementID"), request.Reason, time.Now().UTC()); err != nil {
		writeProductError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listUserProductAssignments(w http.ResponseWriter, r *http.Request) {
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	assignments, err := s.store.ListUserProductAssignments(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"))
	if err != nil {
		writeProductError(w, err)
		return
	}
	response := make([]userProductAssignmentResponse, 0, len(assignments))
	for _, assignment := range assignments {
		response = append(response, userProductAssignmentResponseFromDomain(assignment))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) assignUserProduct(w http.ResponseWriter, r *http.Request) {
	if !s.verifyMutationCSRF(w, r) {
		return
	}
	var request productAssignmentRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	assignment, err := s.store.AssignUserProduct(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), request.UserID, request.ProductKey, request.Reason, time.Now().UTC())
	if err != nil {
		writeProductError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, userProductAssignmentResponseFromDomain(assignment))
}

func (s *Server) revokeUserProduct(w http.ResponseWriter, r *http.Request) {
	if !s.verifyMutationCSRF(w, r) {
		return
	}
	var request organisationReasonRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if err := s.store.RevokeUserProduct(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), chi.URLParam(r, "assignmentID"), request.Reason, time.Now().UTC()); err != nil {
		writeProductError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) proveProductAccess(w http.ResponseWriter, r *http.Request) {
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	access, err := s.store.ProveProductAccess(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), chi.URLParam(r, "productKey"), chi.URLParam(r, "workspaceID"))
	if err != nil {
		writeProductError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, productAccessResponse{Allowed: true, UserID: access.UserID, OrganisationID: access.OrganisationID, ProductKey: access.ProductKey, WorkspaceID: access.WorkspaceID, OrganisationRole: access.OrganisationRole, WorkspaceRole: access.WorkspaceRole})
}

func (s *Server) issueProductAssertion(w http.ResponseWriter, r *http.Request) {
	if !s.verifyMutationCSRF(w, r) {
		return
	}
	var request productAssertionRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if s.cfg.AssertionIssuer == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	access, err := s.store.ProveProductAccess(r.Context(), session.User.ID, chi.URLParam(r, "organisationID"), chi.URLParam(r, "productKey"), chi.URLParam(r, "workspaceID"))
	if err != nil {
		writeProductError(w, err)
		return
	}
	token, claims, err := s.cfg.AssertionIssuer.Issue(assertion.IssueRequest{
		Audience:       request.Audience,
		PlatformUserID: access.UserID,
		OrganisationID: access.OrganisationID,
		WorkspaceID:    access.WorkspaceID,
	}, time.Now().UTC())
	if err != nil {
		writeProductError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, productAssertionResponse{Assertion: token, AssertionVersion: claims.AssertionVersion, IssuedAt: claims.IssuedAt, ExpiresAt: claims.ExpiresAt})
}

func (s *Server) assertionKeys(w http.ResponseWriter, _ *http.Request) {
	if s.cfg.AssertionIssuer == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	keys := s.cfg.AssertionIssuer.PublicKeys()
	response := make([]assertionKeyResponse, 0, len(keys))
	for _, key := range keys {
		response = append(response, assertionKeyResponse{KeyID: key.KeyID, Algorithm: key.Algorithm, PublicKey: base64.RawURLEncoding.EncodeToString(key.PublicKey)})
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) verifyMutationCSRF(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		http.Error(w, "request could not be validated", http.StatusForbidden)
		return false
	}
	token := r.Header.Get("X-CSRF-Token")
	if token == "" && !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		token = r.FormValue("csrf")
	}
	valid, err := s.store.VerifySessionCSRF(r.Context(), cookie.Value, token)
	if err != nil || !valid {
		http.Error(w, "request could not be validated", http.StatusForbidden)
		return false
	}
	return true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, defaultRequestBodyLimit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeOrganisationError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, store.ErrOrganisationNotFound), errors.Is(err, store.ErrOrganisationArchived), errors.Is(err, store.ErrMembershipNotFound), errors.Is(err, store.ErrUnauthorisedOrganisationAction):
		status = http.StatusNotFound
	case errors.Is(err, store.ErrOwnerMutationNotAuthorised):
		status = http.StatusForbidden
	case errors.Is(err, store.ErrDuplicateOrganisationMember):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrInvalidOrganisation), errors.Is(err, domain.ErrInvalidOrganisationName), errors.Is(err, domain.ErrInvalidOrganisationRole), errors.Is(err, domain.ErrInvalidReason):
		status = http.StatusBadRequest
	}
	if status == http.StatusInternalServerError {
		http.Error(w, "service unavailable", status)
		return
	}
	http.Error(w, http.StatusText(status), status)
}

func writeWorkspaceError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, store.ErrAccessScopeDenied), errors.Is(err, store.ErrWorkspaceNotFound), errors.Is(err, store.ErrWorkspaceArchived), errors.Is(err, store.ErrWorkspaceMembershipNotFound), errors.Is(err, store.ErrUnauthorisedWorkspaceAction):
		status = http.StatusNotFound
	case errors.Is(err, store.ErrDuplicateWorkspaceMember):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrInvalidWorkspace), errors.Is(err, domain.ErrInvalidWorkspaceID), errors.Is(err, domain.ErrInvalidWorkspaceName), errors.Is(err, domain.ErrInvalidWorkspaceRole), errors.Is(err, domain.ErrInvalidWorkspaceMember), errors.Is(err, domain.ErrInvalidReason):
		status = http.StatusBadRequest
	}
	if status == http.StatusInternalServerError {
		http.Error(w, "service unavailable", status)
		return
	}
	http.Error(w, http.StatusText(status), status)
}

func writeProductError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, store.ErrProductNotFound), errors.Is(err, store.ErrEntitlementNotFound), errors.Is(err, store.ErrUnauthorisedProductAction), errors.Is(err, store.ErrOrganisationNotFound), errors.Is(err, store.ErrOrganisationArchived), errors.Is(err, store.ErrUnauthorisedOrganisationAction), errors.Is(err, store.ErrProductAssignmentNotFound), errors.Is(err, store.ErrProductAccessDenied):
		status = http.StatusNotFound
	case errors.Is(err, store.ErrProductRetired), errors.Is(err, store.ErrEntitlementInactive), errors.Is(err, store.ErrDuplicateOrganisationEntitlement), errors.Is(err, store.ErrProductAssignmentInactive), errors.Is(err, store.ErrDuplicateUserProductAssignment):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrInvalidProduct), errors.Is(err, domain.ErrInvalidProductKey), errors.Is(err, domain.ErrInvalidProductStatus), errors.Is(err, domain.ErrInvalidEntitlement), errors.Is(err, domain.ErrInvalidEntitlementID), errors.Is(err, domain.ErrInvalidEntitlementStatus), errors.Is(err, domain.ErrInvalidProductAssignment), errors.Is(err, domain.ErrInvalidProductAssignmentID), errors.Is(err, domain.ErrInvalidProductAssignmentStatus), errors.Is(err, domain.ErrInvalidReason), errors.Is(err, assertion.ErrInvalidAudience), errors.Is(err, assertion.ErrInvalidScope), errors.Is(err, assertion.ErrInvalidAssertion):
		status = http.StatusBadRequest
	}
	if status == http.StatusInternalServerError {
		http.Error(w, "service unavailable", status)
		return
	}
	http.Error(w, http.StatusText(status), status)
}

func organisationResponseFromDomain(organisation domain.Organisation) organisationResponse {
	return organisationResponse{ID: organisation.ID, Name: organisation.Name, Status: organisation.Status, CreatedAt: organisation.CreatedAt, ArchivedAt: organisation.ArchivedAt}
}

func workspaceResponseFromDomain(workspace domain.Workspace) workspaceResponse {
	return workspaceResponse{ID: workspace.ID, OrganisationID: workspace.OrganisationID, Name: workspace.Name, Status: workspace.Status, CreatedAt: workspace.CreatedAt, ArchivedAt: workspace.ArchivedAt}
}

func productResponseFromDomain(product domain.Product) productResponse {
	return productResponse{Key: product.Key, DisplayName: product.DisplayName, Status: product.Status, CreatedAt: product.CreatedAt, RetiredAt: product.RetiredAt}
}

func organisationProductEntitlementResponseFromDomain(entitlement domain.OrganisationProductEntitlement) organisationProductEntitlementResponse {
	return organisationProductEntitlementResponse{ID: entitlement.ID, OrganisationID: entitlement.OrganisationID, ProductKey: entitlement.ProductKey, Status: entitlement.Status, Active: entitlement.Active, GrantedBy: entitlement.GrantedBy, GrantedAt: entitlement.GrantedAt, RevokedBy: entitlement.RevokedBy, RevokedAt: entitlement.RevokedAt, RevocationReason: entitlement.RevocationReason}
}

func userProductAssignmentResponseFromDomain(assignment domain.UserProductAssignment) userProductAssignmentResponse {
	return userProductAssignmentResponse{ID: assignment.ID, UserID: assignment.UserID, OrganisationID: assignment.OrganisationID, ProductKey: assignment.ProductKey, Status: assignment.Status, Active: assignment.Active, AssignedBy: assignment.AssignedBy, AssignedAt: assignment.AssignedAt, RevokedBy: assignment.RevokedBy, RevokedAt: assignment.RevokedAt, RevocationReason: assignment.RevocationReason}
}

func (s *Server) mfaSetup(w http.ResponseWriter, r *http.Request) {
	if err := parseBoundedForm(w, r); err != nil {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	csrfCookie, valid, err := s.sessionCSRF(r)
	if err != nil || !valid {
		http.Error(w, "request could not be validated", http.StatusForbidden)
		return
	}
	if session.User.MFAEnabled {
		http.Error(w, "MFA is already enabled", http.StatusConflict)
		return
	}
	secret, err := auth.GenerateTOTPSecret()
	if err != nil {
		s.serverError(w, err)
		return
	}
	enrollmentToken, err := s.store.CreateMFAEnrollment(r.Context(), session.User.ID, secret, time.Now().UTC().Add(10*time.Minute))
	if err != nil {
		s.serverError(w, err)
		return
	}
	if err := render(w, r, templates.MFASetup(session.User, csrfCookie.Value, enrollmentToken, secret, "")); err != nil {
		http.Error(w, "service unavailable", http.StatusInternalServerError)
	}
}

func (s *Server) mfaConfirm(w http.ResponseWriter, r *http.Request) {
	if err := parseBoundedForm(w, r); err != nil {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	csrfCookie, valid, err := s.sessionCSRF(r)
	if err != nil || !valid {
		http.Error(w, "request could not be validated", http.StatusForbidden)
		return
	}
	codes, err := auth.GenerateRecoveryCodes(10)
	if err != nil {
		s.serverError(w, err)
		return
	}
	hashes := make([]string, 0, len(codes))
	for _, code := range codes {
		hash, hashErr := auth.HashPassword(auth.NormalizeRecoveryCode(code))
		if hashErr != nil {
			s.serverError(w, hashErr)
			return
		}
		hashes = append(hashes, hash)
	}
	completed, err := s.store.CompleteMFAEnrollment(r.Context(), session.User.ID, r.FormValue("enrollment_token"), r.FormValue("code"), hashes, time.Now().UTC())
	if err != nil {
		s.serverError(w, err)
		return
	}
	if !completed {
		if err := render(w, r, templates.MFASetup(session.User, csrfCookie.Value, r.FormValue("enrollment_token"), "", "The code was invalid or the setup window expired.")); err != nil {
			http.Error(w, "service unavailable", http.StatusInternalServerError)
		}
		return
	}
	if err := render(w, r, templates.MFARecovery(session.User, csrfCookie.Value, codes)); err != nil {
		http.Error(w, "service unavailable", http.StatusInternalServerError)
	}
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if err := parseBoundedForm(w, r); err != nil {
		return
	}
	session, ok := s.session(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		http.Error(w, "request could not be validated", http.StatusForbidden)
		return
	}
	valid, err := s.store.VerifySessionCSRF(r.Context(), cookie.Value, r.FormValue("csrf"))
	if err != nil || !valid {
		http.Error(w, "request could not be validated", http.StatusForbidden)
		return
	}
	now := time.Now().UTC()
	if err := s.store.RevokeSessionAndRecordLogout(r.Context(), cookie.Value, session.User.ID, now); err != nil {
		s.serverError(w, err)
		return
	}
	setCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode})
	setCookie(w, &http.Cookie{Name: "platform_csrf", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) sessionCSRF(r *http.Request) (*http.Cookie, bool, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, false, err
	}
	csrfCookie, err := r.Cookie("platform_csrf")
	if err != nil {
		return nil, false, err
	}
	valid, err := s.store.VerifySessionCSRF(r.Context(), cookie.Value, r.FormValue("csrf"))
	return csrfCookie, valid, err
}

func (s *Server) requireWorkspaceScope(adminOnly bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, ok := s.session(r)
			if !ok {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}
			scope, err := s.store.ProveWorkspaceScope(r.Context(), session.User.ID, chi.URLParam(r, "workspaceID"), adminOnly)
			if errors.Is(err, store.ErrAccessScopeDenied) {
				// Cross-tenant, stale, archived and tampered identifiers share the
				// same least-knowledge response at the HTTP boundary.
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			if err != nil {
				s.serverError(w, err)
				return
			}
			ctx := context.WithValue(r.Context(), workspaceScopeContextKey{}, scope)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (s *Server) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := s.loadSession(r)
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if err != nil {
			s.serverError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionContextKey{}, session)))
	})
}

func (s *Server) session(r *http.Request) (store.AuthenticatedSession, bool) {
	value, ok := r.Context().Value(sessionContextKey{}).(store.AuthenticatedSession)
	if ok {
		return value, true
	}
	loaded, err := s.loadSession(r)
	return loaded, err == nil
}

func (s *Server) loadSession(r *http.Request) (store.AuthenticatedSession, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return store.AuthenticatedSession{}, sql.ErrNoRows
	}
	return s.store.AuthenticatedSession(r.Context(), cookie.Value, time.Now().UTC())
}

func parseBoundedForm(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, defaultRequestBodyLimit)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return err
	}
	return nil
}

func render(w http.ResponseWriter, r *http.Request, component templ.Component) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return component.Render(r.Context(), w)
}

func setCookie(w http.ResponseWriter, cookie *http.Cookie) { http.SetCookie(w, cookie) }

func (s *Server) serverError(w http.ResponseWriter, err error) {
	_ = err
	http.Error(w, "service unavailable", http.StatusInternalServerError)
}
