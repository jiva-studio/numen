/**
 * Every situation the deck editor has to survive. Also the test corpus: each
 * story is run in a browser by `@storybook/addon-vitest`.
 *
 * What the editor returns is applied here, which is the application's part.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent } from 'storybook/test'
import { ref, watch } from 'vue'
import DeckEditor from './DeckEditor.vue'
import {
  getBlanks,
  HEAD,
  type DeckSection,
  type DeckCard,
  type DeckProblems,
} from '../../../../lib/cards'
import { getDeclaredFields, type InsertionPoint } from '../../../../lib/cards'
import type { CardModel } from '../../../../lib/cards'

interface Corpus {
  readonly cards: readonly DeckCard[]
  readonly cuts: readonly CardModel[]
  readonly sections?: readonly DeckSection[]
  readonly wrong?: {
    readonly at?: Readonly<Record<string, readonly string[]>>
    readonly under?: Readonly<Record<string, Readonly<Record<string, readonly string[]>>>>
  }
}

const createProblems = (corpus: Corpus): DeckProblems => ({
  at: new Map(Object.entries(corpus.wrong?.at ?? {})),
  under: new Map(
    Object.entries(corpus.wrong?.under ?? {}).map(([card, fields]) => [
      card,
      new Map(Object.entries(fields)),
    ]),
  ),
})

const ANIMAL: CardModel = { name: 'Animal', fields: ['Name', 'Height', 'Weight', 'Life span'] }
const WORD: CardModel = { name: 'Word', fields: ['Word', 'Meaning', 'Example'] }
const ASKED: CardModel = { name: 'Basic', fields: ['Question', 'Answer'] }

const UNBROKEN = 'supercalifragilisticexpialidociousandthensomemoreofitwithnothingtobreakatanywhere'

const many = (count: number): readonly DeckCard[] =>
  Array.from({ length: count }, (_, index) =>
    index % 2 === 0
      ? {
          id: `beast00${index}`,
          section: null,
          cardModel: 'Animal',
          filled: [
            { field: 'Name', text: `Beast ${index + 1}` },
            { field: 'Height', text: 'about 45"' },
            { field: 'Weight', text: '- bull: 350 kg\n- cow: 300 kg\n- calf: 40 kg' },
            { field: 'Life span', text: 'about 20 years' },
          ],
        }
      : {
          id: `asked00${index}`,
          section: null,
          cardModel: 'Basic',
          filled: [{ field: 'Question', text: `Question ${index + 1}` }],
        },
  )

const CORPORA = {
  'a deck': {
    cuts: [ANIMAL, WORD],
    cards: [
      {
        id: 'k7m2xq9fzp',
        section: null,
        cardModel: 'Animal',
        filled: [
          { field: 'Name', text: 'Llama' },
          { field: 'Height', text: 'about 45" *at the shoulder*' },
          { field: 'Life span', text: 'about **20 years**' },
        ],
      },
      {
        id: '3n8vr4tqch',
        section: null,
        cardModel: 'Animal',
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
        cardModel: 'Word',
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
        cardModel: 'Basic',
        filled: [
          { field: 'Question', text: 'What is a deck for?' },
          { field: 'Answer', text: 'Cards, and the questions on them.' },
        ],
      },
      {
        id: 'r2t7y5k9wq',
        section: 'roots',
        cardModel: 'Basic',
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
        cardModel: 'Animal',
        filled: [
          { field: 'Name', text: 'Earthworm' },
          { field: 'Life span', text: 'about 2 years' },
        ],
      },
    ],
  },
  'far too many': { cuts: [ANIMAL, ASKED], cards: many(300) },
  /* Values that are not Latin, beside a field name and a value with nothing in
     them to break at. */
  'awkward text': {
    cuts: [
      { name: 'Слово', fields: ['Слово', 'Перевод', 'Пример'] },
      { name: UNBROKEN, fields: [UNBROKEN, 'Long'] },
    ],
    cards: [
      {
        id: 'j2b6t8n4vw',
        section: null,
        cardModel: 'Слово',
        filled: [
          { field: 'Слово', text: 'Компост' },
          { field: 'Перевод', text: 'compost, перепревшие листья и трава' },
          { field: 'Пример', text: 'बगीचे की खाद और हरी खाद' },
        ],
      },
      {
        id: 'h5r1w7z3qm',
        section: null,
        cardModel: UNBROKEN,
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
  /* Every fault a tile can carry: a card under no card model, a card naming a
     card model that was not handed in, a card writing one field twice, a card from
     somebody else carrying marks that must never be drawn as marks, and two
     cards the file names alike, which are two cards and stand as two. */
  'what is wrong': {
    cuts: [ANIMAL, WORD, ASKED],
    cards: [
      { id: 'b8k4n2vqz6', section: null, cardModel: null, filled: [] },
      {
        id: 'm3t9w5rj1x',
        section: null,
        cardModel: 'Gone',
        filled: [{ field: 'Whatever it had', text: 'still here, still readable' }],
      },
      {
        id: 'd6q2z8hn4v',
        section: null,
        cardModel: 'Animal',
        filled: [
          { field: 'Name', text: 'Llama' },
          { field: 'Name', text: 'Alpaca' },
          { field: 'Height', text: 'about 45"' },
        ],
      },
      {
        id: 'y1v5b9kt3n',
        section: null,
        cardModel: 'Basic',
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
      /* A card copied manually and the card it was copied from: the file names
         the two alike, and the window draws each under an identity of its own
         so both are read and both are typed into. */
      {
        id: 'copied-one',
        section: null,
        cardModel: 'Basic',
        filled: [
          { field: 'Question', text: 'Compost, what is it made of' },
          { field: 'Answer', text: 'Leaves and peelings' },
        ],
      },
      {
        id: 'copied-again',
        section: null,
        cardModel: 'Basic',
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
        y1v5b9kt3n: { Answer: ['this value is not in the card model this card is cut by'] },
      },
    },
  },
} satisfies Record<string, Corpus>

type Corpora = keyof typeof CORPORA

interface Knobs {
  corpus: Corpora
  name: string
  width: string
  cards?: never
  cuts?: never
  sections?: never
  words?: never
}

const meta: Meta<Knobs> = {
  title: 'Flash Cards/Deck',
  component: DeckEditor,
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
    components: { DeckEditor },
    setup() {
      const cards = ref<readonly DeckCard[]>(CORPORA[args.corpus].cards)
      const cuts = ref<readonly CardModel[]>(CORPORA[args.corpus].cuts)
      const sections = ref<readonly DeckSection[]>(getSections(CORPORA[args.corpus]))
      const wrong = ref<DeckProblems>(createProblems(CORPORA[args.corpus]))

      watch(
        () => args.corpus,
        (next) => {
          cards.value = CORPORA[next].cards
          cuts.value = CORPORA[next].cuts
          sections.value = getSections(CORPORA[next])
          wrong.value = createProblems(CORPORA[next])
        },
      )

      /** The cards, with one of them changed. */
      const changeCard = (id: string, into: (card: DeckCard) => DeckCard): readonly DeckCard[] =>
        cards.value.map((card) => (card.id === id ? into(card) : card))

      /** The section the last of them is, which is where a new card is made. */
      const lastSection = (): string | null => sections.value.at(-1)?.id ?? null

      return {
        args,
        cards,
        cuts,
        sections,
        wrong,
        onAdd: (cardModel: string) => {
          const id = `made00000${cards.value.length}`
          const matchingCut = cuts.value.find((each) => each.name === cardModel)
          const filled = getBlanks(getDeclaredFields(matchingCut?.fields ?? []))
          cards.value = [...cards.value, { id, section: lastSection(), cardModel, filled }]
        },
        onRemove: (id: string) => {
          cards.value = cards.value.filter((card) => card.id !== id)
        },
        onMove: (id: string, target: InsertionPoint) => {
          cards.value = moveCard(cards.value, sections.value, id, target)
        },
        onWrite: (id: string, field: string, index: number, text: string) => {
          cards.value = changeCard(id, (card) => {
            // The one written is the one counted off under its own field.
            let underCount = 0
            const filled = card.filled.map((each) => {
              if (each.field !== field) return each
              underCount += 1
              return underCount === index ? { field, text } : each
            })
            if (underCount < index) filled.push({ field, text })
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
        <DeckEditor
          :cards="cards"
          :cardModels="cuts"
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

const getSections = (corpus: Corpus): readonly DeckSection[] => corpus.sections ?? []

const moveCard = (
  cards: readonly DeckCard[],
  sections: readonly DeckSection[],
  id: string,
  targetPoint: InsertionPoint,
): readonly DeckCard[] => {
  const card = cards.find((card) => card.id === id)
  if (!card) return cards
  const remainingCards = cards.filter((card) => card.id !== id)

  if (targetPoint === HEAD) return [{ ...card, section: null }, ...remainingCards]

  const before =
    targetPoint === null ? -1 : remainingCards.findIndex((card) => card.id === targetPoint)
  if (before !== -1) {
    const underItem = { ...card, section: remainingCards[before]?.section ?? null }
    return [...remainingCards.slice(0, before), underItem, ...remainingCards.slice(before)]
  }

  const rank = (section: string | null): number =>
    section === null ? 0 : sections.findIndex((each) => each.id === section) + 1

  if (targetPoint === null) {
    return [...remainingCards, { ...card, section: sections.at(-1)?.id ?? null }]
  }
  if (!sections.some((each) => each.id === targetPoint)) return cards

  const seat = remainingCards.findIndex((card) => rank(card.section) >= rank(targetPoint))
  const where = seat === -1 ? remainingCards.length : seat
  return [
    ...remainingCards.slice(0, where),
    { ...card, section: targetPoint },
    ...remainingCards.slice(where),
  ]
}

export default meta
type Story = StoryObj<Knobs>

/**
 * Three cards, two card models, and one card missing a field. The plus button prompts for the
 * card model, and the card it creates has empty fields.
 */
export const ADeck: Story = {
  play: async ({ canvasElement }) => {
    await expect(canvasElement.querySelectorAll('[data-cut]')).toHaveLength(0)

    await userEvent.click(found(canvasElement, '[data-plus] button'))
    await expect(canvasElement.querySelectorAll('[data-cut]')).toHaveLength(2)

    await userEvent.click(found(canvasElement, '[data-cut="Word"]'))

    const tiles = [...canvasElement.querySelectorAll('[data-card]')]
    const createdItem = tiles[tiles.length - 1]

    // Every field the card model declares is placed under its own name, the first
    // included, and all of them stand empty.
    const boxes = [...(createdItem?.querySelectorAll<HTMLTextAreaElement>('textarea') ?? [])]
    await expect(boxes.map((box) => box.getAttribute('data-value'))).toEqual([
      'Word',
      'Meaning',
      'Example',
    ])
    await expect(boxes.every((box) => box.value === '')).toBe(true)
  },
}

/**
 * A deck grown past one list: cards standing before the first section, a
 * section holding several, and a section holding none. A card is dropped at the
 * head of a section, an empty one included, and a section taken away leaves its
 * cards at their positions.
 */
export const InSections: Story = {
  args: { corpus: 'a deck in sections' },
  play: async ({ canvasElement }) => {
    const heads = [...canvasElement.querySelectorAll('[data-section-head]')]
    await expect(heads.map((head) => head.getAttribute('data-section-head'))).toEqual([
      'roots',
      'leaves',
    ])

    // A section holding no card keeps its heading.
    await expect(canvasElement.querySelectorAll('[data-section="leaves"]')).toHaveLength(0)

    // The heading is a rule with the name typed on it, and the way to be rid of
    // it at the end.
    await expect(found(canvasElement, '[data-section-head="roots"] .divider')).toBeTruthy()
    await expect(
      found(canvasElement, '[data-section-head="roots"] input').getAttribute('value'),
    ).toBe('Roots')

    // Each card is placed under the section it is in, and the ones before the
    // first stand under none.
    await expect(canvasElement.querySelectorAll('[data-section="roots"]')).toHaveLength(2)
    await expect(
      found(canvasElement, '[data-card="p4h6c8vzn2"]').getAttribute('data-section'),
    ).toBeNull()

    // A section made at the end, under a name nothing has taken.
    await userEvent.click(found(canvasElement, '[data-add-section]'))
    const createdItem = [...canvasElement.querySelectorAll('[data-section-head]')]
    await expect(createdItem).toHaveLength(3)
    await expect(createdItem[2]?.querySelector('input')?.value).toBe('Section 1')

    // Taking a section away takes away its heading and nothing else: its cards
    // stand under the heading above them now.
    await userEvent.click(found(canvasElement, '[data-section-head="roots"] .remove-button'))
    await expect(canvasElement.querySelectorAll('[data-section-head="roots"]')).toHaveLength(0)
    await expect(canvasElement.querySelectorAll('[data-card]')).toHaveLength(3)
    await expect(
      found(canvasElement, '[data-card="z3f1m6b4dt"]').getAttribute('data-section'),
    ).toBeNull()
  },
}

/** Three hundred cards cut by two card models of different sizes: the deck draws its tiles. */
export const FarTooMany: Story = {
  args: { corpus: 'far too many' },
  play: async ({ canvasElement }) => {
    await expect(canvasElement.querySelectorAll('[data-card]').length).toBeGreaterThanOrEqual(3)
  },
}

/** Values that are not Latin, and a field name with nothing to break at. */
export const AwkwardText: Story = { args: { corpus: 'awkward text' } }

/** No cards, and one plus alone in the empty deck. */
export const NothingAtAll: Story = {
  args: { corpus: 'nothing at all' },
  play: async ({ canvasElement }) => {
    const button = found(canvasElement, '[data-plus] button')

    await expect(canvasElement.querySelectorAll('[data-card]')).toHaveLength(0)

    // It is named for assistive technology and has no visible text.
    await expect(button.getAttribute('aria-label')).toBe('Add a card')
    await expect(button.textContent?.trim()).toBe('')

    // The pressed button opens the prompt inside the same tile.
    await userEvent.click(button)
    const promptElement = found(canvasElement, '[data-plus] .deck__asking')
    await expect(promptElement).toBeVisible()
  },
}

/**
 * A deck the vault found five things wrong with. Every mark relies on the card
 * it was read against and on no other, and what a card is drawn as is the
 * card's own.
 */
export const WhatIsWrongWithACard: Story = {
  args: { corpus: 'what is wrong' },
  play: async ({ canvasElement }) => {
    const getText = (card: string, selector: string): string =>
      found(canvasElement, `[data-card="${card}"] ${selector}`).textContent ?? ''

    await expect(getText('b8k4n2vqz6', '[data-wrong]')).toContain('not a lone wikilink')
    await expect(getText('d6q2z8hn4v', '[data-wrong]')).toContain('writes one field twice')
    await expect(getText('d6q2z8hn4v', '[data-wrong-value="Name"]')).toContain('writes Name twice')
    await expect(getText('y1v5b9kt3n', '[data-wrong-value="Answer"]')).toContain(
      'not in the card model',
    )

    // The card with no problems has no mark.
    await expect(canvasElement.querySelector('[data-card="m3t9w5rj1x"] [data-wrong]')).toBeNull()
    await expect(canvasElement.querySelectorAll('[data-wrong]')).toHaveLength(4)

    // Two cards of one card id stand as two tiles, each carrying what is wrong
    // with it and each typed into on its own.
    await expect(getText('copied-one', '[data-wrong]')).toContain('carry the mark')
    await expect(getText('copied-again', '[data-wrong]')).toContain('carry the mark')

    const box = (card: string): HTMLTextAreaElement =>
      found(canvasElement, `[data-card="${card}"] [data-value="Answer"]`) as HTMLTextAreaElement

    await userEvent.type(box('copied-one'), ' and left to rot down')

    await expect(box('copied-one').value).toBe('Leaves and peelings and left to rot down')
    await expect(box('copied-again').value).toBe('Leaves and peelings, turned')
  },
}

const found = (canvas: HTMLElement, selector: string): HTMLElement => {
  const element = canvas.querySelector<HTMLElement>(selector)
  if (!element) throw new Error(`nothing matching ${selector}`)
  return element
}
