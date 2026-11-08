package web

// Qreative fork: server-side enforcement of the per-user requirePasswordChange
// ("must change password") flag for already-authenticated requests.
//
// Upstream already refuses to issue a new session at login while the flag is
// set (POST /api/auth/login returns 403 and the user completes the change via
// POST /api/auth/password/change-required). This file closes the remaining
// gap: a session or API token that existed before the flag was set (or a token
// minted any other way) must not keep working. While the flag is set, an
// authenticated password user may only fetch their own user record and log
// out; every other authenticated API answers 403 {"mustChangePassword": true}.
// The password change itself goes through /api/auth/password/change-required,
// which re-verifies the current password and clears the flag.

import (
	"net/http"
	"strings"

	"github.com/gtsteffaniak/filebrowser/backend/internal/database/users"
	"github.com/gtsteffaniak/filebrowser/backend/internal/errors"
)

// userMustChangePassword reports whether u is a password-login user whose
// account is flagged requirePasswordChange.
func userMustChangePassword(u *users.User) bool {
	return u != nil && u.RequirePasswordChange && u.LoginMethod == users.LoginMethodPassword
}

// passwordChangeGate blocks an authenticated request when the user must change
// their password and the route has not been allowlisted with
// allowPendingPasswordChange.
func passwordChangeGate(data *requestContext) (int, error) {
	if data == nil || data.allowPendingPasswordChange {
		return 0, nil
	}
	if userMustChangePassword(data.User) {
		return http.StatusForbidden, errors.ErrPasswordChangeRequired
	}
	return 0, nil
}

// allowPendingPasswordChange marks a route as reachable while the
// authenticated user still has to change their password. It must wrap the
// authentication middleware (withUserHelper / withOrWithoutUserHelper).
func allowPendingPasswordChange(fn handleFunc) handleFunc {
	return func(w http.ResponseWriter, r *http.Request, data *requestContext) (int, error) {
		data.allowPendingPasswordChange = true
		return fn(w, r, data)
	}
}

// selfOnlyWhilePasswordChangePending restricts GET /users to the caller's own
// record while the caller must change their password (no user listing or
// lookups of other accounts).
func selfOnlyWhilePasswordChangePending(fn handleFunc) handleFunc {
	return func(w http.ResponseWriter, r *http.Request, data *requestContext) (int, error) {
		if userMustChangePassword(data.User) {
			q := r.URL.Query()
			username := strings.TrimSpace(q.Get("username"))
			if strings.TrimSpace(q.Get("id")) != "" || (username != "self" && username != data.User.Username) {
				return http.StatusForbidden, errors.ErrPasswordChangeRequired
			}
		}
		return fn(w, r, data)
	}
}

// userGetRoute is GET /users: reachable while a password change is pending,
// but then limited to the caller's own record.
func userGetRoute() http.HandlerFunc {
	return wrapHandler(allowPendingPasswordChange(withUserHelper(selfOnlyWhilePasswordChangePending(userGetHandler))))
}

// logoutRoute is POST /auth/logout: always reachable so a user who must change
// their password can still end the session.
func logoutRoute() http.HandlerFunc {
	return wrapHandler(allowPendingPasswordChange(withOrWithoutUserHelper(withRateLimitChain(AuthRateLimitModerate, logoutHandler))))
}
