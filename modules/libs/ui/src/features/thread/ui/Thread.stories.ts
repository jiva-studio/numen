/**
 * What the thread looks like. What it does is asserted in `Thread.test.ts`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'
import Thread from './Thread.vue'
import type { Turn } from '../lib/turn'
import { ARABIC, DEVANAGARI, LINK, LONG, MULTILINE, RUSSIAN, UNBREAKABLE } from '@/shared/fixtures/prose'

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

const createAsked = (id: string, text: string, state?: Turn['state']): Turn =>
  state === undefined ? { id, voice: 'asked', text } : { id, voice: 'asked', text, state }

const back = (id: string, text: string, state?: Turn['state']): Turn =>
  state === undefined ? { id, voice: 'answered', text } : { id, voice: 'answered', text, state }

const renderThread = (turns: readonly Turn[]): Render => () => ({
  components: { Thread },
  setup: () => ({ turns }),
  template: `
    <div class="numen h-[460px] w-[420px] max-w-[calc(100vw-2rem)] rounded-panel border border-rule bg-surface px-4 py-3">
      <Thread :turns="turns" class="h-full" />
    </div>
  `,
})

/** Nothing said yet. */
export const Silent: Story = { render: renderThread([]) }

/**
 * The ordinary case: questions and answers, a run of turns in one voice, and
 * the line breaks that were typed.
 */
export const Playground: Story = {
  render: renderThread([
    createAsked('1', 'What does a plex draw?'),
    back('2', 'One node in focus, and everything else placed by its seat.'),
    createAsked('3', 'Two things.'),
    createAsked('4', 'And where do the seats come from?'),
    back('5', MULTILINE),
  ]),
}

/** An answer still arriving, and a turn that did not go. */
export const InFlight: Story = {
  render: renderThread([
    createAsked('1', 'And the seats?', 'failed'),
    createAsked('2', 'What does a plex draw?'),
    back('3', 'One node in focus, and everything else', 'arriving'),
  ]),
}

/** Words with nowhere to break, scripts that are not Latin, and one that runs
 *  the other way. */
export const AwkwardText: Story = {
  render: renderThread([
    createAsked('1', UNBREAKABLE),
    back('2', LINK),
    createAsked('3', DEVANAGARI),
    back('4', RUSSIAN),
    createAsked('5', ARABIC),
    createAsked('6', ''),
  ]),
}

/**
 * An agent at work: what it reached for sits between what it said, and the
 * prose is read as it was marked up rather than shown with its marks.
 */
export const Working: Story = {
  render: renderThread([
    createAsked('1', 'Add ten children to this note.'),
    back('2', "I'll look at **Harmonic oscillator** first."),
    { id: '3', voice: 'doing', text: 'note_neighbourhood' },
    { id: '4', voice: 'doing', text: 'note_read' },
    back(
      '5',
      'The style is clear:\n\n- a `# Title` line\n- one evocative sentence\n- a `part of` link to the parent\n\nCreating the ten children now.',
    ),
    { id: '6', voice: 'doing', text: 'note_create', state: 'arriving' },
  ]),
}

/** Far too many. What the scrolling is for. */
export const FarTooMany: Story = {
  render: renderThread(
    Array.from({ length: 200 }, (_, index) =>
      index % 2 === 0
        ? createAsked(`${index}`, `Question ${index / 2 + 1}. ${RUSSIAN}`)
        : back(`${index}`, `Answer ${(index + 1) / 2}. ${LONG}`),
    ),
  ),
}

/** A caller rendering the body of a turn its own way. */
export const OwnTurn: Story = {
  render: () => ({
    components: { Thread },
    setup: () => ({
      turns: [createAsked('1', 'Show me the note.'), back('2', 'entropy.md')] as readonly Turn[],
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

interface Position {
  readonly x: number
  readonly y: number
}

/** The middle of a word, in the coordinates of the page it is drawn on. */
const wordAt = (root: Element, word: string): Position => {
  const walk = document.createTreeWalker(root, NodeFilter.SHOW_TEXT)
  for (let node = walk.nextNode(); node; node = walk.nextNode()) {
    const at = (node.nodeValue ?? '').indexOf(word)
    if (at < 0) continue

    const range = document.createRange()
    range.setStart(node, at)
    range.setEnd(node, at + word.length)
    const box = range.getBoundingClientRect()
    return { x: box.x + box.width / 2, y: box.y + box.height / 2 }
  }
  throw new Error(`“${word}” is nowhere in the thread`)
}

/** The same place told to the window the story is framed in. */
const mapToFrame = (at: Position): Position => {
  const frame = window.frameElement as HTMLElement | null
  if (!frame) return at

  const box = frame.getBoundingClientRect()
  return {
    x: box.x + at.x * (box.width / window.innerWidth),
    y: box.y + at.y * (box.height / window.innerHeight),
  }
}

/**
 * A drag the browser makes itself, so the selection it leaves is the browser's
 * own. The places are told to the window the story is framed in, which is where
 * the pointer is driven.
 */
const dragged = async (from: Position, to: Position): Promise<string | null> => {
  const context = await import('vitest/browser').catch(() => null)
  if (!context) return null

  window.getSelection()?.removeAllRanges()
  await context.commands.sweep(mapToFrame(from), mapToFrame(to))
  await new Promise((done) => setTimeout(done, 16))

  return String(window.getSelection() ?? '')
}

/**
 * A selection dragged across the turns. What is taken runs from where the
 * pointer went down to where it came up, through everything between and
 * nothing above. It is asserted only where a browser pointer can be driven.
 */
export const Selecting: Story = {
  render: renderThread([
    createAsked('1', 'Which of these are worth keeping?'),
    back(
      '2',
      'Three of them:\n\n- the harmonic one\n- the damped one\n- the driven one\n\nThe rest repeat what those already say.',
    ),
    createAsked('3', 'Then drop the rest.'),
    back('4', 'Dropped. Nine notes are left in the vault.'),
  ]),
  play: async ({ canvasElement }) => {
    const turns = canvasElement.querySelectorAll('.thread__turn')
    const taken = await dragged(wordAt(turns[1]!, 'damped'), wordAt(turns[3]!, 'vault'))
    if (taken === null) return

    await expect(taken).toContain('the driven one')
    await expect(taken).toContain('Then drop the rest')
    await expect(taken).toContain('Nine notes are left in the')
    await expect(taken).not.toContain('the harmonic one')
    await expect(taken).not.toContain('worth keeping')
  },
}
