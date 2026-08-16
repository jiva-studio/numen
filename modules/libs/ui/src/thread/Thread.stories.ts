/**
 * What the thread looks like. What it does is asserted in `Thread.test.ts`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import Thread from './Thread.vue'
import type { Turn } from './model'
import { ARABIC, DEVANAGARI, LINK, LONG, MULTILINE, RUSSIAN, UNBREAKABLE } from '@/fixtures/prose'

const meta = {
  title: 'Chat/Thread',
  component: Thread,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'What was said sits in a bubble; what came back is text on the ' +
          'surface. It scrolls on its own and stays at the foot while an ' +
          'answer arrives. What a turn’s text is made of belongs to whoever ' +
          'renders the thread — the default keeps the line breaks and does ' +
          'nothing else with it.',
      },
    },
  },
  args: { turns: [] },
} satisfies Meta<typeof Thread>

export default meta
type Story = StoryObj<typeof meta>
type Render = NonNullable<Story['render']>

const said = (id: string, text: string, state?: Turn['state']): Turn =>
  state === undefined ? { id, voice: 'asked', text } : { id, voice: 'asked', text, state }

const back = (id: string, text: string, state?: Turn['state']): Turn =>
  state === undefined ? { id, voice: 'answered', text } : { id, voice: 'answered', text, state }

const framed = (turns: readonly Turn[]): Render => () => ({
  components: { Thread },
  setup: () => ({ turns }),
  template: `
    <div class="numen h-[460px] w-[420px] max-w-[calc(100vw-2rem)] rounded-panel border border-rule bg-surface px-4 py-3">
      <Thread :turns="turns" class="h-full" />
    </div>
  `,
})

/** Nothing said yet. */
export const Silent: Story = { render: framed([]) }

/**
 * The ordinary case: questions and answers, a run of turns in one voice, and
 * the line breaks that were typed.
 */
export const Playground: Story = {
  render: framed([
    said('1', 'What does a plex draw?'),
    back('2', 'One node in focus, and everything else placed by its seat.'),
    said('3', 'Two things.'),
    said('4', 'And where do the seats come from?'),
    back('5', MULTILINE),
  ]),
}

/** An answer still arriving, with the caret on its last line, and a turn that
 *  did not go. */
export const InFlight: Story = {
  render: framed([
    said('1', 'And the seats?', 'failed'),
    said('2', 'What does a plex draw?'),
    back('3', 'One node in focus, and everything else', 'arriving'),
  ]),
}

/** Words with nowhere to break, scripts that are not Latin, and one that runs
 *  the other way. */
export const AwkwardText: Story = {
  render: framed([
    said('1', UNBREAKABLE),
    back('2', LINK),
    said('3', DEVANAGARI),
    back('4', RUSSIAN),
    said('5', ARABIC),
    said('6', ''),
  ]),
}

/** Far too many. What the scrolling is for. */
export const FarTooMany: Story = {
  render: framed(
    Array.from({ length: 200 }, (_, index) =>
      index % 2 === 0
        ? said(`${index}`, `Question ${index / 2 + 1}. ${RUSSIAN}`)
        : back(`${index}`, `Answer ${(index + 1) / 2}. ${LONG}`),
    ),
  ),
}

/** A caller rendering the body of a turn its own way. */
export const OwnTurn: Story = {
  render: () => ({
    components: { Thread },
    setup: () => ({
      turns: [said('1', 'Show me the note.'), back('2', 'entropy.md')] as readonly Turn[],
    }),
    template: `
      <div class="numen h-[300px] w-[420px] rounded-panel border border-rule bg-surface px-4 py-3">
        <Thread :turns="turns" class="h-full">
          <template #turn="{ turn }">
            <code v-if="turn.voice === 'answered'" class="rounded-node bg-raised px-1.5 py-0.5">
              {{ turn.text }}
            </code>
            <span v-else>{{ turn.text }}</span>
          </template>
        </Thread>
      </div>
    `,
  }),
}
