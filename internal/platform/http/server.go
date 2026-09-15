package httpserver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Onellan/tockrplatform/internal/auth"
	"github.com/Onellan/tockrplatform/internal/store"
	"github.com/Onellan/tockrplatform/web/templates"
	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
)

const (
	sessionCookieName = "platform_session"
	loginCSRFCookie   = "platform_login_csrf"
)

type Config struct {
	CookieSecure         bool
	AllowInsecureCookies bool
	SessionTTL           time.Duration
	RateLimitEnabled     bool
	StaticDir            string
	LoginLimiter         auth.LoginLimiterConfig
}

type Server struct {
	store   store.PlatformStore
	cfg     Config
	limiter *auth.LoginLimiter
}

type sessionContextKey struct{}

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
	return &Server{store: persistence, cfg: cfg, limiter: auth.NewLoginLimiter(cfg.LoginLimiter)}
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(s.securityHeaders)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(s.cfg.StaticDir))))
	r.Get("/login", s.loginPage)
	r.Post("/login", s.login)
	r.Group(func(protected chi.Router) {
		protected.Use(s.requireSession)
		protected.Get("/", s.accountPage)
		protected.Get("/account", s.accountPage)
		protected.Post("/account/mfa/setup", s.mfaSetup)
		protected.Post("/account/mfa/confirm", s.mfaConfirm)
		protected.Post("/logout", s.logout)
	})
	return r
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
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
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
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
