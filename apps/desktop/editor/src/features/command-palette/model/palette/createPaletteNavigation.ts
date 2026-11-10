/**
 * The step the palette is at, and how it moves between steps.
 */
import type { PaletteSession } from './session'

export function createPaletteNavigation(session: PaletteSession) {
  const { steps } = session
  return {
    crumb: steps.position.crumb,
    step: steps.position.step,
    opensOn: steps.position.opensOn,
    placeholder: steps.position.placeholder,
    applyRenames: steps.moves.applyRenames,
    leaveStep: steps.moves.leaveStep,
    goBack: steps.moves.goBack,
  }
}
