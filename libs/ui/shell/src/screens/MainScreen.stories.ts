/**
 * The screen the desktop opens on: a plugin view in the main area and the agent at
 * the trailing edge, both as tabs that can be moved, split or closed.
 *
 * The pieces are judged together: whether a strip of tabs reads as one line of
 * chrome above two very different contents. Both panes are placeholders: the
 * views and the agent chat are rendered by the plugins' Storybook.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { ref } from 'vue'
import {
  WorkspaceLayout,
  branch,
  pane,
  type Tab,
  type Workspace as State,
} from '@/features/workspace'

const VIEW = 'view'
const AGENT = 'agent'

const TABS: readonly Tab[] = [
  { id: VIEW, title: 'View' },
  { id: AGENT, title: 'Agent' },
]

/** The view in the main area and the agent at the trailing edge. */
const opening = (): State => ({
  root: branch('root', [pane('main', [VIEW]), pane('aside', [AGENT])], [0.72, 0.28]),
  axis: 'horizontal',
  focus: 'main',
})

const meta: Meta = {
  title: 'Application/Main screen',
  parameters: { layout: 'fullscreen' },
  render: () => ({
    components: { WorkspaceLayout },
    setup() {
      const target = ref<State>(opening())
      return { held: target, TABS, VIEW, AGENT }
    },
    template: `
      <div style="height: 100vh">
        <WorkspaceLayout v-model="held" :tabs="TABS">
          <template #tab="{ id }">
            <div
              v-if="id === VIEW"
              class="grid h-full place-items-center text-sm text-hushed"
              data-placeholder
            >
              A plugin's view
            </div>
            <div
              v-else
              class="grid h-full place-items-center text-sm text-hushed"
              data-placeholder
            >
              The agent chat
            </div>
          </template>
        </WorkspaceLayout>
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj

/** What the window opens on. */
export const Opening: Story = {}
