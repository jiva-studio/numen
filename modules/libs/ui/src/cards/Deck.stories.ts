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
import { HEAD, type Banded, type Drawn, type Filled, type Wrong } from './deck'
import type { Landing } from './order'
import type { Cut } from './stencil'

interface Corpus {
  readonly cards: readonly Drawn[]
  readonly cuts: readonly Cut[]
  /** The sections the cards stand under, in the order they stand in the deck. */
  readonly sections?: readonly Banded[]
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
          id: `beast00${at}`,
          section: null,
          stencil: 'Animal',
          filled: [
            { field: 'Name', text: `Beast ${at + 1}` },
            { field: 'Height', text: 'about 45"' },
            { field: 'Weight', text: '- bull: 350 kg\n- cow: 300 kg\n- calf: 40 kg' },
            { field: 'Life span', text: 'about 20 years' },
          ],
        }
      : {
          id: `asked00${at}`,
          section: null,
          stencil: 'Basic',
          filled: [{ field: 'Question', text: `Question ${at + 1}` }],
        },
  )

const CORPORA = {
  'a deck': {
    cuts: [ANIMAL, WORD],
    cards: [
      {
        id: 'k7m2xq9fzp',
        section: null,
        stencil: 'Animal',
        filled: [
          { field: 'Name', text: 'Llama' },
          { field: 'Height', text: 'about 45" *at the shoulder*' },
          { field: 'Life span', text: 'about **20 years**' },
        ],
      },
      {
        id: '3n8vr4tqch',
        section: null,
        stencil: 'Animal',
        filled: [
          { field: 'Name', text: 'Yak' },
          { field: 'Height', text: 'about 63"' },
          { field: 'Weight', text: '- bull: 350 kg\n- cow: 300 kg' },
          { field: 'Life span', text: 'about 22 years' },
        ],
      },
      {
        id: 'w9s5jd2b1k',
        section: null,
        stencil: 'Word',
        filled: [
          { field: 'Word', text: 'llano' },
          { field: 'Meaning', text: 'a plain' },
          { field: 'Example', text: '' },
        ],
      },
    ],
  },
  /* A deck grown past one list: cards standing before the first section, a
     section holding several, and a section a person made and has yet to put a
     card in. */
  'a deck in sections': {
    cuts: [ANIMAL, ASKED],
    sections: [
      { id: 'roots', name: 'Roots' },
      { id: 'leaves', name: 'Leaves' },
    ],
    cards: [
      {
        id: 'p4h6c8vzn2',
        section: null,
        stencil: 'Basic',
        filled: [
          { field: 'Question', text: 'What is a deck for?' },
          { field: 'Answer', text: 'Cards, and the questions on them.' },
        ],
      },
      {
        id: 'r2t7y5k9wq',
        section: 'roots',
        stencil: 'Basic',
        filled: [
          {
            field: 'Question',
            text: 'Compost, what is it made of\n\nAsked of a heap two winters old.',
          },
          { field: 'Answer', text: 'Leaves and peelings, turned and left to rot down' },
        ],
      },
      {
        id: 'z3f1m6b4dt',
        section: 'roots',
        stencil: 'Animal',
        filled: [
          { field: 'Name', text: 'Earthworm' },
          { field: 'Life span', text: 'about 2 years' },
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
        id: 'c5n8q2wxjb',
        section: null,
        stencil: 'Basic',
        filled: [
          { field: 'Question', text: 'What is compost?' },
          {
            field: 'Answer',
            text: 'Leaves, grass and kitchen peelings, turned twice and left under a sheet until the heap has gone dark and crumbly enough to spread on any bed of the plot.',
          },
        ],
      },
      {
        id: 'v7k3d9m1zr',
        section: null,
        stencil: 'Basic',
        filled: [{ field: 'Question', text: 'a short one' }],
      },
    ],
  },
  /* Values that are not Latin, beside a field name and a value with nothing in
     them to break at. */
  'awkward text': {
    cuts: [{ name: 'Слово', fields: ['Слово', 'Перевод', 'Пример'] }, { name: UNBROKEN, fields: [UNBROKEN, 'Long'] }],
    cards: [
      {
        id: 'j2b6t8n4vw',
        section: null,
        stencil: 'Слово',
        filled: [
          { field: 'Слово', text: 'Компост' },
          { field: 'Перевод', text: 'compost, перепревшие листья и трава' },
          { field: 'Пример', text: 'बगीचे की खाद और हरी खाद' },
        ],
      },
      {
        id: 'h5r1w7z3qm',
        section: null,
        stencil: UNBROKEN,
        filled: [
          { field: UNBROKEN, text: UNBROKEN },
          {
            field: 'Long',
            text: 'A value that goes on and on, well past the width any tile drawing it is likely to have, and then a little further.',
          },
        ],
      },
    ],
  },
  'nothing at all': { cuts: [ANIMAL, WORD], cards: [] },
  /* Every fault a tile can carry: a card under no stencil, a card naming a
     stencil that was not handed in, a card writing one field twice, a card from
     somebody else carrying marks that must never be drawn as marks, and two
     cards the file names alike, which are two cards and stand as two. */
  'what is wrong': {
    cuts: [ANIMAL, WORD, ASKED],
    cards: [
      { id: 'b8k4n2vqz6', section: null, stencil: null, filled: [] },
      {
        id: 'm3t9w5rj1x',
        section: null,
        stencil: 'Gone',
        filled: [{ field: 'Whatever it had', text: 'still here, still readable' }],
      },
      {
        id: 'd6q2z8hn4v',
        section: null,
        stencil: 'Animal',
        filled: [
          { field: 'Name', text: 'Llama' },
          { field: 'Name', text: 'Alpaca' },
          { field: 'Height', text: 'about 45"' },
        ],
      },
      {
        id: 'y1v5b9kt3n',
        section: null,
        stencil: 'Basic',
        filled: [
          { field: 'Question', text: 'A deck from somebody else' },
          {
            field: 'Answer',
            text:
              '<script>window.stolen = 1</script>' +
              '<img src="x" onerror="window.stolen = 2">' +
              'Only these words should stand.',
          },
        ],
      },
      /* A card copied by hand and the card it was copied from: the file names
         the two alike, and the window draws each under an identity of its own
         so both are read and both are typed into. */
      {
        id: 'copied-one',
        section: null,
        stencil: 'Basic',
        filled: [
          { field: 'Question', text: 'Compost, what is it made of' },
          { field: 'Answer', text: 'Leaves and peelings' },
        ],
      },
      {
        id: 'copied-again',
        section: null,
        stencil: 'Basic',
        filled: [
          { field: 'Question', text: 'Compost, what is it made of' },
          { field: 'Answer', text: 'Leaves and peelings, turned' },
        ],
      },
    ],
    wrong: {
      at: {
        b8k4n2vqz6: ['the first paragraph under this card is not a lone wikilink'],
        d6q2z8hn4v: ['this card writes one field twice'],
        'copied-one': ['two cards of this deck carry the mark k7m2xq9fzp'],
        'copied-again': ['two cards of this deck carry the mark k7m2xq9fzp'],
      },
      under: {
        d6q2z8hn4v: { Name: ['this card writes Name twice'] },
        y1v5b9kt3n: { Answer: ['this value is not in the stencil this card is cut by'] },
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
  sections?: never
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
    sections: { table: { disable: true } },
    words: { table: { disable: true } },
  },
  args: { corpus: 'a deck', name: 'Deck', width: '100%' },
  render: (args) => ({
    components: { Deck },
    setup() {
      const cards = ref<readonly Drawn[]>(CORPORA[args.corpus].cards)
      const cuts = ref<readonly Cut[]>(CORPORA[args.corpus].cuts)
      const sections = ref<readonly Banded[]>(sectionsOf(CORPORA[args.corpus]))
      const wrong = ref<Wrong>(wrongOf(CORPORA[args.corpus]))

      watch(
        () => args.corpus,
        (next) => {
          cards.value = CORPORA[next].cards
          cuts.value = CORPORA[next].cuts
          sections.value = sectionsOf(CORPORA[next])
          wrong.value = wrongOf(CORPORA[next])
        },
      )

      /** The cards, with one of them changed. */
      const changed = (id: string, into: (card: Drawn) => Drawn): readonly Drawn[] =>
        cards.value.map((card) => (card.id === id ? into(card) : card))

      /** The section the last of them is, which is where a new card is made. */
      const lastSection = (): string | null => sections.value.at(-1)?.id ?? null

      return {
        args,
        cards,
        cuts,
        sections,
        wrong,
        onAdd: (stencil: string, filled: readonly Filled[]) => {
          const id = `made00000${cards.value.length}`
          cards.value = [...cards.value, { id, section: lastSection(), stencil, filled }]
        },
        onRemove: (id: string) => {
          cards.value = cards.value.filter((card) => card.id !== id)
        },
        onMove: (id: string, at: Landing) => {
          cards.value = moved(cards.value, sections.value, id, at)
        },
        onWrite: (id: string, field: string, nth: number, text: string) => {
          cards.value = changed(id, (card) => {
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
        onAddSection: (name: string) => {
          sections.value = [...sections.value, { id: `made-${sections.value.length}`, name }]
        },
        onRenameSection: (id: string, name: string) => {
          sections.value = sections.value.map((each) => (each.id === id ? { ...each, name } : each))
        },
        /* A section is a name, so taking one away takes away its heading and
           nothing else: its cards stand under whatever heading is above them. */
        onRemoveSection: (id: string) => {
          const above = sections.value[sections.value.findIndex((each) => each.id === id) - 1]
          sections.value = sections.value.filter((each) => each.id !== id)
          cards.value = cards.value.map((card) =>
            card.section === id ? { ...card, section: above?.id ?? null } : card,
          )
        },
      }
    },
    template: `
      <div :style="{ height: '100vh', width: args.width }">
        <Deck
          :cards="cards"
          :cuts="cuts"
          :sections="sections"
          :name="args.name"
          :wrong="wrong"
          @add="onAdd"
          @remove="onRemove"
          @move="onMove"
          @write="onWrite"
          @add-section="onAddSection"
          @rename-section="onRenameSection"
          @remove-section="onRemoveSection"
        />
      </div>
    `,
  }),
}

const sectionsOf = (corpus: Corpus): readonly Banded[] => corpus.sections ?? []

/**
 * A card let go before another, at the head of the deck, at the head of a
 * section, or at the end of it. A card takes the section of whatever it lands
 * in front of.
 */
const moved = (
  cards: readonly Drawn[],
  sections: readonly Banded[],
  id: string,
  at: Landing,
): readonly Drawn[] => {
  const held = cards.find((card) => card.id === id)
  if (!held) return cards
  const left = cards.filter((card) => card.id !== id)

  if (at === HEAD) return [{ ...held, section: null }, ...left]

  const before = at === null ? -1 : left.findIndex((card) => card.id === at)
  if (before !== -1) {
    const under = { ...held, section: left[before]?.section ?? null }
    return [...left.slice(0, before), under, ...left.slice(before)]
  }

  const rank = (section: string | null): number =>
    section === null ? 0 : sections.findIndex((each) => each.id === section) + 1

  if (at === null) {
    return [...left, { ...held, section: sections.at(-1)?.id ?? null }]
  }
  if (!sections.some((each) => each.id === at)) return cards

  const seat = left.findIndex((card) => rank(card.section) >= rank(at))
  const where = seat === -1 ? left.length : seat
  return [...left.slice(0, where), { ...held, section: at }, ...left.slice(where)]
}

export default meta
type Story = StoryObj<Knobs>

/**
 * Three cards, two stencils, and one card leaving a field out. The plus asks
 * which stencil, and the card it makes stands on empty fields.
 */
export const ADeck: Story = {
  play: async ({ canvasElement }) => {
    expect(canvasElement.querySelectorAll('[data-cut]')).toHaveLength(0)

    await userEvent.click(found(canvasElement, '[data-plus] button'))
    expect(canvasElement.querySelectorAll('[data-cut]')).toHaveLength(2)

    await userEvent.click(found(canvasElement, '[data-cut="Word"]'))

    const tiles = [...canvasElement.querySelectorAll('[data-card]')]
    const made = tiles[tiles.length - 1]

    // Every field the stencil declares stands under its own name, the first
    // included, and all of them stand empty.
    const boxes = [...(made?.querySelectorAll<HTMLTextAreaElement>('textarea') ?? [])]
    expect(boxes.map((box) => box.getAttribute('data-value'))).toEqual([
      'Word',
      'Meaning',
      'Example',
    ])
    expect(boxes.every((box) => box.value === '')).toBe(true)
  },
}

/**
 * A deck grown past one list: cards standing before the first section, a
 * section holding several, and a section holding none. A card is let go at the
 * head of a section, an empty one included, and a section taken away leaves its
 * cards where they stand.
 */
export const InSections: Story = {
  args: { corpus: 'a deck in sections' },
  play: async ({ canvasElement }) => {
    const bands = [...canvasElement.querySelectorAll('[data-band]')]
    expect(bands.map((band) => band.getAttribute('data-band'))).toEqual(['roots', 'leaves'])

    // A section holding no card keeps its heading, and stands as wide as the
    // grid under it.
    const leaves = found(canvasElement, '[data-band="leaves"]')
    expect(canvasElement.querySelectorAll('[data-section="leaves"]')).toHaveLength(0)
    expect(leaves.getBoundingClientRect().width).toBeGreaterThan(240)

    // The heading is a rule with the name typed on it, and the way to be rid of
    // it at the end.
    expect(found(canvasElement, '[data-band="roots"] .rule')).toBeTruthy()
    expect(found(canvasElement, '[data-band="roots"] input').getAttribute('value')).toBe('Roots')

    // Each card stands under the section it is in, and the ones before the
    // first stand under none.
    expect(canvasElement.querySelectorAll('[data-section="roots"]')).toHaveLength(2)
    expect(
      found(canvasElement, '[data-card="p4h6c8vzn2"]').getAttribute('data-section'),
    ).toBeNull()

    // A section made at the end, under a name nothing has taken.
    await userEvent.click(found(canvasElement, '[data-add-section]'))
    const made = [...canvasElement.querySelectorAll('[data-band]')]
    expect(made).toHaveLength(3)
    expect(made[2]?.querySelector('input')?.value).toBe('Section 1')

    // Taking a section away takes away its heading and nothing else: its cards
    // stand under the heading above them now.
    await userEvent.click(found(canvasElement, '[data-band="roots"] .deed'))
    expect(canvasElement.querySelectorAll('[data-band="roots"]')).toHaveLength(0)
    expect(canvasElement.querySelectorAll('[data-card]')).toHaveLength(3)
    expect(
      found(canvasElement, '[data-card="z3f1m6b4dt"]').getAttribute('data-section'),
    ).toBeNull()
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

/** Values that are not Latin, and a field name with nothing to break at. */
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
 * A deck the vault found five things wrong with. Every mark stands on the card
 * it was read against and on no other, and what a card is drawn as is the
 * card's own.
 */
export const WhatIsWrongWithACard: Story = {
  args: { corpus: 'what is wrong' },
  play: async ({ canvasElement }) => {
    const said = (card: string, selector: string): string =>
      found(canvasElement, `[data-card="${card}"] ${selector}`).textContent ?? ''

    expect(said('b8k4n2vqz6', '[data-wrong]')).toContain('not a lone wikilink')
    expect(said('d6q2z8hn4v', '[data-wrong]')).toContain('writes one field twice')
    expect(said('d6q2z8hn4v', '[data-wrong-value="Name"]')).toContain('writes Name twice')
    expect(said('y1v5b9kt3n', '[data-wrong-value="Answer"]')).toContain('not in the stencil')

    // The card nothing was said against carries no mark at all.
    expect(canvasElement.querySelector('[data-card="m3t9w5rj1x"] [data-wrong]')).toBeNull()
    expect(canvasElement.querySelectorAll('[data-wrong]')).toHaveLength(4)

    // Two cards of one mark stand as two tiles, each carrying what is wrong
    // with it and each typed into on its own.
    expect(said('copied-one', '[data-wrong]')).toContain('carry the mark')
    expect(said('copied-again', '[data-wrong]')).toContain('carry the mark')

    const box = (card: string): HTMLTextAreaElement =>
      found(canvasElement, `[data-card="${card}"] [data-value="Answer"]`) as HTMLTextAreaElement

    await userEvent.type(box('copied-one'), ' and left to rot down')

    expect(box('copied-one').value).toBe('Leaves and peelings and left to rot down')
    expect(box('copied-again').value).toBe('Leaves and peelings, turned')
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

    const box = found(canvasElement, '[data-card="c5n8q2wxjb"] [data-value="Answer"]')

    // It wrapped: the box stands taller than the one line it is written on.
    const line = Number.parseFloat(getComputedStyle(box).lineHeight)
    expect(box.clientHeight).toBeGreaterThan(line * 2)

    expect(box.scrollHeight).toBeLessThanOrEqual(box.clientHeight + 1)
    measure(canvasElement, 2)
  },
}
