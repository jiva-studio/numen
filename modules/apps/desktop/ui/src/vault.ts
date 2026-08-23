/**
 * Asking the application about the vault it is showing.
 *
 * Nothing here describes what an answer looks like: that is the schema, and
 * both halves are generated from it.
 */
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { Owed, Refusal, VaultService, Way as Ways } from '@numen/protocol'
import { asSeat } from './plex'
import type { Asking, Way } from './finding'
import type { Documents, Marked, Sheet } from './reading'
import type { Answered, Core, Made, NewLink, Refused } from './showing'

export const vault = createClient(
  VaultService,
  createConnectTransport({ baseUrl: window.location.origin }),
)

/** The same questions, in the shape the window asks them. */
export const core: Core & Asking = {
  neighbourhood: (path) => vault.neighbourhood({ path }),
  opening: async () => (await vault.opening({})).note ?? null,
  state: () => vault.state({}),
  changes: async function* (signal) {
    for await (const change of vault.changes({}, { signal })) {
      yield {
        paths: change.paths,
        reload: change.reload,
        renamed: change.renamed.map((went) => ({ from: went.from, to: went.to })),
      }
    }
  },
  focus: (signal) => vault.focus({}, { signal }),
  editing: (signal) => vault.editing({}, { signal }),
  async *tasks(signal) {
    for await (const said of vault.tasks({}, { signal })) {
      yield said.tasks.map((at) => ({
        id: at.id,
        doing: at.doing,
        about: at.about,
        done: Number(at.done),
        total: Number(at.total),
        failed: at.failed,
        asked: at.asked,
      }))
    }
  },
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
  /** The names in the vault that match what is typed. */
  names: async (query, limit) => {
    const answer = await vault.names({ query, limit })
    return answer.found.map((one) => ({
      path: one.note?.path ?? '',
      title: one.note?.title ?? '',
      heading: one.heading?.text ?? '',
      // A name with no heading stands on no line of the prose.
      line: one.heading?.line ?? -1,
      at: one.at.map(run),
    }))
  },
  /** The text the vault holds that answers what is typed, asked one way. */
  search: async (query, way, limit) => {
    const answer = await vault.search({ query, limit, way: ways[way] })
    return answer.found.map((one) => ({
      path: one.path,
      title: one.note?.title ?? '',
      // A source that is not a note carries none, and what this window opens
      // one as is the document it is.
      isNote: one.note !== undefined,
      text: one.text,
      start: one.start,
      length: one.length,
      at: one.at.map(run),
    }))
  },
}

/**
 * The documents the vault holds, over the addresses the application serves the
 * window at. A page is a picture at an address of its own, drawn to the width
 * it is asked for in device pixels.
 */
export const documents: Documents = {
  shape: async (path) => {
    const answer = await served(asset(path))
    const said = (await answer.json()) as { pages?: number; sheets?: readonly Sheet[] }
    return { pages: said.pages ?? 0, sheets: said.sheets ?? [] }
  },
  page: (path, at, wide) => `${asset(path)}/pages/${at}?wide=${wide}`,
  marks: async (path, runs) => {
    const where = runs
      .map((one) => `start=${one.start}&length=${one.length}`)
      .join('&')
    const answer = await served(`${asset(path)}/marks?${where}`)
    const said = (await answer.json()) as { runs?: readonly { marks?: readonly Marked[] }[] }
    return runs.map((_, i) => said.runs?.[i]?.marks ?? [])
  },
}

/**
 * Where a file of the vault is asked about. The path is written out whole, so a
 * file in a folder is one part of the address and the facet asked of it is the
 * next.
 */
const asset = (path: string): string => `/assets/${encodeURIComponent(path)}`

/** How often a document that is busy is waited out before it is a refusal. */
const PATIENCE = 3

/**
 * What the application answered. A document held by whoever is drawing from it
 * is asked for again, after the wait it names.
 */
const served = async (address: string): Promise<Response> => {
  for (let asked = 0; ; asked++) {
    const answer = await fetch(address)
    if (answer.ok) return answer
    if (answer.status !== 503 || asked >= PATIENCE) {
      throw new Error((await answer.text()).trim() || `${answer.status}`)
    }
    await sleep(Number(answer.headers.get('Retry-After') ?? 1) * 1000)
  }
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))

/** How a search is asked, as the schema names it. */
const ways: Record<Way, Ways> = {
  words: Ways.WORDS,
  meaning: Ways.MEANING,
}

/** A run of text, kept as the plain pair the window carries it as. */
const run = (span: { from: number; to: number }) => ({ from: span.from, to: span.to })

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
