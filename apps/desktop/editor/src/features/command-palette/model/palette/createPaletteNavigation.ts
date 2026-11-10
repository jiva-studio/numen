/**
 * The step the palette is at, and how it moves between steps.
 */
import type { PaletteSession } from './session'

export function createPaletteNavigation(session: PaletteSession) {
  const { steps } = session
  return {
    crumb: steps.crumb,
    step: steps.step,
    opensOn: steps.opensOn,
    placeholder: steps.placeholder,
    applyRenames: steps.applyRenames,
    leaveStep: steps.leaveStep,
    goBack: steps.goBack,
  }
}
