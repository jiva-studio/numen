/**
 * The reads and writes of a plugin's note tab, in the order the tab made them.
 */
import {
  convertErrorCode,
  ERROR_MESSAGES,
  type NoteResult,
  type WriteResult,
} from '@/entities/note'

/** What the vault is asked, as a plugin's note tab asks it. */
export interface PluginNotesDeps {
  readonly read: (path: string) => Promise<NoteResult>
  readonly write: (path: string, body: string, baseline: null) => Promise<WriteResult>
}

/** What a plugin's note tab is handed. */
export interface PluginNotes {
  readonly readNote: (path: string) => Promise<{ body: string; error?: string }>
  readonly writeNote: (path: string, body: string) => Promise<void>
}

const NOT_WRITTEN = 'the note could not be written'
const UNREADABLE = 'the frontmatter of this note cannot be read'

interface QueuedWrite {
  body: string
  done: Promise<void>
}

/**
 * Each path has one write in the air at a time. Writes made while one is in the
 * air share the next write, which carries the newest body. A read of a path
 * answers after the writes made before it and none made after it, and a write
 * that fails rejects only the callers that asked for it.
 */
export const createPluginNotes = (deps: PluginNotesDeps): PluginNotes => {
  const queued = new Map<string, QueuedWrite>()
  const tails = new Map<string, Promise<void>>()

  const writeNote = (path: string, body: string): Promise<void> => {
    const waiting = queued.get(path)
    if (waiting) {
      waiting.body = body
      return waiting.done
    }
    const job: QueuedWrite = { body, done: Promise.resolve() }
    const run = async (): Promise<void> => {
      if (queued.get(path) === job) queued.delete(path)
      const answered = await deps.write(path, job.body, null)
      if (!answered.ok) throw new Error(NOT_WRITTEN)
    }
    queued.set(path, job)
    const before = tails.get(path)
    job.done = before ? before.then(run) : run()
    const settled: Promise<void> = job.done
      .catch(() => undefined)
      .then(() => {
        if (tails.get(path) === settled) tails.delete(path)
      })
    tails.set(path, settled)
    return job.done
  }

  const readNote = async (path: string): Promise<{ body: string; error?: string }> => {
    const before = tails.get(path)
    queued.delete(path)
    const answered = await (before ? before.then(() => deps.read(path)) : deps.read(path))
    if (answered.ok) return { body: answered.value.body }
    return { body: '', error: ERROR_MESSAGES[convertErrorCode(answered.error)] || UNREADABLE }
  }

  return { readNote, writeNote }
}
