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
  NO_COMMANDS,
  NO_KINDS,
  NO_PLUGINS,
  NO_SLOTS,
  NO_TABS,
  NO_VAULT,
  NO_WINDOW,
  NO_WORK,
  createPluginNotices,
  readManifestSlotEntries,
  readManifestSlotPoints,
  requireMountAccess,
  type BackgroundTaskHost,
  type CommandsHost,
  type createPluginConnections,
  type SlotLayout,
  type VaultRelations,
  type VaultsHost,
  type WindowHost,
} from '@numen/plugin-loader'
import { createNamedIcon } from '@numen/ui'
import type { NoticeQueue, WindowPlatform } from '@numen/workbench'
import { createViewIssueReporter } from '@/entities/plugin'
import type { WindowTabs } from '@/entities/tab'
import { TARGET } from '@/shared/lib/window'

/** The commands, the vaults, the window and its background tasks the plugins read. */
interface ShellServices {
  readonly commands: CommandsHost
  readonly vaults: VaultsHost
  readonly window: WindowHost
  readonly backgroundTasks: BackgroundTaskHost
}

/**
 * What the host is made from: the relationships between notes and the connections of a plugin, the
 * tabs and notices it registers into, the services it offers (a plugin gets none that is missing),
 * and the platform the plugins read, Windows where none is given, as the frame reports without the
 * Wails runtime.
 */
interface ShellHostDeps {
  readonly relations: VaultRelations
  readonly openExternal: (href: string) => Promise<void>
  readonly getConnections: (pluginId: string) => ReturnType<typeof createPluginConnections>
  readonly tabs: Pick<WindowTabs, 'registerKind' | 'closeTab' | 'closePluginTab'>
  readonly notices: NoticeQueue
  readonly services?: Partial<ShellServices>
  readonly getPlatform?: () => WindowPlatform
  readonly getSlotLayout?: (pluginId: string) => SlotLayout | undefined
}

type SlotManifest = Parameters<typeof readManifestSlotPoints>[1]

const readSlotEntries = (manifests: ReadonlyMap<string, SlotManifest>) =>
  [...manifests].flatMap(([pluginId, manifest]) =>
    readManifestSlotEntries(pluginId, manifest, createNamedIcon),
  )

const readSlotPoints = (manifests: ReadonlyMap<string, SlotManifest>) =>
  [...manifests].flatMap(([pluginId, manifest]) => readManifestSlotPoints(pluginId, manifest))

export function createShellHost(deps: ShellHostDeps) {
  const { tabs, notices, services = {}, getPlatform = () => 'windows' } = deps
  const pluginHost = createPluginHost({
    window: {
      ...NO_WINDOW,
      target: TARGET,
      app: createAppInfo({
        version: import.meta.env.VITE_NUMEN_VERSION,
        build: import.meta.env.VITE_NUMEN_BUILD,
        getPlatform,
      }),
      openExternal: deps.openExternal,
      getNotices: (pluginId) => createPluginNotices(notices, pluginId),
      clearNotices: (pluginId) => notices.clear(pluginId),
      getCommands: () => services.commands,
      getWindow: () => services.window,
    },
    plugins: NO_PLUGINS,
    commands: NO_COMMANDS,
    kinds: {
      ...NO_KINDS,
      registerKind: tabs.registerKind,
      closeTabsForPlugin: tabs.closePluginTab,
    },
    tabs: { ...NO_TABS, closeTab: (id) => tabs.closeTab(id) },
    slots: {
      ...NO_SLOTS,
      getSlotEntries: readSlotEntries,
      getSlotPoints: readSlotPoints,
      ...(deps.getSlotLayout ? { getSlotLayout: deps.getSlotLayout } : {}),
      reportViewIssue: createViewIssueReporter(notices),
    },
    vault: { ...NO_VAULT, getVaults: () => services.vaults, getRelations: () => deps.relations },
    work: {
      ...NO_WORK,
      getBackgroundTasks: () => services.backgroundTasks,
      getConnections: (pluginId) => deps.getConnections(pluginId),
    },
  })
  const { views } = pluginHost.contextHost

  return {
    host: pluginHost.lifecycle,
    styles: pluginHost.slotViews.styles,
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
