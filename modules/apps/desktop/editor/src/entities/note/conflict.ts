/**
 * External conflict detection and resolution for open notes.
 */
import { stateOf, type Event, type Tab } from './tab'
import type { PathRename } from '../../shared/core'
import type { ConflictWords } from './noteTypes'
import { ERROR_MESSAGES, STALE_CONFLICT } from './words'

export function isStale(tab: Tab): boolean {
  return stateOf(tab) === 'stale'
}

export function staleOf(tab: Tab | undefined): ConflictWords | null {
  return tab && isStale(tab) ? STALE_CONFLICT : null
}

export function getErrorMessage(tab: Tab | undefined): string {
  const error = tab?.error
  return error ? ERROR_MESSAGES[error] : ''
}

export function createConflictCoordinator(
  turn: (id: string, event: Event) => void,
  getTab: (id: string) => Tab | undefined,
) {
  const keep = (id: string): void => turn(id, { kind: 'keeping' })
  const take = (id: string): void => turn(id, { kind: 'taking' })
  const changed = (
    allIds: () => readonly string[],
    paths: readonly string[],
    renamed: readonly PathRename[] = [],
  ): void => {
    for (const id of allIds()) turn(id, { kind: 'changed', paths, renamed })
  }

  return {
    keep,
    take,
    changed,
    stale: (id: string) => staleOf(getTab(id)),
    getErrorMessage: (id: string) => getErrorMessage(getTab(id)),
  }
}
