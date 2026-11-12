/**
 * A core that holds notes, as a test scripts it, answered through the real note client: the call
 * is encoded, sent over the router transport and mapped back by the entity's own `api`, so only
 * the answers are the test's.
 */
import { NOTE_ERROR_CODE, parseFingerprint, type NoteErrorCodeValue } from '@numen/core-client'
import type { CoreAnswers } from '@numen/core-client/testing'
import { formatFingerprint } from '@numen/plugin-loader'
import type { Result } from '@/shared/lib/result'
import { answerCore } from './coreAnswers'

/** What a read or a write of a note returned: its prose and the file it is in, or why it is not. */
export type NoteAnswer = Result<{ readonly body: string; readonly at?: string }, string>

/** The reads and writes of notes a test answers. */
export interface NoteCore {
  read(path: string): Promise<NoteAnswer>
  write(
    path: string,
    body: string,
    observed: { readonly prose: string; readonly stat: string } | null,
  ): Promise<NoteAnswer>
}

const CODES: Record<string, NoteErrorCodeValue> = {
  notFound: NOTE_ERROR_CODE.NOT_FOUND,
  notANote: NOTE_ERROR_CODE.NOT_A_NOTE,
  notText: NOTE_ERROR_CODE.NOT_TEXT,
  tooLarge: NOTE_ERROR_CODE.TOO_LARGE,
  bodyUnwritable: NOTE_ERROR_CODE.BODY_UNWRITABLE,
  unreadable: NOTE_ERROR_CODE.UNREADABLE,
  alreadyExists: NOTE_ERROR_CODE.ALREADY_EXISTS,
  invalidName: NOTE_ERROR_CODE.INVALID_NAME,
  changed: NOTE_ERROR_CODE.CONFLICT,
}

const toWire = (answer: NoteAnswer) => {
  if (answer.ok) {
    const { body, at: stat } = answer.value
    return { body, ...(stat === undefined ? {} : { at: parseFingerprint(stat) }) }
  }
  const error = CODES[answer.error]
  if (error === undefined) throw new Error(`no error code is named ${answer.error}`)
  return { error }
}

/**
 * The core answers the reads and writes of notes as `core` says until the test ends. A file is
 * named by `<size> <modified> <path>`, as the window carries it.
 */
export const answerNotes = (core: Partial<NoteCore>): void => {
  const { read, write } = core
  const notes: NonNullable<CoreAnswers['notes']> = {}
  if (read) notes.readNote = async (request) => toWire(await read(request.path))
  if (write) {
    notes.writeNote = async (request) =>
      toWire(
        await write(
          request.path,
          request.body,
          request.baseline
            ? {
                prose: request.baseline.prose,
                stat: formatFingerprint(request.baseline.at) ?? '',
              }
            : null,
        ),
      )
  }
  answerCore({ notes })
}
