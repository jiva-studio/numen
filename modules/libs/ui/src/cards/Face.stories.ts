/**
 * Every situation a face has to survive. Also the test corpus: each story is
 * run in a browser by `@storybook/addon-vitest`.
 *
 * The markdown here is assembled already, as it reaches the component.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor } from 'storybook/test'
import { ref, watch } from 'vue'
import Face from './Face.vue'

/** One card, as two halves of markdown. */
interface Corpus {
  readonly front: string
  readonly back: string
}

const LONG = Array.from(
  { length: 24 },
  (_, at) =>
    `Line ${at + 1}. A sentence long enough to wrap, with a **mark** in it and a \`word of code\`.`,
).join('\n\n')

const UNBROKEN =
  'supercalifragilisticexpialidociousandthensomemoreofitwithnothingtobreakatanywherealongitslength'

const CORPORA = {
  'a card': {
    front: '# Llama\n\nHow tall does it stand?',
    back: '**Height:** about 45" at the shoulder\n\n- a list\n- of two things',
  },
  'far too much': { front: LONG, back: LONG },
  /* Text that is not Latin, beside a run of letters with nothing in it to
     break at. */
  'awkward text': {
    front: '# Лама\n\nКакого она роста?',
    back: `**Рост:** около 45″ в холке\n\nधैर्यं सर्वत्र साधनम्\n\n${UNBROKEN} ${UNBROKEN}`,
  },
  'nothing at all': { front: '', back: '   \n  ' },
  /* A card with a link in each half, which is what the caller is handed to
     decide the meaning of. */
  'a card with links': {
    front: 'Where does the [llama](https://example.org/llama) live?',
    back: 'In the [Andes](https://example.org/andes).',
  },
  /* What a card is written with: a table wider than the card, the marks a
     person wrote to be drawn as marks, and the ones a card may not be drawn
     with at all. */
  'what a card carries': {
    front: 'What is a <u>llama</u>, and what does it weigh?',
    back:
      '<p>A <b>camelid</b> of the Andes.</p>\n<ruby>駱駝<rt>rakuda</rt></ruby>\n\n' +
      '<span style="color: teal">Coloured by the card itself.</span>\n\n' +
      '| Animal | Weight | Height | Life span | Where it is kept | What it is kept for |\n' +
      '| --- | --- | --- | --- | --- | --- |\n' +
      '| Llama | 130 kg | 45" | 20 years | the Andes | carrying and wool |\n' +
      '| Yak | 350 kg | 63" | 22 years | the Himalaya | milk, wool and ploughing |\n\n' +
      '<script>window.stolen = 1</script>\n' +
      '<img src="x" onerror="window.stolen = 2">\n' +
      '<a href="javascript:window.stolen = 3">a link</a>\n' +
      '<iframe src="https://example.org"></iframe>\n' +
      'Only these words should stand.',
  },
} satisfies Record<string, Corpus>

type Corpora = keyof typeof CORPORA

interface Knobs {
  corpus: Corpora
  turned: boolean
  name: string
  silence: string
  turning: string
  /** Given by the story, and nothing a reader turns. */
  front?: never
  back?: never
  words?: never
}

const meta: Meta<Knobs> = {
  title: 'Flash Cards/Face',
  component: Face,
  argTypes: {
    corpus: {
      control: 'select',
      options: Object.keys(CORPORA),
      description: 'What the card is drawn from. Changing it starts afresh.',
    },
    turned: { control: 'boolean' },
    name: { control: 'text' },
    silence: { control: 'text' },
    turning: { control: 'text' },
    front: { table: { disable: true } },
    back: { table: { disable: true } },
    words: { table: { disable: true } },
  },
  args: {
    corpus: 'a card',
    turned: false,
    name: 'Card',
    silence: 'Nothing here',
    turning: 'Turn',
  },
  render: (args) => ({
    components: { Face },
    setup() {
      const turned = ref(args.turned)
      watch(
        () => args.turned,
        (next) => {
          turned.value = next
        },
      )
      /** Where the last link pressed pointed, which is all the caller does with it. */
      const went = ref('')
      return {
        args,
        turned,
        went,
        held: CORPORA,
        onTurn: (next: boolean) => {
          turned.value = next
        },
        /* What a link means is the caller's, and a card in a window of its own
           is read where it stands: the press goes no further. */
        onFollow: (href: string, press: MouseEvent) => {
          press.preventDefault()
          went.value = href
        },
      }
    },
    template: `
      <div style="max-width: 28rem; padding: 1.5rem">
        <Face
          :front="held[args.corpus].front"
          :back="held[args.corpus].back"
          :turned="turned"
          :name="args.name"
          :words="{ silence: args.silence, turning: args.turning }"
          @turn="onTurn"
          @follow="onFollow"
        />
        <p v-if="went" data-went>{{ went }}</p>
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

/** A heading, a question, and a back with a list in it. */
export const ACard: Story = {}

/** Far more prose than the card has room for. */
export const FarTooMuch: Story = { args: { corpus: 'far too much', turned: true } }

/** Text that is not Latin, and a run of letters with nothing to break at. */
export const AwkwardText: Story = { args: { corpus: 'awkward text', turned: true } }

/** Both halves empty. */
export const NothingAtAll: Story = { args: { corpus: 'nothing at all', turned: true } }

/** The back arrives when the card is turned, and not before. */
export const Turning: Story = {
  play: async ({ canvasElement }) => {
    const back = () => canvasElement.querySelector('[data-half="back"]')
    expect(back()).toBeNull()

    const button = canvasElement.querySelector<HTMLButtonElement>('.face__turn')
    if (!button) throw new Error('no button to turn it by')
    expect(button.tagName).toBe('BUTTON')
    await userEvent.click(button)

    expect(back()).not.toBeNull()
    expect(back()?.textContent).toContain('45"')
  },
}

/**
 * A link pressed in a card hands the caller where it points and the press
 * itself, so a caller that wants the card to stay where it is can stop it.
 */
export const ALinkPressed: Story = {
  args: { corpus: 'a card with links' },
  play: async ({ canvasElement }) => {
    const link = canvasElement.querySelector<HTMLAnchorElement>('[data-half="front"] a')
    if (!link) throw new Error('no link on the front')
    expect(link.getAttribute('href')).toBe('https://example.org/llama')

    const press = new MouseEvent('click', { bubbles: true, cancelable: true })
    link.dispatchEvent(press)

    // The caller was handed the press, and stopped it: the window stays here.
    expect(press.defaultPrevented).toBe(true)
    await waitFor(() => {
      expect(canvasElement.querySelector('[data-went]')?.textContent).toBe(
        'https://example.org/llama',
      )
    })
  },
}

/**
 * What a card is written with: a table wider than the card, which scrolls
 * inside itself; the marks a person wrote, drawn as marks; and the ones a card
 * may not be drawn with, taken out.
 */
export const WhatACardCarries: Story = {
  args: { corpus: 'what a card carries', turned: true },
  play: async ({ canvasElement }) => {
    const back = canvasElement.querySelector('[data-half="back"]')
    if (!back) throw new Error('no back to read')

    // The marks a person wrote stand as marks.
    expect(back.querySelector('b')).not.toBeNull()
    expect(back.querySelector('ruby')).not.toBeNull()

    // The table is read sideways inside itself, and the card does not widen.
    const table = back.querySelector('table')
    if (!table) throw new Error('no table on the back')
    expect(getComputedStyle(table).overflowX).toBe('auto')
    expect(table.scrollWidth).toBeGreaterThan(table.clientWidth)
    expect(back.scrollWidth).toBeLessThanOrEqual(back.clientWidth + 1)

    // And the ones a card may not be drawn with are nowhere.
    expect(back.querySelector('script')).toBeNull()
    expect(back.querySelector('iframe')).toBeNull()
    expect(back.querySelector('[onerror]')).toBeNull()
    expect(back.querySelector('a')?.getAttribute('href')).toBeNull()
    expect((window as unknown as Record<string, unknown>)['stolen']).toBeUndefined()
    expect(back.textContent).toContain('Only these words should stand.')
  },
}
