/** biome-ignore-all lint/correctness/noUnresolvedImports: false positives */
// Qreative fork: in-session "must change password" prompt.

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createApp } from 'vue';

const { store, router, notify, globals } = vi.hoisted(() => {
  const state = { route: { query: {} }, user: null, sessionId: 'test-session' };
  return {
    store: {
      state,
      getters: { isShare: () => false, isLoggedIn: () => state.user !== null },
      mutations: {
        showPrompt: vi.fn(),
        closeTopPrompt: vi.fn(),
        updateCurrentUser: vi.fn(),
        setCurrentUser: vi.fn((user) => { state.user = user; }),
        setSession: vi.fn(),
        syncEnforcedUserDefaults: vi.fn(),
        syncSidebarLinkDefaultsPolicy: vi.fn(),
        syncToolAccessDefaultsPolicy: vi.fn(),
        syncShareDefaultsPolicy: vi.fn(),
      },
    },
    router: { push: vi.fn() },
    notify: { showError: vi.fn(), showSuccessToast: vi.fn() },
    globals: { baseURL: '/', recaptcha: false, passwordMinLength: 12 },
  };
});

vi.mock('@/store', () => store);
vi.mock('@/store/state.ts', () => ({ state: store.state }));
vi.mock('@/store/mutations.ts', () => ({ mutations: store.mutations }));
vi.mock('@/router', () => ({ default: router, router }));
vi.mock('@/notify', () => ({ notify }));
vi.mock('@/utils/constants', () => ({ globalVars: globals }));
vi.mock('@/utils/constants.js', () => ({ globalVars: globals }));
vi.mock('@/api/utils', () => ({ fetchJSON: vi.fn(), fetchURL: vi.fn() }));
vi.mock('@/api', async () => ({ authApi: await import('@/api/auth.js') }));

import { mustChangePassword, stopSessionKeepAlive } from '@/utils/auth.js';
import RequirePasswordChange from './RequirePasswordChange.vue';

const current = 'bootstrap password &+%';
const strong = 'a-new-strong-password';
const user = { username: 'qreative', loginMethod: 'password', requirePasswordChange: false };
let app;
let fetchMock;
let reloadMock;

function response(status, body = {}) {
  return { status, ok: status === 200, text: async () => JSON.stringify(body), json: async () => body };
}

function mountPrompt(props) {
  app = createApp(RequirePasswordChange, { username: user.username, promptId: 7, ...props });
  app.config.globalProperties.$t = (key) => key;
  app.config.globalProperties.$router = router;
  app.directive('focus', {});
  return app.mount(document.createElement('div'));
}

describe('mustChangePassword', () => {
  it('is true only for flagged password-login users with auth enabled', () => {
    expect(mustChangePassword({ loginMethod: 'password', requirePasswordChange: true })).toBe(true);
    expect(mustChangePassword({ loginMethod: 'password', requirePasswordChange: true }, true)).toBe(false);
    expect(mustChangePassword({ loginMethod: 'password', requirePasswordChange: false })).toBe(false);
    expect(mustChangePassword({ loginMethod: 'oidc', requirePasswordChange: true })).toBe(false);
    expect(mustChangePassword(null)).toBe(false);
  });
});

describe('RequirePasswordChange prompt (in-session)', () => {
  beforeEach(() => {
    stopSessionKeepAlive();
    vi.clearAllMocks();
    store.state.user = null;
    fetchMock = vi.fn();
    vi.stubGlobal('fetch', (url, options) => {
      if (new URL(url, 'http://localhost').pathname === '/api/auth/renew') {
        return Promise.resolve(response(200));
      }
      return fetchMock(url, options);
    });
    reloadMock = vi.fn();
    vi.stubGlobal('location', { ...window.location, reload: reloadMock, href: 'http://localhost/files/' });
    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    stopSessionKeepAlive();
    app?.unmount();
    app = undefined;
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('asks for the current password when opened without one', () => {
    const prompt = mountPrompt({});
    expect(prompt.askCurrentPassword).toBe(true);
    expect(prompt.$el.parentNode.querySelector('#current-password')).not.toBeNull();
  });

  it('enforces minLength, matching confirmation and a different password', () => {
    const prompt = mountPrompt({});
    prompt.currentPassword = current;
    prompt.newPassword = 'short-pass';
    prompt.passwordConfirm = 'short-pass';
    expect(prompt.canSubmit).toBe(false); // 10 < 12

    prompt.newPassword = strong;
    prompt.passwordConfirm = `${strong}x`;
    expect(prompt.canSubmit).toBe(false); // mismatch

    prompt.newPassword = current;
    prompt.passwordConfirm = current;
    expect(prompt.canSubmit).toBe(false); // same as current

    prompt.currentPassword = '';
    prompt.newPassword = strong;
    prompt.passwordConfirm = strong;
    expect(prompt.canSubmit).toBe(false); // no current password

    prompt.currentPassword = current;
    expect(prompt.canSubmit).toBe(true);
  });

  it('changes the password, refreshes the user and closes the locked prompt', async () => {
    fetchMock
      .mockResolvedValueOnce(response(200))
      .mockResolvedValueOnce(response(200, user));
    const prompt = mountPrompt({});
    prompt.currentPassword = current;
    prompt.newPassword = strong;
    prompt.passwordConfirm = strong;
    await prompt.submit();

    const [url, options] = fetchMock.mock.calls[0];
    const parsed = new URL(url, 'http://localhost');
    expect(parsed.pathname).toBe('/api/auth/password/change-required');
    expect(parsed.searchParams.get('username')).toBe(user.username);
    expect(options).toMatchObject({ method: 'POST', headers: { 'X-Password': encodeURIComponent(current) } });
    expect(JSON.parse(options.body)).toEqual({ password: strong, passwordConfirm: strong });
    expect(store.mutations.setCurrentUser).toHaveBeenCalledWith(user);
    expect(store.mutations.closeTopPrompt).toHaveBeenCalledWith(7, { force: true });
    expect(reloadMock).toHaveBeenCalledOnce();
  });

  it('keeps the prompt open when the server refuses the change', async () => {
    fetchMock.mockResolvedValueOnce(response(401, { message: 'user unauthorized' }));
    const prompt = mountPrompt({});
    prompt.currentPassword = 'wrong current password';
    prompt.newPassword = strong;
    prompt.passwordConfirm = strong;
    await prompt.submit();

    expect(prompt.error).toBe('user unauthorized');
    expect(store.mutations.closeTopPrompt).not.toHaveBeenCalled();
    expect(reloadMock).not.toHaveBeenCalled();
  });

  it('keeps the login-form flow (password passed in) without a current-password field', async () => {
    fetchMock
      .mockResolvedValueOnce(response(200))
      .mockResolvedValueOnce(response(200, user));
    const prompt = mountPrompt({ password: current, redirect: '/files/' });
    expect(prompt.askCurrentPassword).toBe(false);
    prompt.newPassword = strong;
    prompt.passwordConfirm = strong;
    await prompt.submit();
    expect(router.push).toHaveBeenCalledWith('/files/');
    expect(reloadMock).not.toHaveBeenCalled();
  });
});
