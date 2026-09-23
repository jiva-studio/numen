/**
 * Note domain methods for the window core.
 */
import { asFailure, asValue } from '@numen/wire'
import { notes } from '@/shared/clients'
import {
  mapBaseline,
  mapLink,
  mapMoveResult,
  mapNeighbourhood,
  mapNoteResult,
  mapWriteResult,
  run,
  writes,
} from './words'
import { errorIn, staleIn } from '@/shared/answers'
import type { NotePort } from '@/app/ports/notes'
import type { VaultPort } from '@/app/ports/vault'

export type NoteOperations = Pick<
  NotePort,
  | 'neighbourhood'
  | 'watchEdits'
  | 'read'
  | 'write'
  | 'create'
  | 'join'
  | 'rename'
  | 'headings'
  | 'resolve'
> &
  Pick<VaultPort, 'getInitialOpenPath'>

export const notesCore: NoteOperations = {
  neighbourhood: async (path) => mapNeighbourhood(await notes.getNeighbourhood({ path })),
  getInitialOpenPath: async () => (await notes.getOpeningNote({})).note ?? null,
  async *watchEdits(signal) {
    for await (const said of notes.watchEdits({}, { signal })) {
      yield {
        change: said.change,
        path: said.path,
        span: run(said.span ?? { from: 0, to: 0 }),
        text: said.text,
        isComplete: said.isFinal,
      }
    }
  },
  read: async (path) => mapNoteResult(await notes.readNote({ path })),
  write: async (path, body, seen) =>
    mapWriteResult(
      await notes.writeNote({ path, body, ...(seen ? { seen: mapBaseline(seen) } : {}) }),
    ),
  create: async (note) => {
    const answer = await notes.createNote({
      title: note.title,
      path: note.folder,
      links: note.links.map(mapLink),
    })
    const error = errorIn(answer)
    return error ? asFailure(error) : asValue({ path: answer.path })
  },
  join: async (path, link) => errorIn(await notes.writeLink({ path, link: mapLink(link) })),
  rename: async (path, title) => {
    const answer = await notes.renameNote({ path, title })
    if (staleIn(answer)) return asFailure('changed')
    const error = errorIn(answer)
    if (error) return asFailure(error)
    return asValue({
      path: answer.path,
      title: answer.title,
      hasFrontmatter: writes[answer.by],
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
  resolve: async (from, writtenAddresses) => {
    const answer = await notes.resolveAddresses({ from, written: [...writtenAddresses] })
    return new Map(
      answer.resolved.filter((one) => !one.isCrossed).map((one) => [one.written, one.path]),
    )
  },
}
