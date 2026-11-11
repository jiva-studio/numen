import type { Command, CommandTarget } from '@/entities/command'
import { isFieldFocused } from './isFieldFocused'
import type { WindowFactsSource } from './hostPort'

export interface WindowFactsDeps {
  readonly getAppCommands: () => readonly Command[]
  readonly getTarget: () => CommandTarget
  readonly isPaletteOpen: () => boolean
}

/** What the conditions of the commands of the plugins read of the window. */
export function createWindowFacts(deps: WindowFactsDeps): WindowFactsSource {
  return {
    getAppCommands: deps.getAppCommands,
    getValues: () => {
      const inFront = deps.getTarget()
      return {
        'window.isVaultOpen': inFront.vault.id !== '',
        'window.isVaultReady': inFront.isReady,
        'window.isTabOpen': inFront.tab !== '',
        'window.isFieldFocused': isFieldFocused(),
        'window.isPaletteOpen': deps.isPaletteOpen(),
      }
    },
  }
}
