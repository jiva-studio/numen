/** The braces, and the field name allowed between them. */
export const SLOT = /\{\{([^{}]*)\}\}/g

/** One slot at its current position in a template. */
interface Slot {
  readonly field: string
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
export const wrapBraces = (field: string): string => `{{${field}}}`
