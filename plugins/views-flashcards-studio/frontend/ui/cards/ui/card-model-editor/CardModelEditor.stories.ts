/**
 * Every situation the card model editor has to survive. Also the test corpus:
 * each story is run in a browser by `@storybook/addon-vitest`.
 *
 * What the editor returns is applied here, which is the application's part.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor } from 'storybook/test'
import { ref, watch } from 'vue'
import CardModel from './CardModelEditor.vue'
import { orderNames, reorderFields, type Half, type InsertionPoint } from '../../lib/order'
import type { CardModelTemplate } from '../../lib/card-model'
import { renameField } from '../../lib/fill'

interface Corpus {
  readonly fields: readonly string[]
  readonly templates: readonly CardModelTemplate[]
}

const UNBROKEN = 'supercalifragilisticexpialidociousandthensomemoreofitwithnothingtobreakatanywhere'

const many = (count: number): readonly string[] =>
  Array.from({ length: count }, (_, index) => `Field ${index + 1}`)

const CORPORA = {
  'a card model': {
    fields: ['Name', 'Height', 'Weight', 'Life span'],
    templates: [
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
    templates: Array.from({ length: 12 }, (_, index) => ({
      id: `template-${index}`,
      name: `Template ${index + 1}`,
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
    templates: [
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
  'nothing at all': { fields: [], templates: [] },
  /* Everything a file written by hand can hold that the editor did not make:
     one name declared twice, where the first stands and one row is drawn for
     it; a template naming slots the fields do not; marks a person wrote; and a
     template with nothing in it at all. */
  'what is wrong': {
    fields: ['Name', 'Height', 'Height'],
    templates: [
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

/** One corpus, read as the card model is handed it. */
const getCorpus = (key: Corpora): Corpus => CORPORA[key]

interface Knobs {
  corpus: Corpora
  name: string
  /** How wide the window drawing the card model is. */
  width: string
  /** Given by the story, and nothing a reader turns. */
  fields?: never
  templates?: never
  words?: never
}

const meta: Meta<Knobs> = {
  title: 'Flash Cards/Card model',
  component: CardModel,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    corpus: {
      control: 'select',
      options: Object.keys(CORPORA),
      description: 'What the card model starts as. Changing it starts afresh.',
    },
    name: { control: 'text' },
    width: { control: 'text' },
    fields: { table: { disable: true } },
    templates: { table: { disable: true } },
    words: { table: { disable: true } },
  },
  args: { corpus: 'a card model', name: 'Card model', width: '100%' },
  render: (args) => ({
    components: { CardModel },
    setup() {
      const target = getCorpus(args.corpus)
      const fields = ref<readonly string[]>(target.fields)
      const templates = ref<readonly CardModelTemplate[]>(target.templates)

      watch(
        () => args.corpus,
        (next) => {
          const target = getCorpus(next)
          fields.value = target.fields
          templates.value = target.templates
        },
      )

      /** Every template, with one half of one of them rewritten. */
      const rewriteTemplate = (
        id: string,
        half: Half,
        text: string,
      ): readonly CardModelTemplate[] =>
        templates.value.map((template) =>
          template.id === id ? { ...template, [half]: text } : template,
        )

      return {
        args,
        fields,
        templates,
        onAddField: (name: string) => {
          fields.value = [...fields.value, name]
        },
        onRenameField: (field: string, name: string) => {
          fields.value = fields.value.map((each) => (each === field ? name : each))
          templates.value = templates.value.map((template) => ({
            ...template,
            front: renameField(template.front, field, name),
            back: renameField(template.back, field, name),
          }))
        },
        onRemoveField: (field: string) => {
          fields.value = fields.value.filter((each) => each !== field)
        },
        onMoveField: (field: string, target: InsertionPoint) => {
          fields.value = reorderFields(fields.value, field, target)
        },
        onAddTemplate: (name: string) => {
          templates.value = [
            ...templates.value,
            { id: `template-${templates.value.length}`, name, front: '', back: '' },
          ]
        },
        onRenameTemplate: (id: string, name: string) => {
          templates.value = templates.value.map((template) =>
            template.id === id ? { ...template, name } : template,
          )
        },
        onRemoveTemplate: (id: string) => {
          templates.value = templates.value.filter((template) => template.id !== id)
        },
        /* Nothing among the templates is fixed, so any of them lands anywhere. */
        onMoveTemplate: (id: string, target: InsertionPoint) => {
          const order = orderNames(
            templates.value.map((template) => template.id),
            id,
            target,
          )
          templates.value = order.flatMap((each) =>
            templates.value.filter((template) => template.id === each),
          )
        },
        onWrite: (id: string, half: Half, text: string) => {
          templates.value = rewriteTemplate(id, half, text)
        },
      }
    },
    template: `
      <div :style="{ height: '100vh', width: args.width }">
        <CardModel
          :fields="fields"
          :templates="templates"
          :name="args.name"
          @add-field="onAddField"
          @rename-field="onRenameField"
          @remove-field="onRemoveField"
          @move-field="onMoveField"
          @add-template="onAddTemplate"
          @rename-template="onRenameTemplate"
          @remove-template="onRemoveTemplate"
          @move-template="onMoveTemplate"
          @write="onWrite"
        />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const getBox = (canvas: HTMLElement, id: string, half: Half) => {
  const box = canvas.querySelector<HTMLTextAreaElement>(
    `[data-template="${id}"] [data-half="${half}"]`,
  )
  if (!box) throw new Error(`no ${half} of ${id}`)
  return box
}

const found = (canvas: HTMLElement, selector: string): HTMLElement => {
  const element = canvas.querySelector<HTMLElement>(selector)
  if (!element) throw new Error(`nothing matching ${selector}`)
  return element
}

const buttonSaying = (canvas: HTMLElement, text: string) => {
  const element = [...canvas.querySelectorAll<HTMLElement>('button')].find(
    (each) => each.textContent?.trim() === text,
  )
  if (!element) throw new Error(`no button saying ${text}`)
  return element
}

const getDrawnFields = (canvas: HTMLElement): readonly (string | null)[] =>
  [...canvas.querySelectorAll('[data-field]')].map((row) => row.getAttribute('data-field'))

/**
 * Four fields and two templates, one of them asking the other way round.
 *
 * A field's row is one block: the line and the ground are the row's, and the
 * handle, the box and the way to remove it stand inside it carrying neither.
 * The first field names every card, so it keeps the shape the others have with
 * middle of a rule as wide as the list.
 */
export const ACardModel: Story = {
  args: { width: '64rem' },
  play: async ({ canvasElement }) => {
    const row = found(canvasElement, '[data-field="Height"] .card-model__row')
    const box = row.querySelector('input')
    const grip = row.querySelector('[data-grip]')
    const away = row.querySelector('button')
    if (!box || !grip || !away) throw new Error('a row missing part of itself')

    // neither.
    const renderedStyle = getComputedStyle(row)
    expect(Number.parseFloat(renderedStyle.borderTopWidth)).toBeGreaterThan(0)
    expect(renderedStyle.backgroundColor).not.toBe('rgba(0, 0, 0, 0)')
    const inside = getComputedStyle(box)
    expect(Number.parseFloat(inside.borderTopWidth)).toBe(0)
    expect(inside.backgroundColor).toBe('rgba(0, 0, 0, 0)')

    // And all three stand within the row's own bounds.
    const bounds = row.getBoundingClientRect()
    for (const part of [grip, box, away]) {
      const rect = part.getBoundingClientRect()
      expect(rect.left).toBeGreaterThanOrEqual(bounds.left - 1)
      expect(rect.right).toBeLessThanOrEqual(bounds.right + 1)
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
    second
      .querySelector('[data-grip]')
      ?.dispatchEvent(new DragEvent('dragstart', { bubbles: true }))
    first.dispatchEvent(new DragEvent('dragover', { bubbles: true, cancelable: true }))
    first.dispatchEvent(new DragEvent('drop', { bubbles: true }))
    await new Promise((resolved) => {
      setTimeout(resolved, 0)
    })
    expect(getDrawnFields(canvasElement)).toEqual(['Name', 'Height', 'Weight', 'Life span'])

    // the list, and the line gives way to it either side.
    const divider = found(canvasElement, '.card-model__part .divider')
    const list = found(canvasElement, '.card-model__fields')
    const adding = found(canvasElement, '.card-model__part .divider button')
    const axisRect = divider.getBoundingClientRect()
    expect(Math.round(axisRect.width)).toBe(Math.round(list.getBoundingClientRect().width))
    const addingRect = adding.getBoundingClientRect()
    expect(addingRect.left + addingRect.width / 2).toBeCloseTo(
      axisRect.left + axisRect.width / 2,
      0,
    )
    for (const side of ['::before', '::after']) {
      expect(Number.parseFloat(getComputedStyle(divider, side).width)).toBeGreaterThan(0)
    }
    expect(canvasElement.querySelectorAll('.divider')).toHaveLength(2)
  },
}

/**
 * What a person does to a card model: a field written into one template and no other,
 * a field asked for, and a template dragged to another place. The order of the
 * templates is the order a card's repetitions are taken from it, so nothing among
 * them is fixed.
 */
export const WhatAPersonDoesToIt: Story = {
  play: async ({ canvasElement }) => {
    // A field pressed in one template's strip is written into that template alone.
    const front = getBox(canvasElement, 'recognise', 'front')
    front.focus()
    front.setSelectionRange(front.value.length, front.value.length)

    await userEvent.click(
      found(canvasElement, '[data-template="recognise"] [data-insert="Weight"]'),
    )
    expect(getBox(canvasElement, 'recognise', 'front').value).toBe('{{Name}}{{Weight}}')
    expect(getBox(canvasElement, 'name-it', 'back').value).toBe('{{Name}}')

    // A field asked for is placed under a name nothing had taken, below the last.
    await userEvent.click(buttonSaying(canvasElement, 'Add a field'))
    expect(getDrawnFields(canvasElement)).toEqual([
      'Name',
      'Height',
      'Weight',
      'Life span',
      'Field 1',
    ])

    // The last template is let go over the first, and takes its place.
    const getDrawnTemplates = (): readonly (string | null)[] =>
      [...canvasElement.querySelectorAll('[data-template]')].map((each) =>
        each.getAttribute('data-template'),
      )
    expect(getDrawnTemplates()).toEqual(['recognise', 'name-it'])

    const header = found(canvasElement, '[data-template="name-it"] .card-header')
    expect(header.getAttribute('draggable')).toBe('true')
    header.dispatchEvent(new DragEvent('dragstart', { bubbles: true }))
    const onto = found(canvasElement, '[data-template="recognise"]')
    onto.dispatchEvent(new DragEvent('dragover', { bubbles: true, cancelable: true }))

    // The one on its way is quiet, and the line it would land on is drawn.
    const dragged = found(canvasElement, '[data-template="name-it"]')
    await waitFor(() => {
      expect(Number.parseFloat(getComputedStyle(dragged).opacity)).toBeLessThan(1)
      expect(Number.parseFloat(getComputedStyle(onto, '::before').blockSize)).toBeGreaterThan(0)
    })

    onto.dispatchEvent(new DragEvent('drop', { bubbles: true }))

    await waitFor(() => expect(getDrawnTemplates()).toEqual(['name-it', 'recognise']))
  },
}

/**
 * Forty fields and twelve templates, far more than the window has room for. Every
 * row and every template is drawn, and none of it is drawn sideways.
 */
export const FarTooMany: Story = {
  args: { corpus: 'far too many' },
  play: async ({ canvasElement }) => {
    expect(getDrawnFields(canvasElement)).toHaveLength(40)
    expect(canvasElement.querySelectorAll('[data-template]')).toHaveLength(12)

    const editor = found(canvasElement, '.cardModel')
    expect(editor.scrollWidth).toBeLessThanOrEqual(editor.clientWidth + 1)
  },
}

/** Names and prose that are not Latin, and a name with nothing to break at. */
export const AwkwardText: Story = { args: { corpus: 'awkward text' } }

/** No fields and no templates. */
export const NothingAtAll: Story = { args: { corpus: 'nothing at all' } }

/**
 * Everything a file can hold that the editor did not make: one name declared
 * twice, a template naming a slot the fields do not, marks a person wrote that
 * must never be drawn as marks, and a template with nothing in it at all.
 */
export const WhatIsWrong: Story = {
  args: { corpus: 'what is wrong', width: '64rem' },
  play: async ({ canvasElement }) => {
    // The first of two fields of one name stands, and there is no second row
    // and no second chip going nowhere.
    expect(getDrawnFields(canvasElement)).toEqual(['Name', 'Height'])
    expect(canvasElement.querySelectorAll('.card-model__field input')).toHaveLength(2)
    const chips = [...canvasElement.querySelectorAll('[data-template="stray"] [data-insert]')]
    expect(chips.map((chip) => chip.getAttribute('data-insert'))).toEqual(['Name', 'Height'])

    // What the caller found wrong with one template is placed under that template's name,
    // and under no other's.
    expect(
      found(
        canvasElement,
        '[data-template="stray"] [data-pane="front-written"]',
      ).textContent?.trim(),
    ).toContain('Not a field: Colour')
    expect(
      canvasElement.querySelector('[data-template="tagged"] [data-pane] [role="alert"]'),
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
    const editor = found(canvasElement, '.cardModel')
    expect(editor.scrollWidth).toBeLessThanOrEqual(editor.clientWidth + 1)
  },
}
