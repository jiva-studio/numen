/**
 * The braces a face is written with, read and filled as plain values.
 *
 * A face is text with `{{Field}}` standing where a value goes. A card is named
 * by its first field, which is named in the braces like any other. Everything
 * here is the same text on any machine on any day.
 */

import type { Filled } from './deck'

/** The braces, and what a person may write between them. */
const SLOT = /\{\{([^{}]*)\}\}/g

/** One slot where it stands in a face. */
export interface Slot {
  /** The name between the braces, which is a field's name written exactly. */
  readonly field: string
  /** Where the braces begin and end, in UTF-16 code units. */
  readonly from: number
  readonly to: number
}

/** Every slot a face names, in the order it stands. */
export function slotsIn(template: string): readonly Slot[] {
  const found: Slot[] = []
  for (const match of template.matchAll(SLOT)) {
    found.push({
      field: match[1] ?? '',
      from: match.index,
      to: match.index + match[0].length,
    })
  }
  return found
}

/** The braces a field is written as. */
export const braced = (field: string): string => `{{${field}}}`

/**
 * A face with every slot standing what fills it. The name between the braces
 * is a field's name written exactly, so a slot nothing was handed for — a name
 * with space around it among them — stands empty.
 */
export function fill(template: string, values: readonly Filled[]): string {
  return template.replace(SLOT, (_, inside: string) =>
    values.find((each) => each.field === inside)?.text ?? '',
  )
}

/** Text standing as text where the marks around it are read as marks. */
const escaped = (text: string): string =>
  text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

/**
 * A face as its preview shows it. A slot the fields do not name keeps its
 * braces and is marked where it stands, so what is wrong is read in its place.
 */
export function previewed(
  template: string,
  values: readonly Filled[],
  fields: readonly string[],
): string {
  return template.replace(SLOT, (whole: string, inside: string) => {
    if (!fields.includes(inside)) return `<mark>${escaped(whole)}</mark>`
    return values.find((each) => each.field === inside)?.text ?? ''
  })
}

/**
 * The slots a face names that the fields do not, each said once, in the order
 * they stand.
 */
export function strayIn(
  template: string,
  fields: readonly string[],
): readonly string[] {
  const said = new Set<string>()
  for (const slot of slotsIn(template)) {
    if (fields.includes(slot.field)) continue
    said.add(slot.field)
  }
  return [...said]
}

/** A field put into a face: the text it comes to, and where the caret lands. */
export interface Inserted {
  readonly text: string
  readonly caret: number
}

/**
 * A field written into a face at the caret, the caret landing past the closing
 * brace. A caret outside the text is taken to the end nearest it.
 */
export function insert(template: string, at: number, field: string): Inserted {
  const where = Math.max(0, Math.min(at, template.length))
  const written = braced(field)
  return {
    text: template.slice(0, where) + written + template.slice(where),
    caret: where + written.length,
  }
}

/** A face whose every slot naming one field names another. */
export function renamedIn(template: string, from: string, to: string): string {
  return template.replace(SLOT, (whole, inside: string) =>
    inside === from ? braced(to) : whole,
  )
}

/** What a preview stands in the slots: each field under its own name. */
export const sampled = (fields: readonly string[]): readonly Filled[] =>
  fields.map((field) => ({ field, text: field }))
