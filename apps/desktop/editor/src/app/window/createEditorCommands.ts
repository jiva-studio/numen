/**
 * The kinds of tab and the commands of the window, and the few requests one makes of the other.
 */
import { formatError } from '@/shared/api/errors'
import { computed } from 'vue'
import type { PluginHost, SlotHost } from '@numen/plugin-loader'
import { cancelTask } from '@/entities/window'
import {
  createBackgroundTaskHost,
  resolveTaskAction,
  useNoticeTasks,
} from '@/features/background-tasks'
import type { WindowTabs } from '@/entities/tab'
import { searchCore, type NoteLookup, type runSupport } from '@/features/command-palette'
import type { Command, CommandTarget } from '@/entities/command'
import { getNamedIcon } from '@/shared/ui/icons'
import { isFieldFocused } from './isFieldFocused'
import type { createMessageLog, MessageWriter } from '@/shared/model/notices/messages'
import { WORDS as words } from '@/shared/ui/words'
import {
  type CommandItemLists,
  getPaletteId,
  type useDeclaredCommands,
  createCommandNoteReader,
} from '@/features/plugin-commands'
import { createCommandNotes } from '../tabs/commandDeps'
import { createCommands } from './createCommands'
import {
  collectFeatureCommands,
  createFeatureDeps,
  registerFeatureCommands,
} from './createFeatureCommands'
import { createNoteLookup } from '../tabs/editorTabs'
import type { createNoteEditors } from '../tabs/createNoteEditors'
import type { WindowCore } from './windowCore'
import type { WindowServices } from './createWindowServices'
import type { TabsHold } from '@/features/vault-commands'
import type { useSettings } from '@/pages/settings'
import type { useVaults } from '@/entities/vault'
import type { createWindowTabMoves } from './createTabsPart'
import { createTargetResolver } from './displayOptions'
import { createTabKinds } from './createTabKinds'

interface WindowCommandsOptions {
  windowTabs: WindowTabs
  getTarget: () => CommandTarget
  knows: NoteLookup
  settings: ReturnType<typeof useSettings>
  runs: ReturnType<typeof runSupport>
  vaultsModule: ReturnType<typeof useVaults>
  editing: ReturnType<typeof createNoteEditors>
  writeMessage: MessageWriter
  log: ReturnType<typeof createMessageLog>
  pluginHost: PluginHost
  declaredCommands: ReturnType<typeof useDeclaredCommands>
  slots?: SlotHost | undefined
  getAppCommands: () => readonly Command[]
  readCommandNote: ReturnType<typeof createCommandNoteReader>
}

function initWindowCommands(deps: WindowCommandsOptions) {
  const { windowTabs, settings, vaultsModule, editing, writeMessage } = deps
  return createCommands({
    declaredCommands: deps.declaredCommands,
    ...(deps.slots ? { slots: deps.slots } : {}),
    getAppCommands: deps.getAppCommands,
    readCommandNote: deps.readCommandNote,
    core: searchCore,
    words,
    log: deps.log,
    windowTabs,
    getTarget: deps.getTarget,
    knows: deps.knows,
    paletteLists: settings.paletteLists,
    runs: deps.runs,
    coverage: vaultsModule.coverage,
    reached: editing.reached,
    settingPalette: settings.settingPalette,
    settings: settings.rest,
    writeMessage,
    pluginHost: deps.pluginHost,
  })
}

export const createNoteKindReader = (itemLists: CommandItemLists) => async (path: string) => {
  const kind = (await itemLists.fileKinds([path])).get(path)
  return {
    noteType: kind?.type ?? '',
    ...(kind?.properties ? { properties: kind.properties } : {}),
  }
}

function isPaletteOpen(commands: ReturnType<typeof createCommands>): boolean {
  return commands.palette.open.value || commands.commands.palette.open.value
}

function createFeatureCommands(
  base: WindowCore,
  services: WindowServices,
  moves: ReturnType<typeof createWindowTabMoves>,
  tabsHold: TabsHold,
) {
  const { editing, window, navigation, vaults } = services
  const { windowTabs, tabLinks } = base.tabs
  const featureCommands = collectFeatureCommands(
    createFeatureDeps({
      making: editing.making,
      notes: createCommandNotes(editing.reached, windowTabs),
      windowTabs,
      travel: navigation.travel,
      leavePath: navigation.leavePath,
      getOpeningNote: () => window.opening.path.value,
      tabsHold,
      setVaultName: vaults.setVaultName,
      reload: vaults.reload,
      writeMessage: base.writeMessage,
      tabs: {
        tabLinks,
        getTabs: () => windowTabs.tabs.tabs.value,
        getKind: (id) => windowTabs.tabs.getTab(id)?.kind.kind,
        isLinkable: (id) => windowTabs.tabs.getTab(id)?.kind.navigate !== undefined,
        windowId: moves.windowId,
        isAsking: moves.isTabAsking,
        detach: moves.detachTabInPlace,
        move: moves.moveTab,
      },
      listVersionItems: base.plugins.itemLists.listItems,
    }),
    words,
  )
  registerFeatureCommands(featureCommands)
  return featureCommands
}

export function createEditorCommands(
  base: WindowCore,
  services: WindowServices,
  moves: ReturnType<typeof createWindowTabMoves>,
  tabsHold: TabsHold,
) {
  const { editing, settings, window, vaults } = services
  const getTarget = createTargetResolver(base.tabs.windowTabs, vaults, window)
  const featureCommands = createFeatureCommands(base, services, moves, tabsHold)
  const knows: NoteLookup = createNoteLookup(editing, base.tabs.windowTabs)

  const commands = initWindowCommands({
    windowTabs: base.tabs.windowTabs,
    getTarget,
    knows,
    settings,
    runs: base.runs,
    vaultsModule: vaults,
    editing,
    writeMessage: base.writeMessage,
    log: base.log,
    pluginHost: base.plugins.host.pluginHost,
    declaredCommands: base.plugins.declaredCommands,
    slots: base.plugins.host.pluginHost.slots,
    getAppCommands: () => featureCommands.commands,
    readCommandNote: createCommandNoteReader(base.plugins.itemLists),
  })
  services.renames.subscribe((renamed) => commands.commands.navigation.applyRenames(renamed))

  base.plugins.appCommands.current = () => featureCommands.commands
  base.plugins.slotWindowValues.current = () => {
    const inFront = getTarget()
    return {
      'window.isVaultOpen': inFront.vault.id !== '',
      'window.isVaultReady': inFront.isReady,
      'window.isTabOpen': inFront.tab !== '',
      'window.isFieldFocused': isFieldFocused(),
      'window.isPaletteOpen': isPaletteOpen(commands),
    }
  }
  return { getTarget, featureCommands, commands }
}

/** The kinds of tab the window declares, and the services the plugins are given from them. */
export function createWindowKinds(
  base: WindowCore,
  services: WindowServices,
  parts: Pick<ReturnType<typeof createEditorCommands>, 'getTarget' | 'commands'>,
) {
  const { editing, settings, window, tabSync } = services
  const { getTarget, commands } = parts
  const kinds = createTabKinds({
    core: searchCore,
    log: base.log,
    tabOpeners: base.tabs.tabOpeners,
    windowTabs: base.tabs.windowTabs,
    editing,
    pluginNotes: base.plugins.pluginNotes,
    settings,
    window,
    getTarget,
    runCommand: commands.runCommand,
    commandDeps: () => commands.commandDeps,
    tabLinks: base.tabs.tabLinks,
    tabSync,
    getPluginCommandIcon: (id) => {
      const name =
        base.plugins.host.pluginHost.commands.getPaletteCommands().find((one) => one.id === id)
          ?.icon ??
        base.plugins.declaredCommands.commands.value?.find((one) => getPaletteId(one) === id)?.icon
      return name ? getNamedIcon(name) : undefined
    },
  })

  Object.assign(base.plugins.pluginServices, kinds.services, {
    taskList: window.tasks,
    backgroundTasks: createBackgroundTasks(base, window, settings),
  })
  return kinds
}

type WindowSettings = ReturnType<typeof useSettings>

function createBackgroundTasks(
  base: WindowCore,
  window: WindowServices['window'],
  settings: WindowSettings,
) {
  const noticeTasks = useNoticeTasks({
    log: base.log,
    plugins: base.plugins.pluginNotices,
    window: {
      lost: window.health.lost,
      unwatched: window.vault.unwatched,
      error: window.vault.error,
    },
    settings,
  })
  return createBackgroundTaskHost({
    tasks: computed(() => [...window.tasks.value, ...noticeTasks.value]),
    cancelTask,
    reportCancelError: (error) =>
      base.writeMessage(`The task was not cancelled: ${formatError(error)}`, 'error'),
    resolveAction: resolveTaskAction,
    openSettings: (pluginId) => settings.configured.openSettings(pluginId),
  })
}
