import { fn } from 'storybook/test'
import { ref } from 'vue'
import { asFailure, asValue } from '@numen/result'
import { createMockPluginInfo } from '@numen/plugin-manager/testing'
import type { Installation, SettingsTabState } from '../types'
import type { CatalogEntry } from '@/entities/plugin'
import { createKeyFileStore } from '@/entities/key-file'

export const createInertKeyFile = () =>
  createKeyFileStore(
    {
      getState: () =>
        Promise.resolve({
          isInUse: false,
          mode: 'unset',
          isChoiceNeeded: false,
          isLocked: false,
          warning: '',
        }),
      choose: () => Promise.resolve(asValue(undefined)),
      unlock: () => Promise.resolve(asValue(undefined)),
    },
    () => undefined,
  )

type Flat<T> = { [K in keyof T]: T[K] }

/** What a test puts over the defaults of an installation, named without the group it belongs to. */
export type InstallationOverrides = Flat<
  Partial<Installation['settings']> &
    Partial<Omit<Installation['plugin'], 'files' | 'health'>> &
    Partial<Installation['plugin']['files']> &
    Partial<Installation['plugin']['health']> &
    Partial<Omit<Installation['connections'], 'edits'>> &
    Partial<Installation['connections']['edits']> & {
      readonly licenses?: Installation['licenses']
      readonly identity?: Installation['identity']
    }
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
  'appGroups',
  'catalog',
  'retryPlugin',
  'choosePath',
  'getSettingsViews',
  'getIdentityViews',
  'getSlotGroups',
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
    appGroups: ref([]),
    catalog: ref([]),
    health: {
      items: ref([]),
      canRunCommand: () => true,
      runCommand: fn(),
      ...pickOf(overrides, HEALTH_KEYS),
    },
    getSettingsViews: () => new Map(),
    getIdentityViews: () => [],
    files: {
      getPluginData: () => Promise.resolve(0),
      deletePluginData: () => Promise.resolve(''),
      getReadme: () => Promise.resolve(''),
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
  identity: overrides.identity ?? {
    keyFile: createInertKeyFile(),
    identity: ref(undefined),
    load: fn(() => Promise.resolve()),
    create: fn(() => Promise.resolve(asFailure('Not kept.'))),
    delete: fn(() => Promise.resolve(asFailure('Not deleted.'))),
    recovery: {
      generateWords: fn(() => Promise.resolve(asFailure('No words.'))),
      restore: fn(() => Promise.resolve(asFailure('Not restored.'))),
      saveKit: fn(() => Promise.resolve(asFailure('Not saved.'))),
    },
    devices: createInertDevices(),
  },
  licenses: overrides.licenses ?? {
    list: ref([]),
    load: fn(() => Promise.resolve()),
    watch: fn(() => Promise.resolve()),
    activate: fn(() => Promise.resolve(asFailure('Not activated.'))),
    deactivate: fn(() => Promise.resolve(asFailure('Not deactivated.'))),
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
  markedSetting: ref(null),
  ...overrides,
})

/** The devices and the link of an identity that does nothing: no device, and every call refused. */
export function createInertDevices() {
  const createRefusal = () => Promise.resolve(asFailure('Not done.'))
  return {
    list: ref([]),
    load: fn(() => Promise.resolve()),
    revoke: fn(createRefusal),
    link: {
      start: fn(createRefusal),
      get: fn(createRefusal),
      decide: fn(createRefusal),
      confirmJoin: fn(createRefusal),
      cancel: fn(createRefusal),
    },
  }
}
