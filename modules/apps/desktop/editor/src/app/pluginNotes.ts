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

const UNREADABLE = 'the frontmatter of this note cannot be read'

/**
 * Each path has one write in the air at a time, and the newest body queued
 * behind it is the next write. A read of a path answers once the writes of that
 * path have landed.
 */
export const createPluginNotes = (deps: PluginNotesDeps): PluginNotes => {
  const queued = new Map<string, string>()
  const running = new Map<string, Promise<void>>()

  const writeQueued = async (path: string): Promise<void> => {
    let failure: unknown = null
    for (let body = queued.get(path); body !== undefined; body = queued.get(path)) {
      queued.delete(path)
      try {
        await deps.write(path, body, null)
        failure = null
      } catch (error) {
        failure = error
      }
    }
    running.delete(path)
    if (failure !== null) throw failure
  }

  const writeNote = (path: string, body: string): Promise<void> => {
    queued.set(path, body)
    const inFlight = running.get(path) ?? writeQueued(path)
    running.set(path, inFlight)
    return inFlight
  }

  const waitForWrites = async (path: string): Promise<void> => {
    await running.get(path)?.catch(() => undefined)
  }

  const readNote = async (path: string): Promise<{ body: string; error?: string }> => {
    await waitForWrites(path)
    const answered = await deps.read(path)
    if (answered.ok) return { body: answered.value.body }
    return { body: '', error: ERROR_MESSAGES[convertErrorCode(answered.error)] || UNREADABLE }
  }

  return { readNote, writeNote }
}
