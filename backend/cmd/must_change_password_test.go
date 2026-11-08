package cmd

import (
	"path/filepath"
	"testing"

	"github.com/gtsteffaniak/filebrowser/backend/internal/app"
	"github.com/gtsteffaniak/filebrowser/backend/internal/state"
	"github.com/gtsteffaniak/filebrowser/backend/internal/utils"
	"github.com/gtsteffaniak/filebrowser/backend/pkg/settings"
)

// TestBootstrapAdminMustChangePasswordLifecycle covers the Qreative fork flag
// lifecycle for the config/env bootstrap admin password:
//   - the admin created on first start is flagged requirePasswordChange;
//   - every startup that resets the password from config/env flags it again;
//   - once the admin changed the password and the config/env password is removed,
//     restarts neither reset the password nor set the flag.
func TestBootstrapAdminMustChangePasswordLifecycle(t *testing.T) {
	t.Setenv("FILEBROWSER_ONLYOFFICE_SECRET", "")
	t.Setenv("FILEBROWSER_DISABLE_AUTOMATIC_BACKUP", "true")
	settings.Initialize("../../_docker/src/sharing/backend/config.yaml")
	settings.Env.IsPlaywright = false
	t.Cleanup(func() { settings.Env.IsPlaywright = false })

	const bootstrap = "bootstrap-password-from-env"
	const chosen = "founder-chosen-password-123"
	origNested := settings.Config.Auth.Methods.PasswordAuth.AdminPassword
	origLegacy := settings.Config.Auth.AdminPassword
	t.Cleanup(func() {
		settings.Config.Auth.Methods.PasswordAuth.AdminPassword = origNested
		settings.Config.Auth.AdminPassword = origLegacy
	})
	settings.Config.Auth.Methods.PasswordAuth.Enabled = true
	settings.Config.Auth.Methods.PasswordAuth.AdminPassword = bootstrap
	settings.Config.Auth.AdminPassword = ""
	adminName := settings.PasswordAdminUsername()

	dbPath := filepath.Join(t.TempDir(), "filebrowser.sqlite")
	settings.Config.Server.DatabaseV2.Path = dbPath
	if _, err := state.Initialize(dbPath); err != nil {
		t.Fatal(err)
	}
	app.MustWireServices(state.Default())
	t.Cleanup(func() { _ = state.Close() })

	admin, err := state.GetUserByUsername(adminName)
	if err != nil {
		t.Fatal(err)
	}
	if !admin.RequirePasswordChange {
		t.Fatal("bootstrap admin created without requirePasswordChange")
	}

	// An existing admin without the flag (e.g. a database from before this
	// feature) gets flagged by the startup reset from config/env.
	admin.RequirePasswordChange = false
	if err = state.UpdateUser(&admin, "", "requirePasswordChange"); err != nil {
		t.Fatal(err)
	}
	validateUserInfo(false)
	admin, err = state.GetUserByUsername(adminName)
	if err != nil {
		t.Fatal(err)
	}
	if !admin.RequirePasswordChange {
		t.Fatal("startup reset from config/env did not set requirePasswordChange")
	}
	if utils.CheckPwd(bootstrap, admin.Password) != nil {
		t.Fatal("startup reset did not apply the config/env password")
	}

	// The admin changes the password (clears the flag), then the env var is removed.
	admin.RequirePasswordChange = false
	if err = state.UpdateUser(&admin, chosen, "password", "requirePasswordChange"); err != nil {
		t.Fatal(err)
	}
	settings.Config.Auth.Methods.PasswordAuth.AdminPassword = ""
	validateUserInfo(false)
	admin, err = state.GetUserByUsername(adminName)
	if err != nil {
		t.Fatal(err)
	}
	if admin.RequirePasswordChange {
		t.Fatal("restart without config/env password set requirePasswordChange")
	}
	if utils.CheckPwd(chosen, admin.Password) != nil {
		t.Fatal("restart without config/env password changed the admin password")
	}
}
