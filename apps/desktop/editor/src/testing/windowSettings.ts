/**
 * The settings of a window made the way the window makes them: the state its appearance and palette
 * read, and the settings tab.
 */
import type { SettingsStore } from '@/entities/settings'
import type { WindowTabs } from '@/entities/tab'
import { createWindowSettings } from '@/features/settings-commands'
import { createSettingsPage } from '@/pages/settings'
import type { MessageLog } from '@/shared/model/notices/messages'

type PageDeps = Parameters<typeof createSettingsPage>[0]

export interface WindowSettingsDeps {
  readonly rest: SettingsStore
  readonly words: Parameters<typeof createWindowSettings>[0]['words']
  readonly log: MessageLog
  readonly windowTabs: WindowTabs
  readonly onSizeChanged: () => void
  readonly getSettingsViews: PageDeps['getSettingsViews']
  readonly getPlatform: PageDeps['getPlatform']
}

export function createWindowSettingsPart(deps: WindowSettingsDeps) {
  const { rest, words, log, onSizeChanged } = deps
  const windowSettings = createWindowSettings({ rest, words, log, onSizeChanged })
  const page = createSettingsPage({
    rest,
    appearance: windowSettings.appearance,
    log,
    windowTabs: deps.windowTabs,
    getSettingsViews: deps.getSettingsViews,
    getPlatform: deps.getPlatform,
  })
  return {
    rest,
    appearance: windowSettings.appearance,
    settingPalette: windowSettings.settingPalette,
    paletteLists: windowSettings.paletteLists,
    start: windowSettings.start,
    close: windowSettings.close,
    configured: page.configured,
    kinds: page.kinds,
  }
}
