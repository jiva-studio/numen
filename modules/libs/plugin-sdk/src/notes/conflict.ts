/**
 * External conflict detection and resolution for open notes.
 */
import { getState, type Event, type Tab } from './tab'
import type { PathRename } from '../workspace'
import type { ConflictWords } from './types'
import { ERROR_MESSAGES, STALE_CONFLICT } from './words'

export function isStale(tab: Tab): boolean {
  return getState(tab) === 'stale'
}

export function findStaleConflict(tab: Tab | undefined): ConflictWords | null {
  return tab && isStale(tab) ? STALE_CONFLICT : null
}

export function getErrorMessage(tab: Tab | undefined): string {
  const error = tab?.error
  return error ? ERROR_MESSAGES[error] : ''
}

export function createConflictCoordinator(
  dispatch: (id: string, event: Event) => void,
  getTab: (id: string) => Tab | undefined,
) {
  const keep = (id: string): void => dispatch(id, { kind: 'keepLocal' })
  const take = (id: string): void => dispatch(id, { kind: 'takeFile' })
  const notifyChanged = (
    allIds: () => readonly string[],
    paths: readonly string[],
    renames: readonly PathRename[] = [],
  ): void => {
    for (const id of allIds()) dispatch(id, { kind: 'changed', paths, renamed: renames })
  }

  return {
    keep,
    take,
    notifyChanged,
    stale: (id: string) => findStaleConflict(getTab(id)),
    getErrorMessage: (id: string) => getErrorMessage(getTab(id)),
  }
}
