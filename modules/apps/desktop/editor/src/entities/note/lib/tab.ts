/**
 * Pure state machine transitions and effects for note tab editing and auto-save.
 */
import {
  isDirty,
  stateOf,
  waiting,
  MENDABLE_ERRORS,
  type Event,
  type Move,
  type NoteBaseline,
  type ReadResult,
  type Tab,
  type Transition,
  type WriteLimits,
  type WriteResult,
} from './tabState'

export * from './tabState'

/** An event that asks for something and carries nothing with it. */
type AskedKind = Exclude<Event['kind'], 'read' | 'typed' | 'written' | 'changed'>

export const tabAfter = (tab: Tab, event: Event, limits: WriteLimits = waiting): Transition => {
  switch (event.kind) {
    case 'read':
      return applyRead(tab, event.generation, event.answer)
    case 'typed':
      return applyEdit(tab, event.body, event.at, limits)
    case 'written':
      return applyWrite(tab, event.answer)
    case 'changed':
      return applyChange(tab, event.paths, event.renamed)
    default:
      return applyAsked(tab, event.kind)
  }
}

const applyAsked = (tab: Tab, kind: AskedKind): Transition => {
  switch (kind) {
    case 'fired':
      return applyTimer(tab)
    case 'saving':
      return applySave(tab)
    case 'settling':
      return applySettle(tab)
    case 'keeping':
      return applyKeep(tab)
    case 'taking':
      return applyTake(tab)
    case 'closing':
      return applyClose(tab)
  }
}

const keepTab = (tab: Tab): Transition => ({ tab, effects: [] })

/**
 * A body equal to what is shown replaces nothing.
 */
const applyBody = (tab: Tab, body: string, filePath: string | null): Transition => ({
  tab: { ...tab, written: body, filePath, shown: body, since: null },
  effects: body === tab.shown ? [] : [{ kind: 'replace', body }],
})

/** What the tab last saw, for a write to present. */
const getBaseline = (tab: Tab): NoteBaseline | null =>
  tab.written === null || tab.filePath === null ? null : { prose: tab.written, at: tab.filePath }

/** A write of what is on screen now, presenting what it is given. */
const beginWrite = (tab: Tab, baseline: NoteBaseline | null): Transition => ({
  tab: { ...tab, pendingWrite: tab.shown, hasPendingWrite: false, isStale: false },
  effects: [{ kind: 'write', path: tab.path, body: tab.shown, seen: baseline }],
})

/**
 * A read is applied to a tab with nothing unsaved and only for the generation asked for last.
 */
const applyRead = (tab: Tab, generation: number, answer: ReadResult): Transition => {
  const loading = stateOf(tab) === 'loading'
  const stale = generation !== tab.reading
  if (answer.kind === 'error') {
    return loading && !stale ? { tab: { ...tab, error: answer.error }, effects: [] } : keepTab(tab)
  }
  if (answer.kind === 'missing') {
    if (stale) return keepTab(tab)
    return loading ? applyBody(tab, '', null) : keepTab({ ...tab, isDeleted: true })
  }
  return applyReadBody(tab, stale, answer.body, answer.at)
}

/** The text the file holds, taken where the tab holds nothing unsaved of its own. */
const applyReadBody = (tab: Tab, stale: boolean, body: string, at: string): Transition => {
  const state = stateOf(tab)
  if (state === 'loading') return applyBody(tab, body, at)
  if (state === 'gone') return applyBody({ ...tab, isDeleted: false }, body, at)
  if (stale) return keepTab(tab)
  if (state === 'stale') return applyBody({ ...tab, isStale: false }, body, at)
  if (isDirty(tab)) return keepTab(tab)
  return applyBody(tab, body, at)
}

/**
 * When the write happens: once still for the quiet interval, and within bound of first unwritten change.
 */
const armFor = (since: number, at: number, limits: WriteLimits): number =>
  Math.max(0, Math.min(limits.quiet, since + limits.bound - at))

/**
 * An error about the body is cleared by typing when there is a written baseline.
 */
const isMendable = (tab: Tab): boolean =>
  tab.written !== null && tab.error !== null && MENDABLE_ERRORS.includes(tab.error)

const applyEdit = (tab: Tab, body: string, at: number, limits: WriteLimits): Transition => {
  const state = stateOf(tab)
  if (state === 'loading') return keepTab(tab)
  const error = isMendable(tab) ? null : tab.error
  const since = tab.since ?? at
  const next: Tab = { ...tab, shown: body, since, error }
  if (state === 'stale') return keepTab(next)
  return { tab: next, effects: [{ kind: 'arm', after: armFor(since, at, limits) }] }
}

const applyTimer = (tab: Tab): Transition => {
  const state = stateOf(tab)
  if (state === 'loading') return keepTab(tab)
  if (state === 'unsaved') return beginWrite(tab, getBaseline(tab))
  if (state === 'saving') return { tab: { ...tab, hasPendingWrite: true }, effects: [] }
  return keepTab(tab)
}

const applyWrite = (tab: Tab, answer: WriteResult): Transition => {
  if (tab.pendingWrite === null) return keepTab(tab)
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
  return tab.hasPendingWrite ? beginWrite(written, getBaseline(written)) : keepTab(written)
}

const applyChange = (tab: Tab, paths: readonly string[], renames: readonly Move[]): Transition => {
  const went = renames.find((one) => one.from === tab.path)
  const next = went ? { ...tab, path: went.to, isDeleted: false } : tab

  const mine = paths.length === 0 || paths.includes(next.path) || went !== undefined
  if (!mine || stateOf(next) !== 'clean') return keepTab(next)
  const reading = next.reading + 1
  return {
    tab: { ...next, reading },
    effects: [{ kind: 'read', path: next.path, generation: reading }],
  }
}

/**
 * A save asked for now writes what is unsaved and owes one to a write in the air.
 */
const applySave = (tab: Tab): Transition => {
  const state = stateOf(tab)
  if (state === 'unsaved') return beginWrite(tab, getBaseline(tab))
  if (state === 'saving') return { tab: { ...tab, hasPendingWrite: true }, effects: [] }
  return keepTab(tab)
}

/**
 * Settling disarms the timer and writes unsaved text immediately.
 */
const applySettle = (tab: Tab): Transition => {
  const state = stateOf(tab)
  if (state === 'unsaved') {
    const going = beginWrite(tab, getBaseline(tab))
    return { tab: going.tab, effects: [{ kind: 'disarm' }, ...going.effects] }
  }
  if (state === 'saving') {
    return { tab: { ...tab, hasPendingWrite: isDirty(tab) }, effects: [{ kind: 'disarm' }] }
  }
  return { tab, effects: [{ kind: 'disarm' }] }
}

/** Keep: what is on screen goes to the file. */
const applyKeep = (tab: Tab): Transition => {
  const state = stateOf(tab)
  if (state !== 'stale' && state !== 'gone') return keepTab(tab)
  return beginWrite(tab, null)
}

/** Take: the file is read again, replacing the buffer. */
const applyTake = (tab: Tab): Transition => {
  const state = stateOf(tab)
  if (state !== 'stale') return keepTab(tab)
  const reading = tab.reading + 1
  return {
    tab: { ...tab, reading },
    effects: [{ kind: 'read', path: tab.path, generation: reading }],
  }
}

/**
 * Closes the tab, saving unsaved changes or holding if writes or conflicts are pending.
 */
const applyClose = (tab: Tab): Transition => {
  const state = stateOf(tab)
  if (tab.error && state === 'stuck') {
    if (tab.written !== null && isDirty(tab) && MENDABLE_ERRORS.includes(tab.error)) {
      return { tab, effects: [{ kind: 'say', error: tab.error }, { kind: 'hold' }] }
    }
    return { tab, effects: [{ kind: 'say', error: tab.error }, { kind: 'close' }] }
  }
  if (state === 'loading') return { tab, effects: [{ kind: 'close' }] }
  if (state === 'stale') return { tab, effects: [{ kind: 'hold' }] }
  if (state === 'unsaved') {
    const going = beginWrite(tab, getBaseline(tab))
    return { tab: going.tab, effects: [...going.effects, { kind: 'hold' }] }
  }
  if (state === 'saving') {
    return { tab: { ...tab, hasPendingWrite: isDirty(tab) }, effects: [{ kind: 'hold' }] }
  }
  return { tab, effects: [{ kind: 'close' }] }
}
