/**
 * Note domain methods for the window core.
 */
import { asFailure, asValue, findErrorCode, isStaleAnswer, resolveTextLinks } from '@numen/plugin-sdk'
import { notes } from '@/shared/clients'
import {
  mapBaseline,
  mapLink,
  mapMoveResult,
  mapNeighbourhood,
  mapNoteResult,
  mapWriteResult,
  roles,
  mapRange,
  writes,
} from './words'
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
  | 'removeLink'
  | 'replaceLinks'
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
    const answer = await notes.createNote({
      title: note.title,
      path: note.folder,
      links: note.links.map(mapLink),
    })
    const error = findErrorCode(answer)
    return error ? asFailure(error) : asValue({ path: answer.path })
  },
  join: async (path, link) => findErrorCode(await notes.writeLink({ path, link: mapLink(link) })),
  removeLink: async (path, to, role) =>
    findErrorCode(await notes.removeLink({ path, to, ...(role ? { role: roles[role] } : {}) })),
  replaceLinks: async (first, second, links) =>
    findErrorCode(
      await notes.replaceLinks({
        first,
        second,
        links: links.map((link) => ({ ...link, role: roles[link.role] })),
      }),
    ),
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
