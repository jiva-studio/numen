/**
 * Every situation the stencil editor has to survive. Also the test corpus:
 * each story is run in a browser by `@storybook/addon-vitest`.
 *
 * What the editor hands back is applied here, which is the application's part.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor } from 'storybook/test'
import { ref, watch } from 'vue'
import Stencil from './Stencil.vue'
import { ordered, reordered, type Half, type Landing, type Shown } from './model'
import { renamedIn } from './fill'

interface Corpus {
  readonly fields: readonly string[]
  readonly faces: readonly Shown[]
}

const UNBROKEN =
  'supercalifragilisticexpialidociousandthensomemoreofitwithnothingtobreakatanywhere'

const many = (count: number): readonly string[] =>
  Array.from({ length: count }, (_, at) => `Field ${at + 1}`)

const CORPORA = {
  'a stencil': {
    fields: ['Name', 'Height', 'Weight', 'Life span'],
    faces: [
      {
        id: 'recognise',
        name: 'Recognise',
        front: '{{Name}}',
        back: '**Height:** {{Height}}\n\n**Weight:** {{Weight}}\n\n**Life span:** {{Life span}}',
      },
      {
        id: 'name-it',
        name: 'Name it',
        front: 'Which animal stands {{Height}} and lives {{Life span}}?',
        back: '{{Name}}',
      },
    ],
  },
  one: {
    fields: ['Word'],
    faces: [{ id: 'only', name: 'Only', front: '{{Word}}', back: '' }],
  },
  'far too many': {
    fields: many(40),
    faces: Array.from({ length: 12 }, (_, at) => ({
      id: `face-${at}`,
      name: `Face ${at + 1}`,
      front: '{{Field 1}}',
      back: many(40)
        .map((field) => `**${field}:** {{${field}}}`)
        .join('\n\n'),
    })),
  },
  'other scripts': {
    fields: ['Слово', 'Перевод', 'Пример', 'देवनागरी'],
    faces: [
      {
        id: 'слово',
        name: 'Слово',
        front: '{{Слово}}',
        back: '**Перевод:** {{Перевод}}\n\n**Пример:** {{Пример}}\n\n{{देवनागरी}}',
      },
    ],
  },
  unbroken: {
    fields: [UNBROKEN, 'A field name that goes on and on, well past the width anything drawing it has'],
    faces: [
      {
        id: 'run-on',
        name: UNBROKEN,
        front: `{{${UNBROKEN}}}`,
        back: `${UNBROKEN} {{${UNBROKEN}}} ${UNBROKEN}`,
      },
    ],
  },
  'nothing at all': { fields: [], faces: [] },
  'no text at all': {
    fields: ['Blank'],
    faces: [{ id: 'blank', name: '', front: '', back: '' }],
  },
  stray: {
    fields: ['Name', 'Height'],
    faces: [
      {
        id: 'stray',
        name: 'Stray',
        front: '{{Name}} the {{Colour}} one',
        back: '**Height:** {{Height}}\n\n**Weight:** {{Weight}}',
      },
    ],
  },
  tags: {
    fields: ['Question', 'Answer'],
    faces: [
      {
        id: 'asked',
        name: 'Asked',
        front: '<b>{{Question}}</b>',
        back: '<span style="color: teal">{{Answer}}</span>\n\n<script>window.stolen = 1</script>',
      },
    ],
  },
} satisfies Record<string, Corpus>

type Corpora = keyof typeof CORPORA

interface Knobs {
  corpus: Corpora
  name: string
  /** How wide the window drawing the stencil is. */
  width: string
  /** Given by the story, and nothing a reader turns. */
  fields?: never
  faces?: never
  sample?: never
  words?: never
}

const meta: Meta<Knobs> = {
  title: 'Cards/Stencil',
  component: Stencil,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    corpus: {
      control: 'select',
      options: Object.keys(CORPORA),
      description: 'What the stencil starts as. Changing it starts afresh.',
    },
    name: { control: 'text' },
    width: { control: 'text' },
    fields: { table: { disable: true } },
    faces: { table: { disable: true } },
    sample: { table: { disable: true } },
    words: { table: { disable: true } },
  },
  args: { corpus: 'a stencil', name: 'Stencil', width: '100%' },
  render: (args) => ({
    components: { Stencil },
    setup() {
      const fields = ref<readonly string[]>(CORPORA[args.corpus].fields)
      const faces = ref<readonly Shown[]>(CORPORA[args.corpus].faces)

      watch(
        () => args.corpus,
        (next) => {
          fields.value = CORPORA[next].fields
          faces.value = CORPORA[next].faces
        },
      )

      /** Every face, with one half of one of them rewritten. */
      const written = (id: string, half: Half, text: string): readonly Shown[] =>
        faces.value.map((face) => (face.id === id ? { ...face, [half]: text } : face))

      return {
        args,
        fields,
        faces,
        onAddField: (name: string) => {
          fields.value = [...fields.value, name]
        },
        onRenameField: (field: string, name: string) => {
          fields.value = fields.value.map((each) => (each === field ? name : each))
          faces.value = faces.value.map((face) => ({
            ...face,
            front: renamedIn(face.front, field, name),
            back: renamedIn(face.back, field, name),
          }))
        },
        onRemoveField: (field: string) => {
          fields.value = fields.value.filter((each) => each !== field)
        },
        onMoveField: (field: string, at: Landing) => {
          fields.value = reordered(fields.value, field, at)
        },
        onAddFace: (name: string) => {
          faces.value = [...faces.value, { id: `face-${faces.value.length}`, name, front: '', back: '' }]
        },
        onRenameFace: (id: string, name: string) => {
          faces.value = faces.value.map((face) => (face.id === id ? { ...face, name } : face))
        },
        onRemoveFace: (id: string) => {
          faces.value = faces.value.filter((face) => face.id !== id)
        },
        /* Nothing among the faces is fixed, so any of them lands anywhere. */
        onMoveFace: (id: string, at: Landing) => {
          const order = ordered(faces.value.map((face) => face.id), id, at)
          faces.value = order.flatMap((each) => faces.value.filter((face) => face.id === each))
        },
        onWrite: (id: string, half: Half, text: string) => {
          faces.value = written(id, half, text)
        },
      }
    },
    template: `
      <div :style="{ height: '100vh', width: args.width }">
        <Stencil
          :fields="fields"
          :faces="faces"
          :name="args.name"
          @add-field="onAddField"
          @rename-field="onRenameField"
          @remove-field="onRemoveField"
          @move-field="onMoveField"
          @add-face="onAddFace"
          @rename-face="onRenameFace"
          @remove-face="onRemoveFace"
          @move-face="onMoveFace"
          @write="onWrite"
        />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

/** Three fields and two faces, one of them asking the other way round. */
export const AStencil: Story = {}

/** One field and one face, and nothing to reorder. */
export const One: Story = { args: { corpus: 'one' } }

/** Forty fields and twelve faces, far more than the window has room for. */
export const FarTooMany: Story = {
  args: { corpus: 'far too many' },
  play: async ({ canvasElement }) => {
    const bar = found(canvasElement, '[data-face-block="face-0"] .bar')
    const title = found(canvasElement, '[data-face-block="face-0"] .stencil__title')
    const slots = found(canvasElement, '[data-face-block="face-0"] .stencil__slots')

    // Forty fields do not push the heading out of its own strip.
    expect(title.getBoundingClientRect().width).toBeGreaterThan(60)

    // The row scrolls inside the strip instead of growing it.
    expect(slots.scrollWidth).toBeGreaterThan(slots.clientWidth)
    expect(bar.getBoundingClientRect().height).toBeLessThan(40)
    expect(slots.getBoundingClientRect().right).toBeLessThanOrEqual(
      bar.getBoundingClientRect().right + 1,
    )
  },
}

/** Names and prose that are not Latin. */
export const OtherScripts: Story = { args: { corpus: 'other scripts' } }

/** A name far too long, and one with nothing in it to break at. */
export const Unbroken: Story = { args: { corpus: 'unbroken' } }

/** No fields and no faces. */
export const NothingAtAll: Story = { args: { corpus: 'nothing at all' } }

/** A face whose name and both halves are the empty string. */
export const NoTextAtAll: Story = { args: { corpus: 'no text at all' } }

/** A face naming a slot the fields do not, in each half. */
export const Stray: Story = { args: { corpus: 'stray' } }

/** Tags a person wrote among the marks, and a script that does not survive. */
export const Tags: Story = { args: { corpus: 'tags' } }

const boxFor = (canvas: HTMLElement, id: string, half: Half) => {
  const box = canvas.querySelector<HTMLTextAreaElement>(
    `[data-face="${id}"][data-half="${half}"]`,
  )
  if (!box) throw new Error(`no ${half} of ${id}`)
  return box
}

const found = (canvas: HTMLElement, selector: string): HTMLElement => {
  const held = canvas.querySelector<HTMLElement>(selector)
  if (!held) throw new Error(`nothing matching ${selector}`)
  return held
}

const buttonSaying = (canvas: HTMLElement, said: string) => {
  const held = [...canvas.querySelectorAll<HTMLElement>('button')].find(
    (each) => each.textContent?.trim() === said,
  )
  if (!held) throw new Error(`no button saying ${said}`)
  return held
}

const drawnFields = (canvas: HTMLElement): readonly (string | null)[] =>
  [...canvas.querySelectorAll('[data-field]')].map((row) => row.getAttribute('data-field'))

/** One row of fields for the face, written into the half last typed in. */
export const WritesAFieldIn: Story = {
  play: async ({ canvasElement }) => {
    // Nothing has been typed in, so the front is what the row is aimed at.
    const front = boxFor(canvasElement, 'recognise', 'front')
    front.focus()
    front.setSelectionRange(front.value.length, front.value.length)

    const rows = canvasElement.querySelectorAll('[data-face-block="recognise"] .stencil__slots')
    expect(rows).toHaveLength(1)

    await userEvent.click(found(canvasElement, '[data-face-block="recognise"] [data-insert="Weight"]'))
    expect(boxFor(canvasElement, 'recognise', 'front').value).toBe('{{Name}}{{Weight}}')

    // Typing in the back aims the same row at it.
    const back = boxFor(canvasElement, 'recognise', 'back')
    back.focus()
    back.setSelectionRange(0, 0)

    await userEvent.click(found(canvasElement, '[data-face-block="recognise"] [data-insert="Name"]'))
    expect(boxFor(canvasElement, 'recognise', 'back').value.startsWith('{{Name}}')).toBe(true)

    const preview = canvasElement.querySelector('[data-preview="front"]')
    expect(preview?.textContent).toContain('Weight')
    expect(preview?.textContent).not.toContain('{{')
  },
}

/** A field asked for, standing under a name nothing had taken, below the first. */
export const AsksForAField: Story = {
  play: async ({ canvasElement }) => {
    await userEvent.click(buttonSaying(canvasElement, 'Add a field'))
    expect(drawnFields(canvasElement)).toEqual([
      'Name',
      'Height',
      'Weight',
      'Life span',
      'Field 1',
    ])
  },
}

/**
 * The faces are carried by their strips and change places. Their order is the
 * order a card's repetitions are taken from it, so nothing among them is fixed.
 */
export const FacesChangePlaces: Story = {
  play: async ({ canvasElement }) => {
    const drawn = (): readonly (string | null)[] =>
      [...canvasElement.querySelectorAll('[data-face-block]')].map((each) =>
        each.getAttribute('data-face-block'),
      )
    expect(drawn()).toEqual(['recognise', 'name-it'])

    const bar = found(canvasElement, '[data-face-block="name-it"] .bar')
    expect(bar.getAttribute('draggable')).toBe('true')

    // The last face is let go over the first, and takes its place.
    bar.dispatchEvent(new DragEvent('dragstart', { bubbles: true }))
    const onto = found(canvasElement, '[data-face-block="recognise"]')
    onto.dispatchEvent(new DragEvent('dragover', { bubbles: true, cancelable: true }))
    onto.dispatchEvent(new DragEvent('drop', { bubbles: true }))

    await waitFor(() => expect(drawn()).toEqual(['name-it', 'recognise']))
  },
}

/**
 * A field's row is one block: the line and the ground are the row's, and the
 * handle, the box and the way to remove it stand inside it carrying neither.
 */
export const ARowIsOneBlock: Story = {
  play: async ({ canvasElement }) => {
    const row = found(canvasElement, '[data-field="Height"] .stencil__row')
    const box = row.querySelector('input')
    const grip = row.querySelector('[data-grip]')
    const away = row.querySelector('button')
    if (!box || !grip || !away) throw new Error('a row missing part of itself')

    // The row carries the line and the ground.
    const drawn = getComputedStyle(row)
    expect(Number.parseFloat(drawn.borderTopWidth)).toBeGreaterThan(0)
    expect(drawn.backgroundColor).not.toBe('rgba(0, 0, 0, 0)')

    // What stands inside carries neither.
    const inside = getComputedStyle(box)
    expect(Number.parseFloat(inside.borderTopWidth)).toBe(0)
    expect(inside.backgroundColor).toBe('rgba(0, 0, 0, 0)')

    // And all three stand within the row's own bounds, not beside it.
    const bounds = row.getBoundingClientRect()
    for (const part of [grip, box, away]) {
      const at = part.getBoundingClientRect()
      expect(at.left).toBeGreaterThanOrEqual(bounds.left - 1)
      expect(at.right).toBeLessThanOrEqual(bounds.right + 1)
    }
  },
}

/**
 * A field to write in is a small quiet chip, drawn as small quiet actions are
 * drawn everywhere else here: it has a ground of its own at rest, so it reads
 * as something to press without being pointed at.
 */
export const TheFieldsReadAsSomethingToPress: Story = {
  play: async ({ canvasElement }) => {
    const chip = found(canvasElement, '[data-face-block="recognise"] [data-insert="Height"]')
    const title = found(canvasElement, '[data-face-block="recognise"] .stencil__title')

    // It has a ground at rest, and is not merely a word in a line.
    expect(getComputedStyle(chip).backgroundColor).not.toBe('rgba(0, 0, 0, 0)')

    // It is markedly smaller than the heading whose strip it stands in.
    const said = Number.parseFloat(getComputedStyle(chip).fontSize)
    const heading = Number.parseFloat(getComputedStyle(title).fontSize)
    expect(said).toBeLessThan(heading)
  },
}

/** The first field names every card, so it has no handle and no way to go. */
export const TheFirstFieldStaysFirst: Story = {
  play: async ({ canvasElement }) => {
    const first = found(canvasElement, '[data-field="Name"]')
    expect(first.getAttribute('data-names')).toBe('true')

    // The row keeps the shape every other row has; what it may not do is
    // turned off rather than missing, so the boxes line up down the column.
    expect(first.querySelector('[data-grip]')?.getAttribute('draggable')).toBe('false')
    expect(first.querySelector('button')?.disabled).toBe(true)

    const second = found(canvasElement, '[data-field="Height"]')
    expect(second.querySelector('[data-grip]')?.getAttribute('draggable')).toBe('true')
    expect(second.querySelector('button')?.disabled).toBe(false)

    const edge = (row: HTMLElement): number =>
      row.querySelector('input')?.getBoundingClientRect().left ?? 0
    expect(Math.round(edge(first))).toBe(Math.round(edge(second)))

    // Letting a field go on the first row lands it nowhere.
    second.querySelector('[data-grip]')?.dispatchEvent(new DragEvent('dragstart', { bubbles: true }))
    first.dispatchEvent(new DragEvent('dragover', { bubbles: true, cancelable: true }))
    first.dispatchEvent(new DragEvent('drop', { bubbles: true }))

    // A reordering would have been drawn by now.
    await new Promise((settled) => {
      setTimeout(settled, 0)
    })
    expect(drawnFields(canvasElement)).toEqual(['Name', 'Height', 'Weight', 'Life span'])
  },
}

const paneOf = (canvas: HTMLElement, id: string, pane: string): HTMLElement =>
  found(canvas, `[data-face-block="${id}"] [data-pane="${pane}"]`)

/** The four parts of a face, in the order they are drawn. */
const PANES = ['front-written', 'front-preview', 'back-written', 'back-preview'] as const

/**
 * The window of a face is one window: two parts to a row, the writing beside
 * its preview and the front above the back, divided by the lines they share.
 */
export const TwoPartsToARow: Story = {
  args: { width: '64rem' },
  play: async ({ canvasElement }) => {
    const body = found(canvasElement, '[data-face-block="recognise"] .stencil__face-body')
    expect(getComputedStyle(body).gridTemplateColumns.split(' ')).toHaveLength(2)

    const at = (pane: string): DOMRect =>
      paneOf(canvasElement, 'recognise', pane).getBoundingClientRect()

    // The writing stands beside its preview, on one line.
    expect(Math.round(at('front-written').top)).toBe(Math.round(at('front-preview').top))
    expect(at('front-written').right).toBeLessThanOrEqual(at('front-preview').left)

    // The front stands above the back, and both halves are cut the same way.
    expect(at('front-written').bottom).toBeLessThanOrEqual(at('back-written').top)
    expect(Math.round(at('back-written').top)).toBe(Math.round(at('back-preview').top))

    // What divides them is one line, shared by the two it divides.
    const line = Number.parseFloat(getComputedStyle(body).columnGap)
    expect(line).toBeGreaterThan(0)
    expect(at('front-preview').left - at('front-written').right).toBeCloseTo(line, 0)
    expect(at('back-written').top - at('front-written').bottom).toBeCloseTo(line, 0)
  },
}

/**
 * The parts carry no outline of their own: the frame around all four is the
 * block's, and nothing is drawn twice where two of them meet.
 */
export const ThePartsCarryNoOutline: Story = {
  args: { width: '64rem' },
  play: async ({ canvasElement }) => {
    const block = found(canvasElement, '[data-face-block="recognise"]')
    expect(Number.parseFloat(getComputedStyle(block).borderTopWidth)).toBeGreaterThan(0)

    for (const pane of PANES) {
      const drawn = getComputedStyle(paneOf(canvasElement, 'recognise', pane))
      for (const side of ['borderTopWidth', 'borderInlineStartWidth', 'borderBottomWidth', 'borderInlineEndWidth'] as const) {
        expect(Number.parseFloat(drawn[side])).toBe(0)
      }
      expect(drawn.outlineStyle).toBe('none')
    }

    // The notch a name used to sit in is nowhere in the window.
    const body = found(canvasElement, '[data-face-block="recognise"] .stencil__face-body')
    expect(body.querySelectorAll('fieldset, legend')).toHaveLength(0)
  },
}

/** A window too narrow for two parts to a row: one column, each preview under its half. */
export const Narrow: Story = {
  args: { width: '22rem' },
  play: async ({ canvasElement }) => {
    const editor = found(canvasElement, '.stencil')
    const body = found(canvasElement, '[data-face-block="recognise"] .stencil__face-body')
    expect(getComputedStyle(body).gridTemplateColumns.split(' ')).toHaveLength(1)

    const tops = PANES.map((pane) =>
      Math.round(paneOf(canvasElement, 'recognise', pane).getBoundingClientRect().top),
    )
    expect([...tops].sort((first, second) => first - second)).toEqual(tops)
    expect(new Set(tops).size).toBe(4)

    expect(editor.scrollWidth).toBeLessThanOrEqual(editor.clientWidth + 1)
  },
}

/** An empty part says what it is for, in the middle of itself. */
export const AnEmptyPartSaysWhatItIsFor: Story = {
  args: { corpus: 'no text at all', width: '64rem' },
  play: async ({ canvasElement }) => {
    const said: string[] = []
    for (const pane of PANES) {
      const held = paneOf(canvasElement, 'blank', pane)
      const ghost = held.querySelector<HTMLElement>('.stencil__ghost')
      if (!ghost) throw new Error(`nothing said in ${pane}`)
      said.push(ghost.textContent?.trim() ?? '')

      // It stands in the middle of the part, in both directions.
      const box = held.getBoundingClientRect()
      const at = ghost.getBoundingClientRect()
      expect(at.left + at.width / 2).toBeCloseTo(box.left + box.width / 2, 0)
      expect(at.top + at.height / 2).toBeCloseTo(box.top + box.height / 2, 0)
    }
    expect(said).toEqual(['Front', 'Preview', 'Back', 'Preview'])
  },
}

/**
 * A part holding something is drawn as the box alone: what the part is called
 * is what an empty one says in its middle, and nowhere else. Where a part
 * stands in the window is what says which half it is of.
 */
export const NothingStandsOverAFullBox: Story = {
  args: { width: '64rem' },
  play: async ({ canvasElement }) => {
    for (const half of ['front', 'back'] as const) {
      const pane = paneOf(canvasElement, 'recognise', `${half}-written`)
      const box = boxFor(canvasElement, 'recognise', half)
      expect(box.value.trim()).not.toBe('')

      // The box begins where the part does: nothing is drawn above it.
      expect(box.getBoundingClientRect().top - Number.parseFloat(getComputedStyle(box).paddingTop))
        .toBeCloseTo(pane.getBoundingClientRect().top, 0)

      // And the part draws no word of its own, in the middle or anywhere else.
      expect(pane.textContent?.trim()).toBe('')
      expect(pane.querySelector('.stencil__ghost')).toBeNull()
    }
  },
}

/** What stands in a part takes the place of what the part says while it is empty. */
export const WritingInAPartSilencesIt: Story = {
  args: { corpus: 'no text at all', width: '64rem' },
  play: async ({ canvasElement }) => {
    const box = boxFor(canvasElement, 'blank', 'front')
    await userEvent.type(box, 'a word')

    await waitFor(() => {
      expect(paneOf(canvasElement, 'blank', 'front-written').querySelector('.stencil__ghost'))
        .toBeNull()
      expect(paneOf(canvasElement, 'blank', 'front-preview').querySelector('.stencil__ghost'))
        .toBeNull()
    })

    // The half nothing was written in still says what it is for.
    expect(
      paneOf(canvasElement, 'blank', 'back-written').querySelector('.stencil__ghost')?.textContent,
    ).toBe('Back')
  },
}

/**
 * What a list is added to by stands in the middle of a rule that runs the whole
 * width of the list, and the line gives way to it.
 */
export const TheAddingStandsOnARule: Story = {
  args: { width: '64rem' },
  play: async ({ canvasElement }) => {
    const rule = found(canvasElement, '.stencil__part .rule')
    const list = found(canvasElement, '.stencil__fields')
    const button = found(canvasElement, '.stencil__part .rule button')

    // The line runs as wide as the rows above it.
    const along = rule.getBoundingClientRect()
    expect(Math.round(along.width)).toBe(Math.round(list.getBoundingClientRect().width))

    // What it holds stands in the middle, with the line either side of it.
    const at = button.getBoundingClientRect()
    expect(at.left + at.width / 2).toBeCloseTo(along.left + along.width / 2, 0)
    for (const side of ['::before', '::after']) {
      expect(Number.parseFloat(getComputedStyle(rule, side).width)).toBeGreaterThan(0)
    }

    // Both ways of adding are drawn the same way.
    expect(canvasElement.querySelectorAll('.rule')).toHaveLength(2)
  },
}

/** The keyboard landing in a box changes the block, the strip and the parts in nothing. */
export const FocusDrawsNothingAroundIt: Story = {
  args: { width: '64rem' },
  play: async ({ canvasElement }) => {
    const watched = [
      found(canvasElement, '[data-face-block="recognise"]'),
      found(canvasElement, '[data-face-block="recognise"] .bar'),
      ...PANES.map((pane) => paneOf(canvasElement, 'recognise', pane)),
    ]
    const look = (): readonly string[] =>
      watched.map((each) => {
        const drawn = getComputedStyle(each)
        return [
          drawn.backgroundColor,
          drawn.borderColor,
          drawn.borderWidth,
          drawn.boxShadow,
          drawn.outlineStyle,
          drawn.outlineWidth,
        ].join('|')
      })

    const before = look()
    const box = boxFor(canvasElement, 'recognise', 'back')
    box.focus()
    expect(document.activeElement).toBe(box)

    expect(look()).toEqual(before)
  },
}

/** What is wrong is said where the slot itself stands, and under that half alone. */
export const SaysAStraySlotInItsPlace: Story = {
  args: { corpus: 'stray' },
  play: async ({ canvasElement }) => {
    const block = found(canvasElement, '[data-face-block="stray"]')

    expect(block.querySelector('[data-preview="front"] mark')?.textContent).toBe('{{Colour}}')
    expect(block.querySelector('[data-preview="back"] mark')?.textContent).toBe('{{Weight}}')

    expect(
      block.querySelector('[data-pane="front-written"] .stencil__objects')?.textContent?.trim(),
    ).toBe('Not a field: Colour')
    expect(
      block.querySelector('[data-pane="back-written"] .stencil__objects')?.textContent?.trim(),
    ).toBe('Not a field: Weight')

    // Nothing is said about it under the face's name.
    expect(block.querySelector('header .stencil__objects')).toBeNull()
  },
}
