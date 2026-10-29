/**
 * Every situation one card of a deck has to survive. Also the test corpus: each
 * story is run in a browser by `@storybook/addon-vitest`.
 *
 * What the card returns is applied here, which is the application's part.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent } from 'storybook/test'
import { provide, ref, watch } from 'vue'
import Card from './Card.vue'
import { DECK_CONTEXT } from '../../../model/context'
import type { DeckCard } from '../../../lib/deck'
import { getGrid, type Tile } from '../../../lib/grid'
import type { CardModel } from '../../../lib/card'

interface Corpus {
  readonly card: DeckCard
  readonly cut: CardModel | null
  /** What the vault reading the file found wrong with this card. */
  readonly wrong?: readonly string[]
  readonly wrongUnder?: Readonly<Record<string, readonly string[]>>
}

const ANIMAL: CardModel = { name: 'Animal', fields: ['Name', 'Height', 'Weight', 'Life span'] }

const UNBROKEN = 'supercalifragilisticexpialidociousandthensomemoreofitwithnothingtobreakatanywhere'

const CORPORA = {
  'a card': {
    cut: ANIMAL,
    card: {
      id: 'k7m2xq9fzp',
      section: null,
      cardModel: 'Animal',
      filled: [
        { field: 'Name', text: 'Llama' },
        { field: 'Height', text: 'about 45" *at the shoulder*' },
        { field: 'Life span', text: 'about **20 years**' },
      ],
    },
  },
  'a value that wraps': {
    cut: { name: 'Basic', fields: ['Question', 'Answer'] },
    card: {
      id: 'c5n8q2wxjb',
      section: null,
      cardModel: 'Basic',
      filled: [
        { field: 'Question', text: 'What is compost?' },
        {
          field: 'Answer',
          text: 'Leaves, grass and kitchen peelings, turned twice and left under a sheet until the heap has gone dark and crumbly enough to spread on any bed of the plot.',
        },
      ],
    },
  },
  /* A first field of several lines, which is what a heading of one line is read
     from and what no heading could hold. */
  'a first field of many lines': {
    cut: { name: 'Basic', fields: ['Question', 'Answer'] },
    card: {
      id: 'r2t7y5k9wq',
      section: null,
      cardModel: 'Basic',
      filled: [
        {
          field: 'Question',
          text: '> Двух станов не боец, а только гость случайный\n\nWho wrote it, and of whom?',
        },
        { field: 'Answer', text: 'Alexey Konstantinovich Tolstoy, of himself' },
      ],
    },
  },
  /* Values that are not Latin, and a field name with nothing in it to break at. */
  'awkward text': {
    cut: { name: 'Слово', fields: ['Слово', 'Перевод', UNBROKEN] },
    card: {
      id: 'j2b6t8n4vw',
      section: null,
      cardModel: 'Слово',
      filled: [
        { field: 'Слово', text: 'Компост' },
        { field: 'Перевод', text: 'compost, перепревшие листья и трава' },
        { field: UNBROKEN, text: 'बगीचे की खाद और हरी खाद' },
      ],
    },
  },
  /* A card whose every value is the empty string. */
  'no text at all': {
    cut: ANIMAL,
    card: { id: 'b8k4n2vqz6', section: null, cardModel: 'Animal', filled: [] },
  },
  /* A card naming a card model the vault does not hold: the fields are
     unknown, so no boxes are drawn and the card's content stays in the file. */
  'no card model': {
    cut: null,
    card: {
      id: 'm3t9w5rj1x',
      section: null,
      cardModel: 'Gone',
      filled: [{ field: 'Whatever it had', text: 'still here, still readable' }],
    },
  },
  /* A card whose first paragraph is not a lone wikilink. Nothing cuts it, so
     nothing lays it out, and everything it wrote stands as it was written. */
  'cut by nothing': {
    cut: null,
    card: {
      id: 'w4h7p2ctm9',
      section: null,
      cardModel: null,
      filled: [
        { field: 'Question', text: 'What did I mean to put here?' },
        { field: 'Answer', text: 'A note about compost,\nand no card at all.' },
      ],
    },
    wrong: ['the first paragraph of this card is not a lone wikilink, so it names no card model'],
  },
  /* A card with one field written twice. Both values are displayed so the user
     can see them and remove one. */
  'a field written twice': {
    cut: ANIMAL,
    card: {
      id: 'd6q2z8hn4v',
      section: null,
      cardModel: 'Animal',
      filled: [
        { field: 'Name', text: 'Llama' },
        { field: 'Name', text: 'Alpaca' },
        { field: 'Height', text: 'about 45"' },
      ],
    },
    wrong: ['this card writes one field twice'],
    wrongUnder: { Name: ['this card writes Name twice'] },
  },
  /* A card from another source. A value is a box, so no tag is rendered. */
  'tags that must not survive': {
    cut: { name: 'Basic', fields: ['Question', 'Answer'] },
    card: {
      id: 'y1v5b9kt3n',
      section: null,
      cardModel: 'Basic',
      filled: [
        { field: 'Question', text: 'A card from somebody else' },
        {
          field: 'Answer',
          text:
            '<script>window.stolen = 1</script>' +
            '<img src="x" onerror="window.stolen = 2">' +
            'Only these words should stand.',
        },
      ],
    },
  },
} satisfies Record<string, Corpus>

type Corpora = keyof typeof CORPORA

/** The one card of a corpus, laid out against the card model that cuts it. */
const createTile = (corpus: Corpus): Tile => {
  const cardGrid = getGrid(
    { cards: [corpus.card], sections: [], cardModels: corpus.cut ? [corpus.cut] : [] },
    null,
  ).runs[0]?.tiles[0]
  if (!cardGrid) throw new Error('a corpus holding no card')
  return cardGrid
}

interface Knobs {
  corpus: Corpora
  /** How wide the window drawing the card is. */
  width: string
  /** Given by the story, and nothing a reader turns. */
  tile?: never
  wrong?: never
  wrongUnder?: never
  words?: never
}

const meta: Meta<Knobs> = {
  title: 'Flash Cards/Card',
  component: Card,
  parameters: { layout: 'centered' },
  argTypes: {
    corpus: {
      control: 'select',
      options: Object.keys(CORPORA),
      description: 'What the card starts as. Changing it starts afresh.',
    },
    width: { control: 'text' },
    tile: { table: { disable: true } },
    wrong: { table: { disable: true } },
    wrongUnder: { table: { disable: true } },
    words: { table: { disable: true } },
  },
  args: { corpus: 'a card', width: '20rem' },
  render: (args) => ({
    components: { Card },
    setup() {
      const target = ref<Corpus>(CORPORA[args.corpus])
      const tile = ref<Tile>(createTile(CORPORA[args.corpus]))

      watch(
        () => args.corpus,
        (next) => {
          target.value = CORPORA[next]
          tile.value = createTile(CORPORA[next])
        },
      )

      const onWrite = (field: string, index: number, text: string) => {
        const card = target.value.card

        // The one written is the one counted off under its own field.
        let underCount = 0
        const filled = card.filled.map((each) => {
          if (each.field !== field) return each
          underCount += 1
          return underCount === index ? { field, text } : each
        })
        if (underCount < index) filled.push({ field, text })
        const content = { ...target.value, card: { ...card, filled } }
        target.value = content
        tile.value = createTile(content)
      }

      provide(DECK_CONTEXT, {
        write: (_card, { field, index }, text) => onWrite(field, index, text),
      })

      return {
        args,
        tile,
        wrong: () => target.value.wrong ?? [],
        wrongUnder: () => new Map(Object.entries(target.value.wrongUnder ?? {})),
      }
    },
    template: `
      <div :style="{ width: args.width }">
        <Card
          :tile="tile"
          :wrong="wrong()"
          :wrong-under="wrongUnder()"
        />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const found = (canvas: HTMLElement, selector: string): HTMLElement => {
  const element = canvas.querySelector<HTMLElement>(selector)
  if (!element) throw new Error(`nothing matching ${selector}`)
  return element
}

/**
 * A card cut by a card model of four fields, holding two of them.
 *
 * A value is the area under the rule that names it: the name is anchored to the
 * line, the box begins where the line ends, and nothing is drawn around it.
 * Every value is editable, and the strip above them is not.
 */
export const ACard: Story = {
  play: async ({ canvasElement }) => {
    const tile = found(canvasElement, '[data-card="k7m2xq9fzp"]')
    const value = found(canvasElement, '.card__value')
    const rule = value.querySelector('.divider')
    const label = value.querySelector('label')
    if (!rule || !label) throw new Error('no rule and no name on it')

    // One name, and it relies on the rule, so the two share a middle.
    expect(value.querySelectorAll('label')).toHaveLength(1)
    expect(label.closest('.divider')).toBe(rule)
    const line = rule.getBoundingClientRect()
    const name = label.getBoundingClientRect()
    expect(Math.abs((name.top + name.bottom) / 2 - (line.top + line.bottom) / 2)).toBeLessThan(2)

    // The rule spans both edges of the card, past the area of the values.
    const edges = tile.getBoundingClientRect()
    expect(line.left - edges.left).toBeLessThan(2)
    expect(edges.right - line.right).toBeLessThan(2)

    const boxes = [...tile.querySelectorAll<HTMLTextAreaElement>('textarea')]
    expect(boxes).toHaveLength(4)
    for (const box of boxes) {
      const underElement = box.closest('.card__value')?.querySelector('.divider')
      if (!underElement) throw new Error('a box under no rule')
      expect(box.getBoundingClientRect().top).toBeCloseTo(
        underElement.getBoundingClientRect().bottom,
        0,
      )
      expect(box.closest('.card__value')?.querySelector('fieldset')).toBeNull()
    }

    // Empty and one-line boxes stand to one height.
    const heights = boxes.map((box) => Math.round(box.getBoundingClientRect().height))
    expect(new Set(heights).size).toBe(1)

    // The card keeps the surface that both the rules and the boxes are drawn on.
    const cell = found(canvasElement, '.autosize')
    const ground = getComputedStyle(cell).backgroundColor
    expect(ground === 'rgba(0, 0, 0, 0)' || ground === 'transparent').toBe(true)
    expect(getComputedStyle(cell).borderTopWidth).toBe('0px')
    expect(getComputedStyle(tile).backgroundColor).not.toBe('rgba(0, 0, 0, 0)')

    // The strip shows the card model of the card and holds the one control to remove it.
    const header = found(canvasElement, '.card-header')
    expect(found(canvasElement, '[data-cut-of]').textContent?.trim()).toBe('Animal')
    expect(tile.querySelector('[aria-expanded]')).toBeNull()
    expect(header.querySelectorAll('button')).toHaveLength(1)

    // the card holds.
    const cutElement = found(canvasElement, '[data-cut-of]')
    const middle = (each: Element): number => {
      const box = each.getBoundingClientRect()
      return Math.round(box.left + box.width / 2)
    }
    expect(middle(cutElement)).toBe(middle(header))

    const response = Number.parseFloat(getComputedStyle(cutElement).fontSize)
    const content = Number.parseFloat(
      getComputedStyle(found(canvasElement, '[data-value="Height"]')).fontSize,
    )
    expect(response).toBeLessThan(content)

    const box = found(canvasElement, '[data-value="Height"]')
    await userEvent.click(box)
    await userEvent.type(box, '!')
    expect((box as HTMLTextAreaElement).value).toContain('!')

    // Every box keeps what was typed, including line breaks and the first field: a
    // field holds as many lines as are written.
    const first = found(canvasElement, '[data-value="Name"]') as HTMLTextAreaElement
    await userEvent.click(first)
    await userEvent.type(first, '{Enter}b')
    expect(first.value).toContain('\n')
  },
}

/**
 * A first field written over several lines, which is what no heading could
 */
export const AFirstFieldOfManyLines: Story = {
  args: { corpus: 'a first field of many lines' },
  play: async ({ canvasElement }) => {
    const box = found(canvasElement, '[data-value="Question"]')
    const line = Number.parseFloat(getComputedStyle(box).lineHeight)
    expect(box.clientHeight).toBeGreaterThan(line * 2)
    expect(box.scrollHeight).toBeLessThanOrEqual(box.clientHeight + 1)
  },
}

/**
 * A value that wraps: its height is the result of the wrapping, which is
 * computed by the layout of the surrounding element.
 */
export const AValueThatWraps: Story = {
  args: { corpus: 'a value that wraps', width: '18rem' },
  play: async ({ canvasElement }) => {
    const box = found(canvasElement, '[data-value="Answer"]')

    const line = Number.parseFloat(getComputedStyle(box).lineHeight)
    expect(box.clientHeight).toBeGreaterThan(line * 2)
    expect(box.scrollHeight).toBeLessThanOrEqual(box.clientHeight + 1)
  },
}

/** Values that are not Latin, and a field name with nothing to break at. */
export const AwkwardText: Story = { args: { corpus: 'awkward text' } }

/** A card whose every value is the empty string. */
export const NoTextAtAll: Story = { args: { corpus: 'no text at all' } }

/** A card naming a card model the vault does not hold. */
export const NoCardModel: Story = {
  args: { corpus: 'no card model' },
  play: async ({ canvasElement }) => {
    const tile = found(canvasElement, '[data-card="m3t9w5rj1x"]')

    // The card model is unknown, so no boxes are drawn and the missing card model
    // is named.
    expect(tile.textContent).toContain('No card model called Gone')
    expect(tile.querySelectorAll('textarea')).toHaveLength(0)

    // the body keeps, in from the tile's own stroke.
    const body = found(canvasElement, '.card__body')
    const clearance = Number.parseFloat(getComputedStyle(body).paddingInlineStart)
    const stroke = Number.parseFloat(getComputedStyle(tile).borderInlineStartWidth)
    const objections = found(canvasElement, '.card__objections').getBoundingClientRect()
    expect(Math.round(objections.left)).toBe(
      Math.round(tile.getBoundingClientRect().left + stroke + clearance),
    )

    // over.
    const gap = found(canvasElement, '.card__silence').getBoundingClientRect()
    const middle = (box: DOMRect): number => Math.round(box.left + box.width / 2)
    expect(middle(gap)).toBe(middle(body.getBoundingClientRect()))
  },
}

/**
 * A card whose first paragraph is not a lone wikilink.
 *
 * The missing card model is reported as a plain fact on the strip, where the
 * name of a card model is shown for every other card. The problems the vault
 * found with the card are reported once, and the card content is shown where
 * it is edited.
 */
export const CutByNothing: Story = {
  args: { corpus: 'cut by nothing' },
  play: async ({ canvasElement }) => {
    const tile = found(canvasElement, '[data-card="w4h7p2ctm9"]')

    // The strip carries it, in the voice it names a card model in.
    const cutElement = found(canvasElement, '[data-cut-of]')
    expect(cutElement.textContent?.trim()).toBe('Cut by no card model')
    const label = found(canvasElement, '.card__field')
    expect(getComputedStyle(cutElement).color).toBe(getComputedStyle(label).color)

    // The problem is reported once: the body shows what the vault found and nothing else.
    const response = [...tile.querySelectorAll('.card__objections')]
    expect(response).toHaveLength(1)
    expect(response[0]?.textContent).toContain('not a lone wikilink')
    expect(getComputedStyle(cutElement).color).not.toBe(
      getComputedStyle(response[0] as Element).color,
    )

    expect(tile.querySelector('.card__silence')).toBeNull()

    // What the card wrote is read, breaks and all, and typed into nothing.
    expect(tile.querySelectorAll('textarea')).toHaveLength(0)
    const wrote = [...tile.querySelectorAll<HTMLElement>('[data-wrote]')]
    expect(wrote.map((each) => each.getAttribute('data-wrote'))).toEqual(['Question', 'Answer'])
    expect(wrote[1]?.textContent).toContain('\n')

    const edge = (each: Element): number => {
      const box = each.getBoundingClientRect()
      return Math.round(box.left + Number.parseFloat(getComputedStyle(each).paddingInlineStart))
    }
    expect(edge(wrote[0] as Element)).toBe(edge(label))
  },
}

/**
 * A card with one field written twice, with the problems the vault found.
 *
 * Problems with the card are shown under its strip, above the first value;
 * problems with one value are shown under that value, once, at the last box of
 * that value.
 */
export const WhatIsWrongWithIt: Story = {
  args: { corpus: 'a field written twice' },
  play: async ({ canvasElement }) => {
    // Both values are kept, each in a box of its own.
    const boxes = [...canvasElement.querySelectorAll<HTMLTextAreaElement>('[data-value="Name"]')]
    expect(boxes.map((box) => box.value)).toEqual(['Llama', 'Alpaca'])

    const response = found(canvasElement, '[data-wrong]')
    expect(response.textContent).toContain('writes one field twice')
    const first = found(canvasElement, '.card__value')
    expect(response.getBoundingClientRect().bottom).toBeLessThanOrEqual(
      first.getBoundingClientRect().top + 1,
    )

    // It is reported once, under the second of the two boxes for Name.
    const underCards = [...canvasElement.querySelectorAll('[data-wrong-value="Name"]')]
    expect(underCards).toHaveLength(1)
    expect(underCards[0]?.textContent).toContain('this card writes Name twice')
    expect(underCards[0]?.getBoundingClientRect().top).toBeGreaterThanOrEqual(
      boxes[1]?.getBoundingClientRect().bottom ?? 0,
    )

    // The name of a value, the box it is typed in and what is wrong with it
    // all stand over one edge.
    const edge = (each: Element | null | undefined): number => {
      if (!each) throw new Error('nothing to measure')
      const box = each.getBoundingClientRect()
      const renderedStyle = getComputedStyle(each)
      return box.left + Number.parseFloat(renderedStyle.paddingInlineStart)
    }
    const label = found(canvasElement, '.card__value label')
    expect(Math.round(edge(underCards[0]))).toBe(Math.round(edge(boxes[1])))
    expect(Math.round(edge(label))).toBe(Math.round(edge(boxes[1])))
    expect(Math.round(edge(response))).toBe(Math.round(edge(boxes[0])))
  },
}

/** A card from another source. A value is a box, so no tag is rendered. */
export const TagsThatMustNotSurvive: Story = {
  args: { corpus: 'tags that must not survive' },
  play: async ({ canvasElement }) => {
    expect(canvasElement.querySelector('script')).toBeNull()
    expect(canvasElement.querySelector('img')).toBeNull()
    expect(canvasElement.querySelector('[onerror]')).toBeNull()
    expect((window as unknown as Record<string, unknown>)['stolen']).toBeUndefined()

    // The content the user wrote is readable and editable as the text of a box.
    const box = canvasElement.querySelector<HTMLTextAreaElement>('[data-value="Answer"]')
    expect(box?.value).toContain('<script>')
  },
}
