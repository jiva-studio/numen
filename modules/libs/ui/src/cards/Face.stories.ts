/**
 * Every situation a face has to survive. Also the test corpus: each story is
 * run in a browser by `@storybook/addon-vitest`.
 *
 * The markdown here is assembled already, as it reaches the component.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent } from 'storybook/test'
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
  'one word': { front: 'Llama', back: '45"' },
  'far too much': { front: LONG, back: LONG },
  'other scripts': {
    front: '# Лама\n\nКакого она роста?',
    back: '**Рост:** около 45″ в холке\n\nधैर्यं सर्वत्र साधनम्',
  },
  unbroken: { front: UNBROKEN, back: `${UNBROKEN} ${UNBROKEN}` },
  'nothing at all': { front: '', back: '   \n  ' },
  'a table': {
    front: 'What does it weigh?',
    back: '| Animal | Weight | Height | Life span | Where |\n| --- | --- | --- | --- | --- |\n| Llama | 130 kg | 45" | 20 years | the Andes |\n| Yak | 350 kg | 63" | 22 years | the Himalaya |',
  },
  tags: {
    front: 'What is a <u>llama</u>?',
    back: '<p>A <b>camelid</b> of the Andes.</p>\n<ruby>駱駝<rt>rakuda</rt></ruby>\n\n<span style="color: teal">Coloured by the card itself.</span>',
  },
  'tags that must not survive': {
    front: 'A deck from somebody else',
    back:
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
  title: 'Cards/Face',
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
      return {
        args,
        turned,
        held: CORPORA,
        onTurn: (next: boolean) => {
          turned.value = next
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
        />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

/** A heading, a question, and a back with a list in it. */
export const ACard: Story = {}

/** A word on each side, and nothing more. */
export const OneWord: Story = { args: { corpus: 'one word' } }

/** Far more prose than the card has room for. */
export const FarTooMuch: Story = { args: { corpus: 'far too much', turned: true } }

/** Text that is not Latin. */
export const OtherScripts: Story = { args: { corpus: 'other scripts', turned: true } }

/** A run of letters with nothing in it to break at. */
export const Unbroken: Story = { args: { corpus: 'unbroken', turned: true } }

/** Both halves empty. */
export const NothingAtAll: Story = { args: { corpus: 'nothing at all', turned: true } }

/** A table wider than the card, which scrolls inside itself. */
export const ATable: Story = { args: { corpus: 'a table', turned: true } }

/** Tags a person wrote among the marks, drawn as tags. */
export const Tags: Story = { args: { corpus: 'tags', turned: true } }

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

/** A deck from somebody else, with what a card may not be drawn with taken out. */
export const TagsThatMustNotSurvive: Story = {
  args: { corpus: 'tags that must not survive', turned: true },
  play: async ({ canvasElement }) => {
    const back = canvasElement.querySelector('[data-half="back"]')
    if (!back) throw new Error('no back to read')

    expect(back.querySelector('script')).toBeNull()
    expect(back.querySelector('iframe')).toBeNull()
    expect(back.querySelector('[onerror]')).toBeNull()
    expect(back.querySelector('a')?.getAttribute('href')).toBeNull()
    expect((window as unknown as Record<string, unknown>)['stolen']).toBeUndefined()
    expect(back.textContent).toContain('Only these words should stand.')
  },
}
