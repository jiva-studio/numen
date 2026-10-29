/**
 * What the view reads, which is the palette's own state.
 *
 * Every one of these is read again wherever a group is drawn, so a vault moving
 * under an open step is drawn as it is.
 */
import type { Ref } from 'vue'
import type { Vault } from '@/entities/vault'
import type { PendingStep } from '../../lib/step/step'
import type { Command, PaletteLists, RunSupport } from '../../types'
import type { Words } from '../../words'
import type { NameMatch } from '../search/search'

/** Everything the view reads, which is the palette's own state. */
export interface ViewState {
  readonly words: Words
  /** Every command, and the command addressed by its id from another command's row. */
  readonly commands: readonly Command[]
  readonly byId: ReadonlyMap<string, Command>
  readonly runs: RunSupport
  /** The lists the window holds, which a choosing step draws. */
  readonly lists: PaletteLists
  /** The notes a search turned up, and the vaults the installation holds. */
  readonly found: Readonly<Ref<readonly NameMatch[]>>
  readonly known: Readonly<Ref<readonly Vault[]>>
  readonly showing: Readonly<Ref<string>>
  readonly isWorking: Readonly<Ref<boolean>>
  readonly failureMessage: Readonly<Ref<string>>
  /** The step title and label, in the words displayed to the user. */
  readonly getStepTitle: (step: PendingStep) => string
  readonly getStepLabel: (step: PendingStep) => string
}
