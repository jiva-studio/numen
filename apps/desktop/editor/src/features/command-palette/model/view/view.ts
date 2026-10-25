/**
 * What the palette draws at its current position: the commands offered over what is in
 * front, and the groups whichever step is open asks in.
 *
 * Every one of these is the state read again, so a vault moving under an open
 * step is drawn as it is. Nothing here decides anything or asks the
 * application for anything: `palette.ts` holds the steps and does that.
 */
import type { PaletteGroup, PaletteItem } from '@numen/ui'
import { getGroupCommands } from '../../lib/commands/commands'
import type { PendingStep } from '../../lib/step/step'
import type { Command, CommandGroup, CommandTarget } from '../../types'
import { createStepGroups } from './stepGroups'
import type { ViewState } from './viewState'

export type { ViewState } from './viewState'

/** The view one palette draws, over the state that palette holds. */
export function getPaletteView(state: ViewState) {
  const { words, runs } = state
  const steps = createStepGroups(state)

  /** One command as it is drawn, and nothing where the words typed leave it out. */
  const renderItem = (one: Command, target: CommandTarget, word: string): PaletteItem | null => {
    const found = word === '' ? -1 : one.text.toLowerCase().indexOf(word)
    if (word !== '' && found < 0) return null
    const also = one.also ? state.byId.get(one.also) : undefined
    return {
      id: one.id,
      title: one.text,
      ...(found < 0 ? {} : { at: [{ from: found, to: found + word.length }] }),
      ...(one.keys ? { keys: one.keys } : {}),
      actions: [
        { id: one.id, text: one.text },
        ...(also?.isOffered(target, runs) ? [{ id: also.id, text: also.text }] : []),
      ],
    }
  }

  const _getSilenceMessage = (target: CommandTarget): string => {
    if (!target.isReady) return words.noVault
    if (target.path) return words.noneFound
    return words.noNote
  }

  /** Every command offered over what is in front, in the groups it holds. */
  const getCommandGroups = (target: CommandTarget, text: string): readonly PaletteGroup[] => {
    const word = text.trim().toLowerCase()
    const items = (group: CommandGroup): readonly PaletteItem[] =>
      getGroupCommands(state.commands, group)
        .filter((one) => one.isOffered(target, runs))
        .map((one) => renderItem(one, target, word))
        .filter((item) => item !== null)

    const note = items('note')
    const file = items('file')
    const window = items('window')
    const vault = items('vault')

    const groups: PaletteGroup[] = []
    if (note.length > 0) {
      groups.push({ id: 'note', title: words.noteGroupTitle, items: note })
    }
    if (file.length > 0) {
      groups.push({ id: 'file', title: words.overFile, items: file })
    }
    if (window.length > 0) {
      groups.push({ id: 'window', title: words.overWindow, items: window })
    }
    if (vault.length > 0) {
      groups.push({ id: 'vault', title: words.overVault, items: vault })
    }
    return groups
  }

  /** What the palette draws at its current position, over the words typed into it. */
  const createGroups = (step: PendingStep | null, target: CommandTarget, text: string) => {
    if (!step) return getCommandGroups(target, text)
    if (step.step === 'naming') return [steps.getNamingGroup(text)]
    if (step.step === 'arguments') return [steps.getArgumentGroup(step, text)]
    if (step.step === 'address') return [steps.getAddressGroup(text)]
    if (step.step === 'picking') return [steps.getPickingGroup(text)]
    if (step.step === 'choosing') return steps.getChoosingGroups(step, text)
    if (step.step === 'vaults') return [steps.getVaultsGroup(text, step)]
    if (step.step === 'asking') return [steps.getConfirmGroup(step, text)]
    return [steps.getExactlyGroup(step, text)]
  }

  return { createGroups, findNoteType: steps.findNoteType, getVaultAside: steps.getVaultAside }
}
