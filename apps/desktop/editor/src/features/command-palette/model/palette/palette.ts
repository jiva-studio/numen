/**
 * The palette the user types into, put together from its three parts: the palette itself, what it
 * draws, and the steps it moves between.
 *
 * What a step renders is recomputed on every keystroke, so a vault changing under an open step is
 * rendered as it is.
 */
import { createPaletteSession, type CommandPaletteDeps } from './session'
import { createPalette } from './createPalette'
import { createPaletteNavigation } from './createPaletteNavigation'
import { usePaletteView } from './usePaletteView'

export function createCommandPalette(deps: CommandPaletteDeps) {
  const session = createPaletteSession(deps)
  const navigation = createPaletteNavigation(session)
  const view = usePaletteView(session)
  const palette = createPalette(session, view)

  return { palette, view, navigation }
}

export type Commands = ReturnType<typeof createCommandPalette>
