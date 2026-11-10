/**
 * Where the palette stands among its steps, and how it moves between them.
 */
import type { PaletteSession } from './session'

export function usePaletteNavigation(session: PaletteSession) {
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
