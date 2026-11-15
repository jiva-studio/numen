/**
 * The palette commands of the settings of one window: the owners are the settings of the
 * application and of the plugins the store holds, a preview applies to the appearance of the window,
 * and a write of the appearance reads it again.
 */
import type { JsonValue } from '@/shared/api/json'
import {
  INTERFACE_SCALE,
  MODES,
  TEXT_SCALE,
  type Mode,
  type WindowAppearance,
} from '@numen/workbench'
import type { createSettingsStore } from '@/entities/settings'
import { APP_OWNER } from '../lib/settingsConfig'
import { createSettingPalette, type SettingPreview, type SettingWords } from './settingCommands'
import type { Unvalidated } from '@/shared/api/errors'

const SHELL_APPEARANCE = ['app', 'appearance'] as const

const isMode = (value: Unvalidated): value is Mode =>
  typeof value === 'string' && MODES.some((one) => one === value)

const isScale = (key: string | undefined): key is typeof INTERFACE_SCALE | typeof TEXT_SCALE =>
  key === INTERFACE_SCALE || key === TEXT_SCALE

function previewAppearance(
  keyboard: WindowAppearance['preview']['keyboard'],
  key: string | undefined,
  value: JsonValue,
): void {
  if (key === 'theme' && typeof value === 'string') keyboard.previewTheme(value)
  else if (key === 'mode' && isMode(value)) keyboard.previewMode(value)
  else if (isScale(key) && typeof value === 'number') keyboard.previewSize(key, value)
}

function createWindowPreview(appearance: WindowAppearance): SettingPreview {
  return {
    apply: (path, value) => {
      if (path.length !== 3 || !SHELL_APPEARANCE.every((one, index) => path[index] === one)) return
      previewAppearance(appearance.preview.keyboard, path[2], value)
    },
    restore: () => appearance.preview.restore(),
  }
}

export function createWindowSettingPalette(deps: {
  rest: ReturnType<typeof createSettingsStore>
  appearance: WindowAppearance
  words: SettingWords
}) {
  const { rest, appearance, words } = deps
  return createSettingPalette({
    getOwners: () => [
      { id: APP_OWNER, name: '', settings: rest.state.app.value },
      ...rest.state.plugins.value.map((one) => ({
        id: one.id,
        name: one.name,
        settings: one.settings,
      })),
    ],
    words,
    preview: createWindowPreview(appearance),
    write: async (edits) => {
      await rest.write.writeSettings(edits)
      // The window shows what the settings now hold, also where the write was not accepted.
      if (edits.some((one) => one.at[0] === APP_OWNER && one.at[1] === 'appearance')) {
        await appearance.reload()
      }
    },
  })
}
