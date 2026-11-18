/** The headings of the note of a node, read once the pointer rests on the node and put away once the pointer is off it. */
import { getCurrentScope, onScopeDispose, shallowRef } from 'vue'
import { createRequestGuard } from '@numen/ui'
import type { Caught, NoteHeading, VaultNotes } from '@numen/plugin-sdk'
import type { HeadingsSettings } from './settings'

/** How long the pointer rests on a node before its headings come out, in milliseconds. */
export const HOVER_WAIT = 500

/** How long the pointer may be off the node and its list before the list is put away, so it can cross from one to the other, in milliseconds. */
export const LEAVE_GRACE = 150

type Wait = (ms: number) => Promise<void>

export const sleep: Wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

/** What the headings of a node are read with and what choosing one does. */
export interface HeadingsSource {
  readonly notes: Pick<VaultNotes, 'getHeadings'>
  readonly settings: HeadingsSettings
  readonly open: (path: string, line: number) => void
  readonly reportFailure: (error: Caught) => void
  readonly wait: Wait
  readonly hoverWait: number
  readonly leaveGrace: number
}

/**
 * `enter` reads the headings of the note at `path` and brings them out once the pointer has rested for the wait;
 * `leave` puts them away once the pointer has stayed away for the grace, and drops an answer that is still on its
 * way. Nothing is read while the setting is off.
 */
export function useHeadings(source: HeadingsSource, getPath: () => string) {
  const headings = shallowRef<readonly NoteHeading[]>([])
  const guard = createRequestGuard()
  if (getCurrentScope()) onScopeDispose(guard.close)

  let isReading = false
  let departures = 0

  const close = () => {
    departures++
    guard.cancel()
    isReading = false
    headings.value = []
  }

  const leave = async () => {
    const mine = ++departures
    await source.wait(source.leaveGrace)
    if (mine === departures) close()
  }

  const enter = async () => {
    departures++
    if (isReading || headings.value.length > 0) return
    const mine = guard.acquireRequest()
    const path = getPath()
    isReading = true
    try {
      const [found] = await Promise.all([
        source.notes.getHeadings([path]),
        source.wait(source.hoverWait),
      ])
      if (mine.isCurrent) headings.value = found.get(path) ?? []
    } catch (error) {
      if (mine.isCurrent) source.reportFailure(error)
    } finally {
      if (mine.isCurrent) isReading = false
    }
  }

  const choose = (line: number) => source.open(getPath(), line)

  return { headings, enter, leave, choose }
}
