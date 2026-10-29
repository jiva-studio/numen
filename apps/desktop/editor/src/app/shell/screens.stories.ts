/**
 * The window around the settings, the tab the window adds to what the library draws.
 *
 * Nothing here reaches a vault. The tab receives the state it would hold, so it
 * renders in its settled state and can be captured as a screenshot.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { WorkspaceLayout, pane } from '@numen/shell'
import type { Tab, Workspace } from '@numen/shell'
import { ref, type Component } from 'vue'

import { SettingsTab, type Installation } from '@/pages/settings'
import type { Model, PluginInfo } from '@/entities/settings'
import { findTabKindIcon, SETTINGS } from '@/entities/tab'

/** Every tab this file draws, under the identity the window opens it at. */
const TABS: readonly Tab[] = [{ id: SETTINGS, title: 'Settings' }]

/** A tab is drawn as its kind, which is the prefix of the identity. */
const getTabIcon = (id: string) => findTabKindIcon(id.split(':')[0] ?? id)

/** One tab filling the window, drawn in the chrome the window draws it in. */
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
  namedAt: ['agent', 'use'],
  name: 'claude',
  title: 'Claude Code',
  group: '',
  isDefault: true,
  writes: [{ at: ['agent', 'use'], value: '"claude"' }],
  presence: 'present',
}

/** One plugin as the settings list it: active, with no settings, secrets or failure of its own. */
const plugin = (id: string, name: string, description: string): PluginInfo => ({
  id,
  name,
  version: '1.0.0',
  description,
  isActive: true,
  settings: [],
  capabilities: [],
  secrets: [],
  storedSecrets: [],
  consent: 'none',
  hasFrontend: false,
})

/** Plugins of every category the list sorts by, as an installation ships them. */
const PLUGINS: readonly PluginInfo[] = [
  plugin(
    'md.numen.agents.claude',
    'Claude Code Agent',
    'Claude Code CLI agent runner and provider service',
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

const INSTALLATION: Installation = {
  themes: ref([
    { name: 'preset:numen', title: 'numen', isBuiltIn: true, isPinned: false },
    { name: 'preset:dracula', title: 'dracula', isBuiltIn: true, isPinned: true },
    { name: 'preset:solarized', title: 'solarized', isBuiltIn: true, isPinned: false },
    { name: 'mine:sea', title: 'sea', isBuiltIn: false, isPinned: false },
  ]),
  applied: ref('preset:numen'),
  mode: ref('system'),
  isPinned: ref(false),
  sizes: ref({ interfaceScale: 1, textScale: 1.25 }),
  bounds: ref({
    interfaceScale: { min: 0.8, max: 2 },
    textScale: { min: 0.8, max: 1.75 },
  }),
  choose: () => {
    DONE.push('choose')
  },
  isSyncing: ref(true),
  writeSecret: () => {
    DONE.push('key')
  },
  deleteSecret: () => {
    DONE.push('key gone')
  },
  retryPlugin: () => {
    DONE.push('plugin retried')
  },
  choosePath: async () => {
    DONE.push('path chosen')
    return ''
  },
  getSetting: () => undefined,
  getModels: (path) => (path.join('.') === 'agent.use' ? [AGENT_USE] : []),
  write: (edits) => {
    DONE.push(`write ${edits.length}`)
  },
  file: ref('/numen.json'),
  plugins: ref(PLUGINS),
}

/** The settings of the window and of every plugin installed, each reached from its row. */
export const Settings: Story = {
  render: () => window(SETTINGS, SettingsTab, { installation: INSTALLATION }),
}
