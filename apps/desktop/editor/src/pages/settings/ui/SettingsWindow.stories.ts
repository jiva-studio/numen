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
import { ref, type Component } from 'vue'

import type { Installation } from '../types'
import SettingsTab from './SettingsTab.vue'
import type { Model, PluginInfo } from '@/entities/settings'
import { SETTINGS } from '@/entities/tab'
import { createInstallation, createSettingsState } from '@/pages/settings/testing/installation'

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

/** The program an unconfigured installation uses. */
const AGENT_USE: Model = {
  namedAt: ['app', 'agent', 'use'],
  name: 'claude',
  title: 'Claude Code',
  group: '',
  isDefault: true,
  writes: [{ at: ['app', 'agent', 'use'], value: '"claude"' }],
  presence: 'present',
}

const plugin = (id: string, name: string, description: string): PluginInfo => ({
  id,
  name,
  version: '1.0.0',
  description,
  isActive: true,
  settings: [],
  capabilities: [],
  hasFrontend: false,
  hasBackend: false,
})

/** Plugins of every category the list sorts by, as an installation ships them. */
const PLUGINS: readonly PluginInfo[] = [
  plugin(
    'md.numen.agents.acp',
    'ACP Agents',
    'External AI assistants via the Agent Client Protocol',
  ),
  plugin(
    'md.numen.agents.tools.vault',
    'Vault Knowledge Tools',
    'Core vault, note, search and wikilink tools for agents',
  ),
  plugin('md.numen.notes', 'Notes', 'Notes written in Markdown, with wikilinks'),
  plugin(
    'md.numen.engines.embeddings',
    'Vector Embeddings',
    'Vector embeddings engine for semantic search',
  ),
  plugin('md.numen.engines.ocr', 'Document OCR', 'Recognises the text of scans and scanned PDFs'),
  plugin(
    'md.numen.engines.transcription',
    'Audio/Video Transcription',
    'Transcribes the recordings a person asks for',
  ),
  plugin('md.numen.readers.epub', 'EPUB Book Reader', 'EPUB reflowable book reader'),
  plugin('md.numen.readers.pdf', 'PDF Reader', 'PDF document viewer and page renderer'),
  plugin(
    'md.numen.tools.media.importer',
    'Media Importer',
    'Keeps web addresses as links and fetches what is published there',
  ),
  plugin('org.example.sample.studio', 'Sample Studio', 'Samples, forms and the views of them'),
  plugin('md.numen.views.plex.graph', 'Plex Graph', 'Knowledge graph visualizer and plex explorer'),
]

/** The actions the settings requested; a screenshot requests none. */
const DONE: string[] = []

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
      { value: 'mine:sea', title: 'sea', detail: '' },
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

const INSTALLED: Installation['plugin']['plugins'] = ref(PLUGINS)

const INSTALLATION = createInstallation({
  getModels: (path) => (path.join('.') === 'agent.use' ? [AGENT_USE] : []),
  write: (edits) => {
    DONE.push(`write ${edits.length}`)
  },
  file: ref('/numen.json'),
  app: APP_SETTINGS,
  retryPlugin: () => {
    DONE.push('plugin retried')
  },
  choosePath: async () => {
    DONE.push('path chosen')
    return ''
  },
  plugins: INSTALLED,
})

/** The settings of the window and of every plugin installed, each reached from its row. */
export const Settings: Story = {
  render: () => window(SETTINGS, SettingsTab, createSettingsState(INSTALLATION)),
}
