/**
 * The calls of the note service: the notes of the vault, what is written in them, and what they
 * are joined to.
 */
import { findErrorCode, isStaleAnswer, asFailure, asValue } from '@numen/core-client'
import { resolveTextLinks, type TextLink } from '@numen/plugin-loader'
import { notes } from '@/shared/api/clients'
import { mapMoveResult, mapRange } from '@/shared/api/wire'
import type { CreateResult } from '@/shared/model/fileTypes'
import type {
  Neighborhood,
  NewNote,
  NoteEdit,
  NoteHeading,
  NoteResult,
  RenameResult,
  WriteResult,
} from '../lib/note'
import { mapBaseline, mapNeighborhood, mapNoteResult, mapWriteResult, writes } from './noteWords'

/** The notes joined to a note, and how. */
export const getNeighborhood = async (path: string): Promise<Neighborhood> =>
  mapNeighborhood(await notes.getNeighborhood({ path }))

/** The note the vault opens first, and null where it names none. */
export const getInitialOpenPath = async (): Promise<{ path: string } | null> => {
  const note = (await notes.getOpeningNote({})).note
  return note ? { path: note.path } : null
}

/** A change being made to a note's prose, reported while it is being made. */
export async function* watchEdits(signal: AbortSignal): AsyncGenerator<NoteEdit> {
  for await (const update of notes.watchEdits({}, { signal })) {
    yield {
      change: update.change,
      path: update.path,
      span: mapRange(update.range ?? { start: 0, end: 0 }),
      text: update.text,
      isComplete: update.isFinal,
    }
  }
}

/** The prose of a note, below its frontmatter, and the file it came out of. */
export const readNote = async (path: string): Promise<NoteResult> =>
  mapNoteResult(await notes.readNote({ path }))

/**
 * Prose into a note, keeping the frontmatter the file has when it lands.
 *
 * Observed is what a read gave this caller. Prose on disk that the caller never saw is returned as
 * changed, and nothing is written. Nothing observed writes over whatever is there.
 */
export const writeNote = async (
  path: string,
  body: string,
  observed: { prose: string; stat: string } | null,
): Promise<WriteResult> =>
  mapWriteResult(
    await notes.writeNote({
      path,
      body,
      ...(observed ? { baseline: mapBaseline(observed) } : {}),
    }),
  )

/** A note made, named after the title it is given and joined as it is written. */
export const createNote = async (note: NewNote): Promise<CreateResult> => {
  const answer = await notes.createNote({ title: note.title, path: note.folder })
  if (answer.isNameExhausted) return asFailure('alreadyExists')
  const error = findErrorCode(answer)
  return error ? asFailure(error) : asValue({ path: answer.path })
}

/**
 * A note given a different name. Whichever of the title and the filename names it is brought into
 * line, and the file follows where a title and a filename are kept as one name.
 */
export const renameNote = async (path: string, title: string): Promise<RenameResult> => {
  const answer = await notes.renameNote({ path, title })
  if (isStaleAnswer(answer)) return asFailure('changed')
  const error = findErrorCode(answer)
  if (error) return asFailure(error)
  return asValue({
    path: answer.path,
    title: answer.title,
    hasFrontmatter: writes[answer.nameSource],
    moved: answer.moved ? mapMoveResult(answer.moved) : null,
  })
}

/** The headings of each note, keyed by path. A note with no headings is absent. */
export const listHeadings = async (
  paths: readonly string[],
): Promise<ReadonlyMap<string, readonly NoteHeading[]>> => {
  const answer = await notes.listHeadings({ paths: [...paths] })
  return new Map(
    answer.headings.map((one) => [
      one.path,
      one.headings.map((heading) => ({
        text: heading.text,
        level: heading.level,
        line: heading.line,
      })),
    ]),
  )
}

/**
 * The wikilinks of a text, parsed by the core, with the note each resolves to: empty when it
 * resolves to none, or to a note in another vault. A name resolves by a path relative to the note
 * it is written in, which is `from`. `vault` is the id of the vault the text is written in; empty
 * means the vault the window shows.
 */
export const resolveLinks = (
  from: string,
  text: string,
  vault?: string,
): Promise<readonly TextLink[]> => resolveTextLinks(notes, { from, text, vault })

/** The path the note a `name://` address names resolves to from a note, empty when it resolves to none. */
export const resolveNameLink = async (from: string, address: string): Promise<string> => {
  const rawName = address.replace(/^name:\/\//, '')
  return (await resolveLinks(from, `[[${rawName}]]`))[0]?.path ?? ''
}
