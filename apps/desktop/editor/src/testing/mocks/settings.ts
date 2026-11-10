/**
 * The settings entity's part of the mocks: the settings of the vault, and the themes the window
 * lists. The theme module is mocked together with the door that passes it on.
 */
import { vi } from 'vitest'
import { requests } from '../requests'
import { getSettingInForce, keepSetting, mockVaultState, pendingStream } from '../answers'
import type { VaultCore } from '@/app/vault/core'
import type * as SettingsModule from '@/entities/settings'

export const settingsCoreMock = {
  getSettings: () =>
    Promise.resolve({
      written: '{}',
      path: '/vaults/Physics/.numen/settings.json',
      models: [],
      plugins: [],
      app: mockVaultState.app.map(getSettingInForce),
    }),
  updateSettings: (edits: readonly { at: readonly string[]; value: string }[]) => {
    if (mockVaultState.writeError) return Promise.reject(new Error(mockVaultState.writeError))
    for (const one of edits) {
      requests.settingsWritten.push(`${one.at.join('.')} ${one.value}`)
      keepSetting(one.at.join('.'), JSON.parse(one.value))
    }
    return Promise.resolve()
  },
  watchSettings: pendingStream,
} satisfies Partial<VaultCore>

const themesResponse = {
  themes: {
    getAppearance: () =>
      Promise.resolve({
        themes: [
          { name: 'preset:numen', title: 'numen', isBuiltIn: true, pluginId: '', isPinned: false },
          { name: 'mine:sea', title: 'sea', isBuiltIn: false, pluginId: '', isPinned: false },
        ],
        applied: mockVaultState.applied,
        mode: mockVaultState.mode,
        sizes: mockVaultState.sizes,
        bounds: { interfaceScale: { min: 0.8, max: 2 }, textScale: { min: 0.8, max: 1.75 } },
      }),
    readTheme: (name: string) => Promise.resolve(`:root { --numen-surface: ${name} }`),
    watchThemes: pendingStream,
  },
}

vi.mock('@/entities/settings/api/theme', () => themesResponse)

vi.mock('@/entities/settings', async (original) => ({
  ...(await original<typeof SettingsModule>()),
  ...themesResponse,
}))
