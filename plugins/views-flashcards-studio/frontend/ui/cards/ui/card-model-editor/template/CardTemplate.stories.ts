/**
 * Every situation one template of a card model has to survive. Also the test corpus:
 * each story is run in a browser by `@storybook/addon-vitest`.
 *
 * What the template returns is applied here, which is the card model's part.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { computed, ref, watch } from 'vue'
import CardTemplate from './CardTemplate.vue'
import type { FieldValue } from '../../../lib/card'
import { getDeclaredFields, type Half } from '../../../lib/order'
import { templateRows, type TemplateRow, type CardModelTemplate } from '../../../lib/card-model'
import { sampleValues } from '../../../lib/fill'
import { hoverOver, lightness } from '../../../../fixtures/colour'
import { DARK, expectDark } from '../../../../fixtures/theme'

interface Corpus {
  readonly template: CardModelTemplate
  /** The fields the card model declares, which are what may be written into a half. */
  readonly fields: readonly string[]
  readonly sample?: readonly FieldValue[]
  /** The names the other templates carry. */
  readonly taken?: readonly string[]
  /** What the vault reading the file found wrong with this template. */
  readonly wrong?: readonly string[]
}

const ANIMAL = ['Name', 'Height', 'Weight', 'Life span']

const many = (count: number): readonly string[] =>
  Array.from({ length: count }, (_, index) => `Field ${index + 1}`)

const CORPORA = {
  'a template': {
    fields: ANIMAL,
    template: {
      id: 'recognise',
      name: 'Recognise',
      front: '{{Name}}',
      back: '<b>Height:</b> {{Height}}\n<b>Weight:</b> {{Weight}}\n<b>Life span:</b> {{Life span}}',
    },
    sample: [
      { field: 'Name', text: 'Llama' },
      { field: 'Height', text: 'about 45" at the shoulder' },
      { field: 'Weight', text: '130 kg' },
      { field: 'Life span', text: '20 years' },
    ],
    taken: ['Name it'],
  },
  /* A template naming slots the card model does not declare, in both halves. */
  'a stray slot': {
    fields: ['Name', 'Height'],
    template: {
      id: 'stray',
      name: 'Stray',
      front: '{{Name}} the {{Colour}} one',
      back: '<b>Height:</b> {{Height}}\n<b>Weight:</b> {{Weight}}',
    },
  },
  /* A template with nothing in either half, and no name either. */
  'nothing in it': {
    fields: ANIMAL,
    template: { id: 'blank', name: '', front: '', back: '' },
  },
  /* Marks the user wrote, which render as text, and tags, which are rendered as HTML. */
  'marks a person wrote': {
    fields: ['Name', 'Height'],
    template: {
      id: 'tagged',
      name: 'Tagged',
      front: '**{{Name}}** and <b>{{Name}}</b>',
      back: '<span style="color: teal">{{Height}}</span>\n<script>window.stolen = 1</script>',
    },
  },
  /* Forty fields, more than the strip can display. */
  'far too many fields': {
    fields: many(40),
    template: {
      id: 'crowded',
      name: 'Crowded',
      front: '{{Field 1}}',
      back: many(40)
        .map((field) => `<b>${field}:</b> {{${field}}}`)
        .join('\n'),
    },
  },
  /* What the vault found wrong with the template, and a name another template has. */
  'something wrong with it': {
    fields: ANIMAL,
    template: { id: 'amiss', name: 'Recall', front: '{{Name}}', back: '' },
    taken: ['Recognise'],
    wrong: ['this template has no back', 'and nothing fills it'],
  },
} satisfies Record<string, Corpus>

type Corpora = keyof typeof CORPORA

interface Knobs {
  corpus: Corpora
  /** How wide the window drawing the template is. */
  width: string
  /** Given by the story, and nothing a reader turns. */
  template?: never
  fields?: never
  taken?: never
  wrong?: never
  words?: never
}

const meta: Meta<Knobs> = {
  title: 'Flash Cards/Template',
  component: CardTemplate,
  parameters: { layout: 'padded' },
  argTypes: {
    corpus: {
      control: 'select',
      options: Object.keys(CORPORA),
      description: 'What the template starts as. Changing it starts afresh.',
    },
    width: { control: 'text' },
    template: { table: { disable: true } },
    fields: { table: { disable: true } },
    taken: { table: { disable: true } },
    wrong: { table: { disable: true } },
    words: { table: { disable: true } },
  },
  args: { corpus: 'a template', width: '64rem' },
  render: (args) => ({
    components: { CardTemplate },
    setup() {
      const target = ref<Corpus>(CORPORA[args.corpus])
      const content = ref<CardModelTemplate>(CORPORA[args.corpus].template)

      watch(
        () => args.corpus,
        (next) => {
          target.value = CORPORA[next]
          content.value = CORPORA[next].template
        },
      )

      const renderedRow = computed<TemplateRow>(() => {
        const fields = getDeclaredFields(target.value.fields)
        const arrangedRow = templateRows(
          [content.value],
          fields,
          target.value.sample ?? sampleValues(fields),
        )[0]
        if (!arrangedRow) throw new Error('a corpus holding no template')
        return { ...arrangedRow, taken: target.value.taken ?? [] }
      })

      return {
        args,
        drawn: renderedRow,
        wrong: () => target.value.wrong ?? [],
        onRename: (name: string) => {
          content.value = { ...content.value, name }
        },
        onWrite: (half: Half, text: string) => {
          content.value = { ...content.value, [half]: text }
        },
      }
    },
    template: `
      <div :style="{ width: args.width }">
        <CardTemplate
          :template="drawn"
          :wrong="wrong()"
          @rename="onRename"
          @write="onWrite"
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

const getBox = (canvas: HTMLElement, half: Half): HTMLTextAreaElement => {
  const box = canvas.querySelector<HTMLTextAreaElement>(`[data-half="${half}"]`)
  if (!box) throw new Error(`no ${half}`)
  return box
}

const getPane = (canvas: HTMLElement, pane: string): HTMLElement =>
  found(canvas, `[data-pane="${pane}"]`)

/** The four parts of a template, in the order they are drawn. */
const PANES = ['front-written', 'front-preview', 'back-written', 'back-preview'] as const

/**
 * One template of a card model of four fields, its previews standing what a card fills
 * them with.
 *
 * The window is one window: two parts to a row, the writing beside its preview
 * and the front above the back, divided by the lines they share. The frame
 * around all four is the template's, the parts carry none of their own, and the
 * keyboard landing in one changes nothing that is drawn.
 */
export const ATemplate: Story = {
  play: async ({ canvasElement }) => {
    const body = found(canvasElement, '.template__body')
    expect(getComputedStyle(body).gridTemplateColumns.split(' ')).toHaveLength(2)

    const getPaneBox = (pane: string): DOMRect =>
      getPane(canvasElement, pane).getBoundingClientRect()

    // The writing is stored alongside its preview, on one line.
    expect(Math.round(getPaneBox('front-written').top)).toBe(
      Math.round(getPaneBox('front-preview').top),
    )
    expect(getPaneBox('front-written').right).toBeLessThanOrEqual(getPaneBox('front-preview').left)

    // The front is above the back, and both halves are split the same way.
    expect(getPaneBox('front-written').bottom).toBeLessThanOrEqual(getPaneBox('back-written').top)
    expect(Math.round(getPaneBox('back-written').top)).toBe(
      Math.round(getPaneBox('back-preview').top),
    )

    // What divides them is one line, shared by the two it divides.
    const line = Number.parseFloat(getComputedStyle(body).columnGap)
    expect(line).toBeGreaterThan(0)
    expect(getPaneBox('front-preview').left - getPaneBox('front-written').right).toBeCloseTo(
      line,
      0,
    )
    expect(getPaneBox('back-written').top - getPaneBox('front-written').bottom).toBeCloseTo(line, 0)

    // A part is a box to type in, not a line: the box fills the part, so every
    // point of the part focuses the box.
    const part = getPane(canvasElement, 'front-written')
    const box = getBox(canvasElement, 'front')
    const deep = Number.parseFloat(getComputedStyle(box).lineHeight)
    expect(box.getBoundingClientRect().height).toBeGreaterThan(deep * 4)
    expect(box.getBoundingClientRect().height).toBeCloseTo(part.getBoundingClientRect().height, 0)

    // The template is framed and the parts inside it are not.
    const template = found(canvasElement, '[data-template]')
    expect(Number.parseFloat(getComputedStyle(template).borderTopWidth)).toBeGreaterThan(0)
    for (const pane of PANES) {
      const viewElement = getComputedStyle(getPane(canvasElement, pane))
      for (const side of [
        'borderTopWidth',
        'borderInlineStartWidth',
        'borderBottomWidth',
        'borderInlineEndWidth',
      ] as const) {
        expect(Number.parseFloat(viewElement[side])).toBe(0)
      }
      expect(viewElement.outlineStyle).toBe('none')
    }
    expect(body.querySelectorAll('fieldset, legend')).toHaveLength(0)

    // The preview shows the card with the slots filled, without the braces of the
    // markup it came from.
    const preview = found(canvasElement, '[data-preview="back"]')
    expect(preview.textContent).toContain('about 45" at the shoulder')
    expect(preview.textContent).not.toContain('{{')
    expect(preview.querySelector('b')?.textContent).toBe('Height:')

    // A template is HTML, so each of its three bare lines reads as a line of its
    // own: three lines of type, and not one line running into the next.
    const answer = found(canvasElement, '[data-preview="back"] .prose')
    const tall = Number.parseFloat(getComputedStyle(answer).lineHeight)
    expect(tall).toBeGreaterThan(0)
    expect(answer.getBoundingClientRect().height).toBeCloseTo(tall * 3, 0)

    // A field to insert into a half is a small, muted chip with its own background.
    const chip = found(canvasElement, '[data-insert="Height"]')
    const title = found(canvasElement, '.template__title')
    expect(getComputedStyle(chip).backgroundColor).not.toBe('rgba(0, 0, 0, 0)')
    expect(Number.parseFloat(getComputedStyle(chip).fontSize)).toBeLessThan(
      Number.parseFloat(getComputedStyle(title).fontSize),
    )

    // A field is inserted into the half last typed in, at the caret.
    const back = getBox(canvasElement, 'back')
    back.focus()
    back.setSelectionRange(0, 0)
    await userEvent.click(chip)
    await waitFor(() =>
      expect(getBox(canvasElement, 'back').value.startsWith('{{Height}}')).toBe(true),
    )

    // The keyboard landing in a box changes the template, the strip and the parts
    // in nothing.
    const watched = [
      template,
      found(canvasElement, '.card-header'),
      ...PANES.map((pane) => getPane(canvasElement, pane)),
    ]
    const look = (): readonly string[] =>
      watched.map((each) => {
        const renderedRow = getComputedStyle(each)
        return [
          renderedRow.backgroundColor,
          renderedRow.borderColor,
          renderedRow.borderWidth,
          renderedRow.boxShadow,
          renderedRow.outlineStyle,
          renderedRow.outlineWidth,
        ].join('|')
      })

    const before = look()
    const content = getBox(canvasElement, 'front')
    content.focus()
    expect(document.activeElement).toBe(content)
    expect(look()).toEqual(before)
  },
}

/** A window too narrow for two parts to a row: one column, each under its half. */
export const Narrow: Story = {
  args: { width: '22rem' },
  play: async ({ canvasElement }) => {
    const body = found(canvasElement, '.template__body')
    expect(getComputedStyle(body).gridTemplateColumns.split(' ')).toHaveLength(1)

    const tops = PANES.map((pane) =>
      Math.round(getPane(canvasElement, pane).getBoundingClientRect().top),
    )
    expect([...tops].sort((first, second) => first - second)).toEqual(tops)
    expect(new Set(tops).size).toBe(4)
  },
}

/** A template naming slots the card model does not declare, in both halves. */
export const AStraySlot: Story = {
  args: { corpus: 'a stray slot' },
  play: async ({ canvasElement }) => {
    // The error is shown at the slot itself, and under that half only.
    expect(found(canvasElement, '[data-preview="front"] mark').textContent).toBe('{{Colour}}')
    expect(found(canvasElement, '[data-preview="back"] mark').textContent).toBe('{{Weight}}')
    expect(
      found(canvasElement, '[data-pane="front-written"] .template__objections').textContent?.trim(),
    ).toBe('Not a field: Colour')
    expect(
      found(canvasElement, '[data-pane="back-written"] .template__objections').textContent?.trim(),
    ).toBe('Not a field: Weight')
    expect(canvasElement.querySelector('header .template__objections')).toBeNull()

    // The box still fills the part, and a click on the part targets the box.
    const pane = getPane(canvasElement, 'front-written').getBoundingClientRect()
    const box = getBox(canvasElement, 'front').getBoundingClientRect()
    expect(box.height).toBeCloseTo(pane.height, 0)

    const layer = found(canvasElement, '[data-pane="front-written"] .template__amiss')
    expect(getComputedStyle(layer).pointerEvents).toBe('none')

    const response = found(canvasElement, '[data-pane="front-written"] .template__objections')
    const overRect = response.getBoundingClientRect()
    expect(overRect.bottom).toBeLessThanOrEqual(pane.bottom + 1)
    expect(overRect.right).toBeLessThanOrEqual(pane.right + 1)
    expect(getComputedStyle(response).backgroundColor).not.toBe('rgba(0, 0, 0, 0)')
  },
}

/** A template with nothing in either half, and no name either. */
export const NothingInIt: Story = {
  args: { corpus: 'nothing in it' },
  play: async ({ canvasElement }) => {
    // An empty part shows its placeholder, centered.
    const placeholders: string[] = []
    for (const pane of PANES) {
      const element = getPane(canvasElement, pane)
      const ghost = element.querySelector<HTMLElement>('.template__ghost')
      if (!ghost) throw new Error('template not found')
      placeholders.push(ghost.textContent?.trim() ?? '')

      const box = element.getBoundingClientRect()
      const middle = ghost.getBoundingClientRect()
      expect(middle.left + middle.width / 2).toBeCloseTo(box.left + box.width / 2, 0)
      expect(middle.top + middle.height / 2).toBeCloseTo(box.top + box.height / 2, 0)
    }
    expect(placeholders).toEqual(['Front', 'Preview', 'Back', 'Preview'])

    // What is written into a part takes that place, and the part then draws the
    // box alone.
    const content = getBox(canvasElement, 'front')
    await userEvent.type(content, 'a word')
    await waitFor(() => {
      expect(getPane(canvasElement, 'front-written').querySelector('.template__ghost')).toBeNull()
      expect(getPane(canvasElement, 'front-preview').querySelector('.template__ghost')).toBeNull()
    })
    const pane = getPane(canvasElement, 'front-written')
    expect(content.getBoundingClientRect().top).toBeCloseTo(pane.getBoundingClientRect().top, 0)

    // The half with no text still shows its placeholder.
    expect(
      getPane(canvasElement, 'back-written').querySelector('.template__ghost')?.textContent,
    ).toBe('Back')
  },
}

/**
 * A card is HTML: a tag the user wrote is rendered, and the marks of another
 * format render as literal characters.
 */
export const MarksAPersonWrote: Story = {
  args: { corpus: 'marks a person wrote' },
  play: async ({ canvasElement }) => {
    expect(canvasElement.querySelector('script')).toBeNull()
    expect((window as unknown as Record<string, unknown>)['stolen']).toBeUndefined()

    // The tag is drawn; the asterisks around the same value are four characters
    // of the answer, and nothing about them is bold.
    const front = found(canvasElement, '[data-preview="front"]')
    expect(front.textContent).toBe('**Name** and Name')
    expect(front.querySelectorAll('b, strong')).toHaveLength(1)

    // The user's text is editable and shown as the text of a box.
    expect(getBox(canvasElement, 'back').value).toContain('<script>')
  },
}

/**
 * Forty fields, more than the strip can display. The row of fields scrolls
 * inside the strip, which keeps its height and heading.
 */
export const FarTooManyFields: Story = {
  args: { corpus: 'far too many fields' },
  play: async ({ canvasElement }) => {
    const header = found(canvasElement, '.card-header')
    const title = found(canvasElement, '.template__title')
    const slots = found(canvasElement, '.template__slots')

    // Forty fields do not push the heading out of its own strip.
    expect(title.getBoundingClientRect().width).toBeGreaterThan(60)

    expect(slots.scrollWidth).toBeGreaterThan(slots.clientWidth)
    expect(header.getBoundingClientRect().height).toBeLessThan(40)
    expect(slots.getBoundingClientRect().right).toBeLessThanOrEqual(
      header.getBoundingClientRect().right + 1,
    )
  },
}

/**
 * What the vault found wrong with the template, and a name another template carries.
 *
 * The errors found against the template are shown at the end of the strip, above
 * the editor below it, and the error for a name being typed is shown in the same
 * place, at the box.
 */
export const WhatIsWrongWithIt: Story = {
  args: { corpus: 'something wrong with it' },
  play: async ({ canvasElement }) => {
    const response = found(canvasElement, '[data-wrong]')
    expect(response.textContent).toContain('this template has no back')
    expect(response.querySelectorAll('li')).toHaveLength(2)

    // The strip is one row high with all items rendered, and the editor begins
    // where the strip ends: the error overlays the editor and takes no space from
    // it.
    const header = found(canvasElement, '.card-header').getBoundingClientRect()
    const head = found(canvasElement, '.template__head').getBoundingClientRect()
    const body = found(canvasElement, '.template__body').getBoundingClientRect()
    const overRect = response.getBoundingClientRect()
    expect(header.height).toBeLessThan(40)
    expect(body.top).toBeCloseTo(header.bottom, 0)
    expect(overRect.top).toBeGreaterThanOrEqual(head.bottom - 1)
    expect(overRect.bottom).toBeGreaterThan(body.top)

    // It stays inside the template, and a click meant for the editor is received by it.
    const template = found(canvasElement, '[data-template]').getBoundingClientRect()
    expect(overRect.right).toBeLessThanOrEqual(template.right + 1)
    expect(getComputedStyle(found(canvasElement, 'header .template__amiss')).pointerEvents).toBe(
      'none',
    )
    expect(getComputedStyle(response).backgroundColor).not.toBe('rgba(0, 0, 0, 0)')

    // A name that another template uses is rejected, and the box shows the error.
    const name = found(canvasElement, '.template__title') as HTMLInputElement
    await userEvent.clear(name)
    await userEvent.type(name, 'Recognise')
    await waitFor(() => {
      const objections = found(canvasElement, 'header [role="alert"]')
      expect(objections.textContent?.trim()).toBe('That name is taken')
      expect(name.getAttribute('aria-describedby')).toBe(objections.id)
    })
  },
}

/**
 * On the dark set of tokens, where a field chip's hover colour is the template's
 * text colour mixed into the chip's background.
 *
 * The mix is one expression for both sets, so on the dark set it moves the chip
 * towards the text colour, which is lighter there than the background.
 */
export const Dark: Story = {
  globals: DARK,
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    const chip = within(canvasElement).getByRole('button', { name: 'Insert: Height' })

    // The chip's text is lighter than its background, as the dark set defines
    // them.
    const ink = lightness(getComputedStyle(chip).color)
    const restingLightness = lightness(getComputedStyle(chip).backgroundColor)
    expect(ink).toBeGreaterThan(restingLightness)

    await hoverOver(chip)
    await waitFor(() =>
      expect(lightness(getComputedStyle(chip).backgroundColor)).toBeGreaterThan(
        restingLightness + 2,
      ),
    )

    // The hovered chip is lighter than at rest, and still darker than its text.
    expect(lightness(getComputedStyle(chip).backgroundColor)).toBeLessThan(ink)
  },
}
