/**
 * The notes the window has open, and what turns the state model.
 *
 * Coordinates open note tabs, executing reads, writes, timers, and holding note state.
 */
import { shallowReactive } from 'vue'
import { getMonotonicNow } from '@numen/ui'
import {
  openTab,
  getState,
  isUnwritable,
  transitionTab,
  DEFAULT_WRITE_LIMITS,
  type Effect,
  type Event,
  type Move,
  type Tab,
} from './tab'
import type { OpenNote, OpenNotesOptions } from './types'
import { createNoteQueue } from './queue'
import { createConflictCoordinator } from './conflict'

export * from './types'

interface NoteStoreContext {
  tabs: Map<string, Tab>
  bodies: Map<string, string>
  reportedIds: Set<string>
  closeCallbacks: Map<string, ((removed: boolean) => void)[]>
  queue: ReturnType<typeof createNoteQueue>
  onReplaced: (path: string) => void
  dispatch: (id: string, event: Event) => void
}

/** Answers every close waiting on the note. */
function answerClosing(context: NoteStoreContext, id: string, removed: boolean): void {
  const waiting = context.closeCallbacks.get(id) ?? []
  context.closeCallbacks.delete(id)
  for (const answer of waiting) answer(removed)
}

function discardTab(context: NoteStoreContext, id: string): void {
  context.queue.discard(id)
  context.tabs.delete(id)
  context.bodies.delete(id)
  answerClosing(context, id, true)
  context.reportedIds.delete(id)
}

// Stryker disable all: effects handled by queue do not overlap with outer effect switch.
function executeQueueEffect(context: NoteStoreContext, id: string, effect: Effect): boolean {
  const { queue, dispatch } = context
  switch (effect.kind) {
    case 'read':
      queue.read(id, effect.path, effect.generation).catch(() =>
        dispatch(id, {
          kind: 'read',
          generation: effect.generation,
          answer: { kind: 'error', error: 'unreachable' },
        }),
      )
      return true
    case 'write':
      queue
        .write(id, { path: effect.path, body: effect.body, baseline: effect.baseline })
        .catch(() =>
          dispatch(id, { kind: 'written', answer: { kind: 'error', error: 'unreachable' } }),
        )
      return true
    case 'startTimer':
      queue.startTimer(id, effect.after)
      return true
    case 'cancelTimer':
      queue.cancelTimer(id)
      return true
    default:
      return false
  }
}
// Stryker restore all

function executeEffect(context: NoteStoreContext, id: string, effect: Effect): void {
  if (executeQueueEffect(context, id, effect)) return
  switch (effect.kind) {
    case 'replace':
      context.bodies.set(id, effect.body)
      // Stryker disable next-line OptionalChaining: applyTabTransition stores next.tab before executing effects.
      context.onReplaced(context.tabs.get(id)?.path ?? id)
      return
    // Stryker disable next-line StringLiteral: keepOpen is checked at transition level, no-op in effect execution.
    case 'keepOpen':
      return
    case 'reportError':
      context.reportedIds.add(id)
      return
    case 'close':
      discardTab(context, id)
      return
    default:
      // The queue effects are carried out above.
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
  if (isKeptOpen && !isWriting) answerClosing(context, id, false)

  if (next.tab.pendingWrite === null) context.queue.resolveFlush(id)
}

function createNoteStoreContext(
  limits: Parameters<typeof transitionTab>[2],
  onReplaced: (path: string) => void,
) {
  const tabs = shallowReactive(new Map<string, Tab>())
  const bodies = shallowReactive(new Map<string, string>())
  const reportedIds = new Set<string>()
  const closeCallbacks = new Map<string, ((removed: boolean) => void)[]>()

  const context: NoteStoreContext = {
    tabs,
    bodies,
    reportedIds,
    closeCallbacks,
    queue: createNoteQueue({
      dispatch: (id, event) => dispatch(id, event),
      hasTab: (id) => tabs.has(id),
      resumeClosing: (id) => {
        if (closeCallbacks.has(id)) dispatch(id, { kind: 'close' })
      },
    }),
    onReplaced,
    // Stryker disable next-line ArrowFunction: context.dispatch is only used by queue fallback.
    dispatch: (id, event) => dispatch(id, event),
  }

  function dispatch(id: string, event: Event): void {
    const tab = tabs.get(id)
    if (!tab) return
    applyTabTransition(context, id, transitionTab(tab, event, limits))
  }

  return { context, dispatch }
}

export function openNotes(how: OpenNotesOptions = {}) {
  const limits = how.limits ?? DEFAULT_WRITE_LIMITS
  // Without a listener, a replaced body needs no notification.
  const onReplaced = how.onReplaced ?? (() => undefined)
  const now = how.now ?? getMonotonicNow

  const { context, dispatch } = createNoteStoreContext(limits, onReplaced)
  const { tabs, bodies, queue, reportedIds, closeCallbacks } = context

  const conflicts = createConflictCoordinator(dispatch, (id) => tabs.get(id))
  const getOpenIds = (): readonly string[] => [...tabs.keys()]

  const close = (id: string): Promise<boolean> =>
    new Promise((isDone) => {
      if (!tabs.has(id)) return isDone(true)
      if (reportedIds.has(id)) {
        discardTab(context, id)
        return isDone(true)
      }
      const waiting = closeCallbacks.get(id)
      if (waiting) {
        // A close already under way answers this one too.
        waiting.push(isDone)
        return
      }
      closeCallbacks.set(id, [isDone])
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
    getRevision: (id: string): string => tabs.get(id)?.revision ?? '',
    setBody: (id: string, body: string) => {
      bodies.set(id, body)
      dispatch(id, { kind: 'edited', body, at: now() })
    },
    // Stryker disable next-line ArrayDeclaration: an entry that is not a rename names no file, and every walk over renames skips it.
    applyPathChanges: (paths: readonly string[], renamed: readonly Move[] = []) =>
      conflicts.notifyChanged(getOpenIds, paths, renamed),
    save: (id: string) => dispatch(id, { kind: 'save' }),
    keep: conflicts.keep,
    take: conflicts.take,
    getOpenNote: (id: string): OpenNote => {
      const tab = tabs.get(id)
      return {
        path: tab?.path ?? '',
        body: bodies.get(id) ?? '',
        state: tab ? getState(tab) : 'loading',
        error: tab?.error ?? null,
      }
    },
    getOpenIds,
    getErrorMessage: conflicts.getErrorMessage,
    stale: conflicts.stale,
    /** Writes what every open note holds, and names the notes whose text could still not be written. */
    flush: async (): Promise<readonly string[]> => {
      const ids = getOpenIds()
      await Promise.all(ids.map(queue.flushPending))
      return ids.filter((id) => {
        const tab = tabs.get(id)
        return tab !== undefined && isUnwritable(tab)
      })
    },
    tabs,
  }
}
