import type { JsonObject, JsonValue } from '@/shared/api/json'

/** The value at a path of keys in a settings fixture, and nothing where the path leaves it. */
export const getSettingAt = (
  settings: JsonValue | undefined,
  at: readonly string[],
): JsonValue | undefined => {
  let value = settings
  for (const step of at) {
    if (typeof value !== 'object' || value === null || Array.isArray(value)) return undefined
    value = (value as JsonObject)[step]
  }
  return value
}
