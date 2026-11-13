/** The listeners of each drawn note, made once per note, so a note keeps the same functions while it is drawn and is not redrawn for them. */
import type { MenuTrigger } from '@numen/ui'
import { watch } from 'vue'
import type { Position } from '../lib/node'
import type { PlexDestination } from '../../types'

/** What a note reports, each with the id of the note. */
export interface NodeReports {
  readonly activate: (id: string) => void
  readonly show: (id: string, showing: PlexDestination) => void
  readonly reach: (id: string, pointer: PointerEvent) => void
  readonly ask: (id: string) => void
  readonly menu: (id: string, at: Position, trigger: MenuTrigger) => void
  readonly enter: (id: string, heading: string) => void
  readonly attend: (id: string, isAttended: boolean) => void
}

export interface NodeListeners {
  readonly activate: () => void
  readonly show: (showing: PlexDestination) => void
  readonly reach: (pointer: PointerEvent) => void
  readonly ask: () => void
  readonly menu: (at: Position, trigger: MenuTrigger) => void
  readonly enter: (heading: string) => void
  readonly attend: (isAttended: boolean) => void
}

/** `ids` are the notes that are drawn: the listeners of any other note are dropped. */
export function useNodeHandlers(
  reports: NodeReports,
  ids?: () => readonly string[],
): (id: string) => NodeListeners {
  const made = new Map<string, NodeListeners>()

  if (ids) {
    watch(ids, (drawn) => {
      const alive = new Set(drawn)
      for (const id of made.keys()) if (!alive.has(id)) made.delete(id)
    })
  }

  return (id) => {
    let listeners = made.get(id)
    if (!listeners) {
      listeners = {
        activate: () => reports.activate(id),
        show: (showing) => reports.show(id, showing),
        reach: (pointer) => reports.reach(id, pointer),
        ask: () => reports.ask(id),
        menu: (at, trigger) => reports.menu(id, at, trigger),
        enter: (heading) => reports.enter(id, heading),
        attend: (isAttended) => reports.attend(id, isAttended),
      }
      made.set(id, listeners)
    }
    return listeners
  }
}
