/**
 * Save timers, settling coordination, and read/write execution for open notes.
 */
import type { ErrorCode, LinkAddress, NoteResult } from '../../shared/core'
import type { Notes } from './noteTypes'
import type { Event, NoteBaseline, NoteErrorCode } from './tab'

export const errorOf = (from: ErrorCode): NoteErrorCode => {
  if (from === 'deckTooLarge') return 'tooLarge'
  if (from === 'notAStencil' || from === 'notADeck' || from === 'notAPreset') return 'notANote'
  return from === 'missing' || from === 'occupied' || from === 'unnameable' ? 'unreadable' : from
}

export function createNoteQueue(
  core: Notes,
  turn: (id: string, event: Event) => void,
  hasTab: (id: string) => boolean,
  setAddress: (id: string, address: LinkAddress | null) => void,
  onClosingWritten: (id: string) => void,
) {
  const timers = new Map<string, ReturnType<typeof setTimeout>>()
  const settling = new Map<string, () => void>()

  const arm = (id: string, after: number): void => {
    clearTimeout(timers.get(id))
    timers.set(
      id,
      setTimeout(() => {
        timers.delete(id)
        turn(id, { kind: 'fired' })
      }, after),
    )
  }

  const disarm = (id: string): void => {
    clearTimeout(timers.get(id))
    timers.delete(id)
  }

  const settled = (id: string): void => {
    settling.get(id)?.()
    settling.delete(id)
  }

  const settles = (id: string): Promise<void> =>
    new Promise((done) => {
      if (!hasTab(id)) return done()
      settling.set(id, done)
      turn(id, { kind: 'settling' })
    })

  const read = async (id: string, path: string, generation: number): Promise<void> => {
    let answered: NoteResult & { at?: string }
    try {
      answered = await core.read(path)
    } catch {
      // Core read failed.
      turn(id, { kind: 'read', generation, answer: { kind: 'error', error: 'unreachable' } })
      return
    }
    const link = answered.link
    if (link) setAddress(id, link)
    else setAddress(id, null)
    turn(id, {
      kind: 'read',
      generation,
      answer:
        !answered.error
          ? { kind: 'body', body: answered.body, at: answered.at ?? '' }
          : answered.error === 'missing'
            ? { kind: 'missing' }
            : { kind: 'error', error: errorOf(answered.error) },
    })
  }

  const write = async (
    id: string,
    path: string,
    body: string,
    seen: NoteBaseline | null,
  ): Promise<void> => {
    let answered: NoteResult & { at?: string; changed?: boolean }
    try {
      answered = await core.write(path, body, seen)
    } catch {
      // Core write failed.
      turn(id, { kind: 'written', answer: { kind: 'error', error: 'unreachable' } })
      return
    }
    turn(id, {
      kind: 'written',
      answer: answered.changed
        ? { kind: 'changed' }
        : !answered.error
          ? { kind: 'ok', at: answered.at ?? '' }
          : { kind: 'error', error: errorOf(answered.error) },
    })
    onClosingWritten(id)
  }

  const forget = (id: string): void => {
    disarm(id)
    settled(id)
  }

  return {
    arm,
    disarm,
    settled,
    settles,
    read,
    write,
    forget,
  }
}
