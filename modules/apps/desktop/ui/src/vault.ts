/**
 * Asking the application about the vault it is showing.
 *
 * Nothing here describes what an answer looks like: that is the schema, and
 * both halves are generated from it.
 */
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import {
  CardsService,
  Counting,
  Fault as Faults,
  Naming,
  NoteType as NoteTypes,
  Owed,
  Role as Roles,
  SourceKind,
  VaultService,
  VaultsRefusal,
  VaultsService,
  Way as Ways,
} from '@numen/protocol'
import type {
  Card as CardMessage,
  Deck as DeckMessage,
  Entry as EntryMessage,
  Known as KnownMessage,
  Moved as MovedMessage,
  Problem as ProblemMessage,
  Refusal,
  Stencil as StencilMessage,
} from '@numen/protocol'
import { fingerprint, refusalIn, stamp } from './answers'
import type { Asking as Commanding } from './commanding'
import type { Asking, Way } from './finding'
import type { Documents, Marked, Sheet } from './document/reading'
import type { Cue, Recordings } from './recording/listening'
import type {
  Added,
  Answered,
  Cards,
  Carded,
  Core,
  Decked,
  Entry,
  Faced,
  Fault,
  Hanging,
  Known,
  Made,
  Moved,
  Movement,
  NewLink,
  NoteType,
  Offer,
  Problem,
  Refused,
  Removed,
  Renamed,
  Role,
  Source,
  Stencilled,
  VaultRefused,
  Vaults,
} from './core'

const transport = createConnectTransport({ baseUrl: window.location.origin })

export const vault = createClient(VaultService, transport)

const listing = createClient(VaultsService, transport)

const cutting = createClient(CardsService, transport)

/** The vaults this installation holds, in the shape the window asks about them. */
export const vaults: Vaults = {
  list: async () => {
    const answer = await listing.list({})
    return { vaults: answer.vaults.map(held), showing: answer.showing }
  },
  choose: async (title) => {
    const answer = await listing.choose({ title, startingAt: '' })
    return answer.chose ? answer.path : ''
  },
  add: async (path, name) => added(await listing.add({ path, name })),
  rename: async (id, name) => added(await listing.rename({ id, name })),
  forget: async (id) => turnedDown(await listing.forget({ id })),
  erase: async (id) => turnedDown(await listing.erase({ id })),
  open: async (id) => turnedDown(await listing.open({ id })),
}

/** The stencils and the decks of that vault, in the shape the window asks about them. */
export const cards: Cards = {
  stencils: async (limit) => {
    const answer = await cutting.stencils({ limit: limit ?? 0 })
    return { stencils: answer.stencils.map(offered), held: answer.held }
  },
  makeDeck: async (title, folder) => {
    const answer = await cutting.makeDeck({ title, folder })
    return { path: answer.path, refusal: refusalIn(answer) }
  },
  makeStencil: async (title, folder, fields) => {
    const answer = await cutting.makeStencil({ title, folder, fields: [...fields] })
    return { path: answer.path, refusal: refusalIn(answer) }
  },
  renameField: async (path, from, to, seen) => {
    const answer = await cutting.renameField({
      path,
      from,
      to,
      ...(seen === null ? {} : { seen: fingerprint(seen) }),
    })
    return {
      decks: answer.decks,
      cards: answer.cards,
      notWritten: answer.notWritten.map((one) => ({
        path: one.path,
        text: one.problem?.text ?? '',
      })),
      refusal: refusalIn(answer),
      changed: answer.changed,
      at: stamp(answer.at) ?? '',
    }
  },
  readDeck: async (path) => {
    const answer = await cutting.readDeck({ path })
    return {
      deck: answer.deck ? decked(answer.deck) : null,
      refusal: refusalIn(answer),
      at: stamp(answer.at) ?? '',
      bound: Number(answer.bound),
    }
  },
  writeDeck: async (path, deck, seen) => {
    const answer = await cutting.writeDeck({
      path,
      preamble: deck.preamble,
      cards: deck.cards.map(carding),
      sections: deck.sections.map((section) => ({ name: section.name, lead: section.lead })),
      tail: deck.tail,
      ...(seen === null ? {} : { seen: fingerprint(seen) }),
    })
    return {
      refusal: refusalIn(answer),
      changed: answer.changed,
      at: stamp(answer.at) ?? '',
      bound: Number(answer.bound),
    }
  },
  readStencil: async (path) => {
    const answer = await cutting.readStencil({ path })
    return {
      stencil: answer.stencil ? stencilled(answer.stencil) : null,
      refusal: refusalIn(answer),
      at: stamp(answer.at) ?? '',
    }
  },
  writeStencil: async (path, fields, stencil, seen) => {
    const answer = await cutting.writeStencil({
      path,
      fields: [...fields],
      preamble: stencil.preamble,
      faces: stencil.faces.map((face) => ({
        name: face.name,
        lead: face.lead,
        front: face.front,
        back: face.back,
      })),
      tail: stencil.tail,
      ...(seen === null ? {} : { seen: fingerprint(seen) }),
    })
    return {
      refusal: refusalIn(answer),
      changed: answer.changed,
      at: stamp(answer.at) ?? '',
    }
  },
}

/** The same questions, in the shape the window asks them. */
export const core: Core & Asking & Commanding = {
  vaults: () => vaults.list(),
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
  attending: async (open) => {
    await vault.attending({ tabs: open.tabs.map((one) => ({ ...one })), front: open.front })
  },
  editing: (signal) => vault.editing({}, { signal }),
  async *tasks(signal) {
    for await (const said of vault.tasks({}, { signal })) {
      yield said.tasks.map((at) => ({
        id: at.id,
        doing: at.doing,
        about: at.about,
        done: Number(at.done),
        total: Number(at.total),
        counting: at.counting === Counting.BYTES ? ('bytes' as const) : ('things' as const),
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
  rename: async (path, title) => {
    const answer = await vault.rename({ path, title })
    return {
      path: answer.path,
      title: answer.title,
      frontmatter: answer.by === Naming.FRONTMATTER,
      moved: answer.moved ? filed(answer.moved) : null,
      refusal: refusalIn(answer),
      changed: answer.changed,
    } satisfies Renamed
  },
  remove: async (path, destroy) => {
    const answer = await vault.remove({ path, destroy: destroy ?? false })
    return {
      trashed: answer.trashed,
      dangling: answer.dangling,
      refusal: refusalIn(answer),
    } satisfies Removed
  },
  list: async (folder) => (await vault.list({ folder })).entries.map(listed),
  move: async (from, to) => {
    const answer = await vault.move({ from, to })
    return {
      moved: answer.moved ? filed(answer.moved) : null,
      refusal: refusalIn(answer),
    } satisfies Movement
  },
  syncing: async () => (await vault.syncing({})).syncTitleAndFilename,
  choosesSyncing: async (kept) =>
    refusalIn(await vault.chooseSyncing({ syncTitleAndFilename: kept })),
  hanging: async () => {
    const answer = await vault.hanging({})
    return {
      hangs: answer.hangPartsUnderANode,
      parts: answer.partsUnderANode,
    } satisfies Hanging
  },
  // The switch is always sent, and the count only where it is the count being
  // turned.
  choosesHanging: async (hangs, parts) =>
    refusalIn(
      await vault.chooseHanging({ hangPartsUnderANode: hangs, partsUnderANode: parts }),
    ),
  makeFolder: async (path) => refusalIn(await vault.makeFolder({ path })),
  quitting: (signal) => vault.quitting({}, { signal }),
  flushed: async (token, owed) => {
    await vault.flushed({ token, owed: owing[owed ?? 'nothing'] })
  },
  /** What each of the notes asked about is divided into. */
  headings: async (paths) => {
    const answer = await vault.headings({ paths: [...paths] })
    return new Map(
      answer.found.map((one) => [
        one.path,
        one.headings.map((heading) => ({
          text: heading.text,
          level: heading.level,
          line: heading.line,
        })),
      ]),
    )
  },
  /** What the vault holds at each of those paths. */
  standing: async (paths) => {
    const answer = await vault.standing({ paths: [...paths] })
    return new Map(
      answer.found.map((one) => [one.path, { kind: holding[one.kind], type: typed[one.type] }]),
    )
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
      line: one.line,
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
 * The recordings the vault holds, over the same addresses. The player is given
 * an address of its own: the window is drawn from a scheme a browser does not
 * load sound through, and the application answers where it does.
 */
export const recordings: Recordings = {
  listened: async (path) => {
    const answer = await served(asset(path))
    const said = (await answer.json()) as {
      length?: number
      heard?: number
      media?: string
      type?: string
    }
    return {
      length: said.length ?? 0,
      heard: said.heard ?? 0,
      media: said.media ?? '',
      type: said.type ?? '',
    }
  },
  cues: async (path) => {
    const answer = await served(`${asset(path)}/cues`)
    const said = (await answer.json()) as { cues?: readonly Cue[]; editable?: boolean }
    return { cues: said.cues ?? [], editable: said.editable ?? true }
  },
  writes: async (path, cues) => {
    await served(`${asset(path)}/cues`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path, cues }),
    })
  },
  plays: async (path, run) => {
    const where = `start=${run.start}&length=${run.length}`
    const answer = await served(`${asset(path)}/cues?${where}`)
    const said = (await answer.json()) as { cues?: readonly Cue[] }
    return said.cues?.[0]?.from ?? null
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
const served = async (address: string, asking?: RequestInit): Promise<Response> => {
  for (let asked = 0; ; asked++) {
    const answer = await fetch(address, asking)
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

/** What kind of relationship a link is, as the schema names it. */
const roles: Record<Role, Roles> = {
  parent: Roles.PARENT,
  child: Roles.CHILD,
  jump: Roles.JUMP,
  ref: Roles.REF,
  attachment: Roles.ATTACHMENT,
}

/** A link in the shape the schema carries it. */
const written = (link: NewLink) => ({
  to: link.to,
  role: roles[link.role],
  label: link.label ?? '',
})

const seenOf = (seen: { prose: string; at: string }) => ({
  prose: seen.prose,
  at: fingerprint(seen.at),
})

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

/** One row of a listing, kept as the plain value the window carries it as. */
const listed = (one: EntryMessage): Entry => ({
  path: one.path,
  name: one.name,
  folder: one.folder,
  kind: holding[one.kind],
  type: typed[one.type],
})

/** What the vault holds at a path, in the words the window uses. */
const holding: Record<SourceKind, Source> = {
  [SourceKind.UNSPECIFIED]: 'other',
  [SourceKind.NOTE]: 'note',
  [SourceKind.BOOK]: 'book',
  [SourceKind.RECORDING]: 'recording',
}

/** Which of four a note is, in the words the window uses. */
const typed: Record<NoteTypes, NoteType> = {
  [NoteTypes.UNSPECIFIED]: 'note',
  [NoteTypes.DECK]: 'deck',
  [NoteTypes.STENCIL]: 'stencil',
  [NoteTypes.PRESET]: 'preset',
}

/** One stencil of the list, kept as the plain value the window carries it as. */
const offered = (one: { path: string; title: string; fields: string[] }): Offer => ({
  path: one.path,
  title: one.title,
  fields: one.fields,
})

/** A deck as the window carries it. */
const decked = (one: DeckMessage): Decked => ({
  path: one.path,
  title: one.title,
  preamble: one.preamble,
  cards: one.cards.map(carded),
  sections: one.sections.map((section) => ({ name: section.name, lead: section.lead })),
  tail: one.tail,
  problems: one.problems.map(problem),
})

/** A stencil as the window carries it. */
const stencilled = (one: StencilMessage): Stencilled => ({
  path: one.path,
  title: one.title,
  fields: one.fields,
  preamble: one.preamble,
  faces: one.faces.map(
    (face): Faced => ({
      name: face.name,
      lead: face.lead,
      front: face.front,
      back: face.back,
    }),
  ),
  tail: one.tail,
  problems: one.problems.map(problem),
})

const carded = (one: CardMessage): Carded => ({
  mark: one.mark,
  section: one.section ?? null,
  heading: one.heading,
  stencil: one.stencil,
  stencilAt: one.stencilAt,
  lead: one.lead,
  values: one.values.map((value) => ({ field: value.field, text: value.text })),
})

/**
 * One card in the shape the schema carries it. The heading goes back as it
 * came: a write reads it again from the first field, except for the one card
 * whose stencil cannot be read, whose heading is left exactly as it stands.
 */
const carding = (one: Carded) => ({
  mark: one.mark,
  ...(one.section === null ? {} : { section: one.section }),
  heading: one.heading,
  stencil: one.stencil,
  lead: one.lead,
  values: one.values.map((value) => ({ field: value.field, text: value.text })),
})

/** One problem, with where it stands kept as a number or as nothing. */
const problem = (one: ProblemMessage): Problem => ({
  fault: faulted[one.fault],
  card: one.card ?? null,
  face: one.face ?? null,
  field: one.field,
  text: one.text,
})

/** What a problem is, in the words the window uses. */
const faulted: Record<Faults, Fault> = {
  [Faults.UNSPECIFIED]: 'unknown',
  [Faults.FIELD_DECLARED_TWICE]: 'fieldDeclaredTwice',
  [Faults.STENCIL_WITHOUT_FIELDS]: 'stencilWithoutFields',
  [Faults.FACE_MISSING_A_SIDE]: 'faceMissingASide',
  [Faults.PLACEHOLDER_UNDECLARED]: 'placeholderUndeclared',
  [Faults.CARD_WITHOUT_A_STENCIL]: 'cardWithoutAStencil',
  [Faults.STENCIL_IS_NOT_ONE]: 'stencilIsNotOne',
  [Faults.MARK_CARRIED_TWICE]: 'markCarriedTwice',
  [Faults.FIELD_WRITTEN_TWICE]: 'fieldWrittenTwice',
  [Faults.FIELD_NOT_RENAMED]: 'fieldNotRenamed',
}

/** One vault of the list, kept as the plain value the window carries it as. */
const held = (one: KnownMessage): Known => ({
  id: one.id,
  name: one.name,
  path: one.path,
  missing: one.missing,
})

const added = (from: {
  vault?: KnownMessage | undefined
  refusal?: VaultsRefusal | undefined
}): Added => ({
  vault: from.vault ? held(from.vault) : null,
  refusal: turnedDown(from),
})

const turnedDown = (from: { refusal?: VaultsRefusal | undefined }): VaultRefused | null =>
  from.refusal === undefined ? null : unvaulted[from.refusal]

const unvaulted: Record<VaultsRefusal, VaultRefused> = {
  [VaultsRefusal.UNSPECIFIED]: 'unreadable',
  [VaultsRefusal.UNREADABLE]: 'unreadable',
  [VaultsRefusal.COPY]: 'copy',
  [VaultsRefusal.OVERLAPS]: 'overlaps',
  [VaultsRefusal.NAME_TAKEN]: 'nameTaken',
  [VaultsRefusal.LAST_VAULT]: 'lastVault',
  [VaultsRefusal.SHOWING]: 'showing',
  [VaultsRefusal.UNKNOWN]: 'unknown',
  [VaultsRefusal.NO_TRASH]: 'noTrash',
  [VaultsRefusal.ASKING]: 'asking',
}

/** What the file did, in the shape the window carries it. */
const filed = (moved: MovedMessage): Moved => ({
  from: moved.from,
  to: moved.to,
  repaired: moved.repaired,
})

/** What a client has left, as the schema names it. */
const owing: Record<'nothing' | 'written' | 'asking', Owed> = {
  nothing: Owed.UNSPECIFIED,
  written: Owed.WRITTEN,
  asking: Owed.ASKING,
}
