/**
 * Asking the application about the vault it is showing.
 *
 * Nothing here describes what an answer looks like: that is the schema, and
 * both halves are generated from it.
 */
import { Code, createClient } from '@connectrpc/connect'
import type { ConnectError } from '@connectrpc/connect'
import {
  ArtifactService,
  AssetService,
  CardsService,
  Counting as Countings,
  Fault as Faults,
  FileService,
  Naming,
  NoteService,
  NoteType as NoteTypes,
  Owed,
  Presence as Presences,
  Role as Roles,
  SearchService,
  Seat as Seats,
  SettingsService,
  SourceKind,
  State as States,
  VaultService,
  VaultsRefusal,
  VaultsService,
  Way as Ways,
  WindowService,
} from '@numen/protocol'
import type {
  Card as CardMessage,
  Cue as CueMessage,
  Deck as DeckMessage,
  Entry as EntryMessage,
  Known as KnownMessage,
  MoveResult as MovedMessage,
  GetNeighbourhoodResponse as NeighbourhoodMessage,
  Page as PageMessage,
  Problem as ProblemMessage,
  Refusal,
  Stencil as StencilMessage,
} from '@numen/protocol'
import type { Counting } from '@numen/ui'
import { fingerprint, refusalIn, staleIn, stamp } from './answers'
import { DEFAULT_PARTS } from './hanging'
import { DEFAULT_STARTS } from './reviewing'
import { standing as settingAt } from './settings/configuring'
import { write } from './settings/json5'
import type { Asking as Commanding } from './commanding'
import type { Asking, Way } from './finding'
import type { Documents, Page, Sheet } from './document/reading'
import type { Cue, Recordings } from './recording/transcript'
import type {
  Added,
  Answered,
  Artifact as ArtifactOf,
  Cards,
  Carded,
  Core,
  Decked,
  Entry,
  Faced,
  Fault,
  Configured,
  Hanging,
  Known,
  Made,
  Moved,
  Movement,
  Neighbourhood,
  NewLink,
  NoteType,
  Offer,
  Presence,
  Problem,
  Reached,
  Refused,
  Removed,
  Renamed,
  Role,
  Runs,
  Seat,
  Source,
  Stencilled,
  VaultRefused,
  Vaults,
} from './core'
import { transport } from './transport'

export const vault = createClient(VaultService, transport)

/** The tree the vault is filed in: what stands where, and moving it about. */
const files = createClient(FileService, transport)

/** What a note holds, what it is joined to, and every way of writing one. */
const notes = createClient(NoteService, transport)

/** What the vault holds that answers what a person typed. */
const finding = createClient(SearchService, transport)

const vaultsService = createClient(VaultsService, transport)

const cardsService = createClient(CardsService, transport)

/** The file this installation is configured in, which is no vault's. */
const settingsService = createClient(SettingsService, transport)

/** This window itself, which is the editor and not the one cards are run in. */
const windowService = createClient(WindowService, transport)

/** What a model has made from the files of the vault. */
const artifacts = createClient(ArtifactService, transport)

/** What the files of the vault are, for whatever opens one. */
const assets = createClient(AssetService, transport)

/** The window every question about a window names. */
const WINDOW = 'editor'

/**
 * The settings the window turns by name, each where it sits in the file.
 *
 * They are read off the settings whole and written back one at a time, so what
 * the window turns and what the settings screen turns are the one surface.
 */
const SYNCS = ['naming', 'sync_title_and_filename']
const HANGS = ['appearance', 'hang_parts_under_a_node']
const PARTS = ['appearance', 'parts_under_a_node']
const STARTS = ['review', 'day_starts']

/** Every setting as it stands, with the defaults under what the file leaves out. */
const configured = async (): Promise<unknown> =>
  JSON.parse((await settingsService.getSettings({})).written)

/** How many parts a node hangs, and the default where the settings name none. */
const partsIn = (value: unknown): number =>
  typeof value === 'number' ? value : DEFAULT_PARTS

/**
 * Settings written into the file, together or not at all. What could not be
 * written, and nothing where it was.
 */
const puts = async (
  written: readonly { at: readonly string[]; value: unknown }[],
): Promise<string | null> => {
  try {
    await settingsService.writeSettings({
      settings: written.map((one) => ({ at: [...one.at], value: write(one.value) })),
    })
  } catch (thrown) {
    return thrown instanceof Error ? thrown.message : `${thrown}`
  }
  return null
}

/** The vaults this installation holds, in the shape the window asks about them. */
export const vaults: Vaults = {
  // The list is the installation's and what is in front of the person is this
  // window's, so the two are asked of the two and put together here.
  list: async () => {
    const [answer, shown] = await Promise.all([
      vaultsService.listVaults({}),
      windowService.getShownVault({ window: WINDOW }),
    ])
    return { vaults: answer.vaults.map(held), showing: shown.vault }
  },
  choose: async (title) => {
    const answer = await vaultsService.chooseFolder({ title, startingAt: '' })
    return answer.chose ? answer.path : ''
  },
  add: async (path, called) => added(await vaultsService.addVault({ path, displayName: called })),
  rename: async (id, called) =>
    added(await vaultsService.renameVault({ name: id, displayName: called })),
  forget: async (id) => turnedDown(await vaultsService.forgetVault({ name: id })),
  erase: async (id) => turnedDown(await vaultsService.eraseVault({ name: id })),
  open: async (id) => turnedDown(await vaultsService.openVault({ name: id })),
}

/** The stencils and the decks of that vault, in the shape the window asks about them. */
export const cards: Cards = {
  stencils: async (limit) => {
    const answer = await cardsService.listStencils({ limit: limit ?? 0 })
    return { stencils: answer.stencils.map(offered), held: answer.held }
  },
  makeDeck: async (title, folder) => {
    const answer = await cardsService.createDeck({ title, folder })
    return { path: answer.path, refusal: refusalIn(answer) }
  },
  makeStencil: async (title, folder, fields) => {
    const answer = await cardsService.createStencil({ title, folder, fields: [...fields] })
    return { path: answer.path, refusal: refusalIn(answer) }
  },
  renameField: async (path, from, to, seen) => {
    const answer = await cardsService.renameStencilField({
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
      changed: staleIn(answer),
      at: stamp(answer.at) ?? '',
    }
  },
  readDeck: async (path) => {
    const answer = await cardsService.readDeck({ path })
    return {
      deck: answer.deck ? decked(answer.deck) : null,
      refusal: refusalIn(answer),
      at: stamp(answer.at) ?? '',
      bound: Number(answer.bound),
    }
  },
  writeDeck: async (path, deck, seen) => {
    const answer = await cardsService.writeDeck({
      path,
      preamble: deck.preamble,
      cards: deck.cards.map(carding),
      sections: deck.sections.map((section) => ({ name: section.name, lead: section.lead })),
      tail: deck.tail,
      ...(seen === null ? {} : { seen: fingerprint(seen) }),
    })
    return {
      refusal: refusalIn(answer),
      changed: staleIn(answer),
      at: stamp(answer.at) ?? '',
      bound: Number(answer.bound),
    }
  },
  readStencil: async (path) => {
    const answer = await cardsService.readStencil({ path })
    return {
      stencil: answer.stencil ? stencilled(answer.stencil) : null,
      refusal: refusalIn(answer),
      at: stamp(answer.at) ?? '',
    }
  },
  writeStencil: async (path, fields, stencil, seen) => {
    const answer = await cardsService.writeStencil({
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
      changed: staleIn(answer),
      at: stamp(answer.at) ?? '',
    }
  },
}

/** The same questions, in the shape the window asks them. */
export const core: Core & Asking & Commanding = {
  vaults: () => vaults.list(),
  neighbourhood: async (path) => around(await notes.getNeighbourhood({ path })),
  opening: async () => (await notes.getOpeningNote({})).note ?? null,
  state: () => vault.getVaultState({}),
  changes: async function* (signal) {
    for await (const change of vault.watchVaultChanges({}, { signal })) {
      yield {
        paths: change.paths,
        reload: change.reload,
        renamed: change.renamed.map((went) => ({ from: went.from, to: went.to })),
      }
    }
  },
  focus: (signal) => vault.watchFocus({}, { signal }),
  attending: async (open) => {
    await vault.writeOpenTabs({ tabs: open.tabs.map((one) => ({ ...one })), front: open.front })
  },
  editing: (signal) => notes.watchEdits({}, { signal }),
  async *tasks(signal) {
    for await (const said of windowService.watchTasks({ window: WINDOW }, { signal })) {
      yield said.tasks.map((at) => ({
        id: at.id,
        doing: at.doing,
        about: at.about,
        done: Number(at.done),
        total: Number(at.total),
        counting: counted[at.counting] ?? 'things',
        failed: at.failed,
        asked: at.asked,
      }))
    }
  },
  read: async (path) => answered(await notes.readNote({ path })),
  write: async (path, body, seen) =>
    answered(await notes.writeNote({ path, body, ...(seen ? { seen: seenOf(seen) } : {}) })),
  create: async (note) => {
    const answer = await notes.createNote({
      title: note.title,
      folder: note.folder,
      links: note.links.map(written),
    })
    return { path: answer.path, refusal: refusalIn(answer) } satisfies Made
  },
  join: async (path, link) => refusalIn(await notes.writeLink({ path, link: written(link) })),
  rename: async (path, title) => {
    const answer = await notes.renameNote({ path, title })
    return {
      path: answer.path,
      title: answer.title,
      frontmatter: answer.by === Naming.FRONTMATTER,
      moved: answer.moved ? filed(answer.moved) : null,
      refusal: refusalIn(answer),
      changed: staleIn(answer),
    } satisfies Renamed
  },
  remove: async (path, destroy) => {
    const answer = await files.removeFile({ path, destroy: destroy ?? false })
    return {
      trashed: answer.trashed,
      dangling: answer.dangling,
      refusal: refusalIn(answer),
    } satisfies Removed
  },
  list: async (folder) => (await files.listFiles({ folder })).entries.map(listed),
  move: async (from, to) => {
    const answer = await files.moveFile({ from, to })
    return {
      moved: answer.moved ? filed(answer.moved) : null,
      refusal: refusalIn(answer),
    } satisfies Movement
  },
  syncing: async () => settingAt(await configured(), SYNCS) !== false,
  choosesSyncing: (kept) => puts([{ at: SYNCS, value: kept }]),
  hanging: async () => {
    const written = await configured()
    return {
      hangs: settingAt(written, HANGS) !== false,
      parts: partsIn(settingAt(written, PARTS)),
    } satisfies Hanging
  },
  // The switch is always sent, and the count only where it is the count being
  // turned.
  choosesHanging: (hangs, parts) =>
    puts([
      { at: HANGS, value: hangs },
      ...(parts === undefined ? [] : [{ at: PARTS, value: parts }]),
    ]),
  settings: async () => {
    const answer = await settingsService.getSettings({})
    return {
      written: answer.written,
      path: answer.path,
      models: answer.models.map((one) => ({
        namedAt: one.namedAt,
        name: one.name,
        title: one.title,
        shelf: one.shelf,
        byDefault: one.byDefault,
        writes: one.writes.map((write) => ({ at: write.at, value: write.value })),
        presence: standing[one.presence],
      })),
    } satisfies Configured
  },
  choosesSetting: async (written) => {
    await settingsService.writeSettings({
      settings: written.map((one) => ({ at: [...one.at], value: one.value })),
    })
  },
  settingsFile: async () => {
    const answer = await settingsService.readSettingsFile({})
    return { written: answer.written, path: answer.path }
  },
  writesSettingsFile: async (written, seen) => {
    const answer = await settingsService.writeSettingsFile({
      written,
      ...(seen === null ? {} : { seen }),
    })
    return { changed: staleIn(answer) }
  },
  reviewing: async () => {
    const hour = settingAt(await configured(), STARTS)
    return typeof hour === 'string' ? hour : DEFAULT_STARTS
  },
  choosesReviewing: (starts) => puts([{ at: STARTS, value: starts }]),
  makeFolder: async (path) => refusalIn(await files.createFolder({ path })),
  quitting: (signal) => windowService.watchQuit({ window: WINDOW }, { signal }),
  flushed: async (token, owed) => {
    await windowService.reportFlush({ window: WINDOW, token, owed: owing[owed ?? 'nothing'] })
  },
  /** What each of the notes asked about is divided into. */
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
  /** What the vault holds at each of those paths. */
  standing: async (paths) => {
    const answer = await files.listFileKinds({ paths: [...paths] })
    return new Map(
      answer.kinds.map((one) => [one.path, { kind: sourceKind(one.kind), type: noteType(one.type) }]),
    )
  },
  /**
   * Where each of those addresses lands, by the address it was asked about. A
   * note an identifier reaches in another vault is left out: this window puts
   * the vault it is showing in its tabs.
   */
  resolve: async (from, written) => {
    const answer = await notes.resolveAddresses({ from, written: [...written] })
    return new Map(
      answer.reached.filter((one) => !one.crossed).map((one) => [one.written, one.path]),
    )
  },
  /** The names in the vault that match what is typed. */
  names: async (query, limit) => {
    const answer = await finding.searchNames({ query, limit })
    return answer.found.map((one) => ({
      path: one.note?.path ?? '',
      title: one.note?.title ?? '',
      heading: one.heading?.text ?? '',
      // A name with no heading stands on no line of the prose.
      line: one.heading?.line ?? -1,
      at: one.at.map(run),
      type: noteType(one.type),
    }))
  },
  /** The text the vault holds that answers what is typed, asked one way. */
  search: async (query, way, limit) => {
    const answer = await finding.searchPassages({ query, limit, way: ways[way] })
    return answer.found.map((one) => ({
      path: one.path,
      title: one.note?.title ?? '',
      // A source that is not a note carries none, and what this window opens
      // one as is the document it is.
      isNote: one.note !== undefined,
      type: noteType(one.type),
      kind: sourceKind(one.kind),
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
    const answer = await waiting(() => assets.getDocument({ path }))
    return {
      pages: answer.pages,
      sheets: answer.sheets.map((one) => ({ wide: one.wide, high: one.high })),
    }
  },
  page: (path, at, wide) => `${asset(path)}/pages/${at}?wide=${wide}`,
  highlights: async (path, stretches) => {
    const answer = await waiting(() => assets.listHighlights({ path, at: [...stretches] }))
    return stretches.map((_, i) => answer.runs[i]?.pages.map(highlighted) ?? [])
  },
}

/** One page of a highlight, as the window carries it. */
const highlighted = (one: PageMessage): Page => ({
  page: one.index,
  rects: one.rects.map((box) => ({
    minX: box.minX,
    minY: box.minY,
    maxX: box.maxX,
    maxY: box.maxY,
  })),
})

/**
 * The recordings the vault holds, over the same addresses. The player is given
 * an address of its own: the window is drawn from a scheme a browser does not
 * load sound through, and the application answers where it does.
 */
export const recordings: Recordings = {
  listened: async (path) => {
    const answer = await waiting(() => assets.getRecording({ path }))
    return {
      length: answer.length,
      heard: answer.heard,
      media: answer.media,
      type: answer.type,
    }
  },
  cues: async (path) => {
    const answer = await waiting(() => artifacts.readTranscript({ path }))
    return { cues: answer.cues.map(spoken), editable: answer.editable }
  },
  writes: async (path, cues) => {
    await waiting(() => artifacts.writeTranscript({ path, cues: [...cues] }))
  },
  plays: async (path, stretch) => {
    const answer = await waiting(() => artifacts.readTranscript({ path, at: stretch }))
    return answer.cues[0]?.from ?? null
  },
}

/** One stretch of speech, kept as the plain value the window carries it as. */
const spoken = (one: CueMessage): Cue => ({ text: one.text, from: one.from, to: one.to })

/**
 * What a model makes from one file of the vault, asked for by name. Which model
 * does the work follows from the file, so the window names the artifact and
 * never the producer.
 */
export const running: Runs = {
  carries: async (path) => {
    const answer = await artifacts.listArtifacts({ path })
    const held: Record<string, Reached> = {}
    for (const one of answer.artifacts) {
      const of = made[one.name.slice(one.name.lastIndexOf('/artifacts/') + '/artifacts/'.length)]
      if (of) held[of] = reached(one.state)
    }
    return held
  },
  makes: async (path, of) => {
    try {
      const answer = await artifacts.createArtifact({ path, artifactId: ids[of] })
      return {
        able: true,
        of,
        made: reached(answer.artifact?.state),
        error: answer.artifact?.error ?? '',
      }
    } catch (error) {
      // A build that cannot make it at all says so, and it is offered nowhere
      // from then on.
      if (Code.Unimplemented === (error as ConnectError).code) return { able: false }
      throw error
    }
  },
  drops: async (path) => {
    try {
      await artifacts.deleteArtifact({ path, artifactId: ids.transcript })
      return true
    } catch (error) {
      if (Code.Unimplemented === (error as ConnectError).code) return false
      throw error
    }
  },
}

/** What each artifact is asked for under, as the schema names it. */
const ids: Record<ArtifactOf, string> = {
  reading: 'ocr',
  transcript: 'asr',
  corrections: 'asr.corrected',
}

/** And back, for reading the id off the end of an artifact's name. */
const made = Object.fromEntries(
  Object.entries(ids).map(([of, id]) => [id, of as ArtifactOf]),
) as Record<string, ArtifactOf>

/** What has become of an artifact, in the words the window uses. */
const become: Record<States, Reached> = {
  [States.UNSPECIFIED]: 'none',
  [States.NONE]: 'none',
  [States.QUEUED]: 'queued',
  [States.RUNNING]: 'running',
  [States.STOPPED]: 'stopped',
  [States.DONE]: 'done',
  [States.EMPTY]: 'empty',
  [States.FAILED]: 'failed',
}

/** A state this window has no word for is an artifact nothing has made. */
const reached = (state: States | undefined): Reached =>
  (state === undefined ? undefined : become[state]) ?? 'none'

/**
 * Where a file of the vault is asked about. The path is written out whole, so a
 * file in a folder is one part of the address and the facet asked of it is the
 * next.
 */
const asset = (path: string): string => `/assets/${encodeURIComponent(path)}`

/** How often a document that is busy is waited out before it is a refusal. */
const PATIENCE = 3

/** How long the window waits before asking a busy document again. */
const AGAIN = 1000

/**
 * What the application answered. A document held by whoever is drawing from it
 * says so, and is asked again after a wait.
 */
const waiting = async <T>(ask: () => Promise<T>): Promise<T> => {
  for (let asked = 0; ; asked++) {
    try {
      return await ask()
    } catch (error) {
      if (Code.Unavailable !== (error as ConnectError).code || asked >= PATIENCE) throw error
      await sleep(AGAIN)
    }
  }
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))

/**
 * What a piece of work counts, in the words the window uses. One it has no word
 * for is counted one by one.
 */
const counted: Record<Countings, Counting> = {
  [Countings.UNSPECIFIED]: 'things',
  [Countings.THINGS]: 'things',
  [Countings.BYTES]: 'bytes',
  [Countings.SECONDS]: 'seconds',
}

/** How a search is asked, in the words the window uses. */
const asked: Partial<Record<Ways, Way>> = {
  [Ways.EVERY]: 'fused',
  [Ways.WORDS]: 'words',
  [Ways.MEANING]: 'meaning',
  [Ways.NAMES]: 'names',
}

/** How a search is asked, as the schema names it. */
const ways = Object.fromEntries(
  Object.entries(asked).map(([said, way]) => [way, Number(said)]),
) as Record<Way, Ways>

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
  at?: { path: string; size: bigint; mtime: bigint } | undefined
}): Answered & { at?: string; changed: boolean } => {
  const at = stamp(from.at)
  return {
    body: from.body ?? '',
    refusal: refusalIn(from),
    changed: staleIn(from),
    ...(at === undefined ? {} : { at }),
  }
}

/** One row of a listing, kept as the plain value the window carries it as. */
const listed = (one: EntryMessage): Entry => ({
  path: one.path,
  displayName: one.displayName,
  folder: one.folder,
  kind: sourceKind(one.kind),
  type: noteType(one.type),
})

/** What the vault holds at a path, in the words the window uses. */
const holding: Record<SourceKind, Source> = {
  [SourceKind.UNSPECIFIED]: 'other',
  [SourceKind.NOTE]: 'note',
  [SourceKind.BOOK]: 'book',
  [SourceKind.RECORDING]: 'recording',
}

/** A source this window has no word for is a file it holds no source for. */
const sourceKind = (of: SourceKind): Source => holding[of] ?? 'other'

/** What a model's files are on this machine, in the words the window uses. */
const standing: Record<Presences, Presence> = {
  [Presences.UNSPECIFIED]: 'nothing to fetch',
  [Presences.PRESENT]: 'present',
  [Presences.NOT_FETCHED]: 'not fetched',
  [Presences.NOTHING_TO_FETCH]: 'nothing to fetch',
}

/** Where a note sits around the note in focus, in the words the window uses. */
const seated: Record<Seats, Seat | null> = {
  [Seats.UNSPECIFIED]: null,
  [Seats.PARENT]: 'parent',
  [Seats.CHILD]: 'child',
  [Seats.JUMP]: 'jump',
  [Seats.SIBLING]: 'sibling',
}

/**
 * A neighbourhood in the words the window uses. A note the window has no seat
 * for, and one the answer names no note at, is not one of them.
 */
const around = (said: NeighbourhoodMessage): Neighbourhood => ({
  focus: { path: said.focus?.path ?? '', title: said.focus?.title ?? '' },
  focusType: noteType(said.focusType),
  related: said.related.flatMap((one) => {
    const seat = seated[one.seat] ?? null
    if (seat === null || one.note === undefined) return []
    return [
      {
        path: one.note.path,
        title: one.note.title,
        type: noteType(one.type),
        seat,
        label: one.label,
        through: one.through,
        mutual: one.mutual,
      },
    ]
  }),
})

/** Which of four a note is, in the words the window uses. */
const typed: Record<NoteTypes, NoteType> = {
  [NoteTypes.UNSPECIFIED]: 'note',
  [NoteTypes.DECK]: 'deck',
  [NoteTypes.STENCIL]: 'stencil',
  [NoteTypes.PRESET]: 'preset',
}

/** A kind this window has no word for is an ordinary note. */
const noteType = (of: NoteTypes): NoteType => typed[of] ?? 'note'

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
  name: one.name,
  displayName: one.displayName,
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
  nothing: Owed.NOTHING,
  written: Owed.WRITTEN,
  asking: Owed.ASKING,
}
