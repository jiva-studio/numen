/**
 * When a command of the application is offered, and when it is not: its when condition over the
 * values of the palette.
 */
import { parseWhen } from '@numen/plugin-loader'
import { getPaletteValues } from '@/shared/paletteValues'
import type { CommandTarget } from '../model/command'

/**
 * The availability of a command of the application by its condition, read once. Its conditions read
 * the window and whether the target is a note or a file, not content types or settings.
 */
export function createAvailabilityCheck(when: string): (target: CommandTarget) => boolean {
  const parsed = parseWhen(when)
  if (!parsed.ok) throw new Error(parsed.error)
  return (target) => parsed.value.evaluate(getPaletteValues(target, {}))
}

/** The condition of a command of the application and the availability it gives. */
export const withWhen = (
  when: string,
): { readonly when: string; isAvailable(target: CommandTarget): boolean } => ({
  when,
  isAvailable: createAvailabilityCheck(when),
})
