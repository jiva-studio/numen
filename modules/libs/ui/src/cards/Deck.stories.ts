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
import { ordered, type Cut, type Drawn, type Filled, type Landing, type Wrong } from './model'

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
  /* Names and values that are not Latin, beside a name and a value with
     nothing in them to break at. */
  'awkward text': {
    cuts: [{ name: 'Слово', fields: ['Слово', 'Перевод', 'Пример'] }, { name: UNBROKEN, fields: [UNBROKEN, 'Long'] }],
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
        onWrite: (id: string, field: string, nth: number, text: string) => {
          cards.value = changed(id, (card) => {
            const first = cuts.value.find((cut) => cut.name === card.stencil)?.fields[0]
            if (field === first) return { ...card, name: text }

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

/**
 * Three cards, two stencils, and one card leaving a field out.
 *
 * A value is the room under the rule that names it: the name stands on the
 * line, the box begins where the line ends, and nothing is drawn around it.
 * Every one of them is open to typing, and the strip over them opens nothing.
 */
export const ADeck: Story = {
  play: async ({ canvasElement }) => {
    const tile = found(canvasElement, '[data-card="llama"]')
    const value = found(canvasElement, '[data-card="llama"] .deck__value')
    const rule = value.querySelector('.rule')
    const label = value.querySelector('label')
    if (!rule || !label) throw new Error('no rule and no name on it')

    // One name, and it stands on the rule, so the two share a middle.
    expect(value.querySelectorAll('label')).toHaveLength(1)
    expect(label.closest('.rule')).toBe(rule)
    const line = rule.getBoundingClientRect()
    const name = label.getBoundingClientRect()
    expect(Math.abs((name.top + name.bottom) / 2 - (line.top + line.bottom) / 2)).toBeLessThan(2)

    // The rule runs to both edges of the tile, past the room the values keep.
    const edges = tile.getBoundingClientRect()
    expect(line.left - edges.left).toBeLessThan(2)
    expect(edges.right - line.right).toBeLessThan(2)

    const boxes = [...tile.querySelectorAll<HTMLTextAreaElement>('textarea')]
    expect(boxes).toHaveLength(4)
    for (const box of boxes) {
      const under = box.closest('.deck__value')?.querySelector('.rule')
      if (!under) throw new Error('a box under no rule')
      expect(box.getBoundingClientRect().top).toBeCloseTo(under.getBoundingClientRect().bottom, 0)
      expect(box.closest('.deck__value')?.querySelector('fieldset')).toBeNull()
    }

    // Empty and one-line boxes stand to one height.
    const heights = boxes.map((box) => Math.round(box.getBoundingClientRect().height))
    expect(new Set(heights).size).toBe(1)

    // The tile keeps the ground both the rules and the boxes stand on.
    const grown = found(canvasElement, '[data-card="llama"] .grown')
    const ground = getComputedStyle(grown).backgroundColor
    expect(ground === 'rgba(0, 0, 0, 0)' || ground === 'transparent').toBe(true)
    expect(getComputedStyle(grown).borderTopWidth).toBe('0px')
    expect(getComputedStyle(tile).backgroundColor).not.toBe('rgba(0, 0, 0, 0)')

    // The strip holds the one way to remove the card, and opens nothing.
    expect(tile.querySelector('[aria-expanded]')).toBeNull()
    expect(found(canvasElement, '[data-card="llama"] .bar').querySelectorAll('button'))
      .toHaveLength(1)

    const box = found(canvasElement, '[data-card="llama"] [data-value="Height"]')
    await userEvent.click(box)
    await userEvent.type(box, '!')
    expect((box as HTMLTextAreaElement).value).toContain('!')
  },
}

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
    expect(plus.getBoundingClientRect().height).toBeGreaterThan(64)

    // What it opens stands where it was pressed.
    await userEvent.click(button)
    const asking = found(canvasElement, '[data-plus] .deck__asking')
    expect(middle(asking.getBoundingClientRect())).toEqual(middle(plus.getBoundingClientRect()))
  },
}

/**
 * Every fault a tile can carry, one card apiece: a card with nothing in it, a
 * card naming a stencil that was not handed in, and a card carrying the field
 * that names it as a value as well.
 */
export const WhatIsWrongWithACard: Story = {
  args: { corpus: 'what is wrong' },
  play: async ({ canvasElement }) => {
    // The heading names the card; the value is kept and said to be one too many.
    const boxes = [...canvasElement.querySelectorAll<HTMLTextAreaElement>('[data-value="Name"]')]
    expect(boxes.map((box) => box.value)).toEqual(['Llama', 'Alpaca'])

    const row = canvasElement.querySelector('[data-twice]')
    expect(row?.textContent).toContain('The card is named by this field')
    expect(row?.getAttribute('data-stray')).toBeNull()
    expect(canvasElement.textContent).not.toContain('Not a field of this stencil')

    // The card whose stencil was not handed in says so, and keeps its values.
    const orphan = found(canvasElement, '[data-card="orphan"]')
    // Nothing says what its boxes are, so it draws none and says which stencil
    // it asked for.
    expect(orphan.textContent).toContain('Gone')
    expect(orphan.querySelectorAll('textarea')).toHaveLength(0)

    // The card with nothing in it stands as a tile like any other.
    expect(canvasElement.querySelector('[data-card="blank"]')).not.toBeNull()

    // What the vault found wrong with a card is said under its heading, above
    // the first of its values; what it found wrong with one value is said
    // under that value.
    const said = found(canvasElement, '[data-card="blank"] [data-wrong]')
    expect(said.textContent).toContain('this card has no name')
    const first = found(canvasElement, '[data-card="blank"] .deck__value')
    expect(said.getBoundingClientRect().bottom).toBeLessThanOrEqual(
      first.getBoundingClientRect().top + 1,
    )

    // It is said once, under the second of the two boxes standing for Name.
    const under = [
      ...canvasElement.querySelectorAll('[data-card="twice"] [data-wrong-value="Name"]'),
    ]
    expect(under).toHaveLength(1)
    const value = found(canvasElement, '[data-card="twice"] [data-wrong-value="Name"]')
    expect(value.textContent).toContain('this card writes Name twice')
    const box = found(canvasElement, '[data-card="twice"] [data-value="Name"]')
    expect(value.getBoundingClientRect().top).toBeGreaterThanOrEqual(
      box.getBoundingClientRect().bottom - 1,
    )

    // A value is a box, so no tag written into one is ever drawn as a mark.
    expect(canvasElement.querySelector('script')).toBeNull()
    expect(canvasElement.querySelector('img')).toBeNull()
    expect(canvasElement.querySelector('[onerror]')).toBeNull()
    expect((window as unknown as Record<string, unknown>)['stolen']).toBeUndefined()

    // What the person wrote is theirs to read and change, standing as the text
    // of a box.
    const theirs = canvasElement.querySelector<HTMLTextAreaElement>('[data-value="Answer"]')
    expect(theirs?.value).toContain('<script>')
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

    const box = found(canvasElement, '[data-card="faith"] [data-value="Answer"]')

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
