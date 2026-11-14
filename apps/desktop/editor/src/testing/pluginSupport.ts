/**
 * The plugin support of a window made the way the window makes it, from the few parts a test names.
 */
import { ref } from 'vue'
import type {
  HostCommands,
  HostKinds,
  HostSlots,
  HostVault,
  PluginStatus,
} from '@numen/plugin-loader'
import type { NoticeQueue, WindowPlatform } from '@numen/workbench'
import type { SettingsStore } from '@/entities/settings'
import type { WindowTabs } from '@/entities/tab'
import type { Caught } from '@/shared/api/errors'
import {
  createNoticeParts,
  createPluginCatalog,
  createPluginSupport,
  createTabActionParts,
  createTabKindParts,
  createTaskServiceParts,
  createVaultServiceParts,
  createWindowPluginStatus,
  type HostParts,
  type PluginServices,
} from '@/features/plugin-host'

/** What a test names of the plugin support; the rest is left out as a window without it leaves it out. */
export interface TestPluginSupportDeps {
  readonly windowTabs: WindowTabs
  readonly tabOpeners: Parameters<typeof createTabKindParts>[0]['tabOpeners']
  readonly rest: SettingsStore
  readonly getPlatform?: () => WindowPlatform
  readonly readNote?: HostKinds['readNote']
  readonly writeNote?: HostKinds['writeNote']
  readonly resolveLink?: HostVault['resolveLink']
  readonly reportError?: (pluginId: string, error: Caught) => void
  readonly reportLoadError?: (pluginId: string, error: Caught) => void
  readonly reportCommandIssue?: HostCommands['reportCommandIssue']
  readonly reportViewIssue?: HostSlots['reportViewIssue']
  readonly status?: PluginStatus
  readonly getServices?: () => Partial<PluginServices>
  readonly notices?: NoticeQueue
  readonly getCommandDeclarations?: HostCommands['getCommandDeclarations']
  readonly handlers?: HostCommands
  readonly slots?: HostParts
}

const pickDefinedEntries = (entries: Record<string, object | undefined>): Record<string, object> =>
  Object.fromEntries(Object.entries(entries).filter(([, value]) => value !== undefined)) as Record<
    string,
    object
  >

/** The platform of a window that reads none from its frame, as the frame reports without the runtime. */
const NO_PLATFORM: WindowPlatform = 'windows'

const ignoreError = (): void => undefined

export function createTestPluginSupport(deps: TestPluginSupportDeps) {
  const partialServices: Partial<PluginServices> = { taskList: ref([]), ...deps.getServices?.() }
  const services = partialServices as PluginServices
  const named: HostParts = {
    commands: pickDefinedEntries({
      getCommandDeclarations: deps.getCommandDeclarations,
      reportCommandIssue: deps.reportCommandIssue,
    }),
    slots: pickDefinedEntries({ reportViewIssue: deps.reportViewIssue }),
    kinds: pickDefinedEntries({
      readNote: deps.readNote,
      writeNote: deps.writeNote,
      getNoteBuffer: () => services.noteBuffer,
    }),
    vault: pickDefinedEntries({ resolveLink: deps.resolveLink }),
  }
  const optionalParts: readonly (HostParts | undefined)[] = [
    deps.notices ? createNoticeParts({ queue: deps.notices }) : undefined,
    deps.slots,
    deps.handlers && { commands: deps.handlers },
  ]
  return createPluginSupport({
    rest: deps.rest,
    catalog: createPluginCatalog(deps.rest),
    status: deps.status ?? createWindowPluginStatus(deps.rest),
    getPlatform: deps.getPlatform ?? (() => NO_PLATFORM),
    reports: {
      reportError: deps.reportError ?? ignoreError,
      reportLoadError: deps.reportLoadError ?? deps.reportError ?? ignoreError,
    },
    kinds: createTabKindParts(deps),
    tabs: createTabActionParts(deps),
    parts: [
      createVaultServiceParts({ services }),
      createTaskServiceParts({ services }),
      ...optionalParts.filter((one) => one !== undefined),
      named,
    ],
  })
}
