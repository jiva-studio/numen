/**
 * Every situation one face of a stencil has to survive. Also the test corpus:
 * each story is run in a browser by `@storybook/addon-vitest`.
 *
 * What the face returns is applied here, which is the stencil's part.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { computed, ref, watch } from 'vue'
import Face from './Face.vue'
import type { FieldValue } from '../../../lib/card'
import { getDeclaredFields, type Half } from '../../../lib/order'
import { faceRows, type FaceRow, type StencilFace } from '../../../lib/stencil'
import { sampleValues } from '../../../lib/fill'
import { hoverOver, lightness } from '@/shared/fixtures/colour'
import { DARK, expectDark } from '@/shared/fixtures/theme'

interface Corpus {
  readonly face: StencilFace
  /** The fields the stencil declares, which are what may be written into a half. */
  readonly fields: readonly string[]
  readonly sample?: readonly FieldValue[]
  /** The names the other faces carry. */
  readonly taken?: readonly string[]
  /** What the vault reading the file found wrong with this face. */
  readonly wrong?: readonly string[]
}

const ANIMAL = ['Name', 'Height', 'Weight', 'Life span']

const many = (count: number): readonly string[] =>
  Array.from({ length: count }, (_, at) => `Field ${at + 1}`)

const CORPORA = {
  'a face': {
    fields: ANIMAL,
    face: {
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
  /* A face naming slots the stencil does not declare, in both halves. */
  'a stray slot': {
    fields: ['Name', 'Height'],
    face: {
      id: 'stray',
      name: 'Stray',
      front: '{{Name}} the {{Colour}} one',
      back: '<b>Height:</b> {{Height}}\n<b>Weight:</b> {{Weight}}',
    },
  },
  /* A face with nothing in either half, and no name either. */
  'nothing in it': {
    fields: ANIMAL,
    face: { id: 'blank', name: '', front: '', back: '' },
  },
  /* Marks a person wrote, which stand as text, and tags, which are drawn. */
  'marks a person wrote': {
    fields: ['Name', 'Height'],
    face: {
      id: 'tagged',
      name: 'Tagged',
      front: '**{{Name}}** and <b>{{Name}}</b>',
      back: '<span style="color: teal">{{Height}}</span>\n<script>window.stolen = 1</script>',
    },
  },
  /* Forty fields to write in, far more than the strip has room for. */
  'far too many fields': {
    fields: many(40),
    face: {
      id: 'crowded',
      name: 'Crowded',
      front: '{{Field 1}}',
      back: many(40)
        .map((field) => `<b>${field}:</b> {{${field}}}`)
        .join('\n'),
    },
  },
  /* What the vault found wrong with the face, and a name another face has. */
  'something wrong with it': {
    fields: ANIMAL,
    face: { id: 'amiss', name: 'Recall', front: '{{Name}}', back: '' },
    taken: ['Recognise'],
    wrong: ['this face has no back', 'and nothing fills it'],
  },
} satisfies Record<string, Corpus>

type Corpora = keyof typeof CORPORA

interface Knobs {
  corpus: Corpora
  /** How wide the window drawing the face is. */
  width: string
  /** Given by the story, and nothing a reader turns. */
  face?: never
  fields?: never
  taken?: never
  wrong?: never
  words?: never
}

const meta: Meta<Knobs> = {
  title: 'Flash Cards/Face',
  component: Face,
  parameters: { layout: 'padded' },
  argTypes: {
    corpus: {
      control: 'select',
      options: Object.keys(CORPORA),
      description: 'What the face starts as. Changing it starts afresh.',
    },
    width: { control: 'text' },
    face: { table: { disable: true } },
    fields: { table: { disable: true } },
    taken: { table: { disable: true } },
    wrong: { table: { disable: true } },
    words: { table: { disable: true } },
  },
  args: { corpus: 'a face', width: '64rem' },
  render: (args) => ({
    components: { Face },
    setup() {
      const target = ref<Corpus>(CORPORA[args.corpus])
      const content = ref<StencilFace>(CORPORA[args.corpus].face)

      watch(
        () => args.corpus,
        (next) => {
          target.value = CORPORA[next]
          content.value = CORPORA[next].face
        },
      )

      const renderedRow = computed<FaceRow>(() => {
        const fields = getDeclaredFields(target.value.fields)
        const arrangedRow = faceRows([content.value], fields, target.value.sample ?? sampleValues(fields))[0]
        if (!arrangedRow) throw new Error('a corpus holding no face')
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
        <Face
          :face="drawn"
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

/** The four parts of a face, in the order they are drawn. */
const PANES = ['front-written', 'front-preview', 'back-written', 'back-preview'] as const

/**
 * One face of a stencil of four fields, its previews standing what a card fills
 * them with.
 *
 * The window is one window: two parts to a row, the writing beside its preview
 * and the front above the back, divided by the lines they share. The frame
 * around all four is the face's, the parts carry none of their own, and the
 * keyboard landing in one changes nothing that is drawn.
 */
export const AFace: Story = {
  play: async ({ canvasElement }) => {
    const body = found(canvasElement, '.face__body')
    expect(getComputedStyle(body).gridTemplateColumns.split(' ')).toHaveLength(2)

    const getPaneBox = (pane: string): DOMRect =>
      getPane(canvasElement, pane).getBoundingClientRect()

    // The writing is stored alongside its preview, on one line.
    expect(Math.round(getPaneBox('front-written').top)).toBe(
      Math.round(getPaneBox('front-preview').top),
    )
    expect(getPaneBox('front-written').right).toBeLessThanOrEqual(getPaneBox('front-preview').left)

    // The front stands above the back, and both halves are cut the same way.
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

    // A part is a box to write in, not a line: the box fills the part it stands
    // in, so every point of the part is a point to type at.
    const part = getPane(canvasElement, 'front-written')
    const box = getBox(canvasElement, 'front')
    const deep = Number.parseFloat(getComputedStyle(box).lineHeight)
    expect(box.getBoundingClientRect().height).toBeGreaterThan(deep * 4)
    expect(box.getBoundingClientRect().height).toBeCloseTo(part.getBoundingClientRect().height, 0)

    // The face is framed and the parts inside it are not.
    const face = found(canvasElement, '[data-face]')
    expect(Number.parseFloat(getComputedStyle(face).borderTopWidth)).toBeGreaterThan(0)
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

    // The preview stands what a card fills the slots with, and no brace of the
    // markup it came from.
    const preview = found(canvasElement, '[data-preview="back"]')
    expect(preview.textContent).toContain('about 45" at the shoulder')
    expect(preview.textContent).not.toContain('{{')
    expect(preview.querySelector('b')?.textContent).toBe('Height:')

    // A face is HTML, so each of its three bare lines reads as a line of its
    // own: three lines of type, and not one line running into the next.
    const answer = found(canvasElement, '[data-preview="back"] .prose')
    const tall = Number.parseFloat(getComputedStyle(answer).lineHeight)
    expect(tall).toBeGreaterThan(0)
    expect(answer.getBoundingClientRect().height).toBeCloseTo(tall * 3, 0)

    // A field to write into a half is a small quiet chip with a ground of its
    const chip = found(canvasElement, '[data-insert="Height"]')
    const title = found(canvasElement, '.face__title')
    expect(getComputedStyle(chip).backgroundColor).not.toBe('rgba(0, 0, 0, 0)')
    expect(Number.parseFloat(getComputedStyle(chip).fontSize)).toBeLessThan(
      Number.parseFloat(getComputedStyle(title).fontSize),
    )

    // A field is written into the half last typed in, where the caret stands.
    const back = getBox(canvasElement, 'back')
    back.focus()
    back.setSelectionRange(0, 0)
    await userEvent.click(chip)
    await waitFor(() =>
      expect(getBox(canvasElement, 'back').value.startsWith('{{Height}}')).toBe(true),
    )

    // The keyboard landing in a box changes the face, the strip and the parts
    // in nothing.
    const watched = [
      face,
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
    const body = found(canvasElement, '.face__body')
    expect(getComputedStyle(body).gridTemplateColumns.split(' ')).toHaveLength(1)

    const tops = PANES.map((pane) =>
      Math.round(getPane(canvasElement, pane).getBoundingClientRect().top),
    )
    expect([...tops].sort((first, second) => first - second)).toEqual(tops)
    expect(new Set(tops).size).toBe(4)
  },
}

/** A face naming slots the stencil does not declare, in both halves. */
export const AStraySlot: Story = {
  args: { corpus: 'a stray slot' },
  play: async ({ canvasElement }) => {
    // What is wrong is said where the slot itself stands, and under that half
    // alone.
    expect(found(canvasElement, '[data-preview="front"] mark').textContent).toBe('{{Colour}}')
    expect(found(canvasElement, '[data-preview="back"] mark').textContent).toBe('{{Weight}}')
    expect(
      found(canvasElement, '[data-pane="front-written"] .face__objections').textContent?.trim(),
    ).toBe('Not a field: Colour')
    expect(
      found(canvasElement, '[data-pane="back-written"] .face__objections').textContent?.trim(),
    ).toBe('Not a field: Weight')
    expect(canvasElement.querySelector('header .face__objections')).toBeNull()

    // still fills the part, and a press meant for the box reaches it.
    const pane = getPane(canvasElement, 'front-written').getBoundingClientRect()
    const box = getBox(canvasElement, 'front').getBoundingClientRect()
    expect(box.height).toBeCloseTo(pane.height, 0)

    const layer = found(canvasElement, '[data-pane="front-written"] .face__amiss')
    expect(getComputedStyle(layer).pointerEvents).toBe('none')

    const response = found(canvasElement, '[data-pane="front-written"] .face__objections')
    const overRect = response.getBoundingClientRect()
    expect(overRect.bottom).toBeLessThanOrEqual(pane.bottom + 1)
    expect(overRect.right).toBeLessThanOrEqual(pane.right + 1)
    expect(getComputedStyle(response).backgroundColor).not.toBe('rgba(0, 0, 0, 0)')
  },
}

/** A face with nothing in either half, and no name either. */
export const NothingInIt: Story = {
  args: { corpus: 'nothing in it' },
  play: async ({ canvasElement }) => {
    // An empty part says what it is for, in the middle of itself.
    const placeholders: string[] = []
    for (const pane of PANES) {
      const element = getPane(canvasElement, pane)
      const ghost = element.querySelector<HTMLElement>('.face__ghost')
      if (!ghost) throw new Error('face not found')
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
      expect(getPane(canvasElement, 'front-written').querySelector('.face__ghost')).toBeNull()
      expect(getPane(canvasElement, 'front-preview').querySelector('.face__ghost')).toBeNull()
    })
    const pane = getPane(canvasElement, 'front-written')
    expect(content.getBoundingClientRect().top).toBeCloseTo(pane.getBoundingClientRect().top, 0)

    // The half nothing was written in still says what it is for.
    expect(getPane(canvasElement, 'back-written').querySelector('.face__ghost')?.textContent).toBe(
      'Back',
    )
  },
}

/**
 * A card is HTML: a tag a person wrote is drawn, and the marks of another
 * format stand as the characters they are.
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

    // What the person wrote is theirs to read and change, standing as the text
    // of a box.
    expect(getBox(canvasElement, 'back').value).toContain('<script>')
  },
}

/**
 * Forty fields to write in, far more than the strip has room for. The row of
 * fields scrolls inside the strip, which keeps its height and its heading.
 */
export const FarTooManyFields: Story = {
  args: { corpus: 'far too many fields' },
  play: async ({ canvasElement }) => {
    const header = found(canvasElement, '.card-header')
    const title = found(canvasElement, '.face__title')
    const slots = found(canvasElement, '.face__slots')

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
 * What the vault found wrong with the face, and a name another face carries.
 *
 * What stands against the face is said at the end of the strip, over the window
 * under it, and what is wrong with a name being typed is said in the same
 * place, to the box.
 */
export const WhatIsWrongWithIt: Story = {
  args: { corpus: 'something wrong with it' },
  play: async ({ canvasElement }) => {
    const response = found(canvasElement, '[data-wrong]')
    expect(response.textContent).toContain('this face has no back')
    expect(response.querySelectorAll('li')).toHaveLength(2)

    // The strip is one row deep with all items rendered, and the window begins
    // where the strip ends: what is wrong hangs over the window and takes no
    // room from it.
    const header = found(canvasElement, '.card-header').getBoundingClientRect()
    const head = found(canvasElement, '.face__head').getBoundingClientRect()
    const body = found(canvasElement, '.face__body').getBoundingClientRect()
    const overRect = response.getBoundingClientRect()
    expect(header.height).toBeLessThan(40)
    expect(body.top).toBeCloseTo(header.bottom, 0)
    expect(overRect.top).toBeGreaterThanOrEqual(head.bottom - 1)
    expect(overRect.bottom).toBeGreaterThan(body.top)

    // It stays inside the face, and a press meant for the window reaches it.
    const face = found(canvasElement, '[data-face]').getBoundingClientRect()
    expect(overRect.right).toBeLessThanOrEqual(face.right + 1)
    expect(getComputedStyle(found(canvasElement, 'header .face__amiss')).pointerEvents).toBe('none')
    expect(getComputedStyle(response).backgroundColor).not.toBe('rgba(0, 0, 0, 0)')

    // A name another face carries cannot be used, and the box says so.
    const name = found(canvasElement, '.face__title') as HTMLInputElement
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
 * On the dark set of tokens, where a field's chip takes its hover from the
 * face's own ink mixed into the chip's ground.
 *
 * The mix is one expression for both sets, so on the dark set it has to move
 * the chip towards the ink, which is lighter there than the ground under it.
 */
export const Dark: Story = {
  globals: DARK,
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    const chip = within(canvasElement).getByRole('button', { name: 'Insert: Height' })

    // The ink the chip is read by stands above the ground it relies on, which
    // is the way round the dark set is written.
    const ink = lightness(getComputedStyle(chip).color)
    const restingLightness = lightness(getComputedStyle(chip).backgroundColor)
    expect(ink).toBeGreaterThan(restingLightness)

    await hoverOver(chip)
    await waitFor(() =>
      expect(lightness(getComputedStyle(chip).backgroundColor)).toBeGreaterThan(restingLightness + 2),
    )

    // The chip is still a chip under the hand: lighter than it was, and still
    // darker than what is written on it.
    expect(lightness(getComputedStyle(chip).backgroundColor)).toBeLessThan(ink)
  },
}
