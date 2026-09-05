/**
 * The screen the desktop opens on: the plex with the room, the agent along the
 * trailing edge, and both of them tabs that can be moved, split or closed.
 *
 * Where the pieces are judged together — whether the plex has the width it
 * needs beside a conversation, whether a strip of tabs reads as one line of
 * chrome above two very different things.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { onScopeDispose, ref } from 'vue'
import Workspace from '@/workspace/Workspace.vue'
import Plex from '@/plex/Plex.vue'
import Agent from './Agent.vue'
import { branch, pane, type Tab, type Workspace as State } from '@/workspace/node'
import { neighbourhoods } from '@/plex/fixtures/neighbourhoods'
import { LONG, MULTILINE } from '@/fixtures/prose'
import type { PlexNeighbourhood } from '@/plex/neighbourhood'
import type { Turn } from '@/thread/turn'

const PLEX = 'plex'
const AGENT = 'agent'

const TABS: readonly Tab[] = [
  { id: PLEX, title: 'Plex' },
  { id: AGENT, title: 'Agent' },
]

/** The plex with the room, and the agent along the trailing edge. */
const opening = (): State => ({
  root: branch('root', [pane('main', [PLEX]), pane('aside', [AGENT])], [0.72, 0.28]),
  axis: 'horizontal',
  focus: 'main',
})

interface Knobs {
  neighbourhood: PlexNeighbourhood
}

const said = (id: string, text: string): Turn => ({ id, voice: 'asked', text })
const back = (id: string, text: string): Turn => ({ id, voice: 'answered', text })

const OPENING: readonly Turn[] = [
  said('1', 'What is this note linked to?'),
  back('2', MULTILINE),
]

const meta: Meta<Knobs> = {
  title: 'Application/Main screen',
  parameters: { layout: 'fullscreen' },
  argTypes: {
    neighbourhood: {
      control: 'select',
      options: Object.keys(neighbourhoods),
      mapping: neighbourhoods,
      description: 'What the plex is looking at.',
    },
  },
  args: { neighbourhood: neighbourhoods.typical },
  render: (args) => ({
    components: { Workspace, Plex, Agent },
    setup() {
      const held = ref<State>(opening())
      const turns = ref<Turn[]>([...OPENING])
      const text = ref('')
      const working = ref(false)
      let next = OPENING.length
      let tick: ReturnType<typeof setInterval> | undefined

      const settle = () => {
        clearInterval(tick)
        tick = undefined
        working.value = false
      }

      /** An answer that arrives a few characters at a time. */
      const onSubmit = (asked: string) => {
        turns.value.push(said(`${++next}`, asked))
        text.value = ''
        working.value = true

        const id = `${++next}`
        turns.value.push({ id, voice: 'answered', text: '', state: 'arriving' })

        const reply = `About “${asked}”. ${LONG}`
        let at = 0
        tick = setInterval(() => {
          at = Math.min(reply.length, at + 3)
          const done = at === reply.length
          const index = turns.value.findIndex((turn) => turn.id === id)
          if (index >= 0) {
            turns.value[index] = done
              ? { id, voice: 'answered', text: reply.slice(0, at) }
              : { id, voice: 'answered', text: reply.slice(0, at), state: 'arriving' }
          }
          if (done) settle()
        }, 16)
      }

      onScopeDispose(settle)

      return { held, args, turns, text, working, onSubmit, TABS, PLEX, AGENT }
    },
    template: `
      <div style="height: 100vh">
        <Workspace v-model="held" :tabs="TABS">
          <template #tab="{ id }">
            <Plex
              v-if="id === PLEX"
              :neighbourhood="args.neighbourhood"
              :creatable="[]"
            />
            <Agent
              v-else
              v-model="text"
              :turns="turns"
              :working="working"
              placeholder="Ask about the vault"
              @submit="onSubmit"
            />
          </template>
        </Workspace>
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

/** What the window opens on. */
export const Opening: Story = {}
