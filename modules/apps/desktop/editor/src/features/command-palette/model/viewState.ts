/**
 * What the view reads, which is the palette's own state.
 *
 * Every one of these is read again wherever a group is drawn, so a vault moving
 * under an open step is drawn as it is.
 */
import type { Ref } from 'vue'
import type { Vault } from '@/entities/vault'
import type { PendingStep } from '../lib/step'
import type { Command, PaletteLists, RunSupport } from '../types'
import type { Words } from '../words'
import type { NameMatch } from './search'

/** Everything the view reads, which is the palette's own state. */
export interface ViewState {
  readonly words: Words
  /** Every command, and the one another command's row reaches by its id. */
  readonly commands: readonly Command[]
  readonly byId: ReadonlyMap<string, Command>
  readonly runs: RunSupport
  /** The lists the window holds, which a choosing step draws. */
  readonly lists: PaletteLists
  /** The notes a search turned up, and the vaults the installation holds. */
  readonly found: Readonly<Ref<readonly NameMatch[]>>
  readonly known: Readonly<Ref<readonly Vault[]>>
  /** The vault the window is showing, which is the one it will not open again. */
  readonly showing: Readonly<Ref<string>>
  /** Whether the vault is being asked, and what it said when it refused. */
  readonly isWorking: Readonly<Ref<boolean>>
  readonly failureMessage: Readonly<Ref<string>>
  /** What a step is over, in the words a person reads it as. */
  readonly getStepTitle: (step: PendingStep) => string
  readonly getStepLabel: (step: PendingStep) => string
}
