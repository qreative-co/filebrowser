<div align="center">

  [![Codacy Badge](https://app.codacy.com/project/badge/Grade/0b548794f2ac4871a0cf7aa9ecab049f)](https://app.codacy.com/gh/gtsteffaniak/filebrowser/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_grade)
  [![latest version](https://img.shields.io/github/v/release/gtsteffaniak/filebrowser)](https://github.com/gtsteffaniak/filebrowser/releases)
  [![DockerHub Pulls](https://img.shields.io/docker/pulls/gtstef/filebrowser?label=latest%20Docker%20pulls)](https://hub.docker.com/r/gtstef/filebrowser)
  [![Apache-2.0 License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)

  [![Donate](https://www.paypalobjects.com/en_US/i/btn/btn_donate_SM.gif)](https://github.com/gtsteffaniak/filebrowser/wiki/Q&A#is-there-a-way-to-donate-or-support-this-project)

  <img width="150" alt="FileBrowser Quantum logo" src="https://github.com/user-attachments/assets/c40b22c9-33da-47b7-bc4c-ce69bb5cc174">
  <h3>FileBrowser Quantum</h3>
  The best free self-hosted web-based file manager.
  <br/><br/>
  <img width="800" alt="FileBrowser Quantum file listing in dark mode" src="https://filebrowserquantum.com/images/generated/listing/view-mode-normal-dark.jpg">
</div>

> [!NOTE]
> **Qreative fork.** This is [qreative-co/filebrowser](https://github.com/qreative-co/filebrowser), a modified fork of
> [FileBrowser Quantum](https://github.com/gtsteffaniak/filebrowser) v2.1.4-beta under the Apache-2.0 licence (see
> [LICENSE](LICENSE) and [NOTICE](NOTICE)). Our only functional change is **"must change password at first login"**:
> see [Qreative fork: must change password](#qreative-fork-must-change-password) below. Image:
> `ghcr.io/qreative-co/filebrowser:<version>-qreative.<n>`.

## Pinned

:pushpin: [Read The Official Docs](https://filebrowserquantum.com/) (currently english-only)

:pushpin: [v2.0.0 is now stable!](https://filebrowserquantum.com/en/docs/getting-started/v2/about/) --> See [migration support discussion](https://github.com/gtsteffaniak/filebrowser/discussions/3105)

## Qreative fork: must change password

What the fork adds on top of upstream's `requirePasswordChange` user flag:

- **Set automatically for the bootstrap admin.** When the admin account is created on first start with password login
  enabled, and every time startup resets the admin password from `auth.methods.password.adminPassword` or the
  `FILEBROWSER_ADMIN_PASSWORD` environment variable, the admin is flagged `requirePasswordChange`. An admin can set or
  clear the flag for any user (user settings: "Require password change at next login", or
  `PATCH /api/users` with `which: ["requirePasswordChange"]`).
- **Enforced by the backend.** Login with the temporary password does not issue a session (upstream behaviour:
  `POST /api/auth/login` answers 403). Any session or API token that already exists for a flagged password user can only
  call `GET /api/users?username=self` and `POST /api/auth/logout`; every other authenticated API answers
  `403 {"status":403,"message":"password change is required for user","mustChangePassword":true}`, and optional-auth
  routes (share pages) treat the user as anonymous. The password is changed with
  `POST /api/auth/password/change-required?username=<user>` (header `X-Password`: current password, URL-encoded; body
  `{"password": "...", "passwordConfirm": "..."}`), which re-checks the current password, enforces
  `auth.methods.password.minLength`, sets the new password, clears the flag, ends every existing session of the user
  (sessions obtained with the temporary password do not survive) and issues a fresh session.
- **Web UI.** After login (or on any page load with a flagged session) a pinned prompt that cannot be closed asks for the
  current password and the new password twice, enforces `minLength`, calls the endpoint above and clears the flag.

### Removing the bootstrap password env var afterwards

While `FILEBROWSER_ADMIN_PASSWORD` (or `adminPassword` in config) is set, **every restart resets the admin password to
it and sets the flag again**. After the admin has logged in and chosen a new password:

1. In Coolify open the service (here: project INFRASTRUCTURE, service `media-library`) > Environment Variables and
   **delete** `FILEBROWSER_ADMIN_PASSWORD` (never replace it with another value: any non-empty value is applied as the
   admin password on the next start). Also remove the `FILEBROWSER_ADMIN_PASSWORD=${FILEBROWSER_ADMIN_PASSWORD}` line
   from the `files` service environment in the compose so Coolify does not recreate the variable, and make sure
   `config.yaml` has no `adminPassword`.
2. **Restart** the service (Coolify > Restart, or `POST /api/v1/services/<uuid>/restart`).
3. Check the container log no longer shows `Resetting admin user to default username and password.` and that the new
   password still works.

From then on restarts keep the password the admin chose.

## About

FileBrowser Quantum provides an easy way to access and manage your files from the web. It has a modern responsive interface that has many advanced features to manage users, access, sharing, and file preview and editing.

This version is called "Quantum" because it packs tons of advanced features into a tiny and easy-to-run file. Unlike the majority of alternative options, FileBrowser Quantum is simple to install and easy to configure.

The goal for this repo is to become the best open-source self-hosted file browsing application that exists -- **all for free**. This repo will always be free and open-source.

Ready to try it out? See [Getting Started Docs](https://filebrowserquantum.com/en/docs/getting-started/).

## How its different

FileBrowser Quantum is a massive fork of the file browser open-source project with the following changes:

  1. ✅ Better source configuration - multiple sources, include/exclude rules, and [more](https://filebrowserquantum.com/en/docs/configuration/sources/)
  2. ✅ Login support for OIDC, LDAP, JWT, password + 2FA, and proxy.
  3. ✅ Beautiful, Responsive, and Customizable user interface.
  4. ✅ Streamlined configuration via `config.yaml` config file.
  5. ✅ Efficient search
     - Real-time search results as you type.
     - Real-time monitoring and updates in the UI.
     - Search supports file and folder sizes, along with various filters.
  6. ✅ Better listing browsing
     - Thumbnails support includes **office**, **video**, and **album artwork**, and **3D models**.
     - Faster and more responsive views with animations.
     - **Folder sizes** are displayed and support thumbnails
     - Navigating remembers the last scroll position.
     - WebDAV support
     - Granular permissions
  7. ✅ Highly configurable and [customizable sharing options](https://filebrowserquantum.com/en/docs/shares/options/)
     - share expiration time
     - users who can access share (including anonymous)
     - styling and themes
     - file viewing, editing, and uploading permissions
  8. ✅ Access control that can be scoped to user or group and source path.
  9. ✅ Developer API support
     - Ability to create long-lived API Tokens.
     - A helpful Swagger page is available at `/swagger` endpoint for API enabled users.

Notable features that this fork *does not* have (removed):
 - ❌ shell commands are completely removed and will not be returned.

FileBrowser Quantum differs significantly from the original version. Many of these changes required a significant overhaul. Creating a fork was a necessary process to make the program better. There have been many growing pains, but a stable release is planned and coming soon.

## The UI

The UI has a simple three-component navigation system:

  1. (Left) Multi-action button with slide-out panel.
  2. (Middle) The powerful search bar / title
  3. (Right) The view change toggle / overflow menu

All other functions are moved either into the action menu or pop-up menus.
If the action does not depend on context, it will exist in the slide-out
action panel. If the action is available based on context, it will show up as
a pop-up menu.

<img width="1200" height="750" alt="Aug-07-2026 14-54-55" src="https://github.com/user-attachments/assets/fd1c1747-f30f-4e21-b991-f45744416d7e" />

## Official Docs

See the [Official Docs](https://filebrowserquantum.com/). Contributions are welcome and encouraged! See [FilebrowserDocs Github](https://github.com/quantumx-apps/filebrowserDocs).

## Comparison Chart
Application Name | <img width="48" alt="" src="https://github.com/user-attachments/assets/c40b22c9-33da-47b7-bc4c-ce69bb5cc174" > Quantum | <img width="48" alt="" src="https://github.com/filebrowser/filebrowser/blob/master/frontend/public/img/logo.svg" > Filebrowser | <img width="48" alt="" src="https://github.com/mickael-kerjean/filestash/blob/master/public/assets/logo/app_icon.png?raw=true" > Filestash | <img width="48" alt="" src="https://avatars.githubusercontent.com/u/19211038?s=200&v=4" >  Nextcloud | <img width="48" alt="" src="https://cdn.iconscout.com/icon/free/png-256/free-google-drive-logo-icon-svg-download-png-2476481.png" > Google_Drive | <img width="48" alt="" src="https://avatars.githubusercontent.com/u/6422152?v=4" > FileRun
--- | --- | --- | --- | --- | --- | --- |
Filesystem support            | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ |
Linux                         | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ |
Windows                       | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ |
Mac                           | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ |
Self hostable                 | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ |
Has Stable Release?           | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
S3 support                    | ❌ | ❌ | ✅ | ✅ | ❌ | ❌ |
webdav support                | ✅ | ❌ | ✅ | ✅ | ❌ | ✅ |
FTP support                   | ❌ | ❌ | ✅ | ✅ | ❌ | ✅ |
Dedicated docs site?          | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
Multiple sources at once      | ✅ | ❌ | ✅ | ✅ | ❌ | ✅ |
Docker image size             | 180 MB (with ffmpeg) | 31 MB  | 240 MB (main image) | 250 MB | ❌ | > 2 GB |
Min. Memory Requirements      | 512 MB | 128 MB | 128 MB (main image) | 512 MB | ❌ | 512 MB   |
has standalone binary         | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
price                         | free | free | free | free tier | free tier | $99+ |
rich media preview            | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
Upload files from the web?    | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
Advanced Search?              | ✅ | ❌ | ✅ | configurable | ✅ | ✅ |
Indexed Search?               | ✅ | ❌ | ✅ | configurable | ✅ | ✅ |
Content-aware search?         | ❌ | ❌ | ✅ | configurable | ✅ | ✅ |
Custom job support            | :construction: | ✅ | ✅ | ✅ | ❌ | ✅ |
Multiple users                | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
Single sign-on support        | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ |
LDAP sign-on support          | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ |
Long-live API key support     | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ |
API documentation page        | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ |
Mobile App                    | ❌ | ❌ | ❌ | ✅ | ✅ | ❌ |
open source?                  | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
tags support                  | :construction: | ❌ | ✅ | ✅ | ❌ | ✅ |
shareable web links?          | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
Event-based notifications     | :construction: | ❌ | ✅ | ❌ | ❌ | ✅ |
Metrics                       | :construction: | ❌ | ✅ | ✅ | ❌ | ❌ |
file space quotas             | ✅ | ❌ | ✅ | ❌ | ✅ | ✅ |
text-based files editor       | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
Office file support           | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ |
Office file previews          | ✅ | ❌ | ❌ | ✅ | ✅ | ✅ |
Themes                        | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ |
Branding support              | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ |
activity log                  | ✅ | ❌ | ✅ | ✅ | ✅ | ✅
Comments support              | ❌ | ❌ | ✅ | ✅ | ✅ | ✅ |
trash support                 | :construction: | ❌ | ✅ | ✅ | ✅ | ✅ |
Starred/pinned files          | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ |
Chromecast support            | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ |
Share collections of files    | :construction: | ❌ | ❌ | ❌ | ❌ | ✅ |
Can archive selected files    | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ |
Can browse archive files      | :construction: | ❌ | ✅ | ❌ | ❌ | ✅ |
Can convert documents         | :construction: | ❌ | ✅ | ❌ | ❌ | ✅ |
Can convert videos            | :construction: | ❌ | ✅ | ❌ | ❌ | ❌ |
Can convert photos            | :construction: | ❌ | ✅ | ❌ | ❌ | ❌ |
