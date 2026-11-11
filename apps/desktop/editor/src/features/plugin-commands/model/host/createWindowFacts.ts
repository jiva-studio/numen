import type { WhenValues } from '@numen/plugin-loader'
import type { Command } from '@/entities/command'
import type { CommandTarget } from '@/shared/model/command'
import { isFieldFocused } from './isFieldFocused'

/** What the commands part of the window says of the window: its own commands and the when values. */
export interface WindowFactsSource {
  readonly getAppCommands: () => readonly Command[]
  readonly getValues: () => WhenValues
}

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
