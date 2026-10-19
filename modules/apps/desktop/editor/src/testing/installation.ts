import { ref } from 'vue'
import type { PluginInfo } from '@/entities/settings'
import type { Installation } from '@/pages/settings/types'

const nothing = (): void => {}

/** An installation that does nothing and holds the defaults, with what a test cares about put over it. */
export const createInstallation = (over: Partial<Installation> = {}): Installation => ({
  getSetting: () => undefined,
  getModels: () => [],
  write: nothing,
  writeSecret: nothing,
  deleteSecret: nothing,
  retryPlugin: nothing,
  plugins: ref([]),
  file: ref('/numen.json'),
  openFile: nothing,
  themes: ref([]),
  applied: ref('preset:numen'),
  mode: ref('system'),
  isPinned: ref(false),
  sizes: ref({ interfaceScale: 1, textScale: 1 }),
  bounds: ref({
    interfaceScale: { least: 0.8, most: 2 },
    textScale: { least: 0.8, most: 1.75 },
  }),
  choose: nothing,
  isSyncing: ref(true),
  isNesting: ref(true),
  headings: ref(6),
  headingsBounds: ref({ least: 1, most: 12 }),
  chooseHeadings: nothing,
  dayStartTime: ref('04:00'),
  latestDayStartTime: ref('12:00'),
  chooseDayStartTime: nothing,
  ...over,
})

/** A plugin the installation lists, with what a test cares about put over the defaults. */
export const createPluginInfo = (
  id: string,
  name: string,
  over: Partial<PluginInfo> = {},
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
  models: [],
  consent: 'none',
  ...over,
})
