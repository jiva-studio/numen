/**
 * Asking the application about the vault it is showing.
 *
 * Nothing here describes what an answer looks like: that is the schema, and
 * both halves are generated from it.
 */
import { createClient } from '@connectrpc/connect'
import {
  FileService,
  FlushResult,
  NamedBy,
  NoteService,
  NoteType as NoteTypes,
  Presence as Presences,
  Role as Roles,
  SearchMode as Modes,
  SearchService,
  Seat as Seats,
  SettingsService,
  SourceKind,
  Unit as Units,
  VaultService,
  VaultsRefusal,
  VaultsService,
  WindowService,
} from '@numen/protocol'
import type {
  Entry as EntryMessage,
  MoveResult as MoveResultMessage,
  GetNeighbourhoodResponse as NeighbourhoodMessage,
  Refusal,
  Vault as VaultMessage,
} from '@numen/protocol'
import type { TallyUnit } from '@numen/ui'
import { namesOf, transport, troubleWords } from '@numen/wire'
import { fingerprint, refusalIn, staleIn, stamp } from './answers'
import { DEFAULT_PARTS } from './settings/hanging'
import { DEFAULT_STARTS } from './settings/review'
import { settingAt } from './settings/store'
import { write } from './settings/json5'
import type { CommandsDeps } from './command/commands'
import type { SearchDeps, SearchMode } from './command/search'
import type {
  Core,
  Entry,
  Configuration,
  HangingSettings,
  ReviewSettings,
  MakeResult,
  MoveResult,
  Movement,
  Neighbourhood,
  NewLink,
  NoteResult,
  NoteType,
  Presence,
  RemoveResult,
  RenameResult,
  Role,
  Seat,
  Source,
  Vault,
  VaultRefusalReason,
  VaultResult,
  Vaults,
} from './core'

export const vault = createClient(VaultService, transport)

/** The tree the vault is filed in: what stands where, and moving it about. */
const files = createClient(FileService, transport)

/** What a note holds, what it is joined to, and every way of writing one. */
const notes = createClient(NoteService, transport)

/** What the vault holds that answers what a person typed. */
const finding = createClient(SearchService, transport)

const vaultsService = createClient(VaultsService, transport)

/** The file this installation is configured in, which is no vault's. */
const settingsService = createClient(SettingsService, transport)

/** This window itself, which is the editor and not the one cards are run in. */
const windowService = createClient(WindowService, transport)

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
    return troubleWords(thrown)
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
  remove: async (id, trash) => turnedDown(await vaultsService.removeVault({ name: id, trash })),
  open: async (id) => turnedDown(await vaultsService.openVault({ name: id })),
}

/** The same questions, in the shape the window asks them. */
export const core: Core & SearchDeps & CommandsDeps = {
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
        counting: counted[at.unit] ?? 'things',
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
    return { path: answer.path, refusal: refusalIn(answer) } satisfies MakeResult
  },
  join: async (path, link) => refusalIn(await notes.writeLink({ path, link: written(link) })),
  rename: async (path, title) => {
    const answer = await notes.renameNote({ path, title })
    return {
      path: answer.path,
      title: answer.title,
      frontmatter: writes[answer.by],
      moved: answer.moved ? filed(answer.moved) : null,
      refusal: refusalIn(answer),
      changed: staleIn(answer),
    } satisfies RenameResult
  },
  remove: async (path, destroy) => {
    const answer = await files.removeFile({ path, destroy: destroy ?? false })
    return {
      trashed: answer.trashed,
      dangling: answer.dangling,
      refusal: refusalIn(answer),
    } satisfies RemoveResult
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
    const answer = await settingsService.getSettings({})
    const written = JSON.parse(answer.written)
    const held = answer.partsUnderANodeBounds
    return {
      hangs: settingAt(written, HANGS) !== false,
      parts: partsIn(settingAt(written, PARTS)),
      least: held?.least ?? DEFAULT_PARTS,
      most: held?.most ?? DEFAULT_PARTS,
    } satisfies HangingSettings
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
        presence: fetched[one.presence],
      })),
    } satisfies Configuration
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
    const answer = await settingsService.getSettings({})
    const hour = settingAt(JSON.parse(answer.written), STARTS)
    return {
      starts: typeof hour === 'string' ? hour : DEFAULT_STARTS,
      latest: answer.latestDayStarts,
    } satisfies ReviewSettings
  },
  choosesReviewing: (starts) => puts([{ at: STARTS, value: starts }]),
  makeFolder: async (path) => refusalIn(await files.createFolder({ path })),
  quitting: (signal) => windowService.watchQuit({ window: WINDOW }, { signal }),
  flushed: async (token, owed) => {
    await windowService.reportFlush({ window: WINDOW, token, result: owing[owed ?? 'nothing'] })
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
  fileKinds: async (paths) => {
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
      answer.resolved.filter((one) => !one.crossed).map((one) => [one.written, one.path]),
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
  /** The text the vault holds that answers what is typed, asked in one mode. */
  search: async (query, mode, limit) => {
    const answer = await finding.searchPassages({ query, limit, mode: modes[mode] })
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
 * What a piece of work counts, in the words the window uses. One it has no word
 * for is counted one by one.
 */
const counted: Record<Units, TallyUnit> = {
  [Units.UNSPECIFIED]: 'things',
  [Units.THINGS]: 'things',
  [Units.BYTES]: 'bytes',
  [Units.SECONDS]: 'seconds',
}

/**
 * Whether a rename wrote the title into the note rather than moving its file.
 * Keyed by the schema, so a namer added to it has to be answered here before
 * this compiles.
 */
const writes: Record<NamedBy, boolean> = {
  [NamedBy.UNSPECIFIED]: false,
  [NamedBy.FRONTMATTER]: true,
  [NamedBy.FILENAME]: false,
}

/**
 * How a search is asked, in the words the window uses. Keyed by the schema, so
 * a mode added to it has to be given a word here before this compiles.
 */
const asked: Record<Modes, SearchMode | null> = {
  [Modes.UNSPECIFIED]: null,
  [Modes.HYBRID]: 'hybrid',
  [Modes.WORDS]: 'words',
  [Modes.MEANING]: 'meaning',
  [Modes.NAMES]: 'names',
}

/** How a search is asked, as the schema names it. */
const modes = namesOf<SearchMode, Modes>(asked)

/** A run of text, kept as the plain pair the window carries it as. */
const run = (span: { from: number; to: number }) => ({ from: span.from, to: span.to })

/**
 * What kind of relationship a link is, in the words the window uses. Keyed by
 * the schema, so a role added to it has to be given a word here before this
 * compiles, and the window cannot quietly go on knowing four of five.
 */
const called: Record<Roles, Role | null> = {
  [Roles.UNSPECIFIED]: null,
  [Roles.PARENT]: 'parent',
  [Roles.CHILD]: 'child',
  [Roles.JUMP]: 'jump',
  [Roles.REF]: 'ref',
  [Roles.ATTACHMENT]: 'attachment',
}

/** What kind of relationship a link is, as the schema names it. */
const roles = namesOf<Role, Roles>(called)

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
}): NoteResult & { at?: string; changed: boolean } => {
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
  name: one.name,
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
const fetched: Record<Presences, Presence> = {
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

/** One vault of the list, kept as the plain value the window carries it as. */
const held = (one: VaultMessage): Vault => ({
  name: one.name,
  displayName: one.displayName,
  path: one.path,
  missing: one.missing,
})

const added = (from: {
  vault?: VaultMessage | undefined
  refusal?: VaultsRefusal | undefined
}): VaultResult => ({
  vault: from.vault ? held(from.vault) : null,
  refusal: turnedDown(from),
})

const turnedDown = (from: { refusal?: VaultsRefusal | undefined }): VaultRefusalReason | null =>
  from.refusal === undefined ? null : unvaulted[from.refusal]

const unvaulted: Record<VaultsRefusal, VaultRefusalReason> = {
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
const filed = (moved: MoveResultMessage): MoveResult => ({
  from: moved.from,
  to: moved.to,
  repaired: moved.repaired,
})

/**
 * What a client has left, in the words the window uses. Keyed by the schema,
 * so a result added to it has to be given a word here before this compiles.
 */
const left: Record<FlushResult, 'nothing' | 'written' | 'asking' | null> = {
  [FlushResult.UNSPECIFIED]: null,
  [FlushResult.NOTHING]: 'nothing',
  [FlushResult.WRITTEN]: 'written',
  [FlushResult.ASKING]: 'asking',
}

/** And back, which is the one direction this ever travels in. */
const owing = namesOf<NonNullable<(typeof left)[FlushResult]>, FlushResult>(left)
