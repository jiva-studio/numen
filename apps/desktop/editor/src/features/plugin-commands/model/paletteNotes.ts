/**
 * The type and properties of the note in front of the palette, which the conditions of the plugins'
 * commands read and the palette's target does not hold. Each note is read once from the core and kept
 * until `reset`, which the palette calls as it opens or closes, so a change to the note shows at the
 * next open.
 */
import { shallowRef } from 'vue'
import { runInBackground } from '@numen/ui'
import type { CommandNote } from '@numen/plugin-loader'
import type { CommandTarget } from '@/shared/model/command'
import type { PaletteNote } from '../lib/whenContext'

/** Reads the type and properties of the note at a target. */
export type ReadNote = (
  target: Pick<CommandNote, 'path' | 'title' | 'isFolder' | 'others'>,
) => Promise<CommandNote>

export function createPaletteNotes(readNote: ReadNote) {
  const notes = shallowRef<ReadonlyMap<string, PaletteNote>>(new Map())
  const requests = new Set<string>()
  let generation = 0

  /** The note at the path of the target; undefined while it is read, and the read is begun once. */
  const getNote = (target: CommandTarget): PaletteNote | undefined => {
    const path = target.path || target.file
    const known = notes.value.get(path)
    if (known || requests.has(path)) return known
    requests.add(path)
    const requestGeneration = generation
    const pending = readNote({
      path,
      title: target.title,
      isFolder: false,
      others: target.others ?? [],
    })
    runInBackground(
      pending.then(({ type, properties }) => {
        if (requestGeneration !== generation) return
        notes.value = new Map(notes.value).set(path, { type, ...(properties ? { properties } : {}) })
      }),
      () => undefined,
    )
    return undefined
  }

  /** Forgets the notes read, so the next palette reads them again. */
  const reset = (): void => {
    generation += 1
    requests.clear()
    notes.value = new Map()
  }

  return { getNote, reset }
}
