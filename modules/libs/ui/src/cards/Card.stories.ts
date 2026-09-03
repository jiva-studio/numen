/**
 * Every situation one card of a deck has to survive. Also the test corpus: each
 * story is run in a browser by `@storybook/addon-vitest`.
 *
 * What the card hands back is applied here, which is the application's part.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent } from 'storybook/test'
import { ref, watch } from 'vue'
import Card from './Card.vue'
import { grid, type Drawn, type Tile } from './deck'
import type { Stencil } from './stencil'

interface Corpus {
  readonly card: Drawn
  readonly cut: Stencil | null
  /** What the vault reading the file found wrong with this card. */
  readonly wrong?: readonly string[]
  /** What it found wrong with one of its values, by the field it stands in. */
  readonly wrongUnder?: Readonly<Record<string, readonly string[]>>
}

const ANIMAL: Stencil = { name: 'Animal', fields: ['Name', 'Height', 'Weight', 'Life span'] }

const UNBROKEN =
  'supercalifragilisticexpialidociousandthensomemoreofitwithnothingtobreakatanywhere'

const CORPORA = {
  'a card': {
    cut: ANIMAL,
    card: {
      id: 'k7m2xq9fzp',
      section: null,
      stencil: 'Animal',
      filled: [
        { field: 'Name', text: 'Llama' },
        { field: 'Height', text: 'about 45" *at the shoulder*' },
        { field: 'Life span', text: 'about **20 years**' },
      ],
    },
  },
  /* A value with no newline in it that is far wider than the box, so what the
     box stands at can only be worked out from where the text wraps. */
  'a value that wraps': {
    cut: { name: 'Basic', fields: ['Question', 'Answer'] },
    card: {
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
  },
  /* A first field of several lines, which is what a heading of one line is read
     from and what no heading could hold. */
  'a first field of many lines': {
    cut: { name: 'Basic', fields: ['Question', 'Answer'] },
    card: {
      id: 'r2t7y5k9wq',
      section: null,
      stencil: 'Basic',
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
      stencil: 'Слово',
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
    card: { id: 'b8k4n2vqz6', section: null, stencil: 'Animal', filled: [] },
  },
  /* A card naming a stencil the vault does not hold: nothing says what its
     boxes are, so it draws none and keeps what it holds in the file. */
  'no stencil': {
    cut: null,
    card: {
      id: 'm3t9w5rj1x',
      section: null,
      stencil: 'Gone',
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
      stencil: null,
      filled: [
        { field: 'Question', text: 'What did I mean to put here?' },
        { field: 'Answer', text: 'A note about compost,\nand no card at all.' },
      ],
    },
    wrong: ['the first paragraph of this card is not a lone wikilink, so it names no stencil'],
  },
  /* A card writing one field twice. Both values are kept where a person can see
     them and take one out. */
  'a field written twice': {
    cut: ANIMAL,
    card: {
      id: 'd6q2z8hn4v',
      section: null,
      stencil: 'Animal',
      filled: [
        { field: 'Name', text: 'Llama' },
        { field: 'Name', text: 'Alpaca' },
        { field: 'Height', text: 'about 45"' },
      ],
    },
    wrong: ['this card writes one field twice'],
    wrongUnder: { Name: ['this card writes Name twice'] },
  },
  /* A card from somebody else. A value is a box, so no tag of one is drawn. */
  'tags that must not survive': {
    cut: { name: 'Basic', fields: ['Question', 'Answer'] },
    card: {
      id: 'y1v5b9kt3n',
      section: null,
      stencil: 'Basic',
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

/** The one card of a corpus, laid out against the stencil that cuts it. */
const tileOf = (corpus: Corpus): Tile => {
  const laid = grid([corpus.card], [], corpus.cut ? [corpus.cut] : [], null).runs[0]?.tiles[0]
  if (!laid) throw new Error('a corpus holding no card')
  return laid
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
      const held = ref<Corpus>(CORPORA[args.corpus])
      const tile = ref<Tile>(tileOf(CORPORA[args.corpus]))

      watch(
        () => args.corpus,
        (next) => {
          held.value = CORPORA[next]
          tile.value = tileOf(CORPORA[next])
        },
      )

      return {
        args,
        tile,
        wrong: () => held.value.wrong ?? [],
        wrongUnder: () => new Map(Object.entries(held.value.wrongUnder ?? {})),
        onWrite: (field: string, nth: number, text: string) => {
          const card = held.value.card

          // The one written is the one counted off under its own field.
          let under = 0
          const filled = card.filled.map((each) => {
            if (each.field !== field) return each
            under += 1
            return under === nth ? { field, text } : each
          })
          if (under < nth) filled.push({ field, text })
          const written = { ...held.value, card: { ...card, filled } }
          held.value = written
          tile.value = tileOf(written)
        },
      }
    },
    template: `
      <div :style="{ width: args.width }">
        <Card
          :tile="tile"
          :wrong="wrong()"
          :wrong-under="wrongUnder()"
          @write="onWrite"
        />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const found = (canvas: HTMLElement, selector: string): HTMLElement => {
  const held = canvas.querySelector<HTMLElement>(selector)
  if (!held) throw new Error(`nothing matching ${selector}`)
  return held
}

/**
 * A card cut by a stencil of four fields, holding two of them.
 *
 * A value is the room under the rule that names it: the name stands on the
 * line, the box begins where the line ends, and nothing is drawn around it.
 * Every one of them is open to typing, and the strip over them opens nothing.
 */
export const ACard: Story = {
  play: async ({ canvasElement }) => {
    const tile = found(canvasElement, '[data-card="k7m2xq9fzp"]')
    const value = found(canvasElement, '.card__value')
    const rule = value.querySelector('.rule')
    const label = value.querySelector('label')
    if (!rule || !label) throw new Error('no rule and no name on it')

    // One name, and it stands on the rule, so the two share a middle.
    expect(value.querySelectorAll('label')).toHaveLength(1)
    expect(label.closest('.rule')).toBe(rule)
    const line = rule.getBoundingClientRect()
    const name = label.getBoundingClientRect()
    expect(Math.abs((name.top + name.bottom) / 2 - (line.top + line.bottom) / 2)).toBeLessThan(2)

    // The rule runs to both edges of the card, past the room the values keep.
    const edges = tile.getBoundingClientRect()
    expect(line.left - edges.left).toBeLessThan(2)
    expect(edges.right - line.right).toBeLessThan(2)

    const boxes = [...tile.querySelectorAll<HTMLTextAreaElement>('textarea')]
    expect(boxes).toHaveLength(4)
    for (const box of boxes) {
      const under = box.closest('.card__value')?.querySelector('.rule')
      if (!under) throw new Error('a box under no rule')
      expect(box.getBoundingClientRect().top).toBeCloseTo(under.getBoundingClientRect().bottom, 0)
      expect(box.closest('.card__value')?.querySelector('fieldset')).toBeNull()
    }

    // Empty and one-line boxes stand to one height.
    const heights = boxes.map((box) => Math.round(box.getBoundingClientRect().height))
    expect(new Set(heights).size).toBe(1)

    // The card keeps the ground both the rules and the boxes stand on.
    const grown = found(canvasElement, '.grown')
    const ground = getComputedStyle(grown).backgroundColor
    expect(ground === 'rgba(0, 0, 0, 0)' || ground === 'transparent').toBe(true)
    expect(getComputedStyle(grown).borderTopWidth).toBe('0px')
    expect(getComputedStyle(tile).backgroundColor).not.toBe('rgba(0, 0, 0, 0)')

    // The strip says what cut the card, and holds the one way to remove it.
    const bar = found(canvasElement, '.bar')
    expect(found(canvasElement, '[data-cut-of]').textContent?.trim()).toBe('Animal')
    expect(tile.querySelector('[aria-expanded]')).toBeNull()
    expect(bar.querySelectorAll('button')).toHaveLength(1)

    // It stands in the middle of the strip, and is said more quietly than what
    // the card holds.
    const cut = found(canvasElement, '[data-cut-of]')
    const middle = (each: Element): number => {
      const box = each.getBoundingClientRect()
      return Math.round(box.left + box.width / 2)
    }
    expect(middle(cut)).toBe(middle(bar))

    const said = Number.parseFloat(getComputedStyle(cut).fontSize)
    const written = Number.parseFloat(
      getComputedStyle(found(canvasElement, '[data-value="Height"]')).fontSize,
    )
    expect(said).toBeLessThan(written)

    const box = found(canvasElement, '[data-value="Height"]')
    await userEvent.click(box)
    await userEvent.type(box, '!')
    expect((box as HTMLTextAreaElement).value).toContain('!')

    // Every box keeps what was typed, breaks and all, the first field's among
    // them: a field holds as many lines as a person writes.
    const first = found(canvasElement, '[data-value="Name"]') as HTMLTextAreaElement
    await userEvent.click(first)
    await userEvent.type(first, '{Enter}b')
    expect(first.value).toContain('\n')
  },
}

/**
 * A first field written over several lines, which is what no heading could
 * hold. It stands in a box like any other field, and the box grows to it.
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
 * A value with no newline in it that wraps to several lines. Its box stands at
 * what the wrapping comes to, which is worked out by the ground behind it.
 */
export const AValueThatWraps: Story = {
  args: { corpus: 'a value that wraps', width: '18rem' },
  play: async ({ canvasElement }) => {
    const box = found(canvasElement, '[data-value="Answer"]')

    // It wrapped: the box stands taller than the one line it is written on.
    const line = Number.parseFloat(getComputedStyle(box).lineHeight)
    expect(box.clientHeight).toBeGreaterThan(line * 2)
    expect(box.scrollHeight).toBeLessThanOrEqual(box.clientHeight + 1)
  },
}

/** Values that are not Latin, and a field name with nothing to break at. */
export const AwkwardText: Story = { args: { corpus: 'awkward text' } }

/** A card whose every value is the empty string. */
export const NoTextAtAll: Story = { args: { corpus: 'no text at all' } }

/** A card naming a stencil the vault does not hold. */
export const NoStencil: Story = {
  args: { corpus: 'no stencil' },
  play: async ({ canvasElement }) => {
    const tile = found(canvasElement, '[data-card="m3t9w5rj1x"]')

    // Nothing says what its boxes are, so it draws none and says what it asked
    // for.
    expect(tile.textContent).toContain('No stencil called Gone')
    expect(tile.querySelectorAll('textarea')).toHaveLength(0)

    // What is wrong with it stands over the edge a value would: the clearance
    // the body keeps, in from the tile's own stroke.
    const body = found(canvasElement, '.card__body')
    const clearance = Number.parseFloat(getComputedStyle(body).paddingInlineStart)
    const stroke = Number.parseFloat(getComputedStyle(tile).borderInlineStartWidth)
    const objects = found(canvasElement, '.card__objects').getBoundingClientRect()
    expect(Math.round(objects.left)).toBe(
      Math.round(tile.getBoundingClientRect().left + stroke + clearance),
    )

    // What it says of holding nothing stands in the middle of the room left
    // over.
    const silence = found(canvasElement, '.card__silence').getBoundingClientRect()
    const middle = (box: DOMRect): number => Math.round(box.left + box.width / 2)
    expect(middle(silence)).toBe(middle(body.getBoundingClientRect()))
  },
}

/**
 * A card whose first paragraph is not a lone wikilink.
 *
 * That nothing cut it is a plain fact and stands quietly on the strip, where
 * the name of a stencil stands on every other card. What the vault holds
 * against the card is said once, and what the card holds is read where it
 * would be typed.
 */
export const CutByNothing: Story = {
  args: { corpus: 'cut by nothing' },
  play: async ({ canvasElement }) => {
    const tile = found(canvasElement, '[data-card="w4h7p2ctm9"]')

    // The strip carries it, in the voice it names a stencil in.
    const cut = found(canvasElement, '[data-cut-of]')
    expect(cut.textContent?.trim()).toBe('Cut by no stencil')
    const label = found(canvasElement, '.card__field')
    expect(getComputedStyle(cut).color).toBe(getComputedStyle(label).color)

    // It is said once: the body carries what the vault found and nothing else.
    const said = [...tile.querySelectorAll('.card__objects')]
    expect(said).toHaveLength(1)
    expect(said[0]?.textContent).toContain('not a lone wikilink')
    expect(getComputedStyle(cut).color).not.toBe(getComputedStyle(said[0] as Element).color)

    // No heading stands over a body that holds something.
    expect(tile.querySelector('.card__silence')).toBeNull()

    // What the card wrote is read, breaks and all, and typed into nothing.
    expect(tile.querySelectorAll('textarea')).toHaveLength(0)
    const wrote = [...tile.querySelectorAll<HTMLElement>('[data-wrote]')]
    expect(wrote.map((each) => each.getAttribute('data-wrote'))).toEqual(['Question', 'Answer'])
    expect(wrote[1]?.textContent).toContain('\n')

    // A value stands over the edge the name it is written under stands on.
    const edge = (each: Element): number => {
      const box = each.getBoundingClientRect()
      return Math.round(box.left + Number.parseFloat(getComputedStyle(each).paddingInlineStart))
    }
    expect(edge(wrote[0] as Element)).toBe(edge(label))
  },
}

/**
 * A card writing one field twice, with what the vault found wrong with it.
 *
 * What stands against the card is said under its strip, above the first value;
 * what stands against one value is said under that value, once, at the last box
 * standing for it.
 */
export const WhatIsWrongWithIt: Story = {
  args: { corpus: 'a field written twice' },
  play: async ({ canvasElement }) => {
    // Both values are kept, each in a box of its own.
    const boxes = [...canvasElement.querySelectorAll<HTMLTextAreaElement>('[data-value="Name"]')]
    expect(boxes.map((box) => box.value)).toEqual(['Llama', 'Alpaca'])

    const said = found(canvasElement, '[data-wrong]')
    expect(said.textContent).toContain('writes one field twice')
    const first = found(canvasElement, '.card__value')
    expect(said.getBoundingClientRect().bottom).toBeLessThanOrEqual(
      first.getBoundingClientRect().top + 1,
    )

    // It is said once, under the second of the two boxes standing for Name.
    const under = [...canvasElement.querySelectorAll('[data-wrong-value="Name"]')]
    expect(under).toHaveLength(1)
    expect(under[0]?.textContent).toContain('this card writes Name twice')
    expect(under[0]?.getBoundingClientRect().top).toBeGreaterThanOrEqual(
      boxes[1]?.getBoundingClientRect().bottom ?? 0,
    )

    // The name of a value, the box it is typed in and what is wrong with it
    // all stand over one edge.
    const edge = (each: Element | null | undefined): number => {
      if (!each) throw new Error('nothing to measure')
      const box = each.getBoundingClientRect()
      const drawn = getComputedStyle(each)
      return box.left + Number.parseFloat(drawn.paddingInlineStart)
    }
    const label = found(canvasElement, '.card__value label')
    expect(Math.round(edge(under[0]))).toBe(Math.round(edge(boxes[1])))
    expect(Math.round(edge(label))).toBe(Math.round(edge(boxes[1])))
    expect(Math.round(edge(said))).toBe(Math.round(edge(boxes[0])))
  },
}

/** A card from somebody else. A value is a box, so no tag of one is drawn. */
export const TagsThatMustNotSurvive: Story = {
  args: { corpus: 'tags that must not survive' },
  play: async ({ canvasElement }) => {
    expect(canvasElement.querySelector('script')).toBeNull()
    expect(canvasElement.querySelector('img')).toBeNull()
    expect(canvasElement.querySelector('[onerror]')).toBeNull()
    expect((window as unknown as Record<string, unknown>)['stolen']).toBeUndefined()

    // What the person wrote is theirs to read and change, standing as the text
    // of a box.
    const box = canvasElement.querySelector<HTMLTextAreaElement>('[data-value="Answer"]')
    expect(box?.value).toContain('<script>')
  },
}
