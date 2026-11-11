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
import { orderNames, reorderFields, type Half, type InsertionPoint } from '../../../../lib/cards'
import type { CardModelTemplate } from '../../../../lib/cards'
import { renameField } from '../../../../lib/cards'

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
  /* Everything a manually authored file can contain that the editor did not create:
     one name declared twice, where the first applies and one row is drawn for
     it; a template naming slots the fields do not; marks the user typed; and a
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

const getCorpus = (key: Corpora): Corpus => CORPORA[key]

interface Knobs {
  corpus: Corpora
  name: string
  width: string
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
 * Four fields and two templates, one of them in the reverse direction.
 * The first field stays first and cannot be moved or removed; the others can.
 */
export const ACardModel: Story = {
  args: { width: '64rem' },
  play: async ({ canvasElement }) => {
    // The first field stays first.
    const first = found(canvasElement, '[data-field="Name"]')
    const second = found(canvasElement, '[data-field="Height"]')
    await expect(first.getAttribute('data-names')).toBe('true')
    await expect(first.querySelector('[data-grip]')?.getAttribute('draggable')).toBe('false')
    await expect(first.querySelector('button')?.disabled).toBe(true)
    await expect(second.querySelector('[data-grip]')?.getAttribute('draggable')).toBe('true')
    await expect(second.querySelector('button')?.disabled).toBe(false)
    // Letting a field go on the first row lands it nowhere.
    second
      .querySelector('[data-grip]')
      ?.dispatchEvent(new DragEvent('dragstart', { bubbles: true }))
    first.dispatchEvent(new DragEvent('dragover', { bubbles: true, cancelable: true }))
    first.dispatchEvent(new DragEvent('drop', { bubbles: true }))
    await new Promise((resolved) => {
      setTimeout(resolved, 0)
    })
    await expect(getDrawnFields(canvasElement)).toEqual(['Name', 'Height', 'Weight', 'Life span'])

    await expect(canvasElement.querySelectorAll('.divider')).toHaveLength(2)
  },
}

/**
 * User actions on a card model: a field inserted into one template only, a
 * field added, and a template dragged to another position. The order of the
 * templates is the order in which a card's repetitions are taken, so no
 * template position is fixed.
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
    await expect(getBox(canvasElement, 'recognise', 'front').value).toBe('{{Name}}{{Weight}}')
    await expect(getBox(canvasElement, 'name-it', 'back').value).toBe('{{Name}}')

    // An added field is named with an unused name and placed after the last field.
    await userEvent.click(buttonSaying(canvasElement, 'Add a field'))
    await expect(getDrawnFields(canvasElement)).toEqual([
      'Name',
      'Height',
      'Weight',
      'Life span',
      'Field 1',
    ])

    // The last template is dropped over the first, and takes its place.
    const getDrawnTemplates = (): readonly (string | null)[] =>
      [...canvasElement.querySelectorAll('[data-template]')].map((each) =>
        each.getAttribute('data-template'),
      )
    await expect(getDrawnTemplates()).toEqual(['recognise', 'name-it'])

    const header = found(canvasElement, '[data-template="name-it"] .card-header')
    await expect(header.getAttribute('draggable')).toBe('true')
    header.dispatchEvent(new DragEvent('dragstart', { bubbles: true }))
    const onto = found(canvasElement, '[data-template="recognise"]')
    onto.dispatchEvent(new DragEvent('dragover', { bubbles: true, cancelable: true }))

    onto.dispatchEvent(new DragEvent('drop', { bubbles: true }))

    await waitFor(() => expect(getDrawnTemplates()).toEqual(['name-it', 'recognise']))
  },
}

/**
 * Forty fields and twelve templates, more than fit in the window. Every
 * row and every template is rendered.
 */
export const FarTooMany: Story = {
  args: { corpus: 'far too many' },
  play: async ({ canvasElement }) => {
    await expect(getDrawnFields(canvasElement)).toHaveLength(40)
    await expect(canvasElement.querySelectorAll('[data-template]')).toHaveLength(12)
  },
}

/** Names and prose that are not Latin, and a name with nothing to break at. */
export const AwkwardText: Story = { args: { corpus: 'awkward text' } }

/** No fields and no templates. */
export const NothingAtAll: Story = { args: { corpus: 'nothing at all' } }

/**
 * Everything a file can contain that the editor did not create: one name
 * declared twice, a template naming a slot the fields do not declare, marks
 * written by the user that must not be rendered as marks, and a template with
 * no content.
 */
export const WhatIsWrong: Story = {
  args: { corpus: 'what is wrong', width: '64rem' },
  play: async ({ canvasElement }) => {
    // The first of two fields with one name is used, so there is no second row and
    // no second chip with no target.
    await expect(getDrawnFields(canvasElement)).toEqual(['Name', 'Height'])
    await expect(canvasElement.querySelectorAll('.card-model__field input')).toHaveLength(2)
    const chips = [...canvasElement.querySelectorAll('[data-template="stray"] [data-insert]')]
    await expect(chips.map((chip) => chip.getAttribute('data-insert'))).toEqual(['Name', 'Height'])

    // What the caller found wrong with one template is placed under that template's name,
    // and under no other's.
    await expect(
      found(
        canvasElement,
        '[data-template="stray"] [data-pane="front-written"]',
      ).textContent?.trim(),
    ).toContain('Not a field: Colour')
    await expect(
      canvasElement.querySelector('[data-template="tagged"] [data-pane] [role="alert"]'),
    ).toBeNull()
  },
}
