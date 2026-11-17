/** The headings of the note of a node, read when the pointer comes onto the node and put away when it leaves. */
import { getCurrentScope, onScopeDispose, shallowRef, watch } from 'vue'
import { createRequestGuard } from '@numen/ui'
import type { Caught, NoteHeading, VaultNotes } from '@numen/plugin-sdk'
import type { HeadingsSettings } from './settings'

/** How long the pointer rests on a node before its headings come out, in milliseconds. */
export const HOVER_WAIT = 500

type Wait = (ms: number) => Promise<void>

export const sleep: Wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

/** What the headings of a node are read with and what choosing one does. */
export interface HeadingsSource {
  readonly notes: Pick<VaultNotes, 'getHeadings'>
  readonly settings: HeadingsSettings
  readonly open: (path: string, line: number) => void
  readonly reportFailure: (error: Caught) => void
  readonly wait: Wait
}

/**
 * `enter` reads the headings of the note at `path` and brings them out once the pointer has rested for the wait;
 * `leave` puts them away and drops an answer that is still on its way. Nothing is read while the setting is off.
 */
export function useHeadings(source: HeadingsSource, getPath: () => string) {
  const headings = shallowRef<readonly NoteHeading[]>([])
  const guard = createRequestGuard()
  if (getCurrentScope()) onScopeDispose(guard.close)

  const leave = () => {
    guard.cancel()
    headings.value = []
  }

  const enter = async () => {
    if (!source.settings.isNesting.value) return
    const mine = guard.acquireRequest()
    const path = getPath()
    try {
      const [found] = await Promise.all([source.notes.getHeadings([path]), source.wait(HOVER_WAIT)])
      if (mine.isCurrent) headings.value = found.get(path) ?? []
    } catch (error) {
      if (mine.isCurrent) source.reportFailure(error)
    }
  }

  const choose = (line: number) => source.open(getPath(), line)

  watch(source.settings.isNesting, (isNesting) => {
    if (!isNesting) leave()
  })

  return { headings, enter, leave, choose }
}
