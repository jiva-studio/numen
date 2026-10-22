/**
 * What a step of a command draws while it asks: a name to give, an address to
 * point at, the notes or the vaults to pick from, a list the window holds, and
 * the two answers of a confirmation.
 *
 * Nothing here decides anything or asks the application for anything:
 * `palette.ts` holds the steps and does that.
 */
import type { PaletteGroup, PaletteItem } from '@numen/ui'
import type { NoteType } from '@/entities/file'
import type { Vault } from '@/entities/vault'
import { isWebUrl } from '../lib/url'
import { CHOSEN, EXACT, NAME, NO, OPEN, PICK, YES, type PendingStep } from '../lib/step'
import type { StepRow } from '../types'
import type { ConfirmWords, RetypeWords, Words } from '../words'
import type { NameMatch } from './search'
import type { ViewState } from './viewState'

function getNamingGroup(text: string, words: Words): PaletteGroup {
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

function getAddressGroup(text: string, words: Words): PaletteGroup {
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

function getRowItem(one: StepRow, word: string, words: Words): PaletteItem | null {
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

function getVaultAside(one: Vault, showingId: string, words: Words): string {
  if (one.isMissing) return words.gone
  if (one.id === showingId) return words.current
  return ''
}

function getConfirmGroup(
  step: PendingStep,
  text: string,
  ...[words, label]: [words: Words, label: string]
): PaletteGroup {
  const word = text.trim().toLowerCase()
  const {
    keeps = '',
    kept = '',
    action = '',
    detail = '',
  }: Partial<ConfirmWords> = step.command.answers ?? {}
  const answers: readonly (readonly [string, PaletteItem])[] = [
    [
      keeps,
      {
        id: NO,
        title: keeps,
        detail: kept,
        actions: [{ id: NO, text: keeps }],
      },
    ],
    [
      action,
      {
        id: YES,
        title: `${action} ${label}`,
        detail,
        actions: [{ id: YES, text: action }],
      },
    ],
  ]
  return {
    id: 'asking',
    title: words.asking,
    items: answers
      .filter(([itemText]) => itemText.toLowerCase().includes(word))
      .map(([, item]) => item),
    silence: words.answer,
  }
}

function getExactlyGroup(
  step: PendingStep,
  text: string,
  ...[words, title]: [words: Words, title: string]
): PaletteGroup {
  const { action = '', detail = '' }: Partial<RetypeWords> = step.command.warns ?? {}
  return {
    id: 'exactly',
    title: words.exactly,
    items: [
      {
        id: EXACT,
        title: `${action} “${title}”`,
        detail,
        disabled: text.trim() !== title,
        actions: [{ id: EXACT, text: action }],
      },
    ],
  }
}

function createPickingGroup(
  found: readonly NameMatch[],
  text: string,
  meta: { words: Words; isWorking: boolean; failureMessage: string },
): PaletteGroup {
  const observed = new Set<string>()
  const items: PaletteItem[] = []
  for (const one of found) {
    if (observed.has(one.path)) continue
    observed.add(one.path)
    items.push({
      id: one.path,
      title: one.title || one.path,
      ...(one.heading ? {} : { at: one.at }),
      actions: [{ id: PICK, text: meta.words.travel }],
    })
  }
  return {
    id: 'picking',
    title: meta.words.names,
    items,
    isWorking: meta.isWorking,
    silence: meta.failureMessage || (text.trim() ? meta.words.noneFound : meta.words.typeNote),
  }
}

function createVaultsGroup(
  recognized: readonly Vault[],
  step: PendingStep,
  meta: { text: string; words: Words; showing: string; isWorking: boolean; failureMessage: string },
): PaletteGroup {
  const word = meta.text.trim().toLowerCase()
  const items: PaletteItem[] = recognized
    .filter((one) => one.name.toLowerCase().includes(word))
    .map((one) => {
      const why = getVaultAside(one, meta.showing, meta.words)
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
    title: meta.words.vaults,
    items,
    isWorking: meta.isWorking,
    silence: meta.failureMessage || meta.words.noneFound,
  }
}

/** The groups a step draws, over the state the palette holds. */
export function createStepGroups(state: ViewState) {
  const { words, lists, found, known, showing, isWorking, failureMessage } = state
  const { getStepTitle, getStepLabel } = state

  const findNoteType = (id: string): NoteType | null =>
    found.value.find((one) => one.path === id)?.type ?? null

  const getPickingGroup = (text: string): PaletteGroup =>
    createPickingGroup(found.value, text, {
      words,
      isWorking: isWorking.value,
      failureMessage: failureMessage.value,
    })

  const getChoosingGroups = (step: PendingStep, text: string): readonly PaletteGroup[] => {
    const word = text.trim().toLowerCase()
    return lists.getStepGroups(step.command.id, text).map((group) => ({
      id: group.id,
      title: group.title,
      items: group.items.map((one) => getRowItem(one, word, words)).filter((item) => item !== null),
      silence: group.silence ?? words.noneFound,
    }))
  }

  const getVaultsGroup = (text: string, step: PendingStep): PaletteGroup =>
    createVaultsGroup(known.value, step, {
      text,
      words,
      showing: showing.value,
      isWorking: isWorking.value,
      failureMessage: failureMessage.value,
    })

  return {
    getNamingGroup: (text: string) => getNamingGroup(text, words),
    getAddressGroup: (text: string) => getAddressGroup(text, words),
    getPickingGroup,
    getChoosingGroups,
    getVaultsGroup,
    getConfirmGroup: (step: PendingStep, text: string) =>
      getConfirmGroup(step, text, words, getStepLabel(step)),
    getExactlyGroup: (step: PendingStep, text: string) =>
      getExactlyGroup(step, text, words, getStepTitle(step)),
    findNoteType,
    getVaultAside: (one: Vault) => getVaultAside(one, showing.value, words),
  }
}
