/**
 * The window composed: the vault it reads, the kinds it declares, and the
 * few requests one kind makes of another.
 */
import { formatError } from '@/shared/api/errors'
import { ref, type Ref } from 'vue'
import { useWindowFrame } from '@numen/workbench'
import { runSupport } from '@/features/command-palette'
import { createMessageLog } from '@/shared/notices/messages'
import { useAppHotkeys } from '@/features/hotkeys'
import { watchPluginSettings } from '../plugins/useWindowPlugins'
import { createNoteKindReader, initWindowKindsAndCommands } from './createCommandsPart'
import {
  getCommands,
  getKinds,
  getVaults,
  initWindowServices,
  type WindowContext,
  type WindowCore,
} from './createDisplayPart'
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
import type { useSettings } from './useSettings'

function createWindowCore(settingsHolder: {
  current?: ReturnType<typeof useSettings>
}): WindowCore {
  const log = createMessageLog()
  const writeMessage = log.getWriter('command')
  const frame = useWindowFrame()
  // The plugin loader is made after the tabs; until then every plugin counts as starting.
  const pluginStarts: { current?: (pluginId: string) => boolean } = {}
  const tabs = createTabsCore({ writeMessage, settingsHolder, pluginStarts })
  const runs = runSupport()
  const plugins = createPluginsCore({
    log,
    writeMessage,
    frame,
    windowTabs: tabs.windowTabs,
    tabLinks: tabs.tabLinks,
    tabOpeners: tabs.tabOpeners,
    settingsHolder,
  })
  pluginStarts.current = plugins.pluginLoader.isPluginStarting
  return { frame, log, writeMessage, runs, ...tabs, ...plugins }
}

function createPaletteApi(
  ctx: WindowContext,
  kindsAndCommands: ReturnType<typeof initWindowKindsAndCommands>,
) {
  const { commands, palette, commandDeps, pluginWays } = getCommands(ctx)
  return {
    commands,
    search: palette,
    commandDeps,
    pluginWays,
    destinations: getKinds(ctx).destinations,
    runCommand: kindsAndCommands.runCommand,
    runChoice: kindsAndCommands.runChoice,
    getTarget: kindsAndCommands.getTarget,
  }
}

interface TabsApiParts {
  readonly closeTab: ReturnType<typeof createTabCloser>
  readonly openTabs: ReturnType<typeof useWindowOpenTabs>
  readonly isHighlighted: Ref<boolean>
  readonly moves: ReturnType<typeof createWindowTabMoves>
}

const getTabsApi = (base: WindowCore, parts: TabsApiParts) => ({
  windowTabs: base.windowTabs,
  layout: base.windowTabs.layout,
  closeTab: parts.closeTab,
  getTabIcon: parts.openTabs.getTabIcon,
  tabLinks: base.tabLinks,
  isHighlighted: parts.isHighlighted,
  moves: parts.moves,
})

/** Everything the window is made of, made once and handed to what draws it. */
export const createWindow = () => {
  const settingsHolder: { current?: ReturnType<typeof useSettings> } = {}
  const base = createWindowCore(settingsHolder)
  const ctx: WindowContext = {}
  const services = initWindowServices(base, ctx, settingsHolder)
  const { editing, window, settings } = services
  const closeTab = createTabCloser(base)
  const moves = createWindowTabMoves(base, editing, closeTab)
  const kindsAndCommands = initWindowKindsAndCommands(base, ctx, services, moves)
  watchPluginSettings(settings, base.pluginLoader, (error) =>
    base.writeMessage(`Plugins were not tracked: ${formatError(error)}`, 'error'),
  )
  const isWritingTabs = ref(false)
  const isLayoutReady = ref(false)
  const openTabs = useWindowOpenTabs(base, isWritingTabs)
  ctx.tabsHold = openTabs

  const isHighlighted = ref(false)
  useAppHotkeys(getCommands(ctx).onKeyDown)
  const tabLinkSession = createWindowTabLinks(base, {
    tabSync: services.tabSync,
    closeTab,
    isHighlighted,
  })
  const bootstrap = initAppLifecycle({
    vaultsModule: getVaults(ctx),
    windowTabs: base.windowTabs,
    settings,
    window,
    editing,
    pluginLoader: base.pluginLoader,
    waitForDeclaredCommands: () => base.declaredCommands.waitForFirstList(),
    writeMessage: base.writeMessage,
    tabLinkSession,
    windowId: base.windowId,
    openDetachedTab: tabLinkSession.openMovedTab,
    onLayoutStarted: () => {
      isLayoutReady.value = true
      isWritingTabs.value = !isSecondaryWindow()
    },
  })

  if (isSecondaryWindow()) closeWhenEmpty(base, isLayoutReady)

  const frameServices = createWindowFrameServices(base, ctx, kindsAndCommands)
  return {
    frame: base.frame,
    tabs: getTabsApi(base, { closeTab, openTabs, isHighlighted, moves }),
    palette: createPaletteApi(ctx, kindsAndCommands),
    plugins: {
      slots: frameServices.slots,
      windowCommands: frameServices.windowCommands,
      views: getPluginViews(base.pluginHost),
    },
    notices: {
      failure: window.health.failure,
      fileFlush: editing.fileFlush,
      getTitle: editing.getTitle,
    },
    display: {
      vaultList: getVaults(ctx).vaultList,
      readNoteKind: createNoteKindReader(base.itemLists),
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
