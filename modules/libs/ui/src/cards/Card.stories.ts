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
import type { Cut } from './stencil'

interface Corpus {
  readonly card: Drawn
  readonly cut: Cut | null
  /** What the vault reading the file found wrong with this card. */
  readonly wrong?: readonly string[]
  /** What it found wrong with one of its values, by the field it stands in. */
  readonly wrongUnder?: Readonly<Record<string, readonly string[]>>
}

const ANIMAL: Cut = { name: 'Animal', fields: ['Name', 'Height', 'Weight', 'Life span'] }

const UNBROKEN =
  'supercalifragilisticexpialidociousandthensomemoreofitwithnothingtobreakatanywhere'

const CORPORA = {
  'a card': {
    cut: ANIMAL,
    card: {
      id: 'llama',
      name: 'Llama',
      stencil: 'Animal',
      filled: [
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
  },
  /* A name and a value that are not Latin, and a name with nothing in it to
     break at. */
  'awkward text': {
    cut: { name: 'Слово', fields: ['Слово', 'Перевод', UNBROKEN] },
    card: {
      id: 'компост',
      name: 'Компост',
      stencil: 'Слово',
      filled: [
        { field: 'Перевод', text: 'compost, перепревшие листья и трава' },
        { field: UNBROKEN, text: 'बगीचे की खाद और हरी खाद' },
      ],
    },
  },
  /* A card whose name and every value is the empty string. */
  'no text at all': {
    cut: ANIMAL,
    card: { id: 'blank', name: '', stencil: 'Animal', filled: [] },
  },
  /* A card naming a stencil the vault does not hold: nothing says what its
     boxes are, so it draws none and keeps what it holds in the file. */
  'no stencil': {
    cut: null,
    card: {
      id: 'orphan',
      name: 'Orphan',
      stencil: 'Gone',
      filled: [{ field: 'Whatever it had', text: 'still here, still readable' }],
    },
  },
  /* The heading names the card, and the value under that same field is kept
     where a person can see it and take it out. */
  'named twice': {
    cut: ANIMAL,
    card: {
      id: 'twice',
      name: 'Llama',
      stencil: 'Animal',
      filled: [
        { field: 'Name', text: 'Alpaca' },
        { field: 'Height', text: 'about 45"' },
      ],
    },
    wrong: ['another card is called Llama'],
    wrongUnder: { Name: ['this card writes Name twice'] },
  },
  /* A card from somebody else. A value is a box, so no tag of one is drawn. */
  'tags that must not survive': {
    cut: { name: 'Basic', fields: ['Question', 'Answer'] },
    card: {
      id: 'theirs',
      name: 'A card from somebody else',
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
  },
} satisfies Record<string, Corpus>

type Corpora = keyof typeof CORPORA

/** The one card of a corpus, laid out against the stencil that cuts it. */
const tileOf = (corpus: Corpus): Tile => {
  const laid = grid([corpus.card], corpus.cut ? [corpus.cut] : [], null).tiles[0]
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
        /* A card is named by its first field, and that value stands in the
           heading and in none of the card's values. */
        onWrite: (field: string, nth: number, names: boolean, text: string) => {
          const card = held.value.card
          if (names) {
            tile.value = tileOf({ ...held.value, card: { ...card, name: text } })
            held.value = { ...held.value, card: { ...card, name: text } }
            return
          }

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
    const tile = found(canvasElement, '[data-card="llama"]')
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
  },
}

/**
 * The name is written in a heading, so its box holds one line: a break struck
 * in it is refused, and what the card hands back has none.
 */
export const TheNameHoldsOneLine: Story = {
  play: async ({ canvasElement }) => {
    const name = found(canvasElement, '[data-value="Name"]') as HTMLTextAreaElement
    await userEvent.click(name)
    await userEvent.type(name, '{Enter}b')

    expect(name.value).not.toContain('\n')

    // Every other box keeps what was typed, breaks and all.
    const height = found(canvasElement, '[data-value="Height"]') as HTMLTextAreaElement
    await userEvent.click(height)
    await userEvent.type(height, '{Enter}b')
    expect(height.value).toContain('\n')
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

/** A name and a value that are not Latin, and a name with nothing to break at. */
export const AwkwardText: Story = { args: { corpus: 'awkward text' } }

/** A card whose name and every value is the empty string. */
export const NoTextAtAll: Story = { args: { corpus: 'no text at all' } }

/** A card naming a stencil the vault does not hold. */
export const NoStencil: Story = {
  args: { corpus: 'no stencil' },
  play: async ({ canvasElement }) => {
    const tile = found(canvasElement, '[data-card="orphan"]')

    // Nothing says what its boxes are, so it draws none and says what it asked
    // for.
    expect(tile.textContent).toContain('No stencil called Gone')
    expect(tile.querySelectorAll('textarea')).toHaveLength(0)

    // Its name, what is wrong with it and what it says of holding nothing all
    // stand over one edge.
    const edge = (selector: string): number => {
      const each = found(canvasElement, selector)
      return Math.round(
        each.getBoundingClientRect().left +
          Number.parseFloat(getComputedStyle(each).paddingInlineStart),
      )
    }
    expect(edge('.card__objects')).toBe(edge('.card__said'))
    expect(edge('.card__silence')).toBe(edge('.card__said'))
  },
}

/**
 * A card carrying the field that names it as a value as well, with what the
 * vault found wrong with it.
 *
 * What stands against the card is said under its heading, above the first
 * value; what stands against one value is said under that value, once, at the
 * last box standing for it.
 */
export const WhatIsWrongWithIt: Story = {
  args: { corpus: 'named twice' },
  play: async ({ canvasElement }) => {
    // The heading names the card; the value is kept and said to be one too many.
    const boxes = [...canvasElement.querySelectorAll<HTMLTextAreaElement>('[data-value="Name"]')]
    expect(boxes.map((box) => box.value)).toEqual(['Llama', 'Alpaca'])

    const row = canvasElement.querySelector('[data-twice]')
    expect(row?.textContent).toContain('The card is named by this field')
    expect(row?.getAttribute('data-stray')).toBeNull()

    const said = found(canvasElement, '[data-wrong]')
    expect(said.textContent).toContain('another card is called Llama')
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
