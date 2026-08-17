/**
 * The conversation and the field it is carried on with, in one piece.
 *
 * Where the two are judged against each other: whether the composer stays put
 * under a thread that scrolls, whether the field growing takes its height from
 * the conversation, whether one type size holds across both.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import { onScopeDispose, ref } from 'vue'
import Agent from './Agent.vue'
import type { Turn } from '@/thread/model'
import { framed } from '@/fixtures/frame'
import { LONG, MULTILINE, RUSSIAN } from '@/fixtures/prose'

const meta = {
  title: 'Chat/Agent',
  decorators: [framed],
  parameters: { layout: 'fullscreen' },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>
type Render = NonNullable<Story['render']>

const said = (id: string, text: string): Turn => ({ id, voice: 'asked', text })
const back = (id: string, text: string): Turn => ({ id, voice: 'answered', text })

/**
 * The thread takes what height is left and scrolls inside it; the composer
 * takes what it needs and is written over the foot of it.
 */
const TEMPLATE = `
  <Agent
    v-model="text"
    class="h-full"
    :turns="turns"
    :working="working"
    placeholder="Ask about this note"
    @submit="onSubmit"
  />
`

/**
 * A live one. Sending adds the turn, the dots take the button's place, and
 * the answer arrives a few characters at a time.
 */
const conversation = (start: readonly Turn[]): Render => () => ({
  components: { Agent },
  setup() {
    const turns = ref<Turn[]>([...start])
    const text = ref('')
    const working = ref(false)
    let next = start.length
    let tick: ReturnType<typeof setInterval> | undefined

    /** What has arrived so far, put back in place of what was there. */
    const arrived = (id: string, soFar: string, done: boolean) => {
      const index = turns.value.findIndex((turn) => turn.id === id)
      if (index < 0) return
      turns.value[index] = done
        ? { id, voice: 'answered', text: soFar }
        : { id, voice: 'answered', text: soFar, state: 'arriving' }
    }

    const settle = () => {
      clearInterval(tick)
      tick = undefined
      working.value = false
    }

    const onSubmit = (asked: string) => {
      turns.value.push(said(`${next++}`, asked))
      text.value = ''
      working.value = true

      const id = `${next++}`
      turns.value.push({ id, voice: 'answered', text: '', state: 'arriving' })

      const reply = `You asked about “${asked}”. ${LONG}`
      let at = 0
      tick = setInterval(() => {
        at = Math.min(reply.length, at + 3)
        const done = at === reply.length
        arrived(id, reply.slice(0, at), done)
        if (done) settle()
      }, 16)
    }

    onScopeDispose(settle)

    return { turns, text, working, onSubmit }
  },
  template: TEMPLATE,
})

/** Type into it and press Enter. */
export const Playground: Story = {
  render: conversation([
    said('1', 'What does a plex draw?'),
    back('2', 'One node in focus, and everything else placed by its seat.'),
    said('3', 'And where do the seats come from?'),
    back('4', MULTILINE),
  ]),
}

/** Opened on nothing, which is what a new conversation is. */
export const Fresh: Story = {
  render: conversation([]),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const field = canvas.getByRole('textbox')
    await userEvent.click(field)
    await userEvent.type(field, 'what is a seat')
    await userEvent.keyboard('{Enter}')
    await expect(canvas.getByText('what is a seat')).toBeInTheDocument()
    await expect((field as HTMLTextAreaElement).value).toBe('')
  },
}

/** A long conversation, so the composer is under a thread that scrolls. */
export const LongConversation: Story = {
  render: conversation(
    Array.from({ length: 60 }, (_, index) =>
      index % 2 === 0
        ? said(`${index}`, `Question ${index / 2 + 1}. ${RUSSIAN}`)
        : back(`${index}`, `Answer ${(index + 1) / 2}. ${LONG}`),
    ),
  ),
}
