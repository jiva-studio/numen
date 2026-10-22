import { fn } from 'storybook/test'
import { ref } from 'vue'
import type { Model, PluginInfo } from '@/entities/settings'
import type { Installation } from '@/pages/settings/types'

/** An installation whose calls are recorded and which holds the defaults, with what a test cares about put over it. */
export const createInstallation = (overrides: Partial<Installation> = {}): Installation => ({
  getSetting: () => undefined,
  getModels: () => [],
  write: fn(),
  writeSecret: fn(),
  deleteSecret: fn(),
  retryPlugin: fn(),
  plugins: ref([]),
  file: ref('/numen.json'),
  openFile: fn(),
  themes: ref([]),
  applied: ref('preset:numen'),
  mode: ref('system'),
  isPinned: ref(false),
  sizes: ref({ interfaceScale: 1, textScale: 1 }),
  bounds: ref({
    interfaceScale: { least: 0.8, most: 2 },
    textScale: { least: 0.8, most: 1.75 },
  }),
  choose: fn(),
  isSyncing: ref(true),
  isNesting: ref(true),
  headings: ref(6),
  headingsBounds: ref({ least: 1, most: 12 }),
  chooseHeadings: fn(),
  ...overrides,
})

/** A plugin the installation lists, with what a test cares about put over the defaults. */
export const createPluginInfo = (
  id: string,
  name: string,
  overrides: Partial<PluginInfo> = {},
): PluginInfo => ({
  id,
  name,
  version: '1.0.0',
  description: '',
  isActive: false,
  settings: [],
  capabilities: [],
  secrets: [],
  storedSecrets: [],
  consent: 'none',
  ...overrides,
})

/** A program a plugin lists, and the models it reports. */
export interface ProgramFixture {
  readonly name: string
  readonly title: string
  /** Whether it answers where the settings name none. */
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
    if (settingPath.join('.') === 'agent.use') {
      return [
        ...programs.map((one) =>
          createModel(settingPath, { name: one.name, title: one.title, isDefault: one.isDefault ?? false }),
        ),
        createModel(settingPath, {
          title: 'Nothing answers',
          isDefault: !programs.some((one) => one.isDefault),
        }),
      ]
    }
    const program = programs.find(
      (one) => settingPath.length === 3 && one.name === settingPath[1] && settingPath[2] === 'model',
    )
    if (!program) return []
    return [
      createModel(settingPath, { title: 'Whatever this machine answers with', isDefault: true }),
      ...(program.models ?? []).map((one) => createModel(settingPath, one)),
    ]
  }
