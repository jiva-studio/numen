/**
 * The screen the desktop opens on: a plugin's view with the room, the agent
 * along the trailing edge, and both of them tabs that can be moved, split or
 * closed.
 *
 * Where the pieces are judged together — whether a strip of tabs reads as one
 * line of chrome above two very different things. The view is a placeholder:
 * the views themselves are drawn by the plugins' Storybook.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { onScopeDispose, ref } from 'vue'
import { WorkspaceLayout } from '@/features/workspace'
import { Agent } from './agent'
import { branch, pane, type Tab, type Workspace as State } from '@/features/workspace'
import { LONG, MULTILINE } from '@/shared/fixtures/prose'
import type { Turn } from '@/features/thread'

const VIEW = 'view'
const AGENT = 'agent'

const TABS: readonly Tab[] = [
  { id: VIEW, title: 'View' },
  { id: AGENT, title: 'Agent' },
]

/** The view with the room, and the agent along the trailing edge. */
const opening = (): State => ({
  root: branch('root', [pane('main', [VIEW]), pane('aside', [AGENT])], [0.72, 0.28]),
  axis: 'horizontal',
  focus: 'main',
})

const createAsked = (id: string, text: string): Turn => ({ id, voice: 'prompt', text })
const createAnswered = (id: string, text: string): Turn => ({ id, voice: 'response', text })

const OPENING: readonly Turn[] = [
  createAsked('1', 'What is this note linked to?'),
  createAnswered('2', MULTILINE),
]

const meta: Meta = {
  title: 'Application/Main screen',
  parameters: { layout: 'fullscreen' },
  render: () => ({
    components: { WorkspaceLayout, Agent },
    setup() {
      const target = ref<State>(opening())
      const turns = ref<Turn[]>([...OPENING])
      const text = ref('')
      const isWorking = ref(false)
      let next = OPENING.length
      let tick: ReturnType<typeof setInterval> | undefined

      const settle = () => {
        clearInterval(tick)
        tick = undefined
        isWorking.value = false
      }

      /** An answer that arrives a few characters at a time. */
      const onSubmit = (message: string) => {
        turns.value.push(createAsked(`${++next}`, message))
        text.value = ''
        isWorking.value = true

        const id = `${++next}`
        turns.value.push({ id, voice: 'response', text: '', state: 'arriving' })

        const reply = `About “${message}”. ${LONG}`
        let offset = 0
        tick = setInterval(() => {
          offset = Math.min(reply.length, offset + 3)
          const isDone = offset === reply.length
          const index = turns.value.findIndex((turn) => turn.id === id)
          if (index >= 0) {
            turns.value[index] = isDone
              ? { id, voice: 'response', text: reply.slice(0, offset) }
              : { id, voice: 'response', text: reply.slice(0, offset), state: 'arriving' }
          }
          if (isDone) settle()
        }, 16)
      }

      onScopeDispose(settle)

      return { held: target, turns, text, isWorking, onSubmit, TABS, VIEW, AGENT }
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
            <Agent
              v-else
              v-model="text"
              :turns="turns"
              :is-working="working"
              placeholder="Ask about the vault"
              @submit="onSubmit"
            />
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
