/**
 * Asking the application about the vault it is showing.
 *
 * Nothing here describes what an answer looks like: that is the schema, and
 * both halves are generated from it.
 */
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { Owed, Refusal, VaultService } from '@numen/protocol'
import { asSeat } from './plex'
import type { Answered, Core, Made, NewLink, Refused } from './showing'

export const vault = createClient(
  VaultService,
  createConnectTransport({ baseUrl: window.location.origin }),
)

/** The same questions, in the shape the window asks them. */
export const core: Core = {
  neighbourhood: (path) => vault.neighbourhood({ path }),
  opening: async () => (await vault.opening({})).note ?? null,
  state: () => vault.state({}),
  changes: (signal) => vault.changes({}, { signal }),
  focus: (signal) => vault.focus({}, { signal }),
  editing: (signal) => vault.editing({}, { signal }),
  read: async (path) => answered(await vault.read({ path })),
  write: async (path, body, seen) =>
    answered(await vault.write({ path, body, ...(seen ? { seen: seenOf(seen) } : {}) })),
  create: async (note) => {
    const answer = await vault.create({
      title: note.title,
      folder: note.folder,
      links: note.links.map(written),
    })
    return { path: answer.path, refusal: refusalIn(answer) } satisfies Made
  },
  join: async (path, link) => refusalIn(await vault.join({ path, link: written(link) })),
  quitting: (signal) => vault.quitting({}, { signal }),
  flushed: async (token, owed) => {
    await vault.flushed({ token, owed: owing[owed ?? 'nothing'] })
  },
}

/** A link in the shape the schema carries it. */
const written = (link: NewLink) => ({
  to: link.to,
  seat: asSeat(link.seat),
  label: link.label ?? '',
})

/** The schema's answer in the words the window uses. */
/**
 * A file as one value the window carries about and never reads into.
 *
 * The schema holds the parts; what a tab does with one is present it back
 * unchanged, so the parts stay here and the string goes everywhere else.
 */
const stamp = (at?: { path: string; size: bigint; mtime: bigint }): string | undefined =>
  at && `${at.size} ${at.mtime} ${at.path}`

/** The two numbers first: a path holds spaces, and everything after them is it. */
const seenOf = (seen: { prose: string; at: string }) => {
  const [size = '0', mtime = '0', ...rest] = seen.at.split(' ')
  return {
    prose: seen.prose,
    at: { path: rest.join(' '), size: BigInt(size), mtime: BigInt(mtime) },
  }
}

const answered = (from: {
  body?: string | undefined
  refusal?: Refusal | undefined
  changed?: boolean | undefined
  at?: { path: string; size: bigint; mtime: bigint } | undefined
}): Answered & { at?: string; changed?: boolean } => {
  const at = stamp(from.at)
  return {
    body: from.body ?? '',
    refusal: refusalIn(from),
    ...(at === undefined ? {} : { at }),
    ...(from.changed === undefined ? {} : { changed: from.changed }),
  }
}

const refusalIn = (from: { refusal?: Refusal | undefined }): Refused | null =>
  from.refusal === undefined ? null : refused[from.refusal]

const refused: Record<Refusal, Refused> = {
  [Refusal.UNSPECIFIED]: 'unreadable',
  [Refusal.MISSING]: 'missing',
  [Refusal.NOT_A_NOTE]: 'notANote',
  [Refusal.NOT_TEXT]: 'notText',
  [Refusal.TOO_LARGE]: 'tooLarge',
  [Refusal.BODY_REFUSED]: 'bodyRefused',
  [Refusal.UNREADABLE]: 'unreadable',
  [Refusal.OCCUPIED]: 'occupied',
}

/** What a client has left, as the schema names it. */
const owing: Record<'nothing' | 'written' | 'asking', Owed> = {
  nothing: Owed.UNSPECIFIED,
  written: Owed.WRITTEN,
  asking: Owed.ASKING,
}
