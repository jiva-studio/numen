/**
 * The notes the window has open, and what turns the state model.
 *
 * Coordinates open note tabs, executing reads, writes, timers, and holding note state.
 */
import { shallowReactive } from 'vue'
import {
  openTab,
  getState,
  transitionTab,
  DEFAULT_WRITE_LIMITS,
  type Effect,
  type Event,
  type Move,
  type Tab,
} from '../lib/tab'
import type { LinkAddress } from '../lib/note'
import type { OpenNote, Notes, OpenNotesOptions } from '../lib/noteTypes'
import { createNoteQueue } from './queue'
import { createConflictCoordinator } from './conflict'

export * from '../lib/noteTypes'

export function openNotes(core: Notes, how: OpenNotesOptions) {
  const limits = how.limits ?? DEFAULT_WRITE_LIMITS
  const onReplaced = how.onReplaced ?? (() => {})
  const now = how.now

  const tabs = shallowReactive(new Map<string, Tab>())
  const bodies = shallowReactive(new Map<string, string>())
  const addresses = shallowReactive(new Map<string, LinkAddress>())

  const reportedIds = new Set<string>()
  const closeCallbacks = new Map<string, (gone: boolean) => void>()

  function dispatch(id: string, event: Event): void {
    const tab = tabs.get(id)
    if (!tab) return
    applyTransition(id, transitionTab(tab, event, limits))
  }

  const queue = createNoteQueue(
    core,
    dispatch,
    (id) => tabs.has(id),
    (id, addr) => {
      if (addr) addresses.set(id, addr)
      else addresses.delete(id)
    },
    (id) => {
      if (closeCallbacks.has(id)) dispatch(id, { kind: 'close' })
    },
  )

  const conflicts = createConflictCoordinator(dispatch, (id) => tabs.get(id))

  const open = (id: string, path: string = id): void => {
    if (tabs.has(id)) return
    applyTransition(id, openTab(path))
  }

  const reload = (id: string, path: string = id): void => {
    queue.discard(id)
    bodies.delete(id)
    addresses.delete(id)
    reportedIds.delete(id)
    applyTransition(id, openTab(path))
  }

  const close = (id: string): Promise<boolean> =>
    new Promise((done) => {
      if (!tabs.has(id)) return done(true)
      if (reportedIds.has(id)) {
        discard(id)
        return done(true)
      }
      closeCallbacks.set(id, done)
      dispatch(id, { kind: 'close' })
    })

  const getPath = (id: string): string => tabs.get(id)?.path ?? id
  const has = (id: string): boolean => tabs.has(id)
  const getFilePath = (id: string): string => tabs.get(id)?.filePath ?? ''

  const setBody = (id: string, body: string): void => {
    bodies.set(id, body)
    dispatch(id, { kind: 'edited', body, at: now() })
  }

  const save = (id: string): void => dispatch(id, { kind: 'save' })

  const getOpenNote = (id: string): OpenNote => {
    const tab = tabs.get(id)
    const err = tab?.error ?? null
    return {
      path: tab?.path ?? id,
      body: bodies.get(id) ?? '',
      state: tab ? getState(tab) : 'loading',
      error: err,
    }
  }

  const getOpenIds = (): readonly string[] => [...tabs.keys()]

  function applyTransition(id: string, next: ReturnType<typeof transitionTab>): void {
    tabs.set(id, next.tab)
    for (const effect of next.effects) applyEffect(id, effect)

    const isKeptOpen = next.effects.some((effect) => effect.kind === 'keepOpen')
    const isWriting = next.effects.some((effect) => effect.kind === 'write')
    if (isKeptOpen && !isWriting) closeCallbacks.get(id)?.(false)

    if (next.tab.pendingWrite === null) queue.resolveFlush(id)
  }

  function applyEffect(id: string, effect: Effect): void {
    switch (effect.kind) {
      case 'read':
        void queue.read(id, effect.path, effect.generation)
        return
      case 'write':
        void queue.write(id, effect.path, effect.body, effect.seen)
        return
      case 'startTimer':
        queue.startTimer(id, effect.after)
        return
      case 'cancelTimer':
        queue.cancelTimer(id)
        return
      case 'replace':
        bodies.set(id, effect.body)
        onReplaced(getPath(id))
        return
      case 'keepOpen':
        return
      case 'reportError':
        reportedIds.add(id)
        return
      case 'close':
        discard(id)
        return
    }
  }

  function discard(id: string): void {
    queue.discard(id)
    tabs.delete(id)
    bodies.delete(id)
    addresses.delete(id)
    closeCallbacks.get(id)?.(true)
    closeCallbacks.delete(id)
    reportedIds.delete(id)
  }

  const flush = async (): Promise<void> => {
    await Promise.all(getOpenIds().map(close))
  }

  return {
    open,
    reload,
    close,
    flushPending: queue.flushPending,
    getPath,
    has,
    getFilePath,
    setBody,
    applyPathChanges: (paths: readonly string[], renamed: readonly Move[] = []) =>
      conflicts.notifyChanged(getOpenIds, paths, renamed),
    save,
    keep: conflicts.keep,
    take: conflicts.take,
    getOpenNote,
    link: (id: string): LinkAddress | null => addresses.get(id) ?? null,
    getOpenIds,
    getErrorMessage: conflicts.getErrorMessage,
    stale: conflicts.stale,
    flush,
    tabs,
  }
}
