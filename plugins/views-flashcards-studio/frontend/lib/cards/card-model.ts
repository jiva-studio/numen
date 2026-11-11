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
  getSealedMap,
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
  readonly pinned: string
  readonly stray: (fields: readonly string[]) => string
  readonly objection: (objection: Objection) => string
  readonly templateObjection: (objection: HeadingObjection) => string
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
  readonly at: Problems
  readonly fields: Problems
}

/** Nothing wrong with any template and nothing wrong with any field. */
export const NO_CARD_MODEL_PROBLEMS: CardModelProblems = Object.freeze({
  at: getSealedMap<string, readonly string[]>(),
  fields: getSealedMap<string, readonly string[]>(),
})

/** One field of a card model, as its row is drawn. */
export interface FieldRow {
  readonly field: string
  readonly at: number
  readonly of: number
  readonly hasNames: boolean
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
  readonly at: number
  readonly of: number
  readonly fields: readonly string[]
  readonly taken: readonly string[]
  readonly front: string
  readonly back: string
  readonly frontPreview: string
  readonly backPreview: string
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
type PaneMode = 'written' | 'preview'

/** One part of the window a template is edited in. */
export interface Pane {
  readonly half: Half
  readonly mode: PaneMode
  readonly label: string
  readonly named: string
  readonly text: string
  readonly isBlank: boolean
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
