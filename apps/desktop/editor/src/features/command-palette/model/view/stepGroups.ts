/**
 * What a command step draws while it prompts: a name to enter, the notes or
 * vaults to choose from, a list the window holds, and the two answers of a
 * confirmation.
 *
 * Nothing here makes decisions or calls the application: `palette.ts` holds the
 * steps and does that.
 */
import type { PaletteGroup, PaletteItem } from '@numen/workbench'
import type { Vault } from '@/entities/vault'
import { getPrompt, type PendingStep } from '../../lib/step/step'
import type { StepRow } from '../../types'
import type { Words } from '../../words'
import type { ConfirmWords, RetypeWords } from '@/entities/command'
import type { NameMatch } from '../search/search'
import type { ViewState } from './viewState'

/** The ending of the id of the choice that leaves a confirmation without running the command. */
export const KEEP_SUFFIX = ':keep'

function getNamingGroup(step: PendingStep, text: string, words: Words): PaletteGroup {
  const name = text.trim()
  return {
    id: 'naming',
    title: words.naming,
    items: name
      ? [
          {
            id: step.command.id,
            title: `${words.callIt} “${name}”`,
            actions: [{ id: step.command.id, text: words.callIt }],
          },
        ]
      : [],
    silence: words.typeName,
  }
}

function getArgumentGroup(step: PendingStep, text: string, words: Words): PaletteGroup {
  const prompt = getPrompt(step)
  const value = text.trim()
  const title = prompt?.title ?? ''
  const item = (itemTitle: string, action: string): PaletteItem => ({
    id: step.command.id,
    title: itemTitle,
    actions: [{ id: step.command.id, text: action }],
  })
  let items: PaletteItem[] = []
  if (value) items = [item(`${title}: “${value}”`, words.givesIt)]
  else if (prompt && !prompt.isRequired)
    items = [item(`${words.leavesOut} ${title}`, words.leavesOut)]
  return { id: 'arguments', title, items, silence: title }
}

function getRowItem(
  one: StepRow,
  word: string,
  ctx: { command: string; words: Words },
): PaletteItem | null {
  const found = word === '' ? -1 : one.title.toLowerCase().indexOf(word)
  if (word !== '' && found < 0) return null
  return {
    id: one.id,
    title: one.title,
    ...(found < 0 ? {} : { at: [{ start: found, end: found + word.length }] }),
    ...(one.detail ? { detail: one.detail } : {}),
    ...(one.isDisabled ? { isDisabled: true } : {}),
    actions: [{ id: ctx.command, text: ctx.words.chooses }],
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
    retainedNotice = '',
    action = '',
    detail = '',
  }: Partial<ConfirmWords> = step.command.answers ?? {}
  const answers: readonly (readonly [string, PaletteItem])[] = [
    [
      keeps,
      {
        id: `${step.command.id}${KEEP_SUFFIX}`,
        title: keeps,
        detail: retainedNotice,
        actions: [{ id: `${step.command.id}${KEEP_SUFFIX}`, text: keeps }],
      },
    ],
    [
      action,
      {
        id: step.command.id,
        title: `${action} ${label}`,
        detail,
        actions: [{ id: step.command.id, text: action }],
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
        id: step.command.id,
        title: `${action} “${title}”`,
        detail,
        isDisabled: text.trim() !== title,
        actions: [{ id: step.command.id, text: action }],
      },
    ],
  }
}

function createPickingGroup(
  step: PendingStep,
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
      actions: [{ id: step.command.id, text: meta.words.travel }],
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
        ...(why ? { isDisabled: true } : {}),
        actions: [{ id: step.command.id, text: step.command.text }],
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

/** What the groups of the steps that prompt or confirm read. */
export type StepGroupsState = Pick<ViewState, 'words' | 'lists' | 'getStepTitle' | 'getStepLabel'>

/** What the groups of the steps that choose a note or a vault read. */
export type ChoiceGroupsState = Pick<
  ViewState,
  'words' | 'found' | 'known' | 'showing' | 'isWorking' | 'failureMessage'
>

/** The groups of the steps that ask for a name, an argument, an item or a confirmation. */
export function createStepGroups(state: StepGroupsState) {
  const { words, lists } = state

  const getItemGroups = (step: PendingStep, text: string): readonly PaletteGroup[] => {
    const word = text.trim().toLowerCase()
    return lists.getStepGroups(step.command.id, text).map((group) => ({
      id: group.id,
      title: group.title,
      items: group.items
        .map((one) => getRowItem(one, word, { command: step.command.id, words }))
        .filter((item) => item !== null),
      silence: group.silence ?? words.noneFound,
    }))
  }

  return {
    getNamingGroup: (step: PendingStep, text: string) => getNamingGroup(step, text, words),
    getArgumentGroup: (step: PendingStep, text: string) => getArgumentGroup(step, text, words),
    getItemGroups,
    getConfirmGroup: (step: PendingStep, text: string) =>
      getConfirmGroup(step, text, words, state.getStepLabel(step)),
    getExactlyGroup: (step: PendingStep, text: string) =>
      getExactlyGroup(step, text, words, state.getStepTitle(step)),
  }
}

/** The groups of the steps that choose a note or a vault, over what the search found. */
export function createChoiceGroups(state: ChoiceGroupsState) {
  const { words, found, known, showing, isWorking, failureMessage } = state

  const findTabKind = (id: string): string | null => {
    const note = found.value.find((one) => one.path === id)
    return note ? (note.tabKind ?? '') : null
  }

  const getPickingGroup = (step: PendingStep, text: string): PaletteGroup =>
    createPickingGroup(step, found.value, text, {
      words,
      isWorking: isWorking.value,
      failureMessage: failureMessage.value,
    })

  const getVaultsGroup = (text: string, step: PendingStep): PaletteGroup =>
    createVaultsGroup(known.value, step, {
      text,
      words,
      showing: showing.value,
      isWorking: isWorking.value,
      failureMessage: failureMessage.value,
    })

  return {
    getPickingGroup,
    getVaultsGroup,
    findTabKind,
    getVaultAside: (one: Vault) => getVaultAside(one, showing.value, words),
  }
}
