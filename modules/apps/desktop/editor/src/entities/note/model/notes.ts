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

interface NoteStoreContext {
  tabs: Map<string, Tab>
  bodies: Map<string, string>
  addresses: Map<string, LinkAddress>
  reportedIds: Set<string>
  closeCallbacks: Map<string, (gone: boolean) => void>
  queue: ReturnType<typeof createNoteQueue>
  onReplaced: (path: string) => void
}

function discardTab(context: NoteStoreContext, id: string): void {
  context.queue.discard(id)
  context.tabs.delete(id)
  context.bodies.delete(id)
  context.addresses.delete(id)
  context.closeCallbacks.get(id)?.(true)
  context.closeCallbacks.delete(id)
  context.reportedIds.delete(id)
}

function executeEffect(context: NoteStoreContext, id: string, effect: Effect): void {
  switch (effect.kind) {
    case 'read':
      void context.queue.read(id, effect.path, effect.generation)
      return
    case 'write':
      void context.queue.write(id, { path: effect.path, body: effect.body, baseline: effect.seen })
      return
    case 'startTimer':
      context.queue.startTimer(id, effect.after)
      return
    case 'cancelTimer':
      context.queue.cancelTimer(id)
      return
    case 'replace':
      context.bodies.set(id, effect.body)
      context.onReplaced(context.tabs.get(id)?.path ?? id)
      return
    case 'keepOpen':
      return
    case 'reportError':
      context.reportedIds.add(id)
      return
    case 'close':
      discardTab(context, id)
      return
  }
}

function applyTabTransition(
  context: NoteStoreContext,
  id: string,
  next: ReturnType<typeof transitionTab>,
): void {
  context.tabs.set(id, next.tab)
  for (const effect of next.effects) executeEffect(context, id, effect)

  const isKeptOpen = next.effects.some((effect) => effect.kind === 'keepOpen')
  const isWriting = next.effects.some((effect) => effect.kind === 'write')
  if (isKeptOpen && !isWriting) context.closeCallbacks.get(id)?.(false)

  if (next.tab.pendingWrite === null) context.queue.resolveFlush(id)
}

export function openNotes(core: Notes, how: OpenNotesOptions = {}) {
  const limits = how.limits ?? DEFAULT_WRITE_LIMITS
  const onReplaced = how.onReplaced ?? (() => {})
  const now = how.now ?? (() => Date.now())

  const tabs = shallowReactive(new Map<string, Tab>())
  const bodies = shallowReactive(new Map<string, string>())
  const addresses = shallowReactive(new Map<string, LinkAddress>())
  const reportedIds = new Set<string>()
  const closeCallbacks = new Map<string, (gone: boolean) => void>()

  let context!: NoteStoreContext

  function dispatch(id: string, event: Event): void {
    const tab = tabs.get(id)
    if (!tab) return
    applyTabTransition(context, id, transitionTab(tab, event, limits))
  }

  const queue = createNoteQueue(core, {
    dispatch,
    hasTab: (id) => tabs.has(id),
    setAddress: (id, addr) => {
      if (addr) addresses.set(id, addr)
      else addresses.delete(id)
    },
    resumeClosing: (id) => {
      if (closeCallbacks.has(id)) dispatch(id, { kind: 'close' })
    },
  })

  context = { tabs, bodies, addresses, reportedIds, closeCallbacks, queue, onReplaced }
  const conflicts = createConflictCoordinator(dispatch, (id) => tabs.get(id))
  const getOpenIds = (): readonly string[] => [...tabs.keys()]

  const close = (id: string): Promise<boolean> =>
    new Promise((done) => {
      if (!tabs.has(id)) return done(true)
      if (reportedIds.has(id)) {
        discardTab(context, id)
        return done(true)
      }
      closeCallbacks.set(id, done)
      dispatch(id, { kind: 'close' })
    })

  return {
    open: (id: string, path: string = id) => {
      if (!tabs.has(id)) applyTabTransition(context, id, openTab(path))
    },
    reload: (id: string, path: string = id) => {
      discardTab(context, id)
      applyTabTransition(context, id, openTab(path))
    },
    close,
    flushPending: queue.flushPending,
    getPath: (id: string): string => tabs.get(id)?.path ?? id,
    has: (id: string): boolean => tabs.has(id),
    getFilePath: (id: string): string => tabs.get(id)?.filePath ?? '',
    setBody: (id: string, body: string) => {
      bodies.set(id, body)
      dispatch(id, { kind: 'edited', body, at: now() })
    },
    applyPathChanges: (paths: readonly string[], renamed: readonly Move[] = []) =>
      conflicts.notifyChanged(getOpenIds, paths, renamed),
    save: (id: string) => dispatch(id, { kind: 'save' }),
    keep: conflicts.keep,
    take: conflicts.take,
    getOpenNote: (id: string): OpenNote => {
      const tab = tabs.get(id)
      return {
        path: tab?.path ?? id,
        body: bodies.get(id) ?? '',
        state: tab ? getState(tab) : 'loading',
        error: tab?.error ?? null,
      }
    },
    link: (id: string): LinkAddress | null => addresses.get(id) ?? null,
    getOpenIds,
    getErrorMessage: conflicts.getErrorMessage,
    stale: conflicts.stale,
    flush: async () => {
      await Promise.all(getOpenIds().map(close))
    },
    tabs,
  }
}
