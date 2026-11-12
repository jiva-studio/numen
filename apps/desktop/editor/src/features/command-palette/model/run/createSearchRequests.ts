import { createEmitter } from '@/shared/lib/emitter'
import type { createPaletteSwitch } from './createPaletteSwitch'

/**
 * The requests to open the search, which a command makes before the palettes exist. The palettes
 * are connected once they are made.
 */
export function createSearchRequests() {
  const requests = createEmitter<void>()
  return {
    request: (): void => requests.emit(),
    connect: (paletteSwitch: Pick<ReturnType<typeof createPaletteSwitch>, 'openSearch'>): void => {
      requests.subscribe(paletteSwitch.openSearch)
    },
  }
}
