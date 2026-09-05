/**
 * Every situation the stencil editor has to survive. Also the test corpus:
 * each story is run in a browser by `@storybook/addon-vitest`.
 *
 * What the editor hands back is applied here, which is the application's part.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor } from 'storybook/test'
import { ref, watch } from 'vue'
import Stencil from './StencilEditor.vue'
import { ordered, reordered, type Half, type InsertionPoint } from './order'
import type { StencilFace } from './stencil'
import { renamedIn } from './fill'

interface Corpus {
  readonly fields: readonly string[]
  readonly faces: readonly StencilFace[]
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
        back: '<b>Height:</b> {{Height}}\n<b>Weight:</b> {{Weight}}\n<b>Life span:</b> {{Life span}}',
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
        .map((field) => `<b>${field}:</b> {{${field}}}`)
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
        back: '<b>Перевод:</b> {{Перевод}}\n<b>Пример:</b> {{Пример}}\n{{देवनागरी}}',
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
        back: '<b>Height:</b> {{Height}}\n<b>Weight:</b> {{Weight}}',
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

/** One corpus, read as the stencil is handed it. */
const corpusOf = (key: Corpora): Corpus => CORPORA[key]

interface Knobs {
  corpus: Corpora
  name: string
  /** How wide the window drawing the stencil is. */
  width: string
  /** Given by the story, and nothing a reader turns. */
  fields?: never
  faces?: never
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
    words: { table: { disable: true } },
  },
  args: { corpus: 'a stencil', name: 'Stencil', width: '100%' },
  render: (args) => ({
    components: { Stencil },
    setup() {
      const held = corpusOf(args.corpus)
      const fields = ref<readonly string[]>(held.fields)
      const faces = ref<readonly StencilFace[]>(held.faces)

      watch(
        () => args.corpus,
        (next) => {
          const held = corpusOf(next)
          fields.value = held.fields
          faces.value = held.faces
        },
      )

      /** Every face, with one half of one of them rewritten. */
      const written = (id: string, half: Half, text: string): readonly StencilFace[] =>
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
        onMoveField: (field: string, at: InsertionPoint) => {
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
        onMoveFace: (id: string, at: InsertionPoint) => {
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
    `[data-face="${id}"] [data-half="${half}"]`,
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

    // What the list is added to by stands in the middle of a divider as wide as
    // the list, and the line gives way to it either side.
    const divider = found(canvasElement, '.stencil__part .divider')
    const list = found(canvasElement, '.stencil__fields')
    const adding = found(canvasElement, '.stencil__part .divider button')
    const along = divider.getBoundingClientRect()
    expect(Math.round(along.width)).toBe(Math.round(list.getBoundingClientRect().width))
    const at = adding.getBoundingClientRect()
    expect(at.left + at.width / 2).toBeCloseTo(along.left + along.width / 2, 0)
    for (const side of ['::before', '::after']) {
      expect(Number.parseFloat(getComputedStyle(divider, side).width)).toBeGreaterThan(0)
    }
    expect(canvasElement.querySelectorAll('.divider')).toHaveLength(2)
  },
}

/**
 * What a person does to a stencil: a field written into one face and no other,
 * a field asked for, and a face carried to another place. The order of the
 * faces is the order a card's repetitions are taken from it, so nothing among
 * them is fixed.
 */
export const WhatAPersonDoesToIt: Story = {
  play: async ({ canvasElement }) => {
    // A field pressed in one face's strip is written into that face alone.
    const front = boxFor(canvasElement, 'recognise', 'front')
    front.focus()
    front.setSelectionRange(front.value.length, front.value.length)

    await userEvent.click(
      found(canvasElement, '[data-face="recognise"] [data-insert="Weight"]'),
    )
    expect(boxFor(canvasElement, 'recognise', 'front').value).toBe('{{Name}}{{Weight}}')
    expect(boxFor(canvasElement, 'name-it', 'back').value).toBe('{{Name}}')

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
      [...canvasElement.querySelectorAll('[data-face]')].map((each) =>
        each.getAttribute('data-face'),
      )
    expect(drawn()).toEqual(['recognise', 'name-it'])

    const header = found(canvasElement, '[data-face="name-it"] .card-header')
    expect(header.getAttribute('draggable')).toBe('true')
    header.dispatchEvent(new DragEvent('dragstart', { bubbles: true }))
    const onto = found(canvasElement, '[data-face="recognise"]')
    onto.dispatchEvent(new DragEvent('dragover', { bubbles: true, cancelable: true }))

    // The one on its way is quiet, and the line it would land on is drawn.
    const carried = found(canvasElement, '[data-face="name-it"]')
    await waitFor(() => {
      expect(Number.parseFloat(getComputedStyle(carried).opacity)).toBeLessThan(1)
      expect(Number.parseFloat(getComputedStyle(onto, '::before').blockSize)).toBeGreaterThan(0)
    })

    onto.dispatchEvent(new DragEvent('drop', { bubbles: true }))

    await waitFor(() => expect(drawn()).toEqual(['name-it', 'recognise']))
  },
}

/**
 * Forty fields and twelve faces, far more than the window has room for. Every
 * row and every face is drawn, and none of it is drawn sideways.
 */
export const FarTooMany: Story = {
  args: { corpus: 'far too many' },
  play: async ({ canvasElement }) => {
    expect(drawnFields(canvasElement)).toHaveLength(40)
    expect(canvasElement.querySelectorAll('[data-face]')).toHaveLength(12)

    const editor = found(canvasElement, '.stencil')
    expect(editor.scrollWidth).toBeLessThanOrEqual(editor.clientWidth + 1)
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
    const chips = [...canvasElement.querySelectorAll('[data-face="stray"] [data-insert]')]
    expect(chips.map((chip) => chip.getAttribute('data-insert'))).toEqual(['Name', 'Height'])

    // What the caller found wrong with one face stands under that face's name,
    // and under no other's.
    expect(
      found(canvasElement, '[data-face="stray"] [data-pane="front-written"]')
        .textContent?.trim(),
    ).toContain('Not a field: Colour')
    expect(
      canvasElement.querySelector('[data-face="tagged"] [data-pane] [role="alert"]'),
    ).toBeNull()
  },
}

/**
 * A window too narrow for two parts to a row. The editor is what scrolls, and
 * it never scrolls sideways.
 */
export const Narrow: Story = {
  args: { width: '22rem' },
  play: async ({ canvasElement }) => {
    const editor = found(canvasElement, '.stencil')
    expect(editor.scrollWidth).toBeLessThanOrEqual(editor.clientWidth + 1)
  },
}
