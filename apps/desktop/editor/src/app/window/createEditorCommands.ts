/**
 * The kinds of tab and the commands of the window, and the few requests one makes of the other.
 */
import {
  createCommandNotes,
  createInvocationRunner,
  createNoteLookup,
  runSupport,
  searchCore,
} from '@/features/command-palette'
import { createWindowBackgroundTasks } from '@/features/background-tasks'
import { createBufferHost } from '@/features/file-conflict'
import {
  createCommandIconFinder,
  createCommandNoteReader,
  createFileMakers,
  createPluginServices,
  createWindowFacts,
} from '@/features/plugin-commands'
import { createTargetResolver } from '@/features/window-display'
import type { TabsHold } from '@/features/vault-commands'
import { WORDS as words } from '@/shared/ui/words'
import { createCommands } from './createCommands'
import { collectFeatureCommands, createFeatureDeps } from './createFeatureCommands'
import type { WindowCore } from './windowCore'
import type { WindowServices } from './createWindowServices'
import type { createWindowTabMover } from '@/features/tab-move'

export function createEditorCommands(
  base: WindowCore,
  services: WindowServices,
  moves: ReturnType<typeof createWindowTabMover>,
  tabsHold: TabsHold,
) {
  const { editing, settings, window, vaults } = services
  const { windowTabs, tabLinks } = base.tabs
  const getTarget = createTargetResolver(windowTabs, vaults, window)
  const featureCommands = collectFeatureCommands(
    createFeatureDeps({
      base,
      services,
      moves,
      tabsHold,
      notes: createCommandNotes(editing.reached, windowTabs),
      tabLinks,
    }),
    words,
  )
  base.registry.registerAll(featureCommands.handlers)
  const commands = createCommands({
    declaredCommands: base.plugins.declaredCommands,
    getAppCommands: () => featureCommands.commands,
    readCommandNote: createCommandNoteReader(base.plugins.itemLists),
    core: searchCore,
    words,
    windowTabs,
    getTarget,
    knows: createNoteLookup(editing, windowTabs),
    paletteLists: settings.paletteLists,
    runs: runSupport(),
    registry: base.registry,
    coverage: vaults.coverage,
    reached: editing.reached,
    settingPalette: settings.settingPalette,
    settings: settings.rest,
    writeMessage: base.writeMessage,
    pluginHost: base.plugins.host.pluginHost,
  })
  services.renames.subscribe((renamed) => commands.commands.navigation.applyRenames(renamed))
  const facts = createWindowFacts({
    getAppCommands: () => featureCommands.commands,
    getTarget,
    isPaletteOpen: () => commands.palette.open.value || commands.commands.palette.open.value,
  })
  return { getTarget, featureCommands, commands, facts }
}

/** The services the plugins are given, made from the parts of the window. */
export function createWindowKinds(
  base: WindowCore,
  services: WindowServices,
  parts: Pick<ReturnType<typeof createEditorCommands>, 'getTarget' | 'commands'>,
) {
  const { editing, settings, window, tabSync } = services
  const { windowTabs, tabOpeners } = base.tabs
  const { pluginHost } = base.plugins.host
  windowTabs.groups.registerKinds(settings.kinds)
  const pluginServices = createPluginServices({
    log: base.log,
    windowTabs,
    search: searchCore,
    making: editing.making,
    pluginNotes: base.plugins.pluginNotes,
    tabOpeners,
    tabSync,
    window,
    settings,
    getTarget: parts.getTarget,
    runCommand: parts.commands.runCommand,
    runInvocation: createInvocationRunner(() => parts.commands.commandDeps),
    getPluginCommandIcon: createCommandIconFinder(pluginHost, base.plugins.declaredCommands),
  })
  return {
    findFileIcon: (path: string) => tabOpeners.findFileIcon(path),
    made: createFileMakers({ tabOpeners, windowTabs, writeMessage: base.log.getWriter('command') }),
    services: {
      ...pluginServices,
      buffers: createBufferHost({ fileFlush: editing.fileFlush }),
      noteBuffer: editing.notes,
      taskList: window.tasks,
      backgroundTasks: createWindowBackgroundTasks({
        log: base.log,
        plugins: base.plugins.pluginNotices,
        window: {
          lost: window.health.lost,
          unwatched: window.vault.unwatched,
          error: window.vault.error,
          tasks: window.tasks,
        },
        settings,
        writeMessage: base.writeMessage,
        openSettings: (pluginId) => settings.configured.openSettings(pluginId),
      }),
    },
  }
}
