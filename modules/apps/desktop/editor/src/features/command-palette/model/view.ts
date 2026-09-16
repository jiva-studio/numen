/**
 * What the palette draws where it stands: the commands offered over what is in
 * front, and the groups whichever step is open asks in.
 *
 * Every one of these is the state read again, so a vault moving under an open
 * step is drawn as it is. Nothing here decides anything or asks the
 * application for anything: `palette.ts` holds the steps and does that.
 */
import type { PaletteGroup, PaletteItem } from '@numen/ui'
import { inGroup } from '../lib/commands'
import type { PendingStep } from '../lib/step'
import type { Command, CommandGroup, CommandTarget } from '../types'
import { createStepGroups } from './stepGroups'
import type { ViewState } from './viewState'

export type { ViewState } from './viewState'

/** The view one palette draws, over the state that palette holds. */
export function getPaletteView(state: ViewState) {
  const { words, commands, byId, runs } = state
  const steps = createStepGroups(state)

  /** One command as it is drawn, and nothing where the words typed leave it out. */
  const renderItem = (one: Command, over: CommandTarget, word: string): PaletteItem | null => {
    const found = word === '' ? -1 : one.text.toLowerCase().indexOf(word)
    if (word !== '' && found < 0) return null
    const also = one.also ? byId.get(one.also) : undefined
    return {
      id: one.id,
      title: one.text,
      ...(found < 0 ? {} : { at: [{ from: found, to: found + word.length }] }),
      ...(one.keys ? { keys: one.keys } : {}),
      actions: [
        { id: one.id, text: one.text },
        ...(also && also.isOffered(over, runs) ? [{ id: also.id, text: also.text }] : []),
      ],
    }
  }

  /** Why nothing over the note in front is offered. */
  const getSilenceMessage = (over: CommandTarget): string =>
    !over.ready ? words.noVault : over.path ? words.noneFound : words.noNote

  /** Every command offered over what is in front, in the groups it holds. */
  const getCommandGroups = (over: CommandTarget, text: string): readonly PaletteGroup[] => {
    const word = text.trim().toLowerCase()
    const items = (group: CommandGroup): readonly PaletteItem[] =>
      inGroup(commands, group)
        .filter((one) => one.isOffered(over, runs))
        .map((one) => renderItem(one, over, word))
        .filter((item) => item !== null)

    // The runs are offered over a book and over a recording, and their group
    // stands where one of them is in front.
    const overFile = items('file')

    return [
      { id: 'note', title: words.overNote, items: items('note'), silence: getSilenceMessage(over) },
      ...(overFile.length === 0 ? [] : [{ id: 'file', title: words.overFile, items: overFile }]),
      { id: 'window', title: words.overWindow, items: items('window'), silence: words.noneFound },
      { id: 'vault', title: words.overVault, items: items('vault'), silence: words.noneFound },
    ]
  }

  /** What the palette draws where it stands, over the words typed into it. */
  const groupsOf = (step: PendingStep | null, over: CommandTarget, text: string) => {
    if (!step) return getCommandGroups(over, text)
    if (step.step === 'naming') return [steps.getNamingGroup(text)]
    if (step.step === 'address') return [steps.getAddressGroup(text)]
    if (step.step === 'picking') return [steps.getPickingGroup(text)]
    if (step.step === 'choosing') return steps.getChoosingGroups(step, text)
    if (step.step === 'vaults') return [steps.getVaultsGroup(text, step)]
    if (step.step === 'asking') return [steps.getConfirmGroup(step, text)]
    return [steps.getExactlyGroup(step, text)]
  }

  return { groupsOf, typeOf: steps.typeOf, getVaultAside: steps.getVaultAside }
}
