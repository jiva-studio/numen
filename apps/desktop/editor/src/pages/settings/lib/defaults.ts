import type { JsonValue } from '@/shared/api/json'

/**
 * Whether a setting holds a value other than its default. A setting without a default, or holding
 * no value, is never changed; values compare by their JSON.
 */
export const isChangedFromDefault = (
  defaultValue: JsonValue | undefined,
  value: JsonValue | undefined,
): boolean =>
  defaultValue !== undefined &&
  value !== undefined &&
  JSON.stringify(value) !== JSON.stringify(defaultValue)
