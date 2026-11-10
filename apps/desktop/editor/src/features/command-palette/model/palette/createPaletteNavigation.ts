/**
 * The step the palette is at, and how it moves between steps.
 */
import type { PaletteSession } from './session'

export function createPaletteNavigation(session: PaletteSession) {
  const { steps } = session
  return {
    crumb: steps.shown.crumb,
    step: steps.shown.step,
    opensOn: steps.shown.opensOn,
    placeholder: steps.shown.placeholder,
    applyRenames: steps.moves.applyRenames,
    leaveStep: steps.moves.leaveStep,
    goBack: steps.moves.goBack,
  }
}
