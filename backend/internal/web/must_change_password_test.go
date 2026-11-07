package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gtsteffaniak/filebrowser/backend/internal/database/users"
	"github.com/gtsteffaniak/filebrowser/backend/internal/state"
	"github.com/gtsteffaniak/filebrowser/backend/pkg/settings"
)

const mcpTempPassword = "temporary-bootstrap-pass"

func mcpEnablePasswordAuth(t *testing.T) {
	t.Helper()
	orig := settings.Config.Auth.Methods.PasswordAuth.Enabled
	settings.Config.Auth.Methods.PasswordAuth.Enabled = true
	t.Cleanup(func() { settings.Config.Auth.Methods.PasswordAuth.Enabled = orig })
}

// mcpCreateUser creates a password-login admin and returns it as stored.
func mcpCreateUser(t *testing.T, username string, mustChange bool) users.User {
	t.Helper()
	user := &users.User{
		FrontendUser: users.FrontendUser{
			Username:              username,
			LoginMethod:           users.LoginMethodPassword,
			RequirePasswordChange: mustChange,
			Permissions:           users.Permissions{Admin: true},
		},
	}
	if err := state.CreateUser(user, mcpTempPassword); err != nil {
		t.Fatalf("create user: %v", err)
	}
	stored, err := state.GetUserByUsername(username)
	if err != nil {
		t.Fatal(err)
	}
	if stored.RequirePasswordChange != mustChange {
		t.Fatalf("stored requirePasswordChange = %v, want %v", stored.RequirePasswordChange, mustChange)
	}
	return stored
}

func mcpAuthedRequest(method, target, token string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func mcpDecode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	return body
}

func mcpAssertBlocked(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body %s)", rec.Code, rec.Body.String())
	}
	if got := mcpDecode(t, rec)["mustChangePassword"]; got != true {
		t.Fatalf("mustChangePassword = %v, want true (body %s)", got, rec.Body.String())
	}
}

// probeHandler records whether the protected handler ran and as whom.
type probeHandler struct {
	called   bool
	username string
}

func (p *probeHandler) handle(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	p.called = true
	if d.User != nil {
		p.username = d.User.Username
	}
	return http.StatusOK, nil
}

func TestMustChangePasswordBlocksAuthenticatedAPIs(t *testing.T) {
	setupTestEnv(t)
	mcpEnablePasswordAuth(t)
	user := mcpCreateUser(t, "mcp-blocked", true)
	token := testSessionToken(t, &user, 0)

	for _, wrap := range []struct {
		name string
		mk   func(handleFunc) http.HandlerFunc
	}{
		{"withUser", withUser},
		{"withAdmin", withAdmin},
		{"withPermShare", withPermShare},
	} {
		t.Run(wrap.name, func(t *testing.T) {
			probe := &probeHandler{}
			rec := httptest.NewRecorder()
			wrap.mk(probe.handle)(rec, mcpAuthedRequest(http.MethodGet, "/api/resources?path=/", token))
			mcpAssertBlocked(t, rec)
			if probe.called {
				t.Fatal("protected handler ran for a user who must change their password")
			}
		})
	}

	// WebDAV basic auth (token as password) goes through the same gate.
	probe := &probeHandler{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dav/srv/", nil)
	req.SetBasicAuth("mcp-blocked", token)
	withBasicAuth(probe.handle)(rec, req)
	if rec.Code != http.StatusForbidden || probe.called {
		t.Fatalf("webdav status = %d called = %v, want 403 and not called", rec.Code, probe.called)
	}
}

func TestMustChangePasswordOptionalAuthRoutesSeeAnonymous(t *testing.T) {
	setupTestEnv(t)
	mcpEnablePasswordAuth(t)
	user := mcpCreateUser(t, "mcp-optional", true)
	token := testSessionToken(t, &user, 0)

	probe := &probeHandler{}
	rec := httptest.NewRecorder()
	withOrWithoutUser(probe.handle)(rec, mcpAuthedRequest(http.MethodGet, "/api/share/info", token))
	if !probe.called {
		t.Fatalf("optional-auth handler not called (status %d)", rec.Code)
	}
	if probe.username != users.AnonymousUserName {
		t.Fatalf("optional-auth route saw user %q, want anonymous", probe.username)
	}
}

func TestMustChangePasswordAllowsOwnUserOnly(t *testing.T) {
	setupTestEnv(t)
	mcpEnablePasswordAuth(t)
	user := mcpCreateUser(t, "mcp-self", true)
	mcpCreateUser(t, "mcp-other", false)
	token := testSessionToken(t, &user, 0)
	route := userGetRoute()

	for _, target := range []string{"/api/users?username=self", "/api/users?username=mcp-self"} {
		rec := httptest.NewRecorder()
		route(rec, mcpAuthedRequest(http.MethodGet, target, token))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d body = %s", target, rec.Code, rec.Body.String())
		}
		body := mcpDecode(t, rec)
		if body["username"] != "mcp-self" || body["requirePasswordChange"] != true {
			t.Fatalf("GET %s body = %s", target, rec.Body.String())
		}
	}

	for _, target := range []string{"/api/users", "/api/users?username=mcp-other"} {
		rec := httptest.NewRecorder()
		route(rec, mcpAuthedRequest(http.MethodGet, target, token))
		mcpAssertBlocked(t, rec)
	}
}

func TestMustChangePasswordAllowsLogout(t *testing.T) {
	setupTestEnv(t)
	mcpEnablePasswordAuth(t)
	user := mcpCreateUser(t, "mcp-logout", true)
	token := testSessionToken(t, &user, 0)

	rec := httptest.NewRecorder()
	logoutRoute()(rec, mcpAuthedRequest(http.MethodPost, "/api/auth/logout", token))
	if rec.Code != http.StatusOK {
		t.Fatalf("logout status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !state.IsTokenRevoked(token) {
		t.Fatal("logout did not revoke the session token")
	}
}

func TestMustChangePasswordLoginReturnsFlag(t *testing.T) {
	setupTestEnv(t)
	mcpEnablePasswordAuth(t)
	mcpCreateUser(t, "mcp-login", true)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login?username=mcp-login", nil)
	req.Header.Set("X-Password", mcpTempPassword)
	wrapHandler(loginHelper(loginHandler))(rec, req)
	mcpAssertBlocked(t, rec)
	if cookies := rec.Result().Cookies(); len(cookies) > 0 {
		t.Fatalf("login issued cookies while a password change is required: %v", cookies)
	}
}

func TestMustChangePasswordDoesNotAffectOtherUsers(t *testing.T) {
	setupTestEnv(t)
	mcpEnablePasswordAuth(t)
	user := mcpCreateUser(t, "mcp-normal", false)
	token := testSessionToken(t, &user, 0)

	probe := &probeHandler{}
	rec := httptest.NewRecorder()
	withUser(probe.handle)(rec, mcpAuthedRequest(http.MethodGet, "/api/resources?path=/", token))
	if rec.Code != http.StatusOK || !probe.called {
		t.Fatalf("status = %d called = %v, want 200 and called", rec.Code, probe.called)
	}
}

// TestMustChangePasswordLifecycle: flag set -> blocked; password changed via the
// change-required endpoint -> flag cleared, the old password stops working and
// the session is usable again.
func TestMustChangePasswordLifecycle(t *testing.T) {
	setupTestEnv(t)
	mcpEnablePasswordAuth(t)
	origMin := settings.Config.Auth.Methods.PasswordAuth.MinLength
	settings.Config.Auth.Methods.PasswordAuth.MinLength = 12
	t.Cleanup(func() { settings.Config.Auth.Methods.PasswordAuth.MinLength = origMin })

	user := mcpCreateUser(t, "mcp-life", true)
	token := testSessionToken(t, &user, 0)

	probe := &probeHandler{}
	rec := httptest.NewRecorder()
	withUser(probe.handle)(rec, mcpAuthedRequest(http.MethodGet, "/api/resources?path=/", token))
	mcpAssertBlocked(t, rec)

	change := func(current, next string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"password": next, "passwordConfirm": next})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/password/change-required?username=mcp-life", bytes.NewReader(body))
		req.Header.Set("X-Password", current)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		wrapHandler(withoutUserHelper(changeRequiredPasswordHandler))(rec, req)
		return rec
	}

	// Below minLength is refused and the flag stays set.
	if rec := change(mcpTempPassword, "short-pass"); rec.Code != http.StatusBadRequest {
		t.Fatalf("short password status = %d, want 400", rec.Code)
	}
	// Wrong current password is refused.
	if rec := change("not-the-current-password", "a-much-longer-new-password"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong current password status = %d, want 401", rec.Code)
	}
	if u, _ := state.GetUserByUsername("mcp-life"); !u.RequirePasswordChange {
		t.Fatal("flag cleared by a failed change")
	}

	if rec := change(mcpTempPassword, "a-much-longer-new-password"); rec.Code != http.StatusOK {
		t.Fatalf("change status = %d body = %s", rec.Code, rec.Body.String())
	}
	updated, err := state.GetUserByUsername("mcp-life")
	if err != nil {
		t.Fatal(err)
	}
	if updated.RequirePasswordChange {
		t.Fatal("flag still set after the password change")
	}

	probe = &probeHandler{}
	rec = httptest.NewRecorder()
	withUser(probe.handle)(rec, mcpAuthedRequest(http.MethodGet, "/api/resources?path=/", token))
	if rec.Code != http.StatusOK || !probe.called {
		t.Fatalf("after change: status = %d called = %v, want 200 and called", rec.Code, probe.called)
	}

	// The bootstrap password no longer logs in; the new one does.
	login := func(pw string) int {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login?username=mcp-life", nil)
		req.Header.Set("X-Password", pw)
		rec := httptest.NewRecorder()
		wrapHandler(loginHelper(loginHandler))(rec, req)
		return rec.Code
	}
	if code := login(mcpTempPassword); code != http.StatusUnauthorized {
		t.Fatalf("login with old password status = %d, want 401", code)
	}
	if code := login("a-much-longer-new-password"); code != http.StatusOK {
		t.Fatalf("login with new password status = %d, want 200", code)
	}

	// An admin can set the flag again; it is enforced immediately on live sessions.
	again := updated
	again.RequirePasswordChange = true
	if err := state.UpdateUser(&again, "", "requirePasswordChange"); err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	withUser((&probeHandler{}).handle)(rec, mcpAuthedRequest(http.MethodGet, "/api/resources?path=/", token))
	mcpAssertBlocked(t, rec)
}
