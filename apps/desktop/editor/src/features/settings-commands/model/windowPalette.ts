/**
 * The palette commands of the settings of one window: the owners are the settings of the
 * application and of the plugins the store holds, a preview applies to the appearance of the window,
 * and a write of the appearance reads it again.
 */
import { MODES, type Mode, type createSettingsStore } from '@/entities/settings'
import { APP_OWNER } from '../lib/settingsConfig'
import { INTERFACE_SCALE, TEXT_SCALE } from '../lib/appearanceValues'
import type { createWindowAppearance } from './appearance'
import { createSettingPalette, type SettingPreview, type SettingWords } from './settingCommands'

const SHELL_APPEARANCE = ['app', 'appearance'] as const

const isMode = (value: unknown): value is Mode => MODES.some((one) => one === value)

function createWindowPreview(
  appearance: ReturnType<typeof createWindowAppearance>,
): SettingPreview {
  return {
    apply: (path, value) => {
      if (path.length !== 3 || !SHELL_APPEARANCE.every((one, index) => path[index] === one)) return
      const key = path[2]
      if (key === 'theme' && typeof value === 'string') appearance.preview.previewTheme(value)
      if (key === 'mode' && isMode(value)) appearance.preview.previewMode(value)
      if (key === INTERFACE_SCALE && typeof value === 'number') {
        appearance.preview.previewSize(INTERFACE_SCALE, value)
      }
      if (key === TEXT_SCALE && typeof value === 'number') {
        appearance.preview.previewSize(TEXT_SCALE, value)
      }
    },
    restore: () => appearance.preview.restore(),
  }
}

export function createWindowSettingPalette(deps: {
  rest: ReturnType<typeof createSettingsStore>
  appearance: ReturnType<typeof createWindowAppearance>
  words: SettingWords
}) {
  const { rest, appearance, words } = deps
  return createSettingPalette({
    getOwners: () => [
      { id: APP_OWNER, name: '', settings: rest.state.app.value },
      ...rest.state.plugins.value.map((one) => ({ id: one.id, name: one.name, settings: one.settings })),
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
