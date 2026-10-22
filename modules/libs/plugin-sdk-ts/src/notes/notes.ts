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
} from './tab'
import type { LinkAddress, OpenNote, Notes, OpenNotesOptions } from './types'
import { createNoteQueue } from './queue'
import { createConflictCoordinator } from './conflict'

export * from './types'

interface NoteStoreContext {
  tabs: Map<string, Tab>
  bodies: Map<string, string>
  addresses: Map<string, LinkAddress>
  reportedIds: Set<string>
  closeCallbacks: Map<string, (removed: boolean) => void>
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

// Stryker disable next-line all: effects handled by queue do not overlap with outer effect switch.
function executeQueueEffect(queue: NoteStoreContext['queue'], id: string, effect: Effect): boolean {
  switch (effect.kind) {
    case 'read':
      void queue.read(id, effect.path, effect.generation)
      return true
    case 'write':
      void queue.write(id, { path: effect.path, body: effect.body, baseline: effect.seen })
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

function executeEffect(context: NoteStoreContext, id: string, effect: Effect): void {
  if (executeQueueEffect(context.queue, id, effect)) return
  switch (effect.kind) {
    case 'replace':
      context.bodies.set(id, effect.body)
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
  // Stryker disable next-line OptionalChaining: closeCallbacks entry is deleted only after close dispatch completes.
  if (isKeptOpen && !isWriting) context.closeCallbacks.get(id)?.(false)

  if (next.tab.pendingWrite === null) context.queue.resolveFlush(id)
}

function createNoteStoreContext(
  core: Notes,
  limits: Parameters<typeof transitionTab>[2],
  onReplaced: (path: string) => void,
) {
  const tabs = shallowReactive(new Map<string, Tab>())
  const bodies = shallowReactive(new Map<string, string>())
  const addresses = shallowReactive(new Map<string, LinkAddress>())
  const reportedIds = new Set<string>()
  const closeCallbacks = new Map<string, (removed: boolean) => void>()

  const context: NoteStoreContext = {
    tabs,
    bodies,
    addresses,
    reportedIds,
    closeCallbacks,
    queue: null as unknown as ReturnType<typeof createNoteQueue>,
    onReplaced,
  }

  function dispatch(id: string, event: Event): void {
    const tab = tabs.get(id)
    if (!tab) return
    applyTabTransition(context, id, transitionTab(tab, event, limits))
  }

  context.queue = createNoteQueue(core, {
    dispatch,
    hasTab: (id) => tabs.has(id),
    setAddress: (id, addr) => {
      // Stryker disable next-line ConditionalExpression: deleted key and key set to undefined both evaluate to null via link() fallback.
      if (addr) addresses.set(id, addr)
      else addresses.delete(id)
    },
    resumeClosing: (id) => {
      if (closeCallbacks.has(id)) dispatch(id, { kind: 'close' })
    },
  })

  return { context, dispatch }
}

export function openNotes(core: Notes, how: OpenNotesOptions = {}) {
  const limits = how.limits ?? DEFAULT_WRITE_LIMITS
  const onReplaced = how.onReplaced ?? (() => {})
  const now = how.now ?? (() => Date.now())

  const { context, dispatch } = createNoteStoreContext(core, limits, onReplaced)
  const { tabs, bodies, addresses, queue, reportedIds, closeCallbacks } = context

  const conflicts = createConflictCoordinator(dispatch, (id) => tabs.get(id))
  const getOpenIds = (): readonly string[] => [...tabs.keys()]

  const close = (id: string): Promise<boolean> =>
    new Promise((isDone) => {
      if (!tabs.has(id)) return isDone(true)
      if (reportedIds.has(id)) {
        discardTab(context, id)
        return isDone(true)
      }
      closeCallbacks.set(id, isDone)
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
    // Stryker disable next-line ArrayDeclaration: an entry that is not a rename names no file, and every walk over renames skips it.
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
