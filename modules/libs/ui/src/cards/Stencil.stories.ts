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
  /* Names and prose that are not Latin, beside a name with nothing in it to
     break at. */
  'awkward text': {
    fields: [
      'Слово',
      'Перевод',
      'Пример',
      'देवनागरी',
      UNBROKEN,
      'A field name that goes on and on, well past the width anything drawing it has',
    ],
    faces: [
      {
        id: 'слово',
        name: 'Слово',
        front: '{{Слово}}',
        back: '**Перевод:** {{Перевод}}\n\n**Пример:** {{Пример}}\n\n{{देवनागरी}}',
      },
      {
        id: 'run-on',
        name: UNBROKEN,
        front: `{{${UNBROKEN}}}`,
        back: `${UNBROKEN} {{${UNBROKEN}}} ${UNBROKEN}`,
      },
    ],
  },
  'nothing at all': { fields: [], faces: [] },
  /* Everything a file written by hand can hold that the editor did not make:
     one name declared twice, where the first stands and one row is drawn for
     it; a face naming slots the fields do not; marks a person wrote; and a
     face with nothing in it at all. */
  'what is wrong': {
    fields: ['Name', 'Height', 'Height'],
    faces: [
      {
        id: 'stray',
        name: 'Stray',
        front: '{{Name}} the {{Colour}} one',
        back: '**Height:** {{Height}}\n\n**Weight:** {{Weight}}',
      },
      {
        id: 'tagged',
        name: 'Tagged',
        front: '<b>{{Name}}</b>',
        back: '<span style="color: teal">{{Height}}</span>\n\n<script>window.stolen = 1</script>',
      },
      { id: 'blank', name: '', front: '', back: '' },
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
  title: 'Flash Cards/Stencil',
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

const paneOf = (canvas: HTMLElement, id: string, pane: string): HTMLElement =>
  found(canvas, `[data-face-block="${id}"] [data-pane="${pane}"]`)

/** The four parts of a face, in the order they are drawn. */
const PANES = ['front-written', 'front-preview', 'back-written', 'back-preview'] as const

/**
 * Four fields and two faces, one of them asking the other way round.
 *
 * A field's row is one block: the line and the ground are the row's, and the
 * handle, the box and the way to remove it stand inside it carrying neither.
 * The first field names every card, so it keeps the shape the others have with
 * what it may not do turned off. What the list is added to by stands in the
 * middle of a rule as wide as the list.
 */
export const AStencil: Story = {
  args: { width: '64rem' },
  play: async ({ canvasElement }) => {
    const row = found(canvasElement, '[data-field="Height"] .stencil__row')
    const box = row.querySelector('input')
    const grip = row.querySelector('[data-grip]')
    const away = row.querySelector('button')
    if (!box || !grip || !away) throw new Error('a row missing part of itself')

    // The row carries the line and the ground, and what stands inside carries
    // neither.
    const drawn = getComputedStyle(row)
    expect(Number.parseFloat(drawn.borderTopWidth)).toBeGreaterThan(0)
    expect(drawn.backgroundColor).not.toBe('rgba(0, 0, 0, 0)')
    const inside = getComputedStyle(box)
    expect(Number.parseFloat(inside.borderTopWidth)).toBe(0)
    expect(inside.backgroundColor).toBe('rgba(0, 0, 0, 0)')

    // And all three stand within the row's own bounds.
    const bounds = row.getBoundingClientRect()
    for (const part of [grip, box, away]) {
      const at = part.getBoundingClientRect()
      expect(at.left).toBeGreaterThanOrEqual(bounds.left - 1)
      expect(at.right).toBeLessThanOrEqual(bounds.right + 1)
    }

    // A field to write into a face is a small quiet chip with a ground of its
    // own at rest, markedly smaller than the heading whose strip it stands in.
    const chip = found(canvasElement, '[data-face-block="recognise"] [data-insert="Height"]')
    const title = found(canvasElement, '[data-face-block="recognise"] .stencil__title')
    expect(getComputedStyle(chip).backgroundColor).not.toBe('rgba(0, 0, 0, 0)')
    expect(Number.parseFloat(getComputedStyle(chip).fontSize)).toBeLessThan(
      Number.parseFloat(getComputedStyle(title).fontSize),
    )

    // The first field stays first, and its row lines up with every other.
    const first = found(canvasElement, '[data-field="Name"]')
    const second = found(canvasElement, '[data-field="Height"]')
    expect(first.getAttribute('data-names')).toBe('true')
    expect(first.querySelector('[data-grip]')?.getAttribute('draggable')).toBe('false')
    expect(first.querySelector('button')?.disabled).toBe(true)
    expect(second.querySelector('[data-grip]')?.getAttribute('draggable')).toBe('true')
    expect(second.querySelector('button')?.disabled).toBe(false)
    const edge = (each: HTMLElement): number =>
      each.querySelector('input')?.getBoundingClientRect().left ?? 0
    expect(Math.round(edge(first))).toBe(Math.round(edge(second)))

    // Letting a field go on the first row lands it nowhere.
    second.querySelector('[data-grip]')?.dispatchEvent(new DragEvent('dragstart', { bubbles: true }))
    first.dispatchEvent(new DragEvent('dragover', { bubbles: true, cancelable: true }))
    first.dispatchEvent(new DragEvent('drop', { bubbles: true }))
    await new Promise((settled) => {
      setTimeout(settled, 0)
    })
    expect(drawnFields(canvasElement)).toEqual(['Name', 'Height', 'Weight', 'Life span'])

    // What the list is added to by stands in the middle of a rule as wide as
    // the list, and the line gives way to it either side.
    const rule = found(canvasElement, '.stencil__part .rule')
    const list = found(canvasElement, '.stencil__fields')
    const adding = found(canvasElement, '.stencil__part .rule button')
    const along = rule.getBoundingClientRect()
    expect(Math.round(along.width)).toBe(Math.round(list.getBoundingClientRect().width))
    const at = adding.getBoundingClientRect()
    expect(at.left + at.width / 2).toBeCloseTo(along.left + along.width / 2, 0)
    for (const side of ['::before', '::after']) {
      expect(Number.parseFloat(getComputedStyle(rule, side).width)).toBeGreaterThan(0)
    }
    expect(canvasElement.querySelectorAll('.rule')).toHaveLength(2)
  },
}

/**
 * The window of a face is one window: two parts to a row, the writing beside
 * its preview and the front above the back, divided by the lines they share.
 * The frame around all four is the block's, the parts carry none of their own,
 * and the keyboard landing in one changes nothing that is drawn.
 */
export const TheFaceIsOneWindow: Story = {
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

    // The block is framed and the parts inside it are not.
    const block = found(canvasElement, '[data-face-block="recognise"]')
    expect(Number.parseFloat(getComputedStyle(block).borderTopWidth)).toBeGreaterThan(0)
    for (const pane of PANES) {
      const drawn = getComputedStyle(paneOf(canvasElement, 'recognise', pane))
      for (const side of [
        'borderTopWidth',
        'borderInlineStartWidth',
        'borderBottomWidth',
        'borderInlineEndWidth',
      ] as const) {
        expect(Number.parseFloat(drawn[side])).toBe(0)
      }
      expect(drawn.outlineStyle).toBe('none')
    }
    expect(body.querySelectorAll('fieldset, legend')).toHaveLength(0)

    // The keyboard landing in a box changes the block, the strip and the parts
    // in nothing.
    const watched = [
      block,
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
    const written = boxFor(canvasElement, 'recognise', 'back')
    written.focus()
    expect(document.activeElement).toBe(written)
    expect(look()).toEqual(before)
  },
}

/**
 * What a person does to a stencil: a field written into the half last typed
 * in, a field asked for, and a face carried to another place. The order of the
 * faces is the order a card's repetitions are taken from it, so nothing among
 * them is fixed.
 */
export const WhatAPersonDoesToIt: Story = {
  play: async ({ canvasElement }) => {
    // Nothing has been typed in, so the front is what the row is aimed at.
    const front = boxFor(canvasElement, 'recognise', 'front')
    front.focus()
    front.setSelectionRange(front.value.length, front.value.length)

    const rows = canvasElement.querySelectorAll('[data-face-block="recognise"] .stencil__slots')
    expect(rows).toHaveLength(1)

    await userEvent.click(
      found(canvasElement, '[data-face-block="recognise"] [data-insert="Weight"]'),
    )
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

    // A field asked for stands under a name nothing had taken, below the last.
    await userEvent.click(buttonSaying(canvasElement, 'Add a field'))
    expect(drawnFields(canvasElement)).toEqual([
      'Name',
      'Height',
      'Weight',
      'Life span',
      'Field 1',
    ])

    // The last face is let go over the first, and takes its place.
    const drawn = (): readonly (string | null)[] =>
      [...canvasElement.querySelectorAll('[data-face-block]')].map((each) =>
        each.getAttribute('data-face-block'),
      )
    expect(drawn()).toEqual(['recognise', 'name-it'])

    const bar = found(canvasElement, '[data-face-block="name-it"] .bar')
    expect(bar.getAttribute('draggable')).toBe('true')
    bar.dispatchEvent(new DragEvent('dragstart', { bubbles: true }))
    const onto = found(canvasElement, '[data-face-block="recognise"]')
    onto.dispatchEvent(new DragEvent('dragover', { bubbles: true, cancelable: true }))
    onto.dispatchEvent(new DragEvent('drop', { bubbles: true }))

    await waitFor(() => expect(drawn()).toEqual(['name-it', 'recognise']))
  },
}

/**
 * Forty fields and twelve faces, far more than the window has room for. The
 * row of fields scrolls inside the strip, which keeps its height and its
 * heading.
 */
export const FarTooMany: Story = {
  args: { corpus: 'far too many' },
  play: async ({ canvasElement }) => {
    const bar = found(canvasElement, '[data-face-block="face-0"] .bar')
    const title = found(canvasElement, '[data-face-block="face-0"] .stencil__title')
    const slots = found(canvasElement, '[data-face-block="face-0"] .stencil__slots')

    // Forty fields do not push the heading out of its own strip.
    expect(title.getBoundingClientRect().width).toBeGreaterThan(60)

    expect(slots.scrollWidth).toBeGreaterThan(slots.clientWidth)
    expect(bar.getBoundingClientRect().height).toBeLessThan(40)
    expect(slots.getBoundingClientRect().right).toBeLessThanOrEqual(
      bar.getBoundingClientRect().right + 1,
    )
  },
}

/** Names and prose that are not Latin, and a name with nothing to break at. */
export const AwkwardText: Story = { args: { corpus: 'awkward text' } }

/** No fields and no faces. */
export const NothingAtAll: Story = { args: { corpus: 'nothing at all' } }

/**
 * Everything a file can hold that the editor did not make: one name declared
 * twice, a face naming a slot the fields do not, marks a person wrote that
 * must never be drawn as marks, and a face with nothing in it at all.
 */
export const WhatIsWrong: Story = {
  args: { corpus: 'what is wrong', width: '64rem' },
  play: async ({ canvasElement }) => {
    // The first of two fields of one name stands, and there is no second row
    // and no second chip going nowhere.
    expect(drawnFields(canvasElement)).toEqual(['Name', 'Height'])
    expect(canvasElement.querySelectorAll('.stencil__field input')).toHaveLength(2)
    const chips = [...canvasElement.querySelectorAll('[data-face-block="stray"] [data-insert]')]
    expect(chips.map((chip) => chip.getAttribute('data-insert'))).toEqual(['Name', 'Height'])

    // What is wrong is said where the slot itself stands, and under that half
    // alone.
    const block = found(canvasElement, '[data-face-block="stray"]')
    expect(block.querySelector('[data-preview="front"] mark')?.textContent).toBe('{{Colour}}')
    expect(block.querySelector('[data-preview="back"] mark')?.textContent).toBe('{{Weight}}')
    expect(
      block.querySelector('[data-pane="front-written"] .stencil__objects')?.textContent?.trim(),
    ).toBe('Not a field: Colour')
    expect(
      block.querySelector('[data-pane="back-written"] .stencil__objects')?.textContent?.trim(),
    ).toBe('Not a field: Weight')
    expect(block.querySelector('header .stencil__objects')).toBeNull()

    // A mark a person wrote stands as text, and no tag of one is ever drawn.
    expect(canvasElement.querySelector('script')).toBeNull()
    expect((window as unknown as Record<string, unknown>)['stolen']).toBeUndefined()
    expect(boxFor(canvasElement, 'tagged', 'back').value).toContain('<script>')

    // An empty part says what it is for, in the middle of itself.
    const said: string[] = []
    for (const pane of PANES) {
      const held = paneOf(canvasElement, 'blank', pane)
      const ghost = held.querySelector<HTMLElement>('.stencil__ghost')
      if (!ghost) throw new Error(`nothing said in ${pane}`)
      said.push(ghost.textContent?.trim() ?? '')

      const box = held.getBoundingClientRect()
      const middle = ghost.getBoundingClientRect()
      expect(middle.left + middle.width / 2).toBeCloseTo(box.left + box.width / 2, 0)
      expect(middle.top + middle.height / 2).toBeCloseTo(box.top + box.height / 2, 0)
    }
    expect(said).toEqual(['Front', 'Preview', 'Back', 'Preview'])

    // What is written into a part takes that place, and the part then draws
    // the box alone.
    const written = boxFor(canvasElement, 'blank', 'front')
    await userEvent.type(written, 'a word')
    await waitFor(() => {
      expect(
        paneOf(canvasElement, 'blank', 'front-written').querySelector('.stencil__ghost'),
      ).toBeNull()
      expect(
        paneOf(canvasElement, 'blank', 'front-preview').querySelector('.stencil__ghost'),
      ).toBeNull()
    })
    const pane = paneOf(canvasElement, 'blank', 'front-written')
    expect(written.getBoundingClientRect().top).toBeCloseTo(pane.getBoundingClientRect().top, 0)

    // The half nothing was written in still says what it is for.
    expect(
      paneOf(canvasElement, 'blank', 'back-written').querySelector('.stencil__ghost')?.textContent,
    ).toBe('Back')
  },
}

/** A window too narrow for two parts to a row: one column, each under its half. */
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
