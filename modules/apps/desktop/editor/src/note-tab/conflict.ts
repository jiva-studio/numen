/**
 * External conflict detection and resolution for open notes.
 */
import { stateOf, type Event, type Tab } from './tab'
import type { Move } from '../shared/core'
import type { ConflictWords } from './noteTypes'
import { REFUSAL_WORDS, STALE_CONFLICT } from './words'

export const stale = STALE_CONFLICT

export function isStale(tab: Tab): boolean {
  const s = stateOf(tab)
  return s === 'stale' || (s as string) === 'overtaken'
}

export function staleOf(tab: Tab | undefined): ConflictWords | null {
  return tab && isStale(tab) ? stale : null
}

export function sayingOf(tab: Tab | undefined): string {
  const refusal = tab?.refused
  return refusal ? REFUSAL_WORDS[refusal] : ''
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
    renamed: readonly Move[] = [],
  ): void => {
    for (const id of allIds()) turn(id, { kind: 'changed', paths, renamed })
  }

  return {
    keep,
    take,
    changed,
    stale: (id: string) => staleOf(getTab(id)),
    saying: (id: string) => sayingOf(getTab(id)),
  }
}
