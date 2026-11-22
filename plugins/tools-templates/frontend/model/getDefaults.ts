import type { Field } from '../api/templates'

/** The value each field takes while the person has typed nothing: its default, by label. */
export function getDefaults(fields: readonly Field[]): Record<string, string> {
  return Object.fromEntries(fields.map((field) => [field.label, field.default]))
}
