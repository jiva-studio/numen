/**
 * The preset tab kind and the state of its tabs: a preset edited as its schedule settings.
 */
import { shallowRef, type Component } from 'vue'
import { Gauge } from '@lucide/vue'
import { ERRORS } from '../../lib/wire/errors'
import {
  getFileName,
  type BufferHost,
  type CommandNote,
  type MessageWriter,
  type PathRename,
  type TabKindRegistration,
} from '@numen/plugin-sdk'
import { NO_BOUNDS, type Presets, type SettingsBounds } from '../../lib/presets'
import { WORDS as words } from '../lib/words'
import {
  createOpenPreset,
  createFailureReporter,
  createPresetState,
  readPreset,
  type OpenPreset,
  type PresetStateContext,
} from './open'
import { flushWrites } from './writeQueue'
import type { PresetTabState } from './types'

export type { PresetTabState }

/** The word preset tabs are filed under. */
export const PRESET = 'preset'

/** The note type of a deck, whose preset is the one it is scheduled by. */
const DECK_TYPE = 'deck'

/** Where the preset of a note is, or why there is none to open. */
export type PresetPlace =
  | { readonly ok: true; readonly at: string }
  | { readonly ok: false; readonly text: string; readonly severity: 'error' | 'warning' }

/** What the preset tabs of a window are made from. */
export interface PresetTabDeps {
  core: Presets
  buffers: BufferHost
  closeTab: (id: string) => void
  writeMessage: MessageWriter
  today: () => string
  component: Component
}

function applyPathRenames(
  open: Map<string, OpenPreset>,
  titles: Map<string, string>,
  renames: readonly PathRename[],
): void {
  for (const went of renames) {
    const title = titles.get(went.from)
    // Stryker disable next-line ConditionalExpression: Missing title returns undefined on get either way.
    if (title !== undefined) titles.set(went.to, title)
    titles.delete(went.from)
    const one = open.get(went.from)
    if (!one) continue
    open.delete(went.from)
    one.path.value = went.to
    open.set(went.to, one)
  }
}

interface KindDeps {
  context: PresetStateContext
  open: Map<string, OpenPreset>
  states: WeakMap<PresetTabState, OpenPreset>
  flush: (state: PresetTabState) => Promise<void>
  component: Component
}

function createPresetKind(deps: KindDeps): { kind: TabKindRegistration<PresetTabState> } {
  const { context, open, states, flush, component } = deps
  const { titles, bounds, core, today } = context
  const getTitle = (path: string): string =>
    titles.get(path) || getFileName(path) || words.newPreset

  const kind: TabKindRegistration<PresetTabState> = {
    kind: PRESET,
    component,
    icon: Gauge,
    open: (path) => {
      const one = createOpenPreset(path, today(), bounds.value)
      open.set(path, one)
      readPreset(one, { core, bounds, titles, today: today() }).catch(
        createFailureReporter(context),
      )
      const state = createPresetState(one, path, context)
      states.set(state, one)
      return state
    },
    getTitle: (one) => getTitle(one.id),
    identity: (path) => path,
    getTarget: (one) => ({ path: one.id, title: getTitle(one.id) }),
    getOpenTab: (one) => ({ path: one.id }),
    onClose: (one, id) => {
      one.actions.close(id)
      return false
    },
    // The tab state holds no resource beyond the buffer store.
    onDestroy: () => undefined,
    flushPending: (one) => flush(one),
    createFile: (folder, name) => core.createPreset(name, folder),
  }
  return { kind }
}

/**
 * The presets one window has open.
 */
export function usePresetTab(deps: PresetTabDeps) {
  const { core, buffers, closeTab, writeMessage, today, component } = deps
  const titles = new Map<string, string>()
  const bounds = shallowRef<SettingsBounds>(NO_BOUNDS)
  const open = new Map<string, OpenPreset>()

  const closePreset = (path: string) => {
    open.delete(path)
  }

  const context: PresetStateContext = {
    closeTab,
    onClosed: closePreset,
    core,
    bounds,
    writeMessage,
    today,
    titles,
  }

  const getState = (id: string): PresetTabState | undefined => {
    const one = open.get(id)
    return one ? createPresetState(one, id, context) : undefined
  }

  const getTitle = (path: string): string =>
    titles.get(path) || getFileName(path) || words.newPreset

  const readOne = (one: OpenPreset) => readPreset(one, { core, bounds, titles, today: today() })

  const flushOne = async (one: OpenPreset): Promise<void> => {
    await flushWrites(one.writeQueue, {
      path: one.path.value,
      settings: one.settings.value,
      core,
      writeMessage,
    })
  }

  const states = new WeakMap<PresetTabState, OpenPreset>()

  const flushState = async (state: PresetTabState): Promise<void> => {
    const one = states.get(state)
    if (one) await flushOne(one)
  }

  const { kind } = createPresetKind({
    context,
    open,
    states,
    flush: flushState,
    component,
  })

  const applyPathChanges = (
    paths: readonly string[],
    // Stryker disable next-line ArrayDeclaration: an entry that is not a rename names no file, and every walk over renames skips it.
    renames: readonly PathRename[] = [],
  ): void => {
    applyPathRenames(open, titles, renames)
    for (const path of paths) {
      const one = open.get(path)
      if (one) readOne(one).catch(createFailureReporter({ writeMessage }))
    }
  }

  const flush = async (): Promise<void> => {
    await Promise.all([...open.values()].map(flushOne))
  }

  buffers.addFlushHandler(flush)

  /** The preset of a note: the note itself, or the preset a deck is scheduled by. */
  const findPreset = async (note: CommandNote): Promise<PresetPlace> => {
    if (note.type !== DECK_TYPE) return { ok: true, at: note.path }
    const answer = await core.getDeckPreset(note.path)
    if (!answer.ok) return { ok: false, text: ERRORS[answer.error], severity: 'error' }
    const readItem = answer.value.preset
    if (!readItem?.path) return { ok: false, text: words.noPreset, severity: 'warning' }
    if (readItem.title) titles.set(readItem.path, readItem.title)
    return { ok: true, at: readItem.path }
  }

  /** The preset of a note, in a tab of its own, or a message saying why there is none. */
  const openPresetOf = async (
    note: CommandNote,
    openTab: (kind: string, path: string) => Promise<string> | void,
  ): Promise<void> => {
    const place = await findPreset(note)
    if (place.ok) await openTab(PRESET, place.at)
    else writeMessage(place.text, place.severity)
  }

  return { kind, getState, applyPathChanges, getTitle, flush, findPreset, openPresetOf }
}
