/**
 * What a card model is, as plain values: the slots it names, the templates that show
 * them, and the rows and blocks and panes those are drawn as. What any of it
 * means is the caller's. No DOM, no measurement, no clock.
 */

import type { FieldValue } from './card'
import { renderPreview, getUndeclaredSlots } from './fill'
import {
  getDeclaredFields,
  HALVES,
  createSealedMap,
  type Problems,
  type Half,
  type Objection,
  type HeadingObjection,
} from './order'

/** One way a card model shows a card. */
export interface CardModelTemplate {
  readonly id: string
  readonly name: string
  readonly front: string
  readonly back: string
}

/** The words a card model is drawn with, declared once. */
export interface CardModelWords {
  readonly fields: string
  readonly templates: string
  readonly front: string
  readonly back: string
  readonly preview: string
  readonly addField: string
  readonly addTemplate: string
  readonly remove: string
  readonly drag: string
  readonly insert: string
  readonly noFields: string
  readonly noTemplates: string
  readonly fieldStem: string
  readonly templateStem: string
  /** The message for the first field, which cannot be moved or removed. */
  readonly pinned: string
  /** The message for the slots a template names that the fields do not. */
  readonly stray: (fields: readonly string[]) => string
  /** The message for an unusable field name. */
  readonly objection: (objection: Objection) => string
  /** The message for an unusable template name. */
  readonly templateObjection: (objection: HeadingObjection) => string
  /** What a list of things wrong is called to a reader. */
  readonly wrong: string
}

export const CARD_MODEL_WORDS: CardModelWords = {
  fields: 'Fields',
  templates: 'Templates',
  front: 'Front',
  back: 'Back',
  preview: 'Preview',
  addField: 'Add a field',
  addTemplate: 'Add a template',
  remove: 'Remove',
  drag: 'Reorder',
  insert: 'Insert',
  noFields: 'No fields yet',
  noTemplates: 'No templates yet',
  fieldStem: 'Field',
  templateStem: 'Template',
  pinned: 'The first field names every card, and stays first',
  stray: (fields) => `Not a field: ${fields.join(', ')}`,
  objection: (objection) => {
    if (objection === 'blank') return 'A field needs a name'
    if (objection === 'taken') return 'That name is taken'
    return 'A name cannot hold a brace'
  },
  templateObjection: (objection) =>
    objection === 'blank' ? 'A template needs a name' : 'That name is taken',
  wrong: 'What is wrong',
}

/**
 * What the caller found wrong with a card model. A template's problems are shown
 * beside its name and a field's under that field's row.
 */
export interface CardModelProblems {
  /** What is wrong with each template, under the identity it was drawn by. */
  readonly at: Problems
  /** What is wrong with each field, under the name it is declared by. */
  readonly fields: Problems
}

/** Nothing wrong with any template and nothing wrong with any field. */
export const NO_CARD_MODEL_PROBLEMS: CardModelProblems = Object.freeze({
  at: createSealedMap<string, readonly string[]>(),
  fields: createSealedMap<string, readonly string[]>(),
})

/** One field of a card model, as its row is drawn. */
export interface FieldRow {
  readonly field: string
  /** at its current position, counting from one, which is what it is announced as. */
  readonly at: number
  /** How many fields stand with it. */
  readonly of: number
  /** It is first, so cards of this card model are named by it. */
  readonly hasNames: boolean
  /** It is on its way somewhere else in the order. */
  readonly isDragged: boolean
}

/** The rows a card model's fields are drawn as, one to a field. */
export function fieldRows(
  fields: readonly string[],
  dragField: string | null,
): readonly FieldRow[] {
  const status = getDeclaredFields(fields)
  return status.map((field, index) => ({
    field,
    at: index + 1,
    of: status.length,
    hasNames: index === 0,
    isDragged: field === dragField,
  }))
}

/** One template of a card model, as its row is drawn. */
export interface TemplateRow {
  readonly id: string
  readonly name: string
  /** at its current position, counting from one, which is what it is announced as. */
  readonly at: number
  /** How many templates stand with it. */
  readonly of: number
  /** The fields the card model provides, which may be inserted into a half. */
  readonly fields: readonly string[]
  /** The names the other templates carry, which this one's is measured against. */
  readonly taken: readonly string[]
  /** What is written in the two boxes. */
  readonly front: string
  readonly back: string
  /** The same two, with the sample values standing in the braces. */
  readonly frontPreview: string
  readonly backPreview: string
  /** The slots each half names that the fields do not, each listed once. */
  readonly frontStray: readonly string[]
  readonly backStray: readonly string[]
}

/**
 * A card model's templates as they are drawn, each carrying what its preview shows and
 * what is wrong in each half of it.
 */
export function templateRows(
  templates: readonly CardModelTemplate[],
  fields: readonly string[],
  sample: readonly FieldValue[],
): readonly TemplateRow[] {
  const names = templates.map((each) => each.name)
  return templates.map((template, index) => {
    return {
      id: template.id,
      name: template.name,
      at: index + 1,
      of: templates.length,
      fields,
      taken: names.filter((_, idx) => idx !== index),
      front: template.front,
      back: template.back,
      frontPreview: renderPreview(template.front, sample, fields),
      backPreview: renderPreview(template.back, sample, fields),
      frontStray: getUndeclaredSlots(template.front, fields),
      backStray: getUndeclaredSlots(template.back, fields),
    }
  })
}

/** What one part of the window a template is edited in holds. */
export type PaneMode = 'written' | 'preview'

/** One part of the window a template is edited in. */
export interface Pane {
  readonly half: Half
  readonly mode: PaneMode
  readonly label: string
  /** What the part is announced as. */
  readonly named: string
  /** The markup as it is written, or the sample standing in its braces. */
  readonly text: string
  readonly isBlank: boolean
  /** The slots the half names that the fields do not, listed under the markup. */
  readonly stray: readonly string[]
}

/**
 * The four parts of one template, in display order: each half's markup and then
 * its evaluated preview. Two parts per row place the front above the back; one
 * part per row places each preview below its half.
 */
export function getPanes(
  template: TemplateRow,
  words: CardModelWords = CARD_MODEL_WORDS,
): readonly Pane[] {
  return HALVES.flatMap((half): readonly Pane[] => {
    const transcriptText = half === 'front' ? words.front : words.back
    const content = half === 'front' ? template.front : template.back
    const previewTemplate = half === 'front' ? template.frontPreview : template.backPreview
    return [
      {
        half,
        mode: 'written',
        label: transcriptText,
        named: transcriptText,
        text: content,
        isBlank: content.trim() === '',
        stray: half === 'front' ? template.frontStray : template.backStray,
      },
      {
        half,
        mode: 'preview',
        label: words.preview,
        named: `${words.preview}: ${template.name} ${transcriptText}`,
        text: previewTemplate,
        isBlank: previewTemplate.trim() === '',
        stray: [],
      },
    ]
  })
}
