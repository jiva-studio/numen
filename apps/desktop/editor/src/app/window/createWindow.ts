/**
 * The window composed: the services that read the vault, the commands, the kinds of tab and the
 * plugins, made in the order each needs the one before.
 */
import { formatError } from '@/shared/api/errors'
import { ref, type Ref } from 'vue'
import { useWindowFrame } from '@numen/workbench'
import { runSupport } from '@/features/command-palette'
import type { CommandTarget } from '@/entities/command'
import { createMessageLog } from '@/shared/notices/messages'
import { useAppHotkeys } from '@/features/hotkeys'
import { watchPluginSettings } from '../plugins/useWindowPlugins'
import {
  createEditorCommands,
  createNoteKindReader,
  createWindowKinds,
} from './createEditorCommands'
import { createWindowServices } from './createWindowServices'
import type { WindowCore } from './windowCore'
import { closeWhenEmpty, initAppLifecycle, isSecondaryWindow } from './createLifecyclePart'
import { createPluginsCore, createWindowFrameServices, getPluginViews } from './createPluginsPart'
import {
  createTabCloser,
  createTabsCore,
  createWindowTabLinks,
  createWindowTabMoves,
  openPlacedTabs,
  useWindowOpenTabs,
} from './createTabsPart'
import { createSettingsStore, SETTINGS_WORDS } from '@/entities/settings'

function createWindowCore(): WindowCore {
  const log = createMessageLog()
  const rest = createSettingsStore(SETTINGS_WORDS, log.getWriter('configured'))
  const writeMessage = log.getWriter('command')
  const frame = useWindowFrame()
  // The plugin loader is made after the tabs; until then every plugin counts as starting.
  const pluginStarts: { current?: (pluginId: string) => boolean } = {}
  const tabs = createTabsCore({ writeMessage, rest, pluginStarts })
  const runs = runSupport()
  const plugins = createPluginsCore({
    log,
    writeMessage,
    frame,
    windowTabs: tabs.windowTabs,
    tabLinks: tabs.tabLinks,
    tabOpeners: tabs.tabOpeners,
    rest,
  })
  pluginStarts.current = plugins.host.pluginLoader.isPluginStarting
  return { frame, log, rest, writeMessage, runs, tabs, plugins }
}

function createPaletteApi(
  parts: ReturnType<typeof createEditorCommands>,
  kinds: ReturnType<typeof createWindowKinds>,
) {
  const { palette, commands, commandDeps, pluginWays, runCommand, runChoice } = parts.commands
  return {
    commands,
    search: palette,
    commandDeps,
    pluginWays,
    destinations: kinds.destinations,
    runCommand: (id: string, target: CommandTarget) => runCommand(id, target),
    runChoice,
    getTarget: parts.getTarget,
  }
}

interface TabsApiParts {
  readonly closeTab: ReturnType<typeof createTabCloser>
  readonly openTabs: ReturnType<typeof useWindowOpenTabs>
  readonly isHighlighted: Ref<boolean>
  readonly moves: ReturnType<typeof createWindowTabMoves>
}

const getTabsApi = (base: WindowCore, parts: TabsApiParts) => ({
  windowTabs: base.tabs.windowTabs,
  layout: base.tabs.windowTabs.panes.layout,
  closeTab: parts.closeTab,
  getTabIcon: parts.openTabs.getTabIcon,
  tabLinks: base.tabs.tabLinks,
  isHighlighted: parts.isHighlighted,
  moves: parts.moves,
})

/** Everything the window is made of, made once and handed to what draws it. */
export const createWindow = () => {
  const base = createWindowCore()
  const services = createWindowServices(base)
  const { editing, window, settings, vaults } = services
  const closeTab = createTabCloser(base)
  const moves = createWindowTabMoves(base, editing, closeTab)
  const isWritingTabs = ref(false)
  const isLayoutReady = ref(false)
  const openTabs = useWindowOpenTabs(base, isWritingTabs)
  const commands = createEditorCommands(base, services, moves, {
    pause: async () => openTabs.pause(),
    resume: () => openTabs.resume(),
  })
  const kinds = createWindowKinds(base, services, commands)
  watchPluginSettings(base.rest, base.plugins.host.pluginLoader, (error) =>
    base.writeMessage(`Plugins were not tracked: ${formatError(error)}`, 'error'),
  )

  const isHighlighted = ref(false)
  useAppHotkeys(commands.commands.onKeyDown)
  const tabLinkSession = createWindowTabLinks(base, {
    tabSync: services.tabSync,
    closeTab,
    isHighlighted,
  })
  const bootstrap = initAppLifecycle({
    vaultsModule: vaults,
    windowTabs: base.tabs.windowTabs,
    settings,
    window,
    editing,
    pluginLoader: base.plugins.host.pluginLoader,
    waitForDeclaredCommands: () => base.plugins.declaredCommands.waitForFirstList(),
    writeMessage: base.writeMessage,
    tabLinkSession,
    windowId: base.tabs.windowId,
    openDetachedTab: tabLinkSession.openMovedTab,
    onLayoutStarted: () => {
      isLayoutReady.value = true
      isWritingTabs.value = !isSecondaryWindow()
    },
  })

  if (isSecondaryWindow()) closeWhenEmpty(base, isLayoutReady)

  const frameServices = createWindowFrameServices(base, services, commands)
  return {
    frame: base.frame,
    tabs: getTabsApi(base, { closeTab, openTabs, isHighlighted, moves }),
    palette: createPaletteApi(commands, kinds),
    plugins: {
      slots: frameServices.slots,
      windowCommands: frameServices.windowCommands,
      views: getPluginViews(base.plugins.host.pluginHost),
    },
    notices: {
      failure: window.health.failure,
      fileFlush: editing.fileFlush,
      getTitle: editing.getTitle,
    },
    display: {
      vaultList: vaults.vaultList,
      readNoteKind: createNoteKindReader(base.plugins.itemLists),
    },
    settings: {
      openSettings: (pluginId?: string, filter?: string) =>
        settings.configured.openSettings(pluginId, filter),
    },
    lifecycle: {
      isStarted: bootstrap.isStarted,
      isLayoutReady,
      bootstrapFailure: bootstrap.failure,
      retryBootstrap: bootstrap.retry,
      openCurrentVault: () => openPlacedTabs(base),
    },
  }
}
