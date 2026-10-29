/**
 * The braces a template is written with, read and filled as plain values.
 *
 * A template is text with `{{Field}}` standing where a value goes. A card is named
 * by its first field, which is named in the braces like any other. Everything
 * here is the same text on any machine on any day.
 */

import type { FieldValue } from './card'

/** The braces, and the field name allowed between them. */
const SLOT = /\{\{([^{}]*)\}\}/g

/** One slot at its current position in a template. */
export interface Slot {
  /** The name between the braces, which is a field's name written exactly. */
  readonly field: string
  /** Where the braces begin and end, in UTF-16 code units. */
  readonly from: number
  readonly to: number
}

/** Every slot a template names, in template order. */
export function getSlots(template: string): readonly Slot[] {
  const found: Slot[] = []
  for (const match of template.matchAll(SLOT)) {
    found.push({
      field: match[0].slice(2, -2),
      from: match.index,
      to: match.index + match[0].length,
    })
  }
  return found
}

/** The braces a field is written as. */
export const braceField = (field: string): string => `{{${field}}}`

/** Text standing as text where the marks around it are read as marks. */
const escapeHtml = (text: string): string =>
  text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

/**
 * A template as its preview shows it. A slot the fields do not name keeps its
 * braces and is marked at its current position, so what is wrong is read in its place.
 */
export function renderPreview(
  template: string,
  values: readonly FieldValue[],
  fields: readonly string[],
): string {
  return template.replace(SLOT, (whole: string, inside: string) => {
    if (!fields.includes(inside)) return `<mark>${escapeHtml(whole)}</mark>`
    return values.find((each) => each.field === inside)?.text ?? ''
  })
}

/**
 * The slots a template names that the fields do not declare, each listed once, in
 * template order.
 */
export function getUndeclaredSlots(template: string, fields: readonly string[]): readonly string[] {
  const observed = new Set<string>()
  for (const slot of getSlots(template)) {
    if (fields.includes(slot.field)) continue
    observed.add(slot.field)
  }
  return [...observed]
}

/** A field put into a template: the text it evaluates to, and where the caret lands. */
export interface InsertResult {
  readonly text: string
  readonly caret: number
}

/**
 * A field written into a template at the caret, the caret landing past the closing
 * brace. A caret outside the text is taken to the end nearest it.
 */
export function insert(template: string, pos: number, field: string): InsertResult {
  const where = Math.max(0, Math.min(pos, template.length))
  const content = braceField(field)
  return {
    text: template.slice(0, where) + content + template.slice(where),
    caret: where + content.length,
  }
}

/** A template whose every slot naming one field names another. */
export function renameField(template: string, from: string, to: string): string {
  return template.replace(SLOT, (whole, inside: string) =>
    inside === from ? braceField(to) : whole,
  )
}

export const sampleValues = (fields: readonly string[]): readonly FieldValue[] =>
  fields.map((field) => ({ field, text: field }))
