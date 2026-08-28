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
import type { Drawn, Filled, Wrong } from './deck'
import { ordered, type Landing } from './order'
import type { Cut } from './stencil'

interface Corpus {
  readonly cards: readonly Drawn[]
  readonly cuts: readonly Cut[]
  /** What the vault reading this file found wrong with it, by card and field. */
  readonly wrong?: {
    readonly at?: Readonly<Record<string, readonly string[]>>
    readonly under?: Readonly<Record<string, Readonly<Record<string, readonly string[]>>>>
  }
}

/** What a corpus says is wrong, as the grid takes it. */
const wrongOf = (corpus: Corpus): Wrong => ({
  at: new Map(Object.entries(corpus.wrong?.at ?? {})),
  under: new Map(
    Object.entries(corpus.wrong?.under ?? {}).map(([card, fields]) => [
      card,
      new Map(Object.entries(fields)),
    ]),
  ),
})

const ANIMAL: Cut = { name: 'Animal', fields: ['Name', 'Height', 'Weight', 'Life span'] }
const WORD: Cut = { name: 'Word', fields: ['Word', 'Meaning', 'Example'] }
const ASKED: Cut = { name: 'Basic', fields: ['Question', 'Answer'] }

const UNBROKEN =
  'supercalifragilisticexpialidociousandthensomemoreofitwithnothingtobreakatanywhere'

/**
 * Cards cut by two stencils of different sizes, which is where the tiles differ
 * most: one stencil holds four fields and one holds two, and cards leave fields
 * out.
 */
const many = (count: number): readonly Drawn[] =>
  Array.from({ length: count }, (_, at) =>
    at % 2 === 0
      ? {
          id: `beast-${at}`,
          name: `Beast ${at + 1}`,
          stencil: 'Animal',
          filled: [
            { field: 'Height', text: 'about 45"' },
            { field: 'Weight', text: '- bull: 350 kg\n- cow: 300 kg\n- calf: 40 kg' },
            { field: 'Life span', text: 'about 20 years' },
          ],
        }
      : { id: `asked-${at}`, name: `Question ${at + 1}`, stencil: 'Basic', filled: [] },
  )

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
  'far too many': { cuts: [ANIMAL, ASKED], cards: many(300) },
  /* A value with no newline in it that is far wider than the box, so what the
     box stands at can only be worked out from where the text wraps. */
  'a value that wraps': {
    cuts: [ASKED],
    cards: [
      {
        id: 'compost',
        name: 'What is compost?',
        stencil: 'Basic',
        filled: [
          {
            field: 'Answer',
            text: 'Leaves, grass and kitchen peelings, turned twice and left under a sheet until the heap has gone dark and crumbly enough to spread on any bed of the plot.',
          },
        ],
      },
      { id: 'short', name: 'a short one', stencil: 'Basic', filled: [] },
    ],
  },
  /* Names and values that are not Latin, beside a name and a value with
     nothing in them to break at. */
  'awkward text': {
    cuts: [{ name: 'Слово', fields: ['Слово', 'Перевод', 'Пример'] }, { name: UNBROKEN, fields: [UNBROKEN, 'Long'] }],
    cards: [
      {
        id: 'компост',
        name: 'Компост',
        stencil: 'Слово',
        filled: [
          { field: 'Перевод', text: 'compost, перепревшие листья и трава' },
          { field: 'Пример', text: 'बगीचे की खाद और हरी खाद' },
        ],
      },
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
  /* Every fault a tile can carry, one card apiece: a card with nothing in it,
     a card naming a stencil that was not handed in, a card carrying the field
     that names it as a value as well, and a card from somebody else carrying
     marks that must never be drawn as marks. */
  'what is wrong': {
    cuts: [ANIMAL, WORD, ASKED],
    cards: [
      { id: 'blank', name: '', stencil: 'Word', filled: [] },
      {
        id: 'orphan',
        name: 'Orphan',
        stencil: 'Gone',
        filled: [{ field: 'Whatever it had', text: 'still here, still readable' }],
      },
      {
        id: 'twice',
        name: 'Llama',
        stencil: 'Animal',
        filled: [
          { field: 'Name', text: 'Alpaca' },
          { field: 'Height', text: 'about 45"' },
        ],
      },
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
    wrong: {
      at: {
        blank: ['this card has no name'],
        twice: ['another card is called Llama'],
      },
      under: {
        twice: { Name: ['this card writes Name twice'] },
        theirs: { Answer: ['this value is not in the stencil this card is cut by'] },
      },
    },
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
  title: 'Flash Cards/Deck',
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
      const wrong = ref<Wrong>(wrongOf(CORPORA[args.corpus]))

      watch(
        () => args.corpus,
        (next) => {
          cards.value = CORPORA[next].cards
          cuts.value = CORPORA[next].cuts
          wrong.value = wrongOf(CORPORA[next])
        },
      )

      /** The cards, with one of them changed. */
      const changed = (id: string, into: (card: Drawn) => Drawn): readonly Drawn[] =>
        cards.value.map((card) => (card.id === id ? into(card) : card))

      return {
        args,
        cards,
        cuts,
        wrong,
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
        onWrite: (id: string, field: string, nth: number, names: boolean, text: string) => {
          cards.value = changed(id, (card) => {
            if (names) return { ...card, name: text }

            // The one written is the one counted off under its own field.
            let under = 0
            const filled = card.filled.map((each) => {
              if (each.field !== field) return each
              under += 1
              return under === nth ? { field, text } : each
            })
            if (under < nth) filled.push({ field, text })
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
          :wrong="wrong"
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


/**
 * Three hundred cards cut by two stencils of different sizes, which is where
 * the spread of tile heights is widest. Every tile is the size of the fullest
 * of them, and the deck itself is the only thing that scrolls.
 */
export const FarTooMany: Story = {
  args: { corpus: 'far too many' },
  play: async ({ canvasElement }) => {
    const grid = found(canvasElement, '.deck__grid')

    // The window a story is drawn in is wide, so the tiles stand in more than
    // two columns.
    expect(getComputedStyle(grid).gridTemplateColumns.split(' ').length).toBeGreaterThan(2)

    measure(canvasElement)
  },
}

/** Names and values that are not Latin, and a name with nothing to break at. */
export const AwkwardText: Story = { args: { corpus: 'awkward text' } }

/** No cards, and one plus standing alone in the middle of what it is asked from. */
export const NothingAtAll: Story = {
  args: { corpus: 'nothing at all' },
  play: async ({ canvasElement }) => {
    const plus = found(canvasElement, '[data-plus]')
    const button = found(canvasElement, '[data-plus] button')

    expect(canvasElement.querySelectorAll('[data-card]')).toHaveLength(0)

    // It is named for a reader, and says nothing beside itself.
    expect(button.getAttribute('aria-label')).toBe('Add a card')
    expect(button.textContent?.trim()).toBe('')
    expect(button.querySelector('svg')?.getBoundingClientRect().width).toBeGreaterThan(24)

    const middle = (box: DOMRect): readonly number[] => [
      Math.round(box.left + box.width / 2),
      Math.round(box.top + box.height / 2),
    ]
    expect(middle(button.getBoundingClientRect())).toEqual(middle(plus.getBoundingClientRect()))

    // It is a tile like the cards are: a dashed outline, drawn no smaller than
    // a card, and as wide as the column it stands in.
    const drawn = getComputedStyle(plus)
    expect(drawn.borderTopStyle).toBe('dashed')
    expect(Number.parseFloat(drawn.borderTopWidth)).toBeGreaterThan(0)

    const at = plus.getBoundingClientRect()
    expect(at.height).toBeGreaterThan(64)
    expect(at.width).toBeGreaterThan(240)

    // What it opens stands where it was pressed, inside that same outline.
    await userEvent.click(button)
    const asking = found(canvasElement, '[data-plus] .deck__asking')
    expect(middle(asking.getBoundingClientRect())).toEqual(middle(plus.getBoundingClientRect()))
    expect(getComputedStyle(found(canvasElement, '[data-plus]')).borderTopStyle).toBe('dashed')
  },
}

/**
 * A deck the vault found four things wrong with, one card apiece. Every mark
 * stands on the card it was read against and on no other, and what a card is
 * drawn as is the card's own.
 */
export const WhatIsWrongWithACard: Story = {
  args: { corpus: 'what is wrong' },
  play: async ({ canvasElement }) => {
    const said = (card: string, selector: string): string =>
      found(canvasElement, `[data-card="${card}"] ${selector}`).textContent ?? ''

    expect(said('blank', '[data-wrong]')).toContain('this card has no name')
    expect(said('twice', '[data-wrong]')).toContain('another card is called Llama')
    expect(said('twice', '[data-wrong-value="Name"]')).toContain('this card writes Name twice')
    expect(said('theirs', '[data-wrong-value="Answer"]')).toContain('not in the stencil')

    // The card nothing was said against carries no mark at all.
    expect(canvasElement.querySelector('[data-card="orphan"] [data-wrong]')).toBeNull()
    expect(canvasElement.querySelectorAll('[data-wrong]')).toHaveLength(2)
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

/**
 * A window too narrow for two columns: one column, and no sideways scrolling.
 * A value with no newline in it wraps to several lines, and its box stands at
 * what the wrapping comes to, which is worked out by the ground behind it.
 */
export const Narrow: Story = {
  args: { corpus: 'a value that wraps', width: '22rem' },
  play: async ({ canvasElement }) => {
    const deck = found(canvasElement, '.deck')
    const grid = found(canvasElement, '.deck__grid')

    expect(getComputedStyle(grid).gridTemplateColumns.split(' ')).toHaveLength(1)
    expect(deck.scrollWidth).toBeLessThanOrEqual(deck.clientWidth + 1)

    const box = found(canvasElement, '[data-card="compost"] [data-value="Answer"]')

    // It wrapped: the box stands taller than the one line it is written on.
    const line = Number.parseFloat(getComputedStyle(box).lineHeight)
    expect(box.clientHeight).toBeGreaterThan(line * 2)

    expect(box.scrollHeight).toBeLessThanOrEqual(box.clientHeight + 1)
    measure(canvasElement, 2)
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
