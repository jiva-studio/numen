/**
 * The kinds of tab and the commands of the window, and the few requests one makes of the other.
 */
import { computed } from 'vue'
import type { PluginHost, SlotHost } from '@numen/plugin-loader'
import { runDeclaredHandler } from '@/entities/plugin'
import type { useWindowTabs } from '@/entities/tab'
import type { Command, CommandTarget, NoteLookup, runSupport } from '@/features/command-palette'
import { getNamedIcon } from '@/shared/icons'
import { isFieldFocused } from '@/shared/isFieldFocused'
import type { createMessageLog, MessageWriter } from '@/shared/notices/messages'
import { WORDS as words } from '@/shared/words'
import { core } from '../vault'
import type { TabsHold } from '../commands/leaveVault'
import type { CommandItemLists } from '../commands/commandItemLists'
import { getPaletteId, type useDeclaredCommands } from '../commands/declaredCommands'
import { createCommandNoteReader } from '../commands/usePluginCommands'
import { useCommands } from '../commands/useCommands'
import { createVersionCommands } from '../commands/versionCommands'
import { createBackgroundTaskHost } from '../shell/backgroundTasks'
import { resolveTaskAction } from '../shell/taskActions'
import { useNoticeTasks } from '../shell/useNoticeTasks'
import { createNoteLookup } from '../tabs/editorTabs'
import type { createNoteEditors } from '../tabs/useNoteEditors'
import {
  formatError,
  getCommands,
  getKinds,
  getTabsHold,
  getVaults,
  type WindowContext,
  type WindowCore,
  type WindowServices,
} from './createDisplayPart'
import type { useSettings } from './useSettings'
import type { useVaults } from './useVaults'
import type { useWindowDisplay } from './useWindowDisplay'
import { createTargetResolver } from './useWindowDisplayOptions'
import { useWindowKinds } from './useWindowKinds'

interface WindowCommandsOptions {
  tabsHold: TabsHold
  windowTabs: ReturnType<typeof useWindowTabs>
  getTarget: () => CommandTarget
  knows: NoteLookup
  settings: ReturnType<typeof useSettings>
  runs: ReturnType<typeof runSupport>
  vaultsModule: ReturnType<typeof useVaults>
  editing: ReturnType<typeof createNoteEditors>
  kinds: ReturnType<typeof useWindowKinds>
  window: ReturnType<typeof useWindowDisplay>
  writeMessage: MessageWriter
  log: ReturnType<typeof createMessageLog>
  pluginHost: PluginHost
  declaredCommands: ReturnType<typeof useDeclaredCommands>
  slots?: SlotHost | undefined
  getAppCommands: () => readonly Command[]
  readCommandNote: ReturnType<typeof createCommandNoteReader>
}

function initWindowCommands(deps: WindowCommandsOptions) {
  const { windowTabs, settings, vaultsModule, editing, kinds, window, writeMessage } = deps
  return useCommands({
    declaredCommands: deps.declaredCommands,
    ...(deps.slots ? { slots: deps.slots } : {}),
    getAppCommands: deps.getAppCommands,
    readCommandNote: deps.readCommandNote,
    core,
    words,
    log: deps.log,
    windowTabs,
    getTarget: deps.getTarget,
    knows: deps.knows,
    paletteLists: settings.paletteLists,
    runs: deps.runs,
    coverage: vaultsModule.coverage,
    making: editing.making,
    setVaultName: vaultsModule.setVaultName,
    reload: vaultsModule.reload,
    tabsHold: deps.tabsHold,
    reached: editing.reached,
    settingPalette: settings.settingPalette,
    rest: settings.rest,
    travel: (path) => kinds.travel(path),
    leavePath: (from, to) => kinds.leavePath(from, to),
    getOpeningNote: () => window.opening.value,
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

function isPaletteOpen(ctx: WindowContext): boolean {
  const { commands } = ctx
  return commands ? commands.palette.open.value || commands.commands.open.value : false
}

export function initWindowKindsAndCommands(
  base: WindowCore,
  ctx: WindowContext,
  services: WindowServices,
) {
  const { editing, settings, window, tabSync } = services
  const getTarget = createTargetResolver(base.windowTabs, () => getVaults(ctx), window)
  base.appCommands.current = () => getAppCommandList(ctx)
  base.slotWindowValues.current = () => {
    const inFront = getTarget()
    return {
      'window.isVaultOpen': inFront.vault.id !== '',
      'window.isVaultReady': inFront.isReady,
      'window.isTabOpen': inFront.tab !== '',
      'window.isFieldFocused': isFieldFocused(),
      'window.isPaletteOpen': isPaletteOpen(ctx),
    }
  }

  const runCommand = (id: string, target: CommandTarget) => getCommands(ctx).runCommand(id, target)
  ctx.kinds = useWindowKinds({
    core,
    log: base.log,
    tabOpeners: base.tabOpeners,
    windowTabs: base.windowTabs,
    editing,
    pluginNotes: base.pluginNotes,
    settings,
    window,
    getTarget,
    runCommand,
    commandDeps: () => getCommands(ctx).commandDeps,
    tabLinks: base.tabLinks,
    tabSync,
    getPluginCommandIcon: (id) => {
      const name =
        base.pluginHost.getPaletteCommands().find((one) => one.id === id)?.icon ??
        base.declaredCommands.commands.value?.find((one) => getPaletteId(one) === id)?.icon
      return name ? getNamedIcon(name) : undefined
    },
  })

  Object.assign(base.pluginServices, getKinds(ctx).services, {
    taskList: window.tasks,
    backgroundTasks: createBackgroundTasks(base, window, settings),
  })

  const knows: NoteLookup = createNoteLookup(editing, base.windowTabs)

  ctx.commands = initWindowCommands({
    tabsHold: {
      pause: async () => getTabsHold(ctx).pause(),
      resume: () => getTabsHold(ctx).resume(),
    },
    windowTabs: base.windowTabs,
    getTarget,
    knows,
    settings,
    runs: base.runs,
    vaultsModule: getVaults(ctx),
    editing,
    kinds: getKinds(ctx),
    window,
    writeMessage: base.writeMessage,
    log: base.log,
    pluginHost: base.pluginHost,
    declaredCommands: base.declaredCommands,
    slots: base.pluginHost.slots,
    getAppCommands: () => getAppCommandList(ctx),
    readCommandNote: createCommandNoteReader(base.itemLists),
  })

  return {
    getTarget,
    runCommand,
    runChoice: getCommands(ctx).runChoice,
  }
}

type WindowSettings = ReturnType<typeof useSettings>

function createBackgroundTasks(
  base: WindowCore,
  window: WindowServices['window'],
  settings: WindowSettings,
) {
  const noticeTasks = useNoticeTasks({
    log: base.log,
    plugins: base.pluginNotices,
    window,
    settings,
  })
  return createBackgroundTaskHost({
    tasks: computed(() => [...window.tasks.value, ...noticeTasks.value]),
    cancelTask: (id) => core.cancelTask(id),
    reportCancelError: (error) =>
      base.writeMessage(`The task was not cancelled: ${formatError(error)}`, 'error'),
    resolveAction: resolveTaskAction,
    openSettings: (pluginId) => settings.configured.openSettings(pluginId),
  })
}

export function createWindowVersionCommands(base: WindowCore) {
  const versionCommands = createVersionCommands({
    listItems: base.itemLists.listItems,
    run: (command, path, itemId) =>
      runDeclaredHandler({ pluginId: '', command }, { path, values: {}, itemId }),
  })
  versionCommands.register()
  return versionCommands
}

export const getAppCommandList = (ctx: WindowContext): readonly Command[] => [
  ...(ctx.tabCommands?.commands ?? []),
  ...(ctx.versionCommands?.commands ?? []),
]
