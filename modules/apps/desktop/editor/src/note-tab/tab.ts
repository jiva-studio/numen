/**
 * Pure state machine transitions and effects for note tab editing and auto-save.
 */
import {
  dirty,
  stateOf,
  waiting,
  MENDABLE_ERRORS,
  type Event,
  type FilePath,
  type Move,
  type NoteBaseline,
  type ReadResult,
  type Tab,
  type Transition,
  type WriteLimits,
  type WriteResult,
} from './tabState'

export * from './tabState'

export const tabAfter = (tab: Tab, event: Event, limits: WriteLimits = waiting): Transition => {
  switch (event.kind) {
    case 'read':
      return answered(tab, event.generation, event.answer)
    case 'typed':
      return typed(tab, event.body, event.at, limits)
    case 'fired':
      return fired(tab)
    case 'written':
      return landed(tab, event.answer)
    case 'changed':
      return changed(tab, event.paths, event.renamed)
    case 'saving':
      return saving(tab)
    case 'settling':
      return settling(tab)
    case 'keeping':
      return keeping(tab)
    case 'taking':
      return taking(tab)
    case 'closing':
      return closing(tab)
  }
}

const still = (tab: Tab): Transition => ({ tab, effects: [] })

/**
 * A body equal to what is shown replaces nothing.
 */
const shows = (tab: Tab, body: string, filePath: FilePath | null): Transition => ({
  tab: { ...tab, written: body, filePath, shown: body, since: null },
  effects: body === tab.shown ? [] : [{ kind: 'replace', body }],
})

/** What the tab last saw, for a write to present. */
const seenOf = (tab: Tab): NoteBaseline | null =>
  tab.written === null || tab.filePath === null ? null : { prose: tab.written, path: tab.filePath }

/** A write of what is on screen now, presenting what it is given. */
const begins = (tab: Tab, seen: NoteBaseline | null): Transition => ({
  tab: { ...tab, pendingWrite: tab.shown, hasPendingWrite: false, isStale: false },
  effects: [{ kind: 'write', path: tab.path, body: tab.shown, seen }],
})

/**
 * A read is applied to a tab with nothing unsaved and only for the generation asked for last.
 */
const answered = (tab: Tab, generation: number, answer: ReadResult): Transition => {
  const state = stateOf(tab)
  const loading = state === 'loading'
  const stale = generation !== tab.reading
  if (answer.kind === 'error') {
    return loading && !stale ? { tab: { ...tab, error: answer.error }, effects: [] } : still(tab)
  }
  if (answer.kind === 'missing') {
    if (stale) return still(tab)
    return loading ? shows(tab, '', null) : still({ ...tab, isDeleted: true })
  }
  if (loading) return shows(tab, answer.body, answer.at)
  if (state === 'gone') return shows({ ...tab, isDeleted: false }, answer.body, answer.at)
  if (stale) return still(tab)
  if (state === 'stale') return shows({ ...tab, isStale: false }, answer.body, answer.at)
  if (dirty(tab)) return still(tab)
  return shows(tab, answer.body, answer.at)
}

/**
 * When the write happens: once still for the quiet interval, and within bound of first unwritten change.
 */
const armFor = (since: number, at: number, limits: WriteLimits): number =>
  Math.max(0, Math.min(limits.quiet, since + limits.bound - at))

/**
 * An error about the body is cleared by typing when there is a written baseline.
 */
const mends = (tab: Tab): boolean =>
  tab.written !== null && tab.error !== null && MENDABLE_ERRORS.includes(tab.error)

const typed = (tab: Tab, body: string, at: number, limits: WriteLimits): Transition => {
  const state = stateOf(tab)
  if (state === 'loading') return still(tab)
  const error = mends(tab) ? null : tab.error
  const since = tab.since ?? at
  const next: Tab = { ...tab, shown: body, since, error }
  if (state === 'stale') return still(next)
  return { tab: next, effects: [{ kind: 'arm', after: armFor(since, at, limits) }] }
}

const fired = (tab: Tab): Transition => {
  const state = stateOf(tab)
  if (state === 'loading') return still(tab)
  if (state === 'unsaved') return begins(tab, seenOf(tab))
  if (state === 'saving') return { tab: { ...tab, hasPendingWrite: true }, effects: [] }
  return still(tab)
}

const landed = (tab: Tab, answer: WriteResult): Transition => {
  if (tab.pendingWrite === null) return still(tab)
  if (answer.kind === 'error') {
    return {
      tab: { ...tab, pendingWrite: null, hasPendingWrite: false, error: answer.error },
      effects: [],
    }
  }
  if (answer.kind === 'changed') {
    return {
      tab: { ...tab, pendingWrite: null, hasPendingWrite: false, isStale: true, since: null },
      effects: [],
    }
  }
  const written: Tab = {
    ...tab,
    written: tab.pendingWrite,
    filePath: answer.at,
    pendingWrite: null,
    since: null,
    isDeleted: false,
  }
  return tab.hasPendingWrite ? begins(written, seenOf(written)) : still(written)
}

const changed = (tab: Tab, paths: readonly string[], renamed: readonly Move[]): Transition => {
  const went = renamed.find((one) => one.from === tab.path)
  const next = went ? { ...tab, path: went.to, isDeleted: false } : tab

  const mine = paths.length === 0 || paths.includes(next.path) || went !== undefined
  if (!mine || stateOf(next) !== 'clean') return still(next)
  const reading = next.reading + 1
  return {
    tab: { ...next, reading },
    effects: [{ kind: 'read', path: next.path, generation: reading }],
  }
}

/**
 * A save asked for now writes what is unsaved and owes one to a write in the air.
 */
const saving = (tab: Tab): Transition => {
  const state = stateOf(tab)
  if (state === 'unsaved') return begins(tab, seenOf(tab))
  if (state === 'saving') return { tab: { ...tab, hasPendingWrite: true }, effects: [] }
  return still(tab)
}

/**
 * Settling disarms the timer and writes unsaved text immediately.
 */
const settling = (tab: Tab): Transition => {
  const state = stateOf(tab)
  if (state === 'unsaved') {
    const going = begins(tab, seenOf(tab))
    return { tab: going.tab, effects: [{ kind: 'disarm' }, ...going.effects] }
  }
  if (state === 'saving') {
    return { tab: { ...tab, hasPendingWrite: dirty(tab) }, effects: [{ kind: 'disarm' }] }
  }
  return { tab, effects: [{ kind: 'disarm' }] }
}

/** Keep: what is on screen goes to the file. */
const keeping = (tab: Tab): Transition => {
  const state = stateOf(tab)
  if (state !== 'stale' && state !== 'gone') return still(tab)
  return begins(tab, null)
}

/** Take: the file is read again, replacing the buffer. */
const taking = (tab: Tab): Transition => {
  const state = stateOf(tab)
  if (state !== 'stale') return still(tab)
  const reading = tab.reading + 1
  return {
    tab: { ...tab, reading },
    effects: [{ kind: 'read', path: tab.path, generation: reading }],
  }
}

/**
 * Closes the tab, saving unsaved changes or holding if writes or conflicts are pending.
 */
const closing = (tab: Tab): Transition => {
  const state = stateOf(tab)
  if (tab.error && state === 'stuck') {
    if (tab.written !== null && dirty(tab) && MENDABLE_ERRORS.includes(tab.error)) {
      return { tab, effects: [{ kind: 'say', error: tab.error }, { kind: 'hold' }] }
    }
    return { tab, effects: [{ kind: 'say', error: tab.error }, { kind: 'close' }] }
  }
  if (state === 'loading') return { tab, effects: [{ kind: 'close' }] }
  if (state === 'stale') return { tab, effects: [{ kind: 'hold' }] }
  if (state === 'unsaved') {
    const going = begins(tab, seenOf(tab))
    return { tab: going.tab, effects: [...going.effects, { kind: 'hold' }] }
  }
  if (state === 'saving') {
    return { tab: { ...tab, hasPendingWrite: dirty(tab) }, effects: [{ kind: 'hold' }] }
  }
  return { tab, effects: [{ kind: 'close' }] }
}
