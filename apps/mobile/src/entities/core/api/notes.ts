import type { Transport } from '@connectrpc/connect'
import { getNoteClient } from '@numen/core-client'
import { formatErrorCode } from '@/shared/api/errors'
import type { Core, NoteStamp } from '../lib/core'

type Stat = { path: string; size: bigint; mtime: bigint }

const mapStamp = (stat: Stat | undefined): NoteStamp | null =>
  stat ? { path: stat.path, size: stat.size, mtime: stat.mtime } : null

/** The calls of the note service over `transport`, with every answer in the window's own types. */
export function createNoteApi(
  transport: Transport,
): Pick<Core, 'getOpeningPath' | 'readNote' | 'writeNote'> {
  const notes = getNoteClient(transport)
  return {
    async getOpeningPath() {
      return (await notes.getOpeningNote({})).note?.path ?? null
    },
    async readNote(path) {
      const answer = await notes.readNote({ path })
      if (answer.error) return { ok: false, error: formatErrorCode(answer.error) }
      return { ok: true, value: { prose: answer.body, stamp: mapStamp(answer.at) } }
    },
    async writeNote(path, prose, baseline) {
      const answer = await notes.writeNote({
        path,
        body: prose,
        baseline: baseline ? { prose: baseline.prose, at: baseline.stamp } : undefined,
      })
      if (answer.error) return { ok: false, error: formatErrorCode(answer.error) }
      return { ok: true, value: null }
    },
  }
}
