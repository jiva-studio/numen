/**
 * The notes the window has open, and what turns the state model.
 *
 * Coordinates open note tabs, executing reads, writes, timers, and holding note state.
 */
import { ref, type Ref } from 'vue'
import {
  openTab,
  stateOf,
  tabAfter,
  waiting,
  type Effect,
  type Event,
  type Move,
  type Tab,
} from '../lib/tab'
import type { LinkAddress } from '../lib/note'
import type { OpenNote, Notes, OpenNotesOptions } from '../lib/noteTypes'
import { createNoteQueue } from './queue'
import { createConflictCoordinator } from './conflict'

export * from '../lib/noteTypes'

export function openNotes(core: Notes, how: OpenNotesOptions = {}) {
  const limits = how.limits ?? waiting
  const onReplaced = how.onReplaced ?? (() => {})
  const now = how.now ?? (() => Date.now())

  const tabs = ref(new Map<string, Tab>())
  const bodies = ref(new Map<string, string>())
  const addresses = ref(new Map<string, LinkAddress>())

  const told = new Set<string>()
  const closing = new Map<string, (gone: boolean) => void>()

  function turn(id: string, event: Event): void {
    const tab = tabs.value.get(id)
    if (!tab) return
    carry(id, tabAfter(tab, event, limits))
  }

  const queue = createNoteQueue(
    core,
    turn,
    (id) => tabs.value.has(id),
    (id, addr) => {
      if (addr) addresses.value.set(id, addr)
      else addresses.value.delete(id)
    },
    (id) => {
      if (closing.has(id)) turn(id, { kind: 'closing' })
    },
  )

  const conflicts = createConflictCoordinator(
    turn,
    (id) => tabs.value.get(id),
  )

  const open = (id: string, path: string = id): void => {
    if (tabs.value.has(id)) return
    carry(id, openTab(path))
  }

  const close = (id: string): Promise<boolean> =>
    new Promise((done) => {
      if (!tabs.value.has(id)) return done(true)
      if (told.has(id)) {
        forget(id)
        return done(true)
      }
      closing.set(id, done)
      turn(id, { kind: 'closing' })
    })

  const getPath = (id: string): string => tabs.value.get(id)?.path ?? id
  const has = (id: string): boolean => tabs.value.has(id)
  const getFilePath = (id: string): string => tabs.value.get(id)?.filePath ?? ''

  const setBody = (id: string, body: string): void => {
    bodies.value.set(id, body)
    turn(id, { kind: 'typed', body, at: now() })
  }

  const save = (id: string): void => turn(id, { kind: 'saving' })

  const getOpenNote = (id: string): OpenNote => {
    const tab = tabs.value.get(id)
    const err = tab?.error ?? null
    return {
      path: tab?.path ?? id,
      body: bodies.value.get(id) ?? '',
      state: tab ? stateOf(tab) : 'loading',
      error: err,
    }
  }

  const getOpenIds = (): readonly string[] => [...tabs.value.keys()]

  function carry(id: string, next: ReturnType<typeof tabAfter>): void {
    tabs.value = new Map(tabs.value).set(id, next.tab)
    for (const effect of next.effects) applyEffect(id, effect)

    const held = next.effects.some((effect) => effect.kind === 'hold')
    const writing = next.effects.some((effect) => effect.kind === 'write')
    if (held && !writing) closing.get(id)?.(false)

    if (next.tab.pendingWrite === null) queue.finishSettle(id)
  }

  function applyEffect(id: string, effect: Effect): void {
    switch (effect.kind) {
      case 'read':
        void queue.read(id, effect.path, effect.generation)
        return
      case 'write':
        void queue.write(id, effect.path, effect.body, effect.seen)
        return
      case 'arm':
        queue.arm(id, effect.after)
        return
      case 'disarm':
        queue.disarm(id)
        return
      case 'replace':
        bodies.value.set(id, effect.body)
        onReplaced(getPath(id))
        return
      case 'hold':
        return
      case 'say':
        told.add(id)
        return
      case 'close':
        forget(id)
        return
    }
  }

  function forget(id: string): void {
    queue.forget(id)
    tabs.value.delete(id)
    bodies.value.delete(id)
    addresses.value.delete(id)
    closing.get(id)?.(true)
    closing.delete(id)
    told.delete(id)
  }

  const flush = async (): Promise<void> => {
    await Promise.all(getOpenIds().map(close))
  }

  return {
    open,
    close,
    settle: queue.settle,
    getPath,
    has,
    getFilePath,
    setBody,
    applyPathChanges: (paths: readonly string[], renamed: readonly Move[] = []) =>
      conflicts.notifyChanged(getOpenIds, paths, renamed),
    save,
    keep: conflicts.keep,
    take: conflicts.take,
    getOpenNote,
    link: (id: string): LinkAddress | null => addresses.value.get(id) ?? null,
    getOpenIds,
    getErrorMessage: conflicts.getErrorMessage,
    stale: conflicts.stale,
    flush,
    tabs: tabs as Ref<Map<string, Tab>>,
  }
}
