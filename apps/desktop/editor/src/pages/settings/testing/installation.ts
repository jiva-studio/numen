import { fn } from 'storybook/test'
import { ref } from 'vue'
import { createMockPluginInfo } from '@numen/plugin-manager/testing'
import type { PluginInfo, Secret, SecretMenu, SecretSection } from '@/entities/settings'
import type { Installation, SettingsTabState } from '../types'
import { readPluginReadme, type CatalogEntry } from '@/entities/plugin'

type Flat<T> = { [K in keyof T]: T[K] }

/** What a test puts over the defaults of an installation, named without the group it belongs to. */
export type InstallationOverrides = Flat<
  Partial<Installation['settings']> &
    Partial<Omit<Installation['plugin'], 'files'>> &
    Partial<Installation['plugin']['files']> &
    Partial<Omit<Installation['secrets'], 'edit' | 'access'>> &
    Partial<Installation['secrets']['edit']> &
    Partial<Installation['secrets']['access']>
>

const SETTINGS_KEYS = [
  'getSetting',
  'getConfirmedSetting',
  'isPending',
  'isRequested',
  'getModels',
  'getPlatform',
  'write',
  'file',
]
const PLUGIN_KEYS = ['plugins', 'app', 'catalog', 'retryPlugin', 'choosePath', 'getSettingsViews']
const FILES_KEYS = ['getPluginData', 'deletePluginData', 'getReadme']
const SECRETS_KEYS = ['sections', 'menus', 'states', 'loadStates', 'credentialsPath']
const EDIT_KEYS = [
  'createSecret',
  'renameSecret',
  'writeSecret',
  'deleteSecret',
  'deleteSecretGroup',
  'moveSecret',
  'renameSecretGroup',
]
const ACCESS_KEYS = ['allowPluginSecret', 'denyPluginSecret']

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
    getModels: () => [],
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
    getSettingsViews: () => new Map(),
    files: {
      getPluginData: () => Promise.resolve(0),
      deletePluginData: () => Promise.resolve(''),
      getReadme: readPluginReadme,
      ...pickOf(overrides, FILES_KEYS),
    },
    ...pickOf(overrides, PLUGIN_KEYS),
  },
  secrets: {
    sections: ref([]),
    menus: ref([]),
    states: ref(new Map()),
    loadStates: fn(() => Promise.resolve()),
    credentialsPath: ref(''),
    ...pickOf(overrides, SECRETS_KEYS),
    edit: {
      createSecret: fn(() => Promise.resolve('s00000001')),
      renameSecret: fn(),
      writeSecret: fn(),
      deleteSecret: fn(),
      deleteSecretGroup: fn(),
      moveSecret: fn(),
      renameSecretGroup: fn(),
      ...pickOf(overrides, EDIT_KEYS),
    },
    access: {
      allowPluginSecret: fn(),
      denyPluginSecret: fn(),
      ...pickOf(overrides, ACCESS_KEYS),
    },
  },
})

/**
 * What the core answers for these plugins and secrets, in the order of its page: the groups the
 * plugins ask for by name, then the others by name; the menu of each declaration with its own
 * group first; the state of each value from `hasValue`. Test names stay in ASCII.
 */
export const createSecretAnswer = (
  plugins: readonly PluginInfo[],
  secrets: readonly Secret[],
): Pick<Installation['secrets'], 'sections' | 'menus' | 'states'> => {
  const declarations = plugins.flatMap((plugin) =>
    plugin.secrets.map((one) => ({ pluginId: plugin.id, name: one.name, group: one.group })),
  )
  const askedGroups = [
    ...new Set(declarations.map((one) => one.group).filter((one) => one !== '')),
  ].sort()
  const others = [
    ...new Set(secrets.map((one) => one.group).filter((one) => !askedGroups.includes(one))),
  ].sort()
  const sections: SecretSection[] = [...askedGroups, ...others].map((group) => ({
    group,
    isAsked: askedGroups.includes(group),
    secrets: secrets.filter((one) => one.group === group),
  }))
  const menus: SecretMenu[] = declarations.map((declaration) => ({
    ...declaration,
    secrets: [
      ...sections.filter((one) => declaration.group !== '' && one.group === declaration.group),
      ...sections.filter((one) => declaration.group === '' || one.group !== declaration.group),
    ].flatMap((one) => one.secrets),
  }))
  return {
    sections: ref(sections),
    menus: ref(menus),
    states: ref(new Map(secrets.map((one) => [one.id, one.hasValue ?? false]))),
  }
}

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
  ...overrides,
})
