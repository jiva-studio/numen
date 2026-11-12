import { fn } from 'storybook/test'
import { ref } from 'vue'
import type { Result } from '@/shared/lib/result'
import { createMockPluginInfo } from '@numen/plugin-manager/testing'
import type { Installation, SettingsTabState } from '../types'
import { readPluginReadme, type CatalogEntry } from '@/entities/plugin'

type Flat<T> = { [K in keyof T]: T[K] }

/** What a test puts over the defaults of an installation, named without the group it belongs to. */
export type InstallationOverrides = Flat<
  Partial<Installation['settings']> &
    Partial<Omit<Installation['plugin'], 'files' | 'health'>> &
    Partial<Installation['plugin']['files']> &
    Partial<Installation['plugin']['health']> &
    Partial<Omit<Installation['connections'], 'edits'>> &
    Partial<Installation['connections']['edits']>
>

const SETTINGS_KEYS = [
  'getSetting',
  'getConfirmedSetting',
  'isPending',
  'isRequested',
  'getPlatform',
  'write',
  'file',
]
const PLUGIN_KEYS = [
  'plugins',
  'app',
  'catalog',
  'retryPlugin',
  'choosePath',
  'getSettingsViews',
]
const FILES_KEYS = ['getPluginData', 'deletePluginData', 'getReadme']
const HEALTH_KEYS = ['items', 'canRunCommand', 'runCommand']
const EDITS_KEYS = ['create', 'update', 'delete']
const CONNECTIONS_KEYS = [
  'list',
  'types',
  'operationTitles',
  'load',
  'listRowOptions',
  'countNotes',
]

/** A call that failed, with the reason. */
export const asFailure = <E>(error: E): Result<never, E> => ({ ok: false, error })

/** A call that returned a value. */
export const asValue = <T>(value: T): Result<T, never> => ({ ok: true, value })

const pickOf = (source: object, keys: readonly string[]): object =>
  Object.fromEntries(Object.entries(source).filter(([key]) => keys.includes(key)))

/** An installation whose calls are recorded and which holds the defaults, with what a test cares about put over it. */
export const createInstallation = (overrides: InstallationOverrides = {}): Installation => ({
  settings: {
    getSetting: () => undefined,
    getConfirmedSetting: (path) => (overrides.getSetting ?? (() => undefined))(path),
    isPending: () => false,
    isRequested: () => false,
    getPlatform: () => '',
    write: fn(),
    file: ref('/numen.json'),
    ...pickOf(overrides, SETTINGS_KEYS),
  },
  plugin: {
    retryPlugin: fn(),
    choosePath: fn(() => Promise.resolve('')),
    plugins: ref([]),
    app: ref([]),
    catalog: ref([]),
    health: {
      items: ref([]),
      canRunCommand: () => true,
      runCommand: fn(),
      ...pickOf(overrides, HEALTH_KEYS),
    },
    getSettingsViews: () => new Map(),
    files: {
      getPluginData: () => Promise.resolve(0),
      deletePluginData: () => Promise.resolve(''),
      getReadme: readPluginReadme,
      ...pickOf(overrides, FILES_KEYS),
    },
    ...pickOf(overrides, PLUGIN_KEYS),
  },
  connections: {
    list: ref([]),
    types: ref([]),
    operationTitles: ref({}),
    load: fn(() => Promise.resolve()),
    edits: {
      create: fn(() => Promise.resolve(asFailure('Not saved.'))),
      update: fn(() => Promise.resolve(asFailure('Not saved.'))),
      delete: fn(() => Promise.resolve(asValue(undefined))),
      ...pickOf(overrides, EDITS_KEYS),
    },
    listRowOptions: fn(() => Promise.resolve(asValue([]))),
    countNotes: fn(() => Promise.resolve(0)),
    ...pickOf(overrides, CONNECTIONS_KEYS),
  },
})

/** A plugin the installation lists, with what a test cares about put over the defaults. */
export const createPluginInfo = createMockPluginInfo

/** A catalog entry for the plugin `id`, in no category unless the overrides say otherwise. */
export const createCatalogEntry = (
  id: string,
  overrides: Partial<CatalogEntry> = {},
): CatalogEntry => ({
  id,
  name: id,
  version: '1.0.0',
  description: '',
  category: '',
  feature: '',
  tabKinds: [],
  isAvailable: false,
  ...overrides,
})

/** The state of the settings tab over an installation, with the page and the query a test chooses. */
export const createSettingsState = (
  installation: Installation,
  overrides: Partial<Omit<SettingsTabState, 'installation'>> = {},
): SettingsTabState => ({
  installation,
  initialPluginId: ref<string | null>(null),
  initialSearchQuery: ref(''),
  markedSetting: ref(''),
  ...overrides,
})
