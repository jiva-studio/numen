/**
 * The deps of a plugin host written as one flat object, for a test that names a few members, and
 * the groups the host takes made from it.
 */
import {
  NO_COMMANDS,
  NO_KINDS,
  NO_PLUGINS,
  NO_SLOTS,
  NO_TABS,
  NO_VAULT,
  NO_WINDOW,
  NO_WORK,
  createPluginHost,
  type HostCommands,
  type HostKinds,
  type HostSlots,
  type HostTabs,
  type HostVault,
  type HostWork,
  type PluginHost,
  type PluginHostDeps,
} from '../hosting'

/** Every member of every group of the deps of a plugin host, in one object. */
export type TestHostDeps = Pick<PluginHostDeps['window'], 'target'> &
  Partial<PluginHostDeps['window']> &
  Partial<PluginHostDeps['plugins']> &
  Partial<HostCommands> &
  Partial<HostSlots> &
  Pick<HostKinds, 'registerKind'> &
  Partial<HostKinds> &
  Partial<HostTabs> &
  Partial<HostVault> &
  Partial<HostWork>

type GroupMembers = {
  readonly [G in keyof PluginHostDeps]: Readonly<Record<keyof PluginHostDeps[G], true>>
}

const MEMBERS: GroupMembers = {
  window: {
    target: true,
    app: true,
    openExternal: true,
    onError: true,
    getWorkspace: true,
    getCommands: true,
    getWindow: true,
    getNotices: true,
    getPicker: true,
    getDialogs: true,
    clearNotices: true,
  },
  plugins: {
    hasPluginFrontend: true,
    hasPluginBackend: true,
    isPluginFailed: true,
    startPlugin: true,
    startPluginsOf: true,
    findPluginsOf: true,
    getSetting: true,
    watchSetting: true,
    getEditorSettings: true,
    watchEditorSettings: true,
    hasLicense: true,
    getLicenseKey: true,
    watchLicense: true,
  },
  commands: {
    getCommandDeclarations: true,
    getHandlerDeclarations: true,
    listBackendHandlerItems: true,
    runBackendHandler: true,
    reportCommandIssue: true,
    getContentTypes: true,
    getWindowValues: true,
    whenReader: true,
  },
  slots: {
    getSlotEntries: true,
    getSlotPoints: true,
    getSlotLayout: true,
    createSlotCommandFinder: true,
    getSlotWhenValues: true,
    reportSlotIssue: true,
    reportViewIssue: true,
  },
  kinds: {
    registerKind: true,
    registerExtensionReader: true,
    getReaderExtensions: true,
    registerEditor: true,
    closeTabsForPlugin: true,
    getNoteBuffer: true,
    readNote: true,
    writeNote: true,
  },
  tabs: {
    openTab: true,
    openTabAdjacent: true,
    navigateTab: true,
    closeTab: true,
    getTabState: true,
  },
  vault: {
    getNotes: true,
    getRelations: true,
    getFiles: true,
    getVaults: true,
    getBuffers: true,
    getTexts: true,
    resolveLink: true,
  },
  work: {
    getTasks: true,
    getConnections: true,
    getBackgroundTasks: true,
    getProcesses: true,
    getLogs: true,
    health: true,
  },
}

type TestHostMember = TestHostDeps[keyof TestHostDeps]

/** The groups of the deps of a plugin host, each holding the members the flat deps give. */
export function createTestHostDeps(deps: TestHostDeps): PluginHostDeps {
  const pick = (members: object): TestHostDeps => {
    const picked: Partial<Record<keyof TestHostDeps, TestHostMember>> = {}
    for (const key of Object.keys(members) as (keyof TestHostDeps)[]) {
      if (key in deps) picked[key] = deps[key]
    }
    return picked as TestHostDeps
  }
  return {
    window: { ...NO_WINDOW, ...pick(MEMBERS.window) },
    plugins: { ...NO_PLUGINS, ...pick(MEMBERS.plugins) },
    commands: { ...NO_COMMANDS, ...pick(MEMBERS.commands) },
    slots: { ...NO_SLOTS, ...pick(MEMBERS.slots) },
    kinds: { ...NO_KINDS, ...pick(MEMBERS.kinds) },
    tabs: { ...NO_TABS, ...pick(MEMBERS.tabs) },
    vault: { ...NO_VAULT, ...pick(MEMBERS.vault) },
    work: { ...NO_WORK, ...pick(MEMBERS.work) },
  }
}

/** A plugin host made from flat deps. */
export const createTestPluginHost = (deps: TestHostDeps): PluginHost =>
  createPluginHost(createTestHostDeps(deps))

type GroupsOf<D> = { readonly [G in keyof D]?: Partial<D[G]> }

/** The deps of a plugin host from the members a test names, grouped as the host takes them; the rest is the null object of its group. */
export function fillHostDeps(groups: GroupsOf<PluginHostDeps> = {}): PluginHostDeps {
  return {
    window: { ...NO_WINDOW, target: 'desktop.editor', ...groups.window },
    plugins: { ...NO_PLUGINS, ...groups.plugins },
    commands: { ...NO_COMMANDS, ...groups.commands },
    slots: { ...NO_SLOTS, ...groups.slots },
    kinds: { ...NO_KINDS, registerKind: () => () => undefined, ...groups.kinds },
    tabs: { ...NO_TABS, ...groups.tabs },
    vault: { ...NO_VAULT, ...groups.vault },
    work: { ...NO_WORK, ...groups.work },
  }
}
