/**
 * The plugin host of the window for the target desktop.flashcards. The window draws one tab, of the
 * main-pane kind with the lowest order, opened on the vault the user chose; one list of notices;
 * the views declared for the list of vaults; and the view of the slot titleBar. The slot entries are
 * read from the manifests of the plugins the host activated.
 */
import {
  SLOT,
  createAppInfo,
  createPluginHost,
  createPluginNotices,
  readManifestSlotEntries,
  requireMountAccess,
  type BackgroundTaskHost,
  type CommandsHost,
  type Manifest,
  type VaultsHost,
  type WindowHost,
} from '@numen/plugin-loader'
import { createNamedIcon } from '@numen/ui'
import { openExternalUrl, type NoticeQueue, type WindowPlatform } from '@numen/workbench'
import { createViewIssueReporter, getPluginConnections } from '@/entities/plugin'
import type { WindowTabs } from '@/entities/tab'
import { windowRelations } from '@/entities/vault'
import { loadFrameRuntime } from '@/shared/api/runtime'
import { TARGET } from '@/shared/lib/window'

/** The commands, the vaults, the window and its background tasks the plugins read. */
interface ShellServices {
  readonly commands: CommandsHost
  readonly vaults: VaultsHost
  readonly window: WindowHost
  readonly backgroundTasks: BackgroundTaskHost
}

/**
 * What the host is made from: the tabs and notices it registers into, the services it offers (a plugin
 * gets none that is missing), and the platform the plugins read, Windows where none is given, as the
 * frame reports without the Wails runtime.
 */
interface ShellHostDeps {
  readonly tabs: Pick<WindowTabs, 'registerKind' | 'closeTab' | 'closePluginTab'>
  readonly notices: NoticeQueue
  readonly services?: Partial<ShellServices>
  readonly getPlatform?: () => WindowPlatform
}

const readSlotEntries = (manifests: ReadonlyMap<string, Manifest>) =>
  [...manifests].flatMap(([pluginId, manifest]) =>
    readManifestSlotEntries(pluginId, manifest, createNamedIcon),
  )

export function createShellHost(deps: ShellHostDeps) {
  const { tabs, notices, services = {}, getPlatform = () => 'windows' } = deps
  const pluginHost = createPluginHost({
    target: TARGET,
    app: createAppInfo({
      version: import.meta.env.VITE_NUMEN_VERSION,
      build: import.meta.env.VITE_NUMEN_BUILD,
      getPlatform,
    }),
    openExternal: (url) => openExternalUrl(url, loadFrameRuntime),
    registerKind: tabs.registerKind,
    closeTab: (id) => tabs.closeTab(id),
    closeTabsForPlugin: tabs.closePluginTab,
    getSlotEntries: readSlotEntries,
    reportViewIssue: createViewIssueReporter(notices),
    getNotices: (pluginId) => createPluginNotices(notices, pluginId),
    clearNotices: (pluginId) => notices.clear(pluginId),
    getCommands: () => services.commands,
    getVaults: () => services.vaults,
    getWindow: () => services.window,
    getBackgroundTasks: () => services.backgroundTasks,
    getRelations: () => windowRelations,
    getConnections: (pluginId) => getPluginConnections(pluginId),
  })
  const { views } = pluginHost.contextHost

  return {
    host: pluginHost.lifecycle,
    /** The view a plugin draws in the title bar, or null while none does. */
    titleBarView: pluginHost.slotViews.getSlotView(SLOT.titleBar),
    /** A view a plugin declares, drawn into an element of the window itself. */
    mountView: (capability: string, element: HTMLElement, viewContext: object) =>
      views.mountView(
        capability,
        element,
        viewContext,
        requireMountAccess(capability, { pluginId: TARGET }),
      ),
  }
}
