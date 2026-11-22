/**
 * The settings of a window made the way the window makes them: the state its appearance and palette
 * read, and the settings tab.
 */
import { shallowRef } from 'vue'
import type { WindowApi } from '@/app/window/createWindowApi'
import { createConnectionStore, type ConnectionStore } from '@/entities/connection'
import type { SettingsStore } from '@/entities/settings'
import type { WindowTabs } from '@/entities/tab'
import { createWindowSettings } from '@/features/settings-commands'
import { createSettingsPage } from '@/pages/settings'
import type { MessageLog } from '@/shared/model/notices/messages'

type PageDeps = Parameters<typeof createSettingsPage>[0]

export interface WindowSettingsDeps {
  readonly api: WindowApi
  readonly rest: SettingsStore
  readonly connections?: ConnectionStore
  readonly words: Parameters<typeof createWindowSettings>[0]['words']
  readonly log: MessageLog
  readonly windowTabs: WindowTabs
  readonly onSizeChanged: () => void
  readonly getSettingsViews: PageDeps['plugins']['getSettingsViews']
  readonly getPlatform: PageDeps['getPlatform']
  readonly health?: PageDeps['plugins']['health']
}

export function createWindowSettingsPart(deps: WindowSettingsDeps) {
  const { api, rest, words, log, onSizeChanged } = deps
  const { configuration } = api
  const connections =
    deps.connections ??
    createConnectionStore(configuration.connection, log.getWriter('connections'))
  const windowSettings = createWindowSettings({
    themes: configuration.themes,
    watchSettings: configuration.settings.watchSettings,
    rest,
    connections,
    words,
    log,
    onSizeChanged,
  })
  const page = createSettingsPage({
    api: {
      settings: configuration.settings,
      license: configuration.license,
      identity: configuration.identity,
      listFiles: api.vault.file.listFiles,
      listCatalog: api.plugins.listCatalog,
      readReadme: api.plugins.readReadme,
      listRowOptions: configuration.connection.listRowOptions,
    },
    rest,
    connections,
    appearance: windowSettings.appearance,
    log,
    windowTabs: deps.windowTabs,
    getPlatform: deps.getPlatform,
    plugins: {
      getSettingsViews: deps.getSettingsViews,
      health: deps.health ?? {
        items: shallowRef([]),
        commands: { getIds: () => [], run: () => Promise.resolve('') },
      },
    },
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
