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

const isAppearancePath = (path: readonly string[]): boolean =>
  path[0] === APP_OWNER && path[1] === 'appearance'

const isMode = (value: JsonValue): value is Mode => MODES.some((one) => one === value)

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
      if (path.length !== 3 || !isAppearancePath(path)) return
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
      ...rest.state.plugins.value,
    ],
    words,
    preview: createWindowPreview(appearance),
    write: async (edits) => {
      await rest.write.writeSettings(edits)
      // The window shows what the settings now hold, also where the write was not accepted.
      for (const one of edits) {
        if (isAppearancePath(one.at)) {
          await appearance.reload()
          break
        }
      }
    },
  })
}
