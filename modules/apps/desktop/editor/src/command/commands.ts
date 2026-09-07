/**
 * Every command a person can ask for: what each is called, where it is
 * offered, what it asks for before it happens, and the invocation carrying it out.
 *
 * A command is offered over what is in front of the person, and the table
 * below is the whole of that: one entry a command, in the order they are
 * drawn. Whether a command is offered at all is its own answer to give.
 */
import { shallowRef } from 'vue'
import type { PaletteKeys } from '@numen/ui'
import {
  type ArtifactState,
  type ArtifactStates,
  type Source,
  type VaultList,
} from '../core'
import { keysOf } from './chords'
import type { EmptyWords, NameMatch } from './search'

/** What the commands ask of the application before anything is chosen. */
export interface CommandsDeps {
  /** The names in the vault that match. */
  names(query: string, limit: number): Promise<readonly NameMatch[]>
  /** Every vault the installation holds, and which of them this window shows. */
  vaults(): Promise<VaultList>
}

/** Which group a command is offered in. */
export type CommandGroup = 'note' | 'file' | 'window' | 'vault'

/**
 * Which step the palette is on: one being asked for, or the list of commands.
 * `picking` asks the vault what it holds and `vaults` asks the installation;
 * `choosing` offers a list the window holds already.
 */
export type Step =
  | 'commands'
  | 'naming'
  | 'address'
  | 'picking'
  | 'vaults'
  | 'choosing'
  | 'asking'
  | 'exactly'

/** What a command wants before it can happen, which is the step that asks. */
export type PromptStep = Exclude<Step, 'commands'>

/** The vault a command is over: the identity the list gives it, and its name. */
export interface VaultRef {
  readonly id: string
  readonly name: string
}

/**
 * What is in front of the person, and the note it means. An agent tab means
 * the note the plex is standing on.
 */
export interface CommandTarget {
  /** The tab in front, for a command about the tab itself. */
  readonly tab: string
  /** The word its kind is filed under, and nothing for a tab holding nothing. */
  readonly kind: string | null
  /** The note it means, and nothing where it means none. */
  readonly path: string
  readonly title: string
  /**
   * The file a run is over, and what the vault holds there. A tab holding a
   * book or a recording names the file it holds; a row of the tree names the
   * file the row stands for.
   */
  readonly file: string
  readonly source: Source | null
  /**
   * What that file carries, and what has become of each. A file nothing has
   * been asked about carries nothing here, and a command over it is offered on
   * its kind alone.
   */
  readonly made: ArtifactStates
  /** The other files it is over, beside the one at `path`. */
  readonly others?: readonly string[]
  /** The vault the window is showing, and nothing where it shows none. */
  readonly vault: VaultRef
  /** Whether the vault has been read and can be asked to do anything. */
  readonly ready: boolean
}

/** The words the step that asks for the name typed back is drawn in. */
export interface RetypeWords {
  /** What it does, and what it leaves behind. */
  readonly does: string
  readonly then: string
  /** What stands in the field: the name of the thing, typed back. */
  readonly back: string
}

/** The words the step that confirms is drawn in. */
export interface ConfirmWords {
  /** The answer that changes nothing, and what it leaves. */
  readonly keeps: string
  readonly kept: string
  /** The answer that does it, and what it leaves. */
  readonly does: string
  readonly then: string
}

/** One thing a person can ask for. */
export interface Command {
  readonly id: string
  readonly text: string
  /** The keystroke that reaches it away from the palette. */
  readonly keys?: PaletteKeys
  /** What it asks for before it happens. */
  readonly needs?: PromptStep
  /** The step it asks for once the first one is answered. */
  readonly next?: PromptStep
  /** The group it is offered in. */
  readonly group: CommandGroup
  /** Whether it is offered at all over what is in front, in this window. */
  where(at: CommandTarget, runs: RunSupport): boolean
  /** What stands in the field when its step opens, for the person to replace. */
  filled?(at: CommandTarget): string
  /** What its step says, where that step confirms or asks for the name back. */
  readonly answers?: ConfirmWords
  readonly warns?: RetypeWords
  /** The command Shift and Enter reach on the same row. */
  readonly also?: string
}

/** One command as it is carried out: what it is over, and what was typed for it. */
export interface CommandInvocation {
  readonly id: string
  /** The note it is over. Empty for a command over the window or the vault. */
  readonly path: string
  /** The vault it is over, which is the one the window shows until a step picks another. */
  readonly vault: VaultRef
  /**
   * The identity of the tab holding that note, and nothing where none holds it.
   * A note that moves is at another name by the time the invocation is carried out.
   */
  readonly note: string | null
  readonly title: string
  /** The file a run is over, which is the one `CommandTarget` named. */
  readonly file: string
  /** The other files it is over, beside the one at `path`. */
  readonly others: readonly string[]
  /**
   * What was typed for it: a name to give, or a name typed back. A step that
   * offers a list the window holds puts the one that was chosen here.
   */
  readonly name: string
  /** The kind of tab it was asked from, which is where a note it makes lands. */
  readonly kind: string | null
  /** The tab it was asked from, for a command about the tab itself. */
  readonly tab: string
}

/** Everything the commands say in the window's voice. */
export interface Words extends EmptyWords {
  /** The commands, each in the words it is offered by. */
  readonly read: string
  readonly beside: string
  readonly travel: string
  readonly child: string
  readonly parent: string
  readonly jump: string
  readonly title: string
  readonly remove: string
  readonly destroy: string
  readonly ask: string
  readonly copy: string
  /** The two runs a person asks for over the file in front. */
  readonly transcribe: string
  readonly recognise: string
  /** The transcript of the recording in front, put right by a proofreader. */
  readonly proofread: string
  /** A copy of the video a link note points at, fetched onto this disk. */
  readonly download: string
  /** The transcript of the recording in front, taken away, and the two answers. */
  readonly dropTranscript: string
  readonly keepsTranscript: string
  readonly drops: string
  readonly dropped: string
  /** The note in front, shown where the vault files it. */
  readonly reveal: string
  /** The preset the note in front is, or the one the deck in front is scheduled by. */
  readonly preset: string
  readonly newNote: string
  /** The two files a card is written in: the deck it is one of, and what cuts it. */
  readonly newDeck: string
  readonly newStencil: string
  /** The note that says how the decks pointing at it are scheduled. */
  readonly newPreset: string
  /** The note that points at a web address, made from the address. */
  readonly importUrl: string
  readonly newPlex: string
  /** The folders and files of the vault, put in front of the person. */
  readonly files: string
  readonly newAgent: string
  readonly close: string
  /**
   * The four commands over how the window is drawn: the theme, the halves, and
   * the two sizes.
   */
  readonly appearance: string
  readonly mode: string
  readonly interfaceScale: string
  readonly textScale: string
  /** The command over whether a note's title and its filename are one name. */
  readonly syncing: string
  /** The command over whether a node hangs the parts of its note under it. */
  readonly hanging: string
  /** The command over how many of them stand under a node at once. */
  readonly parts: string
  /** Everything this installation is configured as, in a tab of its own. */
  readonly settings: string
  readonly find: string
  /** The keystroke the search answers to away from the palette. */
  readonly findKeys: PaletteKeys
  readonly first: string
  readonly goto: string
  /** The commands over the vaults this installation holds. */
  readonly openVault: string
  readonly newVault: string
  readonly renameVault: string
  readonly forgetVault: string
  readonly eraseVault: string
  /** The groups the commands are drawn in. */
  readonly overNote: string
  readonly overFile: string
  readonly overWindow: string
  readonly overVault: string
  /** Why nothing can be done to a note: the vault is unread, or none is in front. */
  readonly indexing: string
  readonly noNote: string
  /** The list of commands: the chip beside the field, and what stands in it. */
  readonly command: string
  readonly typeCommand: string
  /** A name asked for, and the one item it offers. */
  readonly naming: string
  readonly typeName: string
  readonly callIt: string
  /**
   * An address asked for: the field, what stands in it, what Enter does, and
   * what is said of words that are no address.
   */
  readonly address: string
  readonly typeAddress: string
  readonly importIt: string
  readonly notAnAddress: string
  /** A note asked for, over the names in the vault. */
  readonly names: string
  readonly typeNote: string
  /**
   * One of a list the window holds: the field, and what Enter does. The groups
   * such a list is drawn in are named by whatever holds it.
   */
  readonly typeChoice: string
  readonly chooses: string
  /** A vault asked for, over the vaults the installation holds. */
  readonly vaults: string
  readonly typeVault: string
  /** The two vaults the list draws and does not offer to choose. */
  readonly gone: string
  /** The row a list of values opens on, which is the value in force. */
  readonly current: string
  /** The two answers to the confirmation: the one that changes nothing, first. */
  readonly asking: string
  /** How many files a command is over, where it is over several. */
  readonly several: (files: number) => string
  readonly answer: string
  readonly kept: string
  /** The two answers over a vault, whose folder is left where it is. */
  readonly keepsVault: string
  readonly forgets: string
  readonly stays: string
  /** The name typed back, which is what destroying asks for. */
  readonly exactly: string
  readonly typeBack: string
  readonly destroys: string
  readonly forever: string
  /** The same, over a vault whose folder goes to the trash this machine keeps. */
  readonly typeVaultBack: string
  readonly erases: string
  readonly binned: string
  /** The note the vault could not find, offered as one to make. */
  readonly creating: string
  readonly creates: string
  /** The seats it can be made in, off the note in front. */
  readonly asChild: string
  readonly asParent: string
  readonly asJump: string
}

/** A command over the note in front, which there has to be one of. */
const onNote = (at: CommandTarget): boolean => at.ready && at.path !== ''

/** A command over the vault in front, which there has to be one of. */
const onVault = (at: CommandTarget): boolean => at.vault.id !== ''

/**
 * The runs this build cannot do at all, as one window has been told them. The
 * application says so the first time one is asked for, and that window offers
 * it nowhere after that.
 */
export interface RunSupport {
  /** Whether this build can do a run at all. A view drawing it follows the answer. */
  canRun(run: string): boolean
  /** A run the application answered it cannot do at all. */
  cannotRun(run: string): void
}

/** The runs one window holds, which is every one of them until it is told otherwise. */
export const runSupport = (): RunSupport => {
  const beyond = shallowRef<ReadonlySet<string>>(new Set())
  return {
    canRun: (run) => !beyond.value.has(run),
    cannotRun: (run) => void (beyond.value = new Set(beyond.value).add(run)),
  }
}

/**
 * A run over the file in front, which the vault has to hold that kind of and
 * this build has to be able to do.
 */
const onSource =
  (run: string, source: Source) =>
  (at: CommandTarget, runs: RunSupport): boolean =>
    at.ready && at.file !== '' && at.source === source && runs.canRun(run)

/**
 * A run over the file in front that is offered on what has been made from it,
 * and not on its kind alone: a book already read is not offered to be read. A
 * file nothing has been asked about carries nothing, and is offered.
 */
const onEvidence =
  (run: string, source: Source, made: (carries: ArtifactStates) => boolean) =>
  (at: CommandTarget, runs: RunSupport): boolean =>
    onSource(run, source)(at, runs) && (isEmpty(at.made) || made(at.made))

const isEmpty = (carries: ArtifactStates): boolean => Object.keys(carries).length === 0

/** An artifact a run over the file would begin. */
const owed = (made: ArtifactState | undefined): boolean =>
  made === undefined || made === 'none' || made === 'stopped'

const always = (): boolean => true

/**
 * Every command, in the order it is drawn. The keyboard it is being read on
 * decides how the keystrokes on it are written.
 *
 * A command reached by Shift and Enter on another one's row is offered here
 * too and drawn nowhere: the row it belongs to is the one that names it.
 */
export const commandsOf = (
  words: Words,
  agent: string = navigator.userAgent,
): readonly Command[] => [
  { id: 'read', text: words.read, group: 'note', where: onNote, also: 'beside' },
  { id: 'beside', text: words.beside, group: 'note', where: onNote },
  { id: 'travel', text: words.travel, ...keysOf('travel', agent), group: 'note', where: onNote },
  {
    id: 'child',
    text: words.child,
    ...keysOf('child', agent),
    group: 'note',
    needs: 'naming',
    where: onNote,
  },
  { id: 'parent', text: words.parent, group: 'note', needs: 'naming', where: onNote },
  { id: 'jump', text: words.jump, group: 'note', needs: 'naming', where: onNote },
  {
    id: 'title',
    text: words.title,
    group: 'note',
    needs: 'naming',
    where: onNote,
    filled: (at) => at.title,
  },
  // The note goes to the vault's .trash folder, and putting it back is a move.
  // Destroy is the one that asks.
  { id: 'remove', text: words.remove, group: 'note', where: onNote, also: 'destroy' },
  {
    id: 'destroy',
    text: words.destroy,
    group: 'note',
    needs: 'exactly',
    where: onNote,
    warns: { does: words.destroys, then: words.forever, back: words.typeBack },
  },
  { id: 'ask', text: words.ask, group: 'note', where: onNote },
  { id: 'copy', text: words.copy, group: 'note', where: onNote },
  { id: 'reveal', text: words.reveal, group: 'note', where: onNote },
  { id: 'preset', text: words.preset, group: 'note', where: onNote },
  {
    id: 'transcribe',
    text: words.transcribe,
    group: 'file',
    where: onEvidence('transcribe', 'recording', (made) => owed(made.transcript)),
  },
  {
    id: 'download',
    text: words.download,
    group: 'file',
    // Only a note pointing at a video carries a copy at all, so the row being
    // there is what says this file is one. An hour of video on somebody's disk
    // is asked for by hand, and one already here is not asked for again.
    where: (at, runs) =>
      at.ready && runs.canRun('download') && at.made.copy !== undefined && owed(at.made.copy),
  },
  {
    id: 'proofread',
    text: words.proofread,
    group: 'file',
    // There is nothing to put right until a model has heard something, and
    // nothing to put right again once it has been put right.
    where: onEvidence(
      'proofread',
      'recording',
      (made) => made.transcript === 'done' && owed(made.corrections),
    ),
  },
  {
    id: 'dropTranscript',
    text: words.dropTranscript,
    group: 'file',
    needs: 'asking',
    // Everything one run of listening left goes, so a run that stopped part way
    // and a recording that gave no words are both taken away here.
    where: onEvidence('dropTranscript', 'recording', (made) => made.transcript !== 'none'),
    answers: {
      keeps: words.keepsTranscript,
      kept: words.kept,
      does: words.drops,
      then: words.dropped,
    },
  },
  {
    id: 'recognise',
    text: words.recognise,
    group: 'file',
    where: onEvidence('recognise', 'book', (made) => owed(made.reading)),
  },
  {
    id: 'note',
    text: words.newNote,
    ...keysOf('note', agent),
    group: 'window',
    needs: 'naming',
    where: (at) => at.ready,
  },
  {
    id: 'deck',
    text: words.newDeck,
    group: 'window',
    needs: 'naming',
    where: (at) => at.ready,
  },
  {
    id: 'stencil',
    text: words.newStencil,
    group: 'window',
    needs: 'naming',
    where: (at) => at.ready,
  },
  {
    id: 'newPreset',
    text: words.newPreset,
    group: 'window',
    needs: 'naming',
    where: (at) => at.ready,
  },
  {
    id: 'importUrl',
    text: words.importUrl,
    group: 'window',
    needs: 'address',
    where: (at) => at.ready,
  },
  { id: 'plex', text: words.newPlex, ...keysOf('plex', agent), group: 'window', where: always },
  { id: 'files', text: words.files, group: 'window', where: always },
  { id: 'agent', text: words.newAgent, ...keysOf('agent', agent), group: 'window', where: always },
  {
    id: 'close',
    text: words.close,
    ...keysOf('close', agent),
    group: 'window',
    where: (at) => at.tab !== '',
  },
  { id: 'find', text: words.find, keys: words.findKeys, group: 'window', where: always },
  { id: 'appearance', text: words.appearance, group: 'window', needs: 'choosing', where: always },
  { id: 'mode', text: words.mode, group: 'window', needs: 'choosing', where: always },
  {
    id: 'interfaceScale',
    text: words.interfaceScale,
    group: 'window',
    needs: 'choosing',
    where: always,
  },
  { id: 'textScale', text: words.textScale, group: 'window', needs: 'choosing', where: always },
  { id: 'syncing', text: words.syncing, group: 'window', needs: 'choosing', where: always },
  { id: 'hanging', text: words.hanging, group: 'window', needs: 'choosing', where: always },
  { id: 'parts', text: words.parts, group: 'window', needs: 'choosing', where: always },
  { id: 'settings', text: words.settings, group: 'window', where: always },
  { id: 'first', text: words.first, group: 'vault', where: (at) => at.ready },
  {
    id: 'goto',
    text: words.goto,
    ...keysOf('goto', agent),
    group: 'vault',
    needs: 'picking',
    where: (at) => at.ready,
  },
  { id: 'openVault', text: words.openVault, group: 'vault', needs: 'vaults', where: always },
  {
    id: 'newVault',
    text: words.newVault,
    ...keysOf('newVault', agent),
    group: 'vault',
    where: always,
  },
  {
    id: 'renameVault',
    text: words.renameVault,
    group: 'vault',
    needs: 'naming',
    where: onVault,
    filled: (at) => at.vault.name,
  },
  {
    id: 'forgetVault',
    text: words.forgetVault,
    group: 'vault',
    needs: 'vaults',
    next: 'asking',
    where: always,
    answers: {
      keeps: words.keepsVault,
      kept: words.kept,
      does: words.forgets,
      then: words.stays,
    },
  },
  {
    id: 'eraseVault',
    text: words.eraseVault,
    group: 'vault',
    needs: 'vaults',
    next: 'exactly',
    where: always,
    warns: { does: words.erases, then: words.binned, back: words.typeVaultBack },
  },
]

/**
 * The commands another one reaches on its own row. They are offered there and
 * drawn nowhere of their own.
 */
const secondary = (commands: readonly Command[]): ReadonlySet<string> =>
  new Set(commands.map((one) => one.also).filter((id) => id !== undefined))

/**
 * The commands of one group, in the order they are drawn, less the ones
 * another command's row reaches.
 */
export const inGroup = (
  commands: readonly Command[],
  group: CommandGroup,
): readonly Command[] => {
  const second = secondary(commands)
  return commands.filter((one) => one.group === group && !second.has(one.id))
}

/** The commands over the note in front, in the order they are drawn. */
export const overNote = (commands: readonly Command[]): readonly Command[] =>
  inGroup(commands, 'note')

/**
 * Whether what was typed asks for the commands: the field held nothing, and
 * what went into it is the one character that means them.
 */
export const asksCommands = (was: string, now: string): boolean => was === '' && now === '>'

/** One command as it is carried out, over what it was asked over. */
export const invocationOf = (
  id: string,
  at: CommandTarget,
  name = '',
  note: string | null = null,
): CommandInvocation => ({
  id,
  path: at.path,
  vault: at.vault,
  note,
  title: at.title,
  file: at.file,
  others: at.others ?? [],
  name,
  kind: at.kind,
  tab: at.tab,
})
