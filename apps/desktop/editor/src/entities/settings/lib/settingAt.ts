/** The value at a path of keys in the settings, and nothing where the path leaves them. */
export const getSettingAt = (settings: unknown, at: readonly string[]): unknown => {
  let value = settings
  for (const step of at) {
    if (typeof value !== 'object' || value === null) return undefined
    value = (value as Record<string, unknown>)[step]
  }
  return value
}
