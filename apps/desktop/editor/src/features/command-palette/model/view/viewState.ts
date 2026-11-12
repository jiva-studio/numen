/**
 * What the view reads, which is the palette's own state.
 *
 * Every one of these is read again wherever a group is drawn, so a vault moving
 * under an open step is drawn as it is.
 */
import type { Ref } from 'vue'
import type { Vault } from '@/entities/vault'
import type { PendingStep } from '../../lib/step/step'
import type { PaletteLists } from '../../types'
import type { Command } from '@/entities/command'
import type { RunSupport } from '@/shared/model/command'
import type { Words } from '../../words'
import type { NameMatch } from '../search/search'

/** Everything the view reads, which is the palette's own state. */
export interface ViewState {
  readonly words: Words
  readonly commands: readonly Command[]
  readonly byId: ReadonlyMap<string, Command>
  readonly runs: RunSupport
  readonly lists: PaletteLists
  readonly found: Readonly<Ref<readonly NameMatch[]>>
  readonly known: Readonly<Ref<readonly Vault[]>>
  readonly showing: Readonly<Ref<string>>
  readonly isWorking: Readonly<Ref<boolean>>
  readonly failureMessage: Readonly<Ref<string>>
  readonly getStepTitle: (step: PendingStep) => string
  readonly getStepLabel: (step: PendingStep) => string
}
