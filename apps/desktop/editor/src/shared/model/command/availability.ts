/**
 * When a command of the application is offered, and when it is not: its when condition over the
 * values of the palette.
 */
import { parseWhen } from '@numen/plugin-loader'
import { getPaletteValues } from './paletteValues'
import type { CommandTarget } from './invocation'

function createAvailabilityCheck(when: string): (target: CommandTarget) => boolean {
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
