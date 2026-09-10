/**
 * Note domain methods for the window core.
 */
import { notes } from './clients'
import { answered, around, filed, run, seenOf, writes, written } from './words'
import { refusalIn, staleIn } from '../../shared/answers'
import type { Core } from '../../shared/core'

export type NoteOperations = Pick<
  Core,
  | 'neighbourhood'
  | 'opening'
  | 'editing'
  | 'read'
  | 'write'
  | 'create'
  | 'join'
  | 'rename'
  | 'headings'
  | 'resolve'
>

export const noteOperations: NoteOperations = {
  neighbourhood: async (path) => around(await notes.getNeighbourhood({ path })),
  opening: async () => (await notes.getOpeningNote({})).note ?? null,
  async *editing(signal) {
    for await (const said of notes.watchEdits({}, { signal })) {
      yield {
        change: said.change,
        path: said.path,
        span: run(said.span ?? { from: 0, to: 0 }),
        text: said.text,
        isComplete: said.done,
        done: said.done,
      }
    }
  },
  read: async (path) => answered(await notes.readNote({ path })),
  write: async (path, body, seen) =>
    answered(await notes.writeNote({ path, body, ...(seen ? { seen: seenOf(seen) } : {}) })),
  create: async (note) => {
    const answer = await notes.createNote({
      title: note.title,
      path: note.folder,
      links: note.links.map(written),
    })
    return { path: answer.path, refusal: refusalIn(answer) }
  },
  join: async (path, link) => refusalIn(await notes.writeLink({ path, link: written(link) })),
  rename: async (path, title) => {
    const answer = await notes.renameNote({ path, title })
    return {
      path: answer.path,
      title: answer.title,
      hasFrontmatter: writes[answer.by],
      frontmatter: writes[answer.by],
      moved: answer.moved ? filed(answer.moved) : null,
      refusal: refusalIn(answer),
      hasChanged: staleIn(answer),
      changed: staleIn(answer),
    }
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
      answer.resolved.filter((one) => !one.crossed).map((one) => [one.written, one.path]),
    )
  },
}

export const notesCore = noteOperations
