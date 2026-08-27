/**
 * Every situation the deck editor has to survive. Also the test corpus: each
 * story is run in a browser by `@storybook/addon-vitest`.
 *
 * What the editor hands back is applied here, which is the application's part.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent } from 'storybook/test'
import { ref, watch } from 'vue'
import Deck from './Deck.vue'
import { ordered, type Cut, type Drawn, type Filled, type Landing } from './model'

interface Corpus {
  readonly cards: readonly Drawn[]
  readonly cuts: readonly Cut[]
}

const ANIMAL: Cut = { name: 'Animal', fields: ['Name', 'Height', 'Weight', 'Life span'] }
const WORD: Cut = { name: 'Word', fields: ['Word', 'Meaning', 'Example'] }
const ASKED: Cut = { name: 'Basic', fields: ['Question', 'Answer'] }

const UNBROKEN =
  'supercalifragilisticexpialidociousandthensomemoreofitwithnothingtobreakatanywhere'

const many = (count: number): readonly Drawn[] =>
  Array.from({ length: count }, (_, at) => ({
    id: `card-${at}`,
    name: `Word ${at + 1}`,
    stencil: 'Word',
    filled: [
      { field: 'Meaning', text: `The ${at + 1}th of them.` },
      { field: 'Example', text: 'A sentence it stands in.' },
    ],
  }))

const CORPORA = {
  'a deck': {
    cuts: [ANIMAL, WORD],
    cards: [
      {
        id: 'llama',
        name: 'Llama',
        stencil: 'Animal',
        filled: [
          { field: 'Height', text: 'about 45" *at the shoulder*' },
          { field: 'Life span', text: 'about **20 years**' },
        ],
      },
      {
        id: 'yak',
        name: 'Yak',
        stencil: 'Animal',
        filled: [
          { field: 'Height', text: 'about 63"' },
          { field: 'Weight', text: '- bull: 350 kg\n- cow: 300 kg' },
          { field: 'Life span', text: 'about 22 years' },
        ],
      },
      {
        id: 'llano',
        name: 'llano',
        stencil: 'Word',
        filled: [
          { field: 'Meaning', text: 'a plain' },
          { field: 'Example', text: '' },
        ],
      },
    ],
  },
  one: {
    cuts: [WORD],
    cards: [
      {
        id: 'only',
        name: 'Only',
        stencil: 'Word',
        filled: [
          { field: 'Meaning', text: 'the one card there is' },
        ],
      },
    ],
  },
  'far too many': { cuts: [WORD], cards: many(300) },
  /* Where the tiles differ most: two stencils, one holding four fields and one
     holding two, and cards leaving fields out. */
  /* A value with no newline in it that is far wider than the box, so what the
     box stands at can only be worked out from where the text wraps. */
  'a value that wraps': {
    cuts: [ASKED],
    cards: [
      {
        id: 'faith',
        name: 'śraddhā',
        stencil: 'Basic',
        filled: [
          {
            field: 'Answer',
            text: 'вера, рождённая из слушания, — та, что приходит не от рассуждения и не от опыта, а от услышанного слова, и держится на нём одном.',
          },
        ],
      },
      { id: 'short', name: 'a short one', stencil: 'Basic', filled: [] },
    ],
  },
  'stencils of different sizes': {
    cuts: [ANIMAL, ASKED],
    cards: [
      ...Array.from({ length: 12 }, (_, at) => ({
        id: `beast-${at}`,
        name: `Beast ${at + 1}`,
        stencil: 'Animal',
        filled: [
          { field: 'Height', text: 'about 45"' },
          { field: 'Weight', text: '- bull: 350 kg\n- cow: 300 kg\n- calf: 40 kg' },
          { field: 'Life span', text: 'about 20 years' },
        ],
      })),
      ...Array.from({ length: 12 }, (_, at) => ({
        id: `asked-${at}`,
        name: `Question ${at + 1}`,
        stencil: 'Basic',
        filled: [],
      })),
    ],
  },
  'other scripts': {
    cuts: [{ name: 'Слово', fields: ['Слово', 'Перевод', 'Пример'] }],
    cards: [
      {
        id: 'лама',
        name: 'Лама',
        stencil: 'Слово',
        filled: [
          { field: 'Перевод', text: 'llama, южноамериканское животное' },
          { field: 'Пример', text: 'धैर्यं सर्वत्र साधनम्' },
        ],
      },
    ],
  },
  unbroken: {
    cuts: [{ name: UNBROKEN, fields: [UNBROKEN, 'Long'] }],
    cards: [
      {
        id: 'run-on',
        name: UNBROKEN,
        stencil: UNBROKEN,
        filled: [
          {
            field: 'Long',
            text: 'A value that goes on and on, well past the width any tile drawing it is likely to have, and then a little further.',
          },
        ],
      },
    ],
  },
  'nothing at all': { cuts: [ANIMAL, WORD], cards: [] },
  'no text at all': {
    cuts: [WORD],
    cards: [{ id: 'blank', name: '', stencil: 'Word', filled: [] }],
  },
  'no stencil': {
    cuts: [ANIMAL],
    cards: [
      {
        id: 'orphan',
        name: 'Orphan',
        stencil: 'Gone',
        filled: [{ field: 'Whatever it had', text: 'still here, still readable' }],
      },
    ],
  },
  /* A card carrying the field that names it as a value as well. The heading
     names it, and the value is kept where a person can see and remove it. */
  'named twice': {
    cuts: [ANIMAL],
    cards: [
      {
        id: 'twice',
        name: 'Llama',
        stencil: 'Animal',
        filled: [
          { field: 'Name', text: 'Alpaca' },
          { field: 'Height', text: 'about 45"' },
        ],
      },
    ],
  },
  'tags that must not survive': {
    cuts: [ASKED],
    cards: [
      {
        id: 'theirs',
        name: 'A deck from somebody else',
        stencil: 'Basic',
        filled: [
          {
            field: 'Answer',
            text:
              '<script>window.stolen = 1</script>' +
              '<img src="x" onerror="window.stolen = 2">' +
              'Only these words should stand.',
          },
        ],
      },
    ],
  },
} satisfies Record<string, Corpus>

type Corpora = keyof typeof CORPORA

interface Knobs {
  corpus: Corpora
  name: string
  /** How wide the window drawing the deck is. */
  width: string
  /** Given by the story, and nothing a reader turns. */
  cards?: never
  cuts?: never
  words?: never
}

const meta: Meta<Knobs> = {
  title: 'Cards/Deck',
  component: Deck,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    corpus: {
      control: 'select',
      options: Object.keys(CORPORA),
      description: 'What the deck starts as. Changing it starts afresh.',
    },
    name: { control: 'text' },
    width: { control: 'text' },
    cards: { table: { disable: true } },
    cuts: { table: { disable: true } },
    words: { table: { disable: true } },
  },
  args: { corpus: 'a deck', name: 'Deck', width: '100%' },
  render: (args) => ({
    components: { Deck },
    setup() {
      const cards = ref<readonly Drawn[]>(CORPORA[args.corpus].cards)
      const cuts = ref<readonly Cut[]>(CORPORA[args.corpus].cuts)

      watch(
        () => args.corpus,
        (next) => {
          cards.value = CORPORA[next].cards
          cuts.value = CORPORA[next].cuts
        },
      )

      /** The cards, with one of them changed. */
      const changed = (id: string, into: (card: Drawn) => Drawn): readonly Drawn[] =>
        cards.value.map((card) => (card.id === id ? into(card) : card))

      return {
        args,
        cards,
        cuts,
        onAdd: (name: string, stencil: string, filled: readonly Filled[]) => {
          const id = `card-${cards.value.length}-${stencil}`
          cards.value = [...cards.value, { id, name, stencil, filled }]
        },
        onRemove: (id: string) => {
          cards.value = cards.value.filter((card) => card.id !== id)
        },
        onMove: (id: string, at: Landing) => {
          const order = ordered(cards.value.map((card) => card.id), id, at)
          cards.value = order.flatMap((each) => cards.value.filter((card) => card.id === each))
        },
        /* A card is named by its first field, and that value stands in the
           heading and in none of the card's values. */
        onWrite: (id: string, field: string, text: string) => {
          cards.value = changed(id, (card) => {
            const first = cuts.value.find((cut) => cut.name === card.stencil)?.fields[0]
            if (field === first) return { ...card, name: text }

            const filled = card.filled.some((each) => each.field === field)
              ? card.filled.map((each) => (each.field === field ? { field, text } : each))
              : [...card.filled, { field, text }]
            return { ...card, filled }
          })
        },
      }
    },
    template: `
      <div :style="{ height: '100vh', width: args.width }">
        <Deck
          :cards="cards"
          :cuts="cuts"
          :name="args.name"
          @add="onAdd"
          @remove="onRemove"
          @move="onMove"
          @write="onWrite"
        />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

/** Three cards, two stencils, and one card leaving a field out. */
export const ADeck: Story = {}

/** One card, and nothing to move it among. */
export const One: Story = { args: { corpus: 'one' } }

/** Three hundred cards, far more than the window has room for. */
export const FarTooMany: Story = { args: { corpus: 'far too many' } }

/** Names and values that are not Latin. */
export const OtherScripts: Story = { args: { corpus: 'other scripts' } }

/** A name and a value with nothing in them to break at. */
export const Unbroken: Story = { args: { corpus: 'unbroken' } }

/** No cards, and only the plus, which is drawn as it is drawn anywhere else. */
export const NothingAtAll: Story = {
  args: { corpus: 'nothing at all' },
  play: async ({ canvasElement }) => {
    const plus = found(canvasElement, '[data-plus]')
    const button = found(canvasElement, '[data-plus] button')

    expect(canvasElement.querySelectorAll('[data-card]')).toHaveLength(0)
    expect(button.textContent?.trim()).toBe('')

    const box = plus.getBoundingClientRect()
    const at = button.getBoundingClientRect()
    expect(Math.round(at.left + at.width / 2)).toBe(Math.round(box.left + box.width / 2))
    expect(Math.round(at.top + at.height / 2)).toBe(Math.round(box.top + box.height / 2))
    expect(box.height).toBeGreaterThan(64)
  },
}

/** A card whose name and every value is the empty string. */
export const NoTextAtAll: Story = { args: { corpus: 'no text at all' } }

/** A card naming a stencil that was not handed in. */
export const NoStencil: Story = { args: { corpus: 'no stencil' } }

/** A card carrying the field that names it as a value as well. */
export const NamedTwice: Story = {
  args: { corpus: 'named twice' },
  play: async ({ canvasElement }) => {
    // The heading names the card; the value is kept and said to be one too many.
    const boxes = [...canvasElement.querySelectorAll<HTMLTextAreaElement>('[data-value="Name"]')]
    expect(boxes.map((box) => box.value)).toEqual(['Llama', 'Alpaca'])

    const row = canvasElement.querySelector('[data-twice]')
    expect(row?.textContent).toContain('The card is named by this field')
    expect(row?.getAttribute('data-stray')).toBeNull()
    expect(canvasElement.textContent).not.toContain('Not a field of this stencil')
  },
}

/** A deck from somebody else. A value is a box, so no tag of one is ever drawn. */
export const TagsThatMustNotSurvive: Story = {
  args: { corpus: 'tags that must not survive' },
  play: async ({ canvasElement }) => {
    expect(canvasElement.querySelector('script')).toBeNull()
    expect(canvasElement.querySelector('img')).toBeNull()
    expect(canvasElement.querySelector('[onerror]')).toBeNull()
    expect((window as unknown as Record<string, unknown>)['stolen']).toBeUndefined()

    // What the person wrote is theirs to read and change, standing as the text
    // of a box and never as marks.
    const box = canvasElement.querySelector<HTMLTextAreaElement>('[data-value="Answer"]')
    expect(box?.value).toContain('<script>')
  },
}

const found = (canvas: HTMLElement, selector: string): HTMLElement => {
  const held = canvas.querySelector<HTMLElement>(selector)
  if (!held) throw new Error(`nothing matching ${selector}`)
  return held
}

/** Every tile is the size of the fullest of them, and nothing inside one scrolls. */
const measure = (canvasElement: HTMLElement, least = 3): void => {
  const tiles = [...canvasElement.querySelectorAll<HTMLElement>('[data-card]')]
  expect(tiles.length).toBeGreaterThanOrEqual(least)

  const sizes = tiles.map((tile) => {
    const box = tile.getBoundingClientRect()
    return `${Math.round(box.width)}x${Math.round(box.height)}`
  })
  expect(new Set(sizes).size).toBe(1)

  // Nothing inside a tile is read by scrolling: not the tile, not a value.
  for (const tile of tiles) {
    expect(tile.scrollHeight).toBeLessThanOrEqual(tile.clientHeight + 1)
    for (const box of tile.querySelectorAll<HTMLElement>('textarea')) {
      expect(box.scrollHeight).toBeLessThanOrEqual(box.clientHeight + 1)
    }
  }
}

/** Two stencils of different sizes, where the spread of tile heights is widest. */
export const StencilsOfDifferentSizes: Story = {
  args: { corpus: 'stencils of different sizes' },
  play: async ({ canvasElement }) => {
    const grid = found(canvasElement, '.deck__grid')

    // The window a story is drawn in is wide, so the tiles stand in more than
    // two columns.
    const columns = getComputedStyle(grid).gridTemplateColumns.split(' ').length
    expect(columns).toBeGreaterThan(2)

    measure(canvasElement)
  },
}

/**
 * A value with no newline in it that wraps to several lines. Its box stands at
 * what the wrapping comes to, which is worked out by the ground behind it and
 * not by counting the lines the value is written on.
 */
export const AValueThatWraps: Story = {
  args: { corpus: 'a value that wraps', width: '24rem' },
  play: async ({ canvasElement }) => {
    const box = found(canvasElement, '[data-card="faith"] [data-value="Answer"]')

    // It wrapped: the box stands taller than the one line it is written on.
    const line = Number.parseFloat(getComputedStyle(box).lineHeight)
    expect(box.clientHeight).toBeGreaterThan(line * 2)

    expect(box.scrollHeight).toBeLessThanOrEqual(box.clientHeight + 1)
    measure(canvasElement, 2)
  },
}

/** Three hundred cards, all of one size, with the deck itself the only scroll. */
export const FarTooManyIsStillOneSize: Story = {
  args: { corpus: 'far too many' },
  play: async ({ canvasElement }) => {
    measure(canvasElement)
  },
}

/** A window too narrow for two columns: one column, and no sideways scrolling. */
export const Narrow: Story = {
  args: { width: '22rem' },
  play: async ({ canvasElement }) => {
    const deck = found(canvasElement, '.deck')
    const grid = found(canvasElement, '.deck__grid')

    expect(getComputedStyle(grid).gridTemplateColumns.split(' ')).toHaveLength(1)
    expect(deck.scrollWidth).toBeLessThanOrEqual(deck.clientWidth + 1)
  },
}

/** The plus asks which stencil, and the card it makes stands on empty fields. */
export const AsksWhichStencil: Story = {
  play: async ({ canvasElement }) => {
    expect(canvasElement.querySelectorAll('[data-cut]')).toHaveLength(0)

    await userEvent.click(found(canvasElement, '[data-plus] button'))
    expect(canvasElement.querySelectorAll('[data-cut]')).toHaveLength(2)

    await userEvent.click(found(canvasElement, '[data-cut="Word"]'))

    const tiles = [...canvasElement.querySelectorAll('[data-card]')]
    const made = tiles[tiles.length - 1]

    // The field naming the card stands first, under a name nothing had taken,
    // and the rest stand empty after it.
    const boxes = [...(made?.querySelectorAll<HTMLTextAreaElement>('textarea') ?? [])]
    expect(boxes.map((box) => box.getAttribute('data-value'))).toEqual([
      'Word',
      'Meaning',
      'Example',
    ])
    expect(boxes[0]?.value).toBe('Card 1')
    expect(boxes.slice(1).every((box) => box.value === '')).toBe(true)
  },
}

/**
 * The name of a box sits on the box's own outline, in the gap the line leaves
 * for it. It is the legend, so the browser puts it there and there is nothing
 * to line up by hand.
 */
export const TheNameSitsOnTheLine: Story = {
  play: async ({ canvasElement }) => {
    const value = found(canvasElement, '[data-card="llama"] .deck__value')
    const outline = value.querySelector('fieldset')
    const label = value.querySelector('label')
    if (!outline || !label) throw new Error('no outline and no name on it')

    // One name, and it is the legend itself rather than a second element beside it.
    expect(value.querySelectorAll('label')).toHaveLength(1)
    expect(label.closest('legend')).not.toBeNull()

    // A legend straddles: the browser paints the line through its middle, half
    // a name's height below the outline's own box top. Anything laid beside the
    // legend instead of in it sits that half-height too high.
    const box = outline.getBoundingClientRect()
    const name = label.getBoundingClientRect()
    const middle = (name.top + name.bottom) / 2
    expect(Math.abs(middle - box.top - name.height / 2)).toBeLessThan(2)

    // The gap is real: the line does not run behind the name.
    expect(name.width).toBeGreaterThan(0)
    expect(name.left).toBeGreaterThan(box.left)
  },
}

/**
 * Every box keeps the name's line clear of its text, on every count of lines,
 * and a box holding nothing stands as tall as a box holding one.
 */
export const TheTextClearsTheLine: Story = {
  args: { corpus: 'a deck' },
  play: async ({ canvasElement }) => {
    const tile = found(canvasElement, '[data-card="llama"]')
    const boxes = [...tile.querySelectorAll<HTMLTextAreaElement>('textarea')]
    expect(boxes.length).toBeGreaterThan(2)

    for (const box of boxes) {
      const outline = box.closest('.notched')?.querySelector('fieldset')
      const label = box.closest('.notched')?.querySelector('label')
      if (!outline || !label) throw new Error('a box with no outline')

      // Where the line is painted, and where the typing starts.
      const line = outline.getBoundingClientRect().top + label.getBoundingClientRect().height / 2
      const text = box.getBoundingClientRect().top + Number.parseFloat(getComputedStyle(box).paddingTop)
      expect(text - line).toBeGreaterThan(4)
    }

    // Empty and one-line boxes stand to one height.
    const heights = boxes.map((box) => Math.round(box.getBoundingClientRect().height))
    expect(new Set(heights).size).toBe(1)
  },
}

/** The plus stands alone in the middle, and what it opens stands there too. */
export const OnePlusInTheMiddle: Story = {
  play: async ({ canvasElement }) => {
    const plus = found(canvasElement, '[data-plus]')
    const button = found(canvasElement, '[data-plus] button')

    // It is named for a reader, and says nothing beside itself.
    expect(button.getAttribute('aria-label')).toBe('Add a card')
    expect(button.textContent?.trim()).toBe('')

    const glyph = button.querySelector('svg')?.getBoundingClientRect()
    expect(glyph?.width).toBeGreaterThan(24)

    const middle = (box: DOMRect): readonly number[] => [
      Math.round(box.left + box.width / 2),
      Math.round(box.top + box.height / 2),
    ]
    expect(middle(button.getBoundingClientRect())).toEqual(middle(plus.getBoundingClientRect()))

    // What it opens stands where it was pressed.
    await userEvent.click(button)
    const asking = found(canvasElement, '[data-plus] .deck__asking')
    expect(middle(asking.getBoundingClientRect())).toEqual(middle(plus.getBoundingClientRect()))
  },
}

/** The box is filled, so it reads as something to type in. */
export const TheBoxIsFilled: Story = {
  play: async ({ canvasElement }) => {
    const tile = found(canvasElement, '[data-card="llama"]')
    const filled = found(canvasElement, '[data-card="llama"] .grown')

    // The box has a ground of its own. Whether it reads darker or lighter than
    // the tile is the theme's business; that it has one at all is not.
    const ground = getComputedStyle(filled).backgroundColor
    expect(ground).not.toBe('rgba(0, 0, 0, 0)')
    expect(ground).not.toBe('transparent')
    expect(getComputedStyle(tile).backgroundColor).not.toBe('rgba(0, 0, 0, 0)')
  },
}

/** Every value is open to typing, and the head holds nothing that opens a tile. */
export const EveryValueIsOpenToTyping: Story = {
  play: async ({ canvasElement }) => {
    const tile = found(canvasElement, '[data-card="llama"]')
    expect(tile.querySelectorAll('textarea')).toHaveLength(4)
    expect(tile.querySelector('[aria-expanded]')).toBeNull()
    expect(found(canvasElement, '[data-card="llama"] .bar').querySelectorAll('button'))
      .toHaveLength(1)

    const box = found(canvasElement, '[data-card="llama"] [data-value="Height"]')
    await userEvent.click(box)
    await userEvent.type(box, '!')
    expect((box as HTMLTextAreaElement).value).toContain('!')
  },
}
