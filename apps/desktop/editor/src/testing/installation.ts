import { fn } from 'storybook/test'
import { ref } from 'vue'
import { asFailure, asValue } from '@numen/core-client'
import { createMockPluginInfo } from '@numen/plugin-manager/testing'
import type { Model, PluginInfo, Secret, SecretMenu, SecretSection } from '@/entities/settings'
import type { Installation } from '@/pages/settings/types'

/** An installation whose calls are recorded and which holds the defaults, with what a test cares about put over it. */
export const createInstallation = (overrides: Partial<Installation> = {}): Installation => ({
  getSetting: () => undefined,
  getModels: () => [],
  write: fn(),
  secretSections: ref([]),
  secretMenus: ref([]),
  secretStates: ref(new Map()),
  loadSecretStates: fn(() => Promise.resolve()),
  credentialsPath: ref(''),
  createSecret: fn(() => Promise.resolve('s00000001')),
  renameSecret: fn(),
  writeSecret: fn(),
  deleteSecret: fn(),
  deleteSecretGroup: fn(),
  moveSecret: fn(),
  renameSecretGroup: fn(),
  allowPluginSecret: fn(),
  denyPluginSecret: fn(),
  connections: ref([]),
  connectionTypes: ref([]),
  loadConnections: fn(() => Promise.resolve()),
  createConnection: fn(() => Promise.resolve(asFailure('Not saved.'))),
  updateConnection: fn(() => Promise.resolve(asFailure('Not saved.'))),
  deleteConnection: fn(() => Promise.resolve(asValue(undefined))),
  retryPlugin: fn(),
  choosePath: fn(() => Promise.resolve('')),
  plugins: ref([]),
  file: ref('/numen.json'),
  ...overrides,
})

/**
 * What the core answers for these plugins and secrets, in the order of its page: the groups the
 * plugins ask for by name, then the others by name; the menu of each declaration with its own
 * group first; the state of each value from `hasValue`. Test names stay in ASCII.
 */
export const createSecretAnswer = (
  plugins: readonly PluginInfo[],
  secrets: readonly Secret[],
): Pick<Installation, 'secretSections' | 'secretMenus' | 'secretStates'> => {
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
    secretSections: ref(sections),
    secretMenus: ref(menus),
    secretStates: ref(new Map(secrets.map((one) => [one.id, one.hasValue ?? false]))),
  }
}

/** A plugin the installation lists, with what a test cares about put over the defaults. */
export const createPluginInfo = createMockPluginInfo

/** A program a plugin lists, and the models it reports. */
export interface ProgramFixture {
  readonly name: string
  readonly title: string
  readonly isDefault?: boolean
  readonly models?: readonly { readonly name: string; readonly title: string }[]
}

const createModel = (at: readonly string[], overrides: Partial<Model>): Model => ({
  namedAt: at,
  name: '',
  title: '',
  group: '',
  isDefault: false,
  writes: [],
  presence: 'nothing to fetch',
  ...overrides,
})

/**
 * What the installation answers `getModels` with for the agent section: the programs under
 * `agent.use`, with the choice of none after them, and the models each reports under its own
 * `model`, with the choice of its own model first.
 */
export const createProgramModels =
  (programs: readonly ProgramFixture[]) =>
  (settingPath: readonly string[]): readonly Model[] => {
    if (settingPath.join('.') === 'app.agent.use') {
      return [
        ...programs.map((one) =>
          createModel(settingPath, {
            name: one.name,
            title: one.title,
            isDefault: one.isDefault ?? false,
          }),
        ),
        createModel(settingPath, {
          title: 'No agent responds',
          isDefault: !programs.some((one) => one.isDefault),
        }),
      ]
    }
    const program = programs.find(
      (one) =>
        settingPath.length === 4 && one.name === settingPath[2] && settingPath[3] === 'model',
    )
    if (!program) return []
    return [
      createModel(settingPath, { title: 'Default model', isDefault: true }),
      ...(program.models ?? []).map((one) => createModel(settingPath, one)),
    ]
  }
