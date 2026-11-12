/**
 * What the thread looks like. What it does is asserted in `Thread.test.ts`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'
import Thread from './Thread.vue'
import type { Turn } from '../../types'
import { STORY_BOXES } from '../../fixtures/frame'
import {
  ARABIC,
  DEVANAGARI,
  LINK,
  LONG,
  MULTILINE,
  RUSSIAN,
  UNBREAKABLE,
} from '../../fixtures/prose'

const meta = {
  title: 'Chat/Thread',
  component: Thread,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'What was said is in a bubble; what came back is text on the ' +
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

const createRequestedTurn = (id: string, text: string, state?: Turn['state']): Turn =>
  state === undefined ? { id, voice: 'prompt', text } : { id, voice: 'prompt', text, state }

const createRespondedTurn = (id: string, text: string, state?: Turn['state']): Turn =>
  state === undefined ? { id, voice: 'response', text } : { id, voice: 'response', text, state }

const renderThread =
  (turns: readonly Turn[]): Render =>
  () => ({
    components: { Thread },
    setup: () => ({ turns, STORY_BOXES }),
    template: `
    <div :class="STORY_BOXES.thread">
      <Thread :turns="turns" class="h-full" />
    </div>
  `,
  })

/** No content yet. */
export const Silent: Story = { render: renderThread([]) }

/**
 * The ordinary case: questions and answers, a run of turns in one voice, and
 * the line breaks that were typed.
 */
export const Playground: Story = {
  render: renderThread([
    createRequestedTurn('1', 'What does a plex draw?'),
    createRespondedTurn('2', 'One node in focus, and everything else placed by its relation.'),
    createRequestedTurn('3', 'Two things.'),
    createRequestedTurn('4', 'And where do the relations come from?'),
    createRespondedTurn('5', MULTILINE),
  ]),
}

/** An answer still arriving, and a turn that did not go. */
export const InFlight: Story = {
  render: renderThread([
    createRequestedTurn('1', 'And the relations?', 'failed'),
    createRequestedTurn('2', 'What does a plex draw?'),
    createRespondedTurn('3', 'One node in focus, and everything else', 'arriving'),
  ]),
}

/** Words with nowhere to break, scripts that are not Latin, and one that runs
 *  the other way. */
export const AwkwardText: Story = {
  render: renderThread([
    createRequestedTurn('1', UNBREAKABLE),
    createRespondedTurn('2', LINK),
    createRequestedTurn('3', DEVANAGARI),
    createRespondedTurn('4', RUSSIAN),
    createRequestedTurn('5', ARABIC),
    createRequestedTurn('6', ''),
  ]),
}

/**
 * An agent at work: the tools it used appear between its messages, and the prose is
 * rendered from its markup and shown without the markup characters.
 */
export const Working: Story = {
  render: renderThread([
    createRequestedTurn('1', 'Add ten children to this note.'),
    createRespondedTurn('2', "I'll look at **Harmonic oscillator** first."),
    { id: '3', voice: 'doing', text: 'link_list' },
    { id: '4', voice: 'doing', text: 'note_read' },
    createRespondedTurn(
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
        ? createRequestedTurn(`${index}`, `Question ${index / 2 + 1}. ${RUSSIAN}`)
        : createRespondedTurn(`${index}`, `Answer ${(index + 1) / 2}. ${LONG}`),
    ),
  ),
}

/** A caller rendering the body of a turn its own way. */
export const OwnTurn: Story = {
  render: () => ({
    components: { Thread },
    setup: () => ({
      turns: [
        createRequestedTurn('1', 'Show me the note.'),
        createRespondedTurn('2', 'entropy.md'),
      ] as readonly Turn[],
      STORY_BOXES,
    }),
    template: `
      <div :class="STORY_BOXES.shortThread">
        <Thread :turns="turns" class="h-full">
          <template #turn="{ turn }">
            <code v-if="turn.voice === 'answered'" :class="STORY_BOXES.code">
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

const getWordPosition = (root: Element, word: string): Position => {
  const walk = document.createTreeWalker(root, NodeFilter.SHOW_TEXT)
  for (let node = walk.nextNode(); node; node = walk.nextNode()) {
    const offset = (node.nodeValue ?? '').indexOf(word)
    if (offset < 0) continue

    const range = document.createRange()
    range.setStart(node, offset)
    range.setEnd(node, offset + word.length)
    const box = range.getBoundingClientRect()
    return { x: box.x + box.width / 2, y: box.y + box.height / 2 }
  }
  throw new Error(`“${word}” is nowhere in the thread`)
}

const mapToFrame = (point: Position): Position => {
  const frame = window.frameElement as HTMLElement | null
  if (!frame) return point

  const box = frame.getBoundingClientRect()
  return {
    x: box.x + point.x * (box.width / window.innerWidth),
    y: box.y + point.y * (box.height / window.innerHeight),
  }
}

const dragAcross = async (from: Position, to: Position): Promise<string | null> => {
  const context = await import('vitest/browser').catch(() => null)
  if (!context) return null

  window.getSelection()?.removeAllRanges()
  await (
    context.commands as unknown as { sweep(from: Position, to: Position): Promise<void> }
  ).sweep(mapToFrame(from), mapToFrame(to))
  await new Promise((isDone) => setTimeout(isDone, 16))

  return String(window.getSelection() ?? '')
}

/**
 * A selection dragged across the turns. What is taken runs from where the
 * pointer went down to where it came up, through everything between and
 * nothing above. It is asserted only where a browser pointer can be driven.
 */
export const Selecting: Story = {
  render: renderThread([
    createRequestedTurn('1', 'Which of these are worth keeping?'),
    createRespondedTurn(
      '2',
      'Three of them:\n\n- the harmonic one\n- the damped one\n- the driven one\n\nThe rest repeat what those already say.',
    ),
    createRequestedTurn('3', 'Then drop the rest.'),
    createRespondedTurn('4', 'Dropped. Nine notes are left in the vault.'),
  ]),
  play: async ({ canvasElement }) => {
    const turns = canvasElement.querySelectorAll('.thread__turn')
    const acquiredItem = await dragAcross(
      getWordPosition(turns[1]!, 'damped'),
      getWordPosition(turns[3]!, 'vault'),
    )
    if (acquiredItem === null) return

    await expect(acquiredItem).toContain('the driven one')
    await expect(acquiredItem).toContain('Then drop the rest')
    await expect(acquiredItem).toContain('Nine notes are left in the')
    await expect(acquiredItem).not.toContain('the harmonic one')
    await expect(acquiredItem).not.toContain('worth keeping')
  },
}
