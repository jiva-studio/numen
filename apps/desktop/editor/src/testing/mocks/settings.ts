/**
 * The settings entity's part of the answers: the settings of the vault, the themes the window
 * lists and the plugin catalog, answered to the real clients over the router transport.
 */
import { THEME_MODE, THEME_ORIGIN } from '@numen/core-client'
import { createLostConnection } from '@numen/core-client/testing'
import { requests } from '../requests'
import { getSettingInForce, keepSetting, mockVaultState, pendingStream } from '../answers'
import { toWireSetting } from '../settingsWire'

const MODE_CODES = {
  system: THEME_MODE.SYSTEM,
  light: THEME_MODE.LIGHT,
  dark: THEME_MODE.DARK,
} as const

export const settingsAnswers = {
  settingsService: {
    getSettings: () => ({
      written: '{}',
      path: '/vaults/Physics/.numen/settings.json',
      plugins: [],
      app: mockVaultState.app.map(getSettingInForce).map(toWireSetting),
    }),
    writeSettings: (request: { settings: readonly { at: readonly string[]; value: string }[] }) => {
      if (mockVaultState.writeError) throw createLostConnection(mockVaultState.writeError)
      for (const one of request.settings) {
        requests.settingsWritten.push(`${one.at.join('.')} ${one.value}`)
        keepSetting(one.at.join('.'), JSON.parse(one.value))
      }
      return {}
    },
    watchSettings: pendingStream,
  },
  theme: {
    listThemes: () => ({
      themes: [
        { name: 'preset:numen', title: 'numen', origin: THEME_ORIGIN.PRESET },
        { name: 'mine:sea', title: 'sea', origin: THEME_ORIGIN.MINE },
      ],
      applied: mockVaultState.applied,
      mode: MODE_CODES[mockVaultState.mode],
      interfaceScale: mockVaultState.sizes.interfaceScale,
      textScale: mockVaultState.sizes.textScale,
      interfaceScaleBounds: { min: 0.8, max: 2 },
      textScaleBounds: { min: 0.8, max: 1.75 },
    }),
    readTheme: (request: { name: string }) => ({
      css: `:root { --numen-surface: ${request.name} }`,
    }),
    watchThemes: pendingStream,
  },
  pluginCatalog: {
    listCatalog: () => ({ entries: [], sourceErrors: [] }),
  },
}
