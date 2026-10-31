/**
 * Note domain methods for the window core.
 */
import {
  asFailure,
  asValue,
  findErrorCode,
  isStaleAnswer,
  resolveTextLinks,
} from '@numen/plugin-sdk'
import { notes } from '@/shared/clients'
import {
  mapBaseline,
  mapMoveResult,
  mapNeighbourhood,
  mapNoteResult,
  mapWriteResult,
  mapRange,
  writes,
} from './words'
import { relationsCore } from './relations'
import type { NotePort } from '@/app/ports/notes'
import type { VaultPort } from '@/app/ports/vault'

export type NoteOperations = Pick<
  NotePort,
  | 'neighbourhood'
  | 'watchEdits'
  | 'read'
  | 'write'
  | 'create'
  | 'relations'
  | 'rename'
  | 'headings'
  | 'resolveLinks'
> &
  Pick<VaultPort, 'getInitialOpenPath'>

export const notesCore: NoteOperations = {
  neighbourhood: async (path) => mapNeighbourhood(await notes.getNeighbourhood({ path })),
  getInitialOpenPath: async () => (await notes.getOpeningNote({})).note ?? null,
  async *watchEdits(signal) {
    for await (const update of notes.watchEdits({}, { signal })) {
      yield {
        change: update.change,
        path: update.path,
        span: mapRange(update.range ?? { start: 0, end: 0 }),
        text: update.text,
        isComplete: update.isFinal,
      }
    }
  },
  read: async (path) => mapNoteResult(await notes.readNote({ path })),
  write: async (path, body, observed) =>
    mapWriteResult(
      await notes.writeNote({
        path,
        body,
        ...(observed ? { baseline: mapBaseline(observed) } : {}),
      }),
    ),
  create: async (note) => {
    const answer = await notes.createNote({ title: note.title, path: note.folder })
    if (answer.isNameExhausted) return asFailure('alreadyExists')
    const error = findErrorCode(answer)
    return error ? asFailure(error) : asValue({ path: answer.path })
  },
  relations: relationsCore,
  rename: async (path, title) => {
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
  },
  headings: async (paths) => {
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
  },
  resolveLinks: (from, text, vault) => resolveTextLinks(notes, { from, text, vault }),
}
