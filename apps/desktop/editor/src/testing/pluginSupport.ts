/**
 * The plugin support of a window made the way the window makes it, from the few parts a test names.
 */
import type { PluginHostDeps, PluginStatus } from '@numen/plugin-loader'
import type { NoticeQueue, WindowPlatform } from '@numen/workbench'
import type { SettingsStore } from '@/entities/settings'
import type { WindowTabs } from '@/entities/tab'
import type { Caught } from '@/shared/api/errors'
import {
  createNoticeParts,
  createPluginSupport,
  createTabActionParts,
  createTabKindParts,
  createTaskServiceParts,
  createVaultServiceParts,
  type HostParts,
  type PluginServices,
} from '@/features/plugin-commands'

/** What a test names of the plugin support; the rest is left out as a window without it leaves it out. */
export interface TestPluginSupportDeps {
  readonly windowTabs: WindowTabs
  readonly tabOpeners: Parameters<typeof createTabKindParts>[0]['tabOpeners']
  readonly rest: SettingsStore
  readonly getPlatform?: () => WindowPlatform
  readonly readNote?: PluginHostDeps['readNote']
  readonly writeNote?: PluginHostDeps['writeNote']
  readonly resolveLink?: PluginHostDeps['resolveLink']
  readonly reportError?: (pluginId: string, error: Caught) => void
  readonly reportLoadError?: (pluginId: string, error: Caught) => void
  readonly reportCommandIssue?: PluginHostDeps['reportCommandIssue']
  readonly reportViewIssue?: PluginHostDeps['reportViewIssue']
  readonly status?: PluginStatus
  readonly getServices?: () => Partial<PluginServices>
  readonly notices?: NoticeQueue
  readonly getCommandDeclarations?: PluginHostDeps['getCommandDeclarations']
  readonly handlers?: HostParts
  readonly slots?: HostParts
}

/** The entries a test names, without the ones it leaves out. */
const pickDefined = (entries: Record<string, object | undefined>): Record<string, object> =>
  Object.fromEntries(Object.entries(entries).filter(([, value]) => value !== undefined)) as Record<
    string,
    object
  >

export function createTestPluginSupport(deps: TestPluginSupportDeps) {
  const getServices = deps.getServices ?? (() => ({}))
  const named: HostParts = pickDefined({
    getCommandDeclarations: deps.getCommandDeclarations,
    reportCommandIssue: deps.reportCommandIssue,
    reportViewIssue: deps.reportViewIssue,
    readNote: deps.readNote,
    writeNote: deps.writeNote,
    resolveLink: deps.resolveLink,
  })
  const given: readonly (HostParts | undefined)[] = [
    deps.notices ? createNoticeParts({ queue: deps.notices }) : undefined,
    deps.slots,
    deps.handlers,
  ]
  return createPluginSupport({
    rest: deps.rest,
    kinds: createTabKindParts(deps),
    tabs: createTabActionParts(deps),
    parts: [
      createVaultServiceParts({ getServices }),
      createTaskServiceParts({ getServices }),
      ...given.filter((one) => one !== undefined),
      named,
    ],
    ...pickDefined({
      status: deps.status,
      getPlatform: deps.getPlatform,
      reportError: deps.reportError,
      reportLoadError: deps.reportLoadError,
    }),
  })
}
