/**
 * A value written into the settings file.
 *
 * The file a person types is JSON5 — comments, trailing commas, single quotes
 * and bare keys — and the vault is what reads it, refusing a value the settings
 * cannot be read out of before the file is touched. What the window writes back
 * is JSON, which JSON5 holds whole.
 */

/** A value written back into the settings, which are JSON. */
export const write = (value: unknown): string => JSON.stringify(value, null, 2)
