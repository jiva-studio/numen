/**
 * The deps of a plugin host written as one flat object, for a test that names a few members, and
 * the groups the host takes made from it.
 */
import type {
  HostCommands,
  HostKinds,
  HostPlugins,
  HostSlots,
  HostTabs,
  HostVault,
  HostWindow,
  HostWork,
  PluginHostDeps,
} from '../hosting/types'
import { createPluginHost, type PluginHost } from '../hosting/host'

/** Every member of every group of the deps of a plugin host, in one object. */
export type TestHostDeps = HostWindow &
  HostPlugins &
  HostCommands &
  HostSlots &
  HostKinds &
  HostTabs &
  HostVault &
  HostWork

type GroupMembers = {
  readonly [G in keyof PluginHostDeps]-?: Readonly<
    Record<keyof NonNullable<PluginHostDeps[G]>, true>
  >
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
    onSettingChange: true,
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
    window: pick(MEMBERS.window),
    plugins: pick(MEMBERS.plugins),
    commands: pick(MEMBERS.commands),
    slots: pick(MEMBERS.slots),
    kinds: pick(MEMBERS.kinds),
    tabs: pick(MEMBERS.tabs),
    vault: pick(MEMBERS.vault),
    work: pick(MEMBERS.work),
  }
}

/** A plugin host made from flat deps. */
export const createTestPluginHost = (deps: TestHostDeps): PluginHost =>
  createPluginHost(createTestHostDeps(deps))
