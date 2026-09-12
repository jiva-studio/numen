/**
 * What a step of a command draws while it asks: a name to give, an address to
 * point at, the notes or the vaults to pick from, a list the window holds, and
 * the two answers of a confirmation.
 *
 * Nothing here decides anything or asks the application for anything:
 * `palette.ts` holds the steps and does that.
 */
import type { PaletteGroup, PaletteItem } from '@numen/ui'
import type { NoteType } from '@/shared/file'
import type { Vault } from '@/shared/vaults'
import { isWebUrl } from '../lib/url'
import { CHOSEN, EXACT, NAME, NO, OPEN, PICK, YES, type PendingStep } from '../lib/step'
import type { StepRow } from '../types'
import type { ConfirmWords, RetypeWords } from '../words'
import type { ViewState } from './viewState'

/** The groups a step draws, over the state the palette holds. */
export function createStepGroups(state: ViewState) {
  const { words, lists, found, known, showing, isWorking, failureMessage } = state
  const { getStepTitle, getStepLabel } = state

  /** A name to give, as the one thing the words typed can be. */
  const getNamingGroup = (text: string): PaletteGroup => {
    const name = text.trim()
    return {
      id: 'naming',
      title: words.naming,
      items: name
        ? [
            {
              id: NAME,
              title: `${words.callIt} “${name}”`,
              actions: [{ id: NAME, text: words.callIt }],
            },
          ]
        : [],
      silence: words.typeName,
    }
  }

  /**
   * An address to point at, as the one thing the words typed can be. What is
   * offered is what a browser would go to; the vault reads it again and is what
   * refuses one it cannot fetch.
   */
  const getAddressGroup = (text: string): PaletteGroup => {
    const raw = text.trim()
    return {
      id: 'address',
      title: words.url,
      items: isWebUrl(raw)
        ? [
            {
              id: NAME,
              title: `${words.importIt} “${raw}”`,
              actions: [{ id: NAME, text: words.importIt }],
            },
          ]
        : [],
      silence: raw ? words.notAnAddress : words.typeAddress,
    }
  }

  /** Which of four the note a row of the picking step stands for is. */
  const typeOf = (id: string): NoteType | null =>
    found.value.find((one) => one.path === id)?.type ?? null

  /**
   * The notes the vault turned up. A note found by a heading is that note, and
   * a note found twice is one row.
   */
  const getPickingGroup = (text: string): PaletteGroup => {
    const seen = new Set<string>()
    const items: PaletteItem[] = []
    for (const one of found.value) {
      if (seen.has(one.path)) continue
      seen.add(one.path)
      items.push({
        id: one.path,
        title: one.title || one.path,
        ...(one.heading ? {} : { at: one.at }),
        actions: [{ id: PICK, text: words.travel }],
      })
    }
    return {
      id: 'picking',
      title: words.names,
      items,
      working: isWorking.value,
      silence: failureMessage.value || (text.trim() ? words.noneFound : words.typeNote),
    }
  }

  /**
   * The groups a list the window holds is drawn in, narrowed by the words typed.
   * They are read again on every keystroke, and a row drawn as not to be chosen
   * is not chosen.
   */
  const getChoosingGroups = (step: PendingStep, text: string): readonly PaletteGroup[] => {
    const word = text.trim().toLowerCase()
    return lists.getStepGroups(step.command.id, text).map((group) => ({
      id: group.id,
      title: group.title,
      items: group.items.map((one) => getRowItem(one, word)).filter((item) => item !== null),
      silence: group.silence ?? words.noneFound,
    }))
  }

  /** One such row, and nothing where the words typed leave it out. */
  const getRowItem = (one: StepRow, word: string): PaletteItem | null => {
    const found = word === '' ? -1 : one.title.toLowerCase().indexOf(word)
    if (word !== '' && found < 0) return null
    return {
      id: one.id,
      title: one.title,
      ...(found < 0 ? {} : { at: [{ from: found, to: found + word.length }] }),
      ...(one.detail ? { detail: one.detail } : {}),
      ...(one.disabled ? { disabled: true } : {}),
      actions: [{ id: CHOSEN, text: words.chooses }],
    }
  }

  /**
   * Why a vault the list holds is drawn and not chosen: its folder is not
   * there, or it is the one the window is showing. A vault that can be chosen
   * is marked with nothing.
   */
  const getVaultAside = (one: Vault): string =>
    one.missing ? words.gone : one.id === showing.value ? words.current : ''

  /**
   * The vaults the installation holds. The two it will not take are marked
   * ahead of their folder, and cannot be chosen. The folder is what tells one
   * vault from another, so it keeps the room.
   */
  const getVaultsGroup = (text: string, step: PendingStep): PaletteGroup => {
    const word = text.trim().toLowerCase()
    const items: PaletteItem[] = known.value
      .filter((one) => word === '' || one.name.toLowerCase().includes(word))
      .map((one) => {
        const why = getVaultAside(one)
        return {
          id: one.id,
          title: one.name,
          detail: why ? `${why} · ${one.path}` : one.path,
          ...(why ? { disabled: true } : {}),
          actions: [{ id: OPEN, text: step.command.text }],
        }
      })
    return {
      id: 'vaults',
      title: words.vaults,
      items,
      working: isWorking.value,
      silence: failureMessage.value || words.noneFound,
    }
  }

  /**
   * The two answers put before a note goes to the trash. The one that changes
   * nothing is drawn first, and it is the one the keyboard opens on. Each is
   * reached by the name it is offered under, as an item of any other step is.
   */
  const getConfirmGroup = (step: PendingStep, text: string): PaletteGroup => {
    const word = text.trim().toLowerCase()
    const { keeps = '', kept = '', action = '', then = '' }: Partial<ConfirmWords> =
      step.command.answers ?? {}
    const items: PaletteItem[] = [
      {
        id: NO,
        title: keeps,
        detail: kept,
        actions: [{ id: NO, text: keeps }],
      },
      {
        id: YES,
        title: `${action} ${getStepLabel(step)}`,
        detail: then,
        actions: [{ id: YES, text: action }],
      },
    ]
    return {
      id: 'asking',
      title: words.asking,
      items: word
        ? items.filter((one) => (one.actions?.[0]?.text ?? '').toLowerCase().includes(word))
        : items,
      silence: words.answer,
    }
  }

  /** The name typed back, which is the one thing that reaches destroying. */
  const getExactlyGroup = (step: PendingStep, text: string): PaletteGroup => {
    const title = getStepTitle(step)
    const { action = '', then = '' }: Partial<RetypeWords> = step.command.warns ?? {}
    return {
      id: 'exactly',
      title: words.exactly,
      items: [
        {
          id: EXACT,
          title: `${action} “${title}”`,
          detail: then,
          disabled: text.trim() !== title,
          actions: [{ id: EXACT, text: action }],
        },
      ],
    }
  }

  return {
    getNamingGroup,
    getAddressGroup,
    getPickingGroup,
    getChoosingGroups,
    getVaultsGroup,
    getConfirmGroup,
    getExactlyGroup,
    typeOf,
    getVaultAside,
  }
}
