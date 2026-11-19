/**
 * The reads and writes of a plugin's note tab, in the order the tab made them.
 */
import {
  convertErrorCode,
  ERROR_MESSAGES,
  type NoteResult,
  type WriteResult,
} from '@/entities/note'

/** The queries a plugin's note tab makes to the vault. */
export interface PluginNotesDeps {
  readonly read: (path: string) => Promise<NoteResult>
  readonly write: (
    path: string,
    body: string,
    baseline: { prose: string; stat: string } | null,
  ) => Promise<WriteResult>
}

/** What a plugin's note tab is handed. */
export interface PluginNotes {
  readonly writeIfUnchanged: (
    path: string,
    body: string,
    observed: { body: string; at: string },
  ) => Promise<string>
  readonly readNote: (path: string) => Promise<{ body: string; at?: string; error?: string }>
  readonly writeNote: (path: string, body: string) => Promise<void>
}

const NOT_WRITTEN = 'the note could not be written'
const UNREADABLE = 'the frontmatter of this note cannot be read'

interface QueuedWrite {
  body: string
  promise: Promise<void>
}

/**
 * Each path has at most one write in flight. Writes made while one is in flight
 * are merged into the next write, which carries the newest body. A read of a
 * path resolves after the writes made before it and none made after it, and a
 * failed write rejects only the callers that requested it.
 */
export const createPluginNotes = (deps: PluginNotesDeps): PluginNotes => {
  const queued = new Map<string, QueuedWrite>()
  const tails = new Map<string, Promise<void>>()

  function enqueue<T>(path: string, run: () => Promise<T>): Promise<T> {
    const before = tails.get(path)
    const result = before ? before.then(run) : run()
    const settled = result
      .then(
        () => undefined,
        () => undefined,
      )
      .then(() => {
        if (tails.get(path) === settled) tails.delete(path)
      })
    tails.set(path, settled)
    return result
  }

  const writeNote = (path: string, body: string): Promise<void> => {
    const pendingItem = queued.get(path)
    if (pendingItem) {
      pendingItem.body = body
      return pendingItem.promise
    }
    const job: QueuedWrite = { body, promise: Promise.resolve() }
    const run = async (): Promise<void> => {
      if (queued.get(path) === job) queued.delete(path)
      const reply = await deps.write(path, job.body, null)
      if (!reply.ok) throw new Error(NOT_WRITTEN)
    }
    queued.set(path, job)
    job.promise = enqueue(path, run)
    return job.promise
  }

  const readNote = async (path: string): Promise<{ body: string; at?: string; error?: string }> => {
    const before = tails.get(path)
    queued.delete(path)
    const reply = await (before ? before.then(() => deps.read(path)) : deps.read(path))
    if (reply.ok) return reply.value
    return { body: '', error: ERROR_MESSAGES[convertErrorCode(reply.error)] || UNREADABLE }
  }

  const writeIfUnchanged: PluginNotes['writeIfUnchanged'] = (path, body, observed) => {
    queued.delete(path)
    return enqueue(path, async () => {
      const reply = await deps.write(path, body, { prose: observed.body, stat: observed.at })
      if (!reply.ok && reply.error === 'changed')
        throw new Error('The note changed on disk. Reopen it before saving.')
      if (!reply.ok || !reply.value.at) throw new Error(NOT_WRITTEN)
      return reply.value.at
    })
  }

  return { readNote, writeNote, writeIfUnchanged }
}
