/** biome-ignore-all lint/correctness/noUnresolvedImports: false positives */
// Qreative fork: locked prompts (the in-session "must change password" prompt)
// cannot be closed by the user, only by code passing { force: true }.

import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.unmock('@/store');

import { mutations } from './mutations.ts';
import { state } from './state.ts';

describe('locked prompts', () => {
  beforeEach(() => {
    state.prompts = [];
  });

  it('ignores closeTopPrompt without force and closes with force', () => {
    mutations.showPrompt({ name: 'requirePasswordChange', pinned: true, locked: true });
    const [prompt] = state.prompts;
    expect(prompt.locked).toBe(true);

    mutations.closeTopPrompt();
    mutations.closeTopPrompt(prompt.id);
    expect(state.prompts).toHaveLength(1);

    mutations.closeTopPrompt(prompt.id, { force: true });
    expect(state.prompts).toHaveLength(0);
  });

  it('leaves ordinary pinned prompts closable', () => {
    mutations.showPrompt({ name: 'generic', pinned: true });
    expect(state.prompts[0].locked).toBe(false);
    mutations.closeTopPrompt();
    expect(state.prompts).toHaveLength(0);
  });
});
