/**
 * The braces a template is written with, read and filled as plain values.
 *
 * A template is text with `{{Field}}` standing where a value goes. A card is named
 * by its first field, which is named in the braces like any other. Everything
 * here is the same text on any machine on any day.
 */

import { wrapBraces, getSlots, SLOT } from './braces'
import type { FieldValue } from './card'

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
  const content = wrapBraces(field)
  return {
    text: template.slice(0, where) + content + template.slice(where),
    caret: where + content.length,
  }
}

export const sampleValues = (fields: readonly string[]): readonly FieldValue[] =>
  fields.map((field) => ({ field, text: field }))
