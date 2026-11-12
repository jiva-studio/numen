/**
 * The window around the settings, the tab the window adds to what the library draws.
 *
 * No vault is contacted. The tab receives its state as input, so it
 * renders in its settled state and can be captured as a screenshot.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { WorkspaceLayout, pane } from '@numen/workbench'
import type { Tab, Workspace } from '@numen/workbench'
import { Settings as SettingsIcon } from '@lucide/vue'
import { computed, reactive, ref, type Component, type Ref } from 'vue'

import type { Installation } from '../types'
import SettingsTab from './SettingsTab.vue'
import type { PluginHealthItem } from '@/entities/plugin'
import type { PluginInfo, SettingEdit } from '@/entities/settings'
import { SETTINGS } from '@/entities/tab'
import { createInstallation, createSettingsState } from '@/pages/settings/testing/installation'
import {
  SHIPPED_CATALOG,
  SHIPPED_PLUGINS,
  SHIPPED_READMES,
} from '@/pages/settings/testing/shippedPlugins'

/** Every tab this file draws, under the identity the window opens it at. */
const TABS: readonly Tab[] = [{ id: SETTINGS, title: 'Settings' }]

const getTabIcon = () => SettingsIcon

const window = (tab: string, tabPane: Component, state: unknown) => ({
  components: { WorkspaceLayout },
  setup() {
    const layout = ref<Workspace>({
      root: pane(
        'main',
        TABS.map((one) => one.id),
        tab,
      ),
      axis: 'horizontal',
      focus: 'main',
    })
    return { layout, TABS, tab, tabPane, state, getTabIcon }
  },
  template: `
    <div style="height: 100vh">
      <WorkspaceLayout v-model="layout" :tabs="TABS">
        <!-- Lucide draws on a 24 grid, and the stroke is given in those units. -->
        <template #icon="{ id }">
          <component
            :is="getTabIcon(id)"
            v-if="getTabIcon(id)"
            style="inline-size: 100%; block-size: 100%; stroke-width: 1.75; opacity: 0.75"
          />
        </template>

        <template #tab="{ id }">
          <component :is="tabPane" v-if="id === tab" :state="state" />
          <div v-else />
        </template>
      </WorkspaceLayout>
    </div>
  `,
})

const meta: Meta = {
  title: 'Desktop/Window',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

/* The settings. ------------------------------------------------------------ */

/** Document OCR suspended after a failed start. */
const HEALTH: readonly PluginHealthItem[] = [
  {
    pluginId: 'md.numen.engines.ocr',
    key: 'start',
    severity: 'error',
    text: 'The engine exited while starting: the model file pp-ocrv6-tiny.onnx is missing.',
    detail: 'open models/pp-ocrv6-tiny.onnx: no such file or directory',
  },
]

/** What numen.json holds: EPUB Book Reader turned off and two values changed from their defaults. */
const VALUES = reactive(
  new Map<string, unknown>([
    ['md.numen.readers.epub/enabled', false],
    ['md.numen.agents.numen/temperature', 0.4],
    ['md.numen.tools.media.importer/copyMaxSizeMb', 2000],
  ]),
)

const readValue = (path: readonly string[]) => VALUES.get(path.join('/')) as never

const writeValues = (edits: readonly SettingEdit[]) => {
  for (const edit of edits) VALUES.set(edit.at.join('/'), JSON.parse(edit.value))
}

const APP_SETTINGS: Installation['plugin']['app'] = ref([
  {
    key: 'appearance.theme',
    type: 'enum',
    title: 'Theme',
    description: 'The stylesheet the window is drawn with.',
    default: 'preset:numen',
    optionsFrom: 'themes',
    preview: true,
    activeOptions: [
      { value: 'preset:numen', title: 'numen', detail: '' },
      { value: 'preset:dracula', title: 'dracula', detail: '' },
    ],
    isShown: true,
    value: 'preset:numen',
  },
  {
    key: 'appearance.textScale',
    type: 'range',
    title: 'Reading size',
    description: 'Scale of note and book text. 1 is the default.',
    default: 1,
    min: 0.8,
    max: 1.75,
    step: 0.1,
    preview: true,
    isShown: true,
    value: 1.25,
  },
])

/** The host decides which settings a condition shows; this does it against the values held. */
const withShownSettings = (plugin: PluginInfo): PluginInfo => {
  const valueOf = (key: string) =>
    VALUES.get(`${plugin.id}/${key}`) ?? plugin.settings.find((one) => one.key === key)?.default
  return {
    ...plugin,
    settings: plugin.settings.map((one) => ({
      ...one,
      isShown: one.when === undefined || one.when.is.includes(String(valueOf(one.when.setting) ?? '')),
    })),
  }
}

const createShippedInstallation = (): Installation =>
  createInstallation({
    getSetting: readValue,
    write: writeValues,
    app: APP_SETTINGS,
    plugins: computed(() => SHIPPED_PLUGINS.map(withShownSettings)) as unknown as Ref<readonly PluginInfo[]>,
    catalog: ref(SHIPPED_CATALOG),
    getReadme: (pluginId) => Promise.resolve(SHIPPED_READMES[pluginId] ?? ''),
    getPluginData: () => Promise.resolve(42),
    health: ref(HEALTH),
  })

/** The settings tab opened on the list. */
const drawSettings = () =>
  window(
    SETTINGS,
    SettingsTab,
    createSettingsState(createShippedInstallation(), { initialPluginId: ref(null) }),
  )

/** The settings of the window and of every plugin installed, each reached from its row. */
export const Settings: Story = {
  render: () => drawSettings(),
}
