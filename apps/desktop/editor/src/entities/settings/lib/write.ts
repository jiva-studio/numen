import type { JsonValue } from '@/shared/api/json'

/**
 * A value written back into the settings file. The file is read as JSON5, and
 * what is written back is JSON laid out to be read.
 */
export const write = (value: JsonValue): string => JSON.stringify(value, null, 2)
