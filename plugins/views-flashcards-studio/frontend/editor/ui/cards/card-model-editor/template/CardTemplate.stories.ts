/**
 * Every situation one template of a card model has to survive. Also the test corpus:
 * each story is run in a browser by `@storybook/addon-vitest`.
 *
 * What the template returns is applied here, which is the card model's part.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor } from 'storybook/test'
import { computed, ref, watch } from 'vue'
import CardTemplate from './CardTemplate.vue'
import type { FieldValue } from '../../../../../lib/cards'
import { getDeclaredFields, type Half } from '../../../../../lib/cards'
import { templateRows, type TemplateRow, type CardModelTemplate } from '../../../../../lib/cards'
import { sampleValues } from '../../../../../lib/cards'

interface Corpus {
  readonly template: CardModelTemplate
  readonly fields: readonly string[]
  readonly sample?: readonly FieldValue[]
  readonly taken?: readonly string[]
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
  width: string
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
 * One template of a card model of four fields, its previews standing for what a card fills them with.
 * The preview shows the card with the slots filled, and a field chip inserts its slot into the half last typed in.
 */
export const ATemplate: Story = {
  play: async ({ canvasElement }) => {
    const body = found(canvasElement, '.template__body')
    await expect(body.querySelectorAll('fieldset, legend')).toHaveLength(0)

    // The preview shows the card with the slots filled, without the braces of the
    // markup it came from.
    const preview = found(canvasElement, '[data-preview="back"]')
    await expect(preview.textContent).toContain('about 45" at the shoulder')
    await expect(preview.textContent).not.toContain('{{')
    await expect(preview.querySelector('b')?.textContent).toBe('Height:')

    // A field is inserted into the half last typed in, at the caret.
    const chip = found(canvasElement, '[data-insert="Height"]')
    const back = getBox(canvasElement, 'back')
    back.focus()
    back.setSelectionRange(0, 0)
    await userEvent.click(chip)
    await waitFor(() =>
      expect(getBox(canvasElement, 'back').value.startsWith('{{Height}}')).toBe(true),
    )

    // The keyboard lands in a box.
    const content = getBox(canvasElement, 'front')
    content.focus()
    await expect(document.activeElement).toBe(content)
  },
}

/** A template naming slots the card model does not declare, in both halves. */
export const AStraySlot: Story = {
  args: { corpus: 'a stray slot' },
  play: async ({ canvasElement }) => {
    // The error is shown at the slot itself, and under that half only.
    await expect(found(canvasElement, '[data-preview="front"] mark').textContent).toBe('{{Colour}}')
    await expect(found(canvasElement, '[data-preview="back"] mark').textContent).toBe('{{Weight}}')
    await expect(
      found(canvasElement, '[data-pane="front-written"] .template__objections').textContent?.trim(),
    ).toBe('Not a field: Colour')
    await expect(
      found(canvasElement, '[data-pane="back-written"] .template__objections').textContent?.trim(),
    ).toBe('Not a field: Weight')
    await expect(canvasElement.querySelector('header .template__objections')).toBeNull()
  },
}

/** A template with nothing in either half, and no name either. */
export const NothingInIt: Story = {
  args: { corpus: 'nothing in it' },
  play: async ({ canvasElement }) => {
    // An empty part shows its placeholder.
    const placeholders: string[] = []
    for (const pane of PANES) {
      const ghost = getPane(canvasElement, pane).querySelector<HTMLElement>('.template__ghost')
      if (!ghost) throw new Error('template not found')
      placeholders.push(ghost.textContent?.trim() ?? '')
    }
    await expect(placeholders).toEqual(['Front', 'Preview', 'Back', 'Preview'])

    // What is written into a part takes that place.
    const content = getBox(canvasElement, 'front')
    await userEvent.type(content, 'a word')
    await waitFor(async () => {
      await expect(
        getPane(canvasElement, 'front-written').querySelector('.template__ghost'),
      ).toBeNull()
      await expect(
        getPane(canvasElement, 'front-preview').querySelector('.template__ghost'),
      ).toBeNull()
    })

    // The half with no text still shows its placeholder.
    await expect(
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
    await expect(canvasElement.querySelector('script')).toBeNull()
    await expect((window as unknown as Record<string, unknown>)['stolen']).toBeUndefined()

    // The tag is drawn; the asterisks around the same value are four characters
    // of the answer, and nothing about them is bold.
    const front = found(canvasElement, '[data-preview="front"]')
    await expect(front.textContent).toBe('**Name** and Name')
    await expect(front.querySelectorAll('b, strong')).toHaveLength(1)

    // The user's text is editable and shown as the text of a box.
    await expect(getBox(canvasElement, 'back').value).toContain('<script>')
  },
}

/** Forty fields, more than the strip can display: every one of them is still there to insert. */
export const FarTooManyFields: Story = {
  args: { corpus: 'far too many fields' },
  play: async ({ canvasElement }) => {
    await expect(canvasElement.querySelectorAll('[data-insert]')).toHaveLength(40)
  },
}

/**
 * What the vault found wrong with the template, and a name another template carries.
 * The errors found against the template are listed, and the error for a name being typed is shown at the box.
 */
export const WhatIsWrongWithIt: Story = {
  args: { corpus: 'something wrong with it' },
  play: async ({ canvasElement }) => {
    const response = found(canvasElement, '[data-wrong]')
    await expect(response.textContent).toContain('this template has no back')
    await expect(response.querySelectorAll('li')).toHaveLength(2)

    // A name that another template uses is rejected, and the box shows the error.
    const name = found(canvasElement, '.template__title') as HTMLInputElement
    await userEvent.clear(name)
    await userEvent.type(name, 'Recognise')
    await waitFor(async () => {
      const objections = found(canvasElement, 'header [role="alert"]')
      await expect(objections.textContent?.trim()).toBe('That name is taken')
      await expect(name.getAttribute('aria-describedby')).toBe(objections.id)
    })
  },
}
