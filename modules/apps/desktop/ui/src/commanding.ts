/**
 * What a person can ask for, and the steps a command asks them for first: what
 * each command is called, where it is offered, and what it wants typed.
 *
 * A command that needs nothing is a deed the moment it is chosen. One that
 * needs a name, a note, a vault or an answer puts the palette on a step of its
 * own, and the step it is on is what the field means.
 */
import { computed, ref, shallowRef } from 'vue'
import type { PaletteBand, PaletteItem, PaletteKeys } from '@numen/ui'
import { asking as latest } from './asking'
import { wentTo, type Known, type Listed, type NoteType, type Source, type Went } from './core'
import { keysOf } from './keying'
import type { Named, Silences } from './finding'

/** What the commands ask of the application before anything is chosen. */
export interface Asking {
  /** The names in the vault that match. */
  names(query: string, limit: number): Promise<readonly Named[]>
  /** Every vault the installation holds, and which of them this window shows. */
  vaults(): Promise<Listed>
}

/** One of a list the window itself holds, as the step that offers it draws it. */
export interface Offered {
  readonly id: string
  readonly title: string
  /** A second line: what is true of this row and not of the ones beside it. */
  readonly detail?: string
  /** Drawn, said, and not chosen. */
  readonly disabled?: boolean
  /** The value the setting this list is of holds now, which is where it opens. */
  readonly inForce?: boolean
}

/** One band of such a list, named by whatever holds it. */
export interface Offering {
  readonly id: string
  readonly title: string
  readonly items: readonly Offered[]
  /** What is said in its place where it holds nothing. */
  readonly silence?: string
}

/**
 * The lists the window holds, and what it does with the one the keyboard is
 * standing on. A list is read again every time the step is drawn, so what the
 * window holds may change while the step stands open.
 */
export interface Holds {
  /**
   * What this command offers now, in the bands it is drawn in. The words typed
   * come too: a list may hold a row made out of them.
   */
  offers(command: string, typed: string): readonly Offering[]
  /** The one the keyboard is standing on, and nothing where it stands on none. */
  shows(command: string, item: string): void
}

/**
 * What the window knows about a note by the name it is filed under. A step
 * stands open while the vault moves under it, and this is read again each time
 * the step is drawn and once more as the deed is made.
 */
export interface Knows {
  /** What it is called now, and nothing where the window names it nothing. */
  called(path: string): string
  /** The identity of the tab holding it, and nothing where none holds it. */
  holding(path: string): string | null
}

/** Which band a command is offered in. */
export type Band = 'note' | 'file' | 'window' | 'vault'

/**
 * Which step the palette is on: one being asked for, or the list of commands.
 * `picking` asks the vault what it holds and `vaults` asks the installation;
 * `choosing` offers a list the window holds already.
 */
export type Step =
  | 'commands'
  | 'naming'
  | 'picking'
  | 'vaults'
  | 'choosing'
  | 'asking'
  | 'exactly'

/** What a command wants before it can happen, which is the step that asks. */
export type Needed = Exclude<Step, 'commands'>

/** The vault a command is over: the identity the list gives it, and its name. */
export interface Shown {
  readonly id: string
  readonly name: string
}

/**
 * What is in front of the person, and the note it means. An agent tab means
 * the note the plex is standing on.
 */
export interface Where {
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
  /** The other files it is over, beside the one at `path`. */
  readonly others?: readonly string[]
  /** The vault the window is showing, and nothing where it shows none. */
  readonly vault: Shown
  /** Whether the vault has been read and can be asked to do anything. */
  readonly ready: boolean
}

/** The words the step that asks for the name typed back is drawn in. */
export interface Warns {
  /** What it does, and what it leaves behind. */
  readonly does: string
  readonly then: string
  /** What stands in the field: the name of the thing, typed back. */
  readonly back: string
}

/** The words the step that confirms is drawn in. */
export interface Answers {
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
  readonly needs?: Needed
  /** The step it asks for once the first one is answered. */
  readonly next?: Needed
  /** The band it is offered in. */
  readonly band: Band
  /** Whether it is offered at all over what is in front, in this window. */
  where(at: Where, runs: Runnable): boolean
  /** What stands in the field when its step opens, for the person to replace. */
  filled?(at: Where): string
  /** What its step says, where that step confirms or asks for the name back. */
  readonly answers?: Answers
  readonly warns?: Warns
  /** The command Shift and Enter reach on the same row. */
  readonly also?: string
}

/** One command as it is carried out: what it is over, and what was typed for it. */
export interface Deed {
  readonly id: string
  /** The note it is over. Empty for a command over the window or the vault. */
  readonly path: string
  /** The vault it is over, which is the one the window shows until a step picks another. */
  readonly vault: Shown
  /**
   * The identity of the tab holding that note, and nothing where none holds it.
   * A note that moves is at another name by the time the deed is carried out.
   */
  readonly note: string | null
  readonly title: string
  /** The file a run is over, which is the one `Where` named. */
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
export interface Words extends Silences {
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
  /** The bands the commands are drawn in. */
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
  /** A note asked for, over the names in the vault. */
  readonly names: string
  readonly typeNote: string
  /**
   * One of a list the window holds: the field, and what Enter does. The bands
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

/** The one item of a step that asks for one thing. */
const NAME = 'name'
const PICK = 'pick'
const OPEN = 'open'
const EXACT = 'exactly'

/** The one thing every item of a list the window holds can be asked. */
const CHOSEN = 'chosen'

/** The two answers of the step that confirms. */
const NO = 'no'
const YES = 'yes'

/** The band and the item that offer to make the note a search did not find. */
export const MAKING = 'creating'

/** How many notes the step that picks one asks for. */
const EACH = 8

/** How long a keystroke waits before the vault is asked. */
const HOLD = 120

/** A command over the note in front, which there has to be one of. */
const onNote = (at: Where): boolean => at.ready && at.path !== ''

/** A command over the vault in front, which there has to be one of. */
const onVault = (at: Where): boolean => at.vault.id !== ''

/**
 * The runs this build cannot do at all, as one window has been told them. The
 * application says so the first time one is asked for, and that window offers
 * it nowhere after that.
 */
export interface Runnable {
  /** Whether this build can do a run at all. A view drawing it follows the answer. */
  canRun(run: string): boolean
  /** A run the application answered it cannot do at all. */
  cannotRun(run: string): void
}

/** The runs one window holds, which is every one of them until it is told otherwise. */
export const runnable = (): Runnable => {
  const beyond = ref<ReadonlySet<string>>(new Set())
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
  (at: Where, runs: Runnable): boolean =>
    at.ready && at.file !== '' && at.source === source && runs.canRun(run)

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
  { id: 'read', text: words.read, band: 'note', where: onNote, also: 'beside' },
  { id: 'beside', text: words.beside, band: 'note', where: onNote },
  { id: 'travel', text: words.travel, ...keysOf('travel', agent), band: 'note', where: onNote },
  {
    id: 'child',
    text: words.child,
    ...keysOf('child', agent),
    band: 'note',
    needs: 'naming',
    where: onNote,
  },
  { id: 'parent', text: words.parent, band: 'note', needs: 'naming', where: onNote },
  { id: 'jump', text: words.jump, band: 'note', needs: 'naming', where: onNote },
  {
    id: 'title',
    text: words.title,
    band: 'note',
    needs: 'naming',
    where: onNote,
    filled: (at) => at.title,
  },
  // The note goes to the vault's .trash folder, and putting it back is a move.
  // Destroy is the one that asks.
  { id: 'remove', text: words.remove, band: 'note', where: onNote, also: 'destroy' },
  {
    id: 'destroy',
    text: words.destroy,
    band: 'note',
    needs: 'exactly',
    where: onNote,
    warns: { does: words.destroys, then: words.forever, back: words.typeBack },
  },
  { id: 'ask', text: words.ask, band: 'note', where: onNote },
  { id: 'copy', text: words.copy, band: 'note', where: onNote },
  { id: 'reveal', text: words.reveal, band: 'note', where: onNote },
  { id: 'preset', text: words.preset, band: 'note', where: onNote },
  {
    id: 'transcribe',
    text: words.transcribe,
    band: 'file',
    where: onSource('transcribe', 'recording'),
  },
  {
    id: 'proofread',
    text: words.proofread,
    band: 'file',
    where: onSource('proofread', 'recording'),
  },
  {
    id: 'dropTranscript',
    text: words.dropTranscript,
    band: 'file',
    needs: 'asking',
    where: onSource('dropTranscript', 'recording'),
    answers: {
      keeps: words.keepsTranscript,
      kept: words.kept,
      does: words.drops,
      then: words.dropped,
    },
  },
  { id: 'recognise', text: words.recognise, band: 'file', where: onSource('recognise', 'book') },
  {
    id: 'note',
    text: words.newNote,
    ...keysOf('note', agent),
    band: 'window',
    needs: 'naming',
    where: (at) => at.ready,
  },
  {
    id: 'deck',
    text: words.newDeck,
    band: 'window',
    needs: 'naming',
    where: (at) => at.ready,
  },
  {
    id: 'stencil',
    text: words.newStencil,
    band: 'window',
    needs: 'naming',
    where: (at) => at.ready,
  },
  {
    id: 'newPreset',
    text: words.newPreset,
    band: 'window',
    needs: 'naming',
    where: (at) => at.ready,
  },
  { id: 'plex', text: words.newPlex, ...keysOf('plex', agent), band: 'window', where: always },
  { id: 'files', text: words.files, band: 'window', where: always },
  { id: 'agent', text: words.newAgent, ...keysOf('agent', agent), band: 'window', where: always },
  {
    id: 'close',
    text: words.close,
    ...keysOf('close', agent),
    band: 'window',
    where: (at) => at.tab !== '',
  },
  { id: 'find', text: words.find, keys: words.findKeys, band: 'window', where: always },
  { id: 'appearance', text: words.appearance, band: 'window', needs: 'choosing', where: always },
  { id: 'mode', text: words.mode, band: 'window', needs: 'choosing', where: always },
  {
    id: 'interfaceScale',
    text: words.interfaceScale,
    band: 'window',
    needs: 'choosing',
    where: always,
  },
  { id: 'textScale', text: words.textScale, band: 'window', needs: 'choosing', where: always },
  { id: 'syncing', text: words.syncing, band: 'window', needs: 'choosing', where: always },
  { id: 'hanging', text: words.hanging, band: 'window', needs: 'choosing', where: always },
  { id: 'parts', text: words.parts, band: 'window', needs: 'choosing', where: always },
  { id: 'settings', text: words.settings, band: 'window', where: always },
  { id: 'first', text: words.first, band: 'vault', where: (at) => at.ready },
  {
    id: 'goto',
    text: words.goto,
    ...keysOf('goto', agent),
    band: 'vault',
    needs: 'picking',
    where: (at) => at.ready,
  },
  { id: 'openVault', text: words.openVault, band: 'vault', needs: 'vaults', where: always },
  {
    id: 'newVault',
    text: words.newVault,
    ...keysOf('newVault', agent),
    band: 'vault',
    where: always,
  },
  {
    id: 'renameVault',
    text: words.renameVault,
    band: 'vault',
    needs: 'naming',
    where: onVault,
    filled: (at) => at.vault.name,
  },
  {
    id: 'forgetVault',
    text: words.forgetVault,
    band: 'vault',
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
    band: 'vault',
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

/** The commands over the note in front, in the order they are drawn. */
export const overNote = (commands: readonly Command[]): readonly Command[] => {
  const second = secondary(commands)
  return commands.filter((one) => one.band === 'note' && !second.has(one.id))
}

/**
 * Whether what was typed asks for the commands: the field held nothing, and
 * what went into it is the one character that means them.
 */
export const asksCommands = (was: string, now: string): boolean => was === '' && now === '>'

/** One command as it is carried out, over what it was asked over. */
export const deedOf = (id: string, at: Where, name = '', note: string | null = null): Deed => ({
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

/**
 * The note a search did not find, made under the words that were looked for.
 * A seat hangs it off the note in front; anything else stands it on its own.
 */
export const creates = (seat: string, name: string, at: Where): Deed =>
  SEATED.includes(seat) && at.path
    ? deedOf(seat, at, name)
    : deedOf('note', { ...at, path: '', title: '' }, name)

/** The seats a note the search did not find can be made in. */
const SEATED: readonly string[] = ['child', 'parent', 'jump']

/**
 * The bands of a search, and the offer to make a note where every one of them
 * answered with nothing. A band still waiting has not answered.
 */
export const offering = (
  bands: readonly PaletteBand[],
  typed: string,
  words: Words,
  at: Where,
): readonly PaletteBand[] => {
  const name = typed.trim()
  const empty = bands.length > 0 && bands.every((one) => one.items.length === 0 && !one.working)
  if (!name || !empty) return bands
  // A note made from a search stands on its own, and the note in front is what
  // it can be joined to as it is made.
  const seats = at.path
    ? [
        { id: 'child', text: words.asChild },
        { id: 'parent', text: words.asParent },
        { id: 'jump', text: words.asJump },
      ]
    : []
  return [
    ...bands,
    {
      id: MAKING,
      title: words.creating,
      items: [
        {
          id: MAKING,
          title: `${words.creates} “${name}”`,
          ...(at.path ? { detail: at.title || at.path } : {}),
          actions: [{ id: MAKING, text: words.creates }, ...seats],
        },
      ],
    },
  ]
}

/** One step of a command: what it asks for, and what it is over. */
interface Asked {
  readonly step: Needed
  readonly command: Command
  readonly on: Where
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))

export function commanding(
  core: Asking,
  words: Words,
  at: () => Where,
  knows: Knows,
  holds: Holds,
  runs: Runnable,
  wait: (ms: number) => Promise<unknown> = sleep,
) {
  /** Whether the commands are drawn at all. */
  const open = ref(false)
  const typed = ref('')

  const commands = commandsOf(words)
  const byId = new Map(commands.map((one) => [one.id, one]))
  const second = secondary(commands)

  /**
   * The steps a command asked for, the last of them the one being asked now.
   * None of them is the list of commands itself.
   */
  const steps = shallowRef<readonly Asked[]>([])

  /** The names the vault answered the step that picks a note with. */
  const found = shallowRef<readonly Named[]>([])
  /** The vaults the installation answered the step that lists them with. */
  const known = shallowRef<readonly Known[]>([])
  /** Which of them that answer said this window is showing. */
  const showing = ref('')
  const working = ref(false)
  /** What the vault could not be asked, in words a person reads. */
  const said = ref('')

  /** A keystroke takes the question over, and only the newest is drawn. */
  const asked = latest()

  /** What the commands are over, as the window stands now. */
  const on = computed<Where>(() => at())

  /** One command as it is carried out, over the note the window holds it by. */
  const deed = (id: string, over: Where, name = ''): Deed =>
    deedOf(id, over, name, over.path ? knows.holding(over.path) : null)

  /**
   * What the thing a step is over is called now. A command over the vault is
   * over the one the window shows; one over a note is over the note at the name
   * it is filed under now.
   */
  const calling = (step: Asked): string =>
    step.command.band === 'vault'
      ? step.on.vault.name
      : knows.called(step.on.path) || step.on.title

  /**
   * What a step is over, as its answer names it: the one thing by the name it
   * carries, or how many things there are.
   */
  const named = (step: Asked): string => {
    const others = step.on.others?.length ?? 0
    return others > 0 ? words.several(others + 1) : `“${calling(step)}”`
  }

  /** The step being asked, and nothing at the list of commands. */
  const here = computed<Asked | null>(() => steps.value.at(-1) ?? null)

  /** Which step this is, in a few words, drawn beside the field. */
  const crumb = computed(() => here.value?.command.text ?? words.command)

  /** What the words typed here will mean, drawn in place of them. */
  const placeholder = computed(() => {
    const step = here.value
    switch (step?.step) {
      case 'naming':
        return words.typeName
      case 'picking':
        return words.typeNote
      case 'choosing':
        return words.typeChoice
      case 'vaults':
        return words.typeVault
      case 'asking':
        return words.answer
      case 'exactly':
        return step.command.warns?.back ?? words.typeBack
      default:
        return words.typeCommand
    }
  })

  /** Which step the palette is on, as one word it hands back. */
  const step = computed(
    () => `${steps.value.length}:${here.value?.command.id ?? ''}:${here.value?.step ?? 'commands'}`,
  )

  /**
   * The row the step opens standing on: the value the list it offers is of.
   * A step offering a list of no value opens where the palette would.
   */
  const opensOn = computed(() => {
    const step = here.value
    if (step?.step !== 'choosing') return ''
    const rows = holds.offers(step.command.id, typed.value).flatMap((band) => band.items)
    return rows.find((one) => one.inForce)?.id ?? ''
  })

  /** Nothing is being asked, and nothing already asked for will be drawn. */
  const drop = () => {
    asked.drop()
    found.value = []
    known.value = []
    showing.value = ''
    working.value = false
    said.value = ''
  }

  /**
   * The names the vault holds that match. The hold is what keeps a question off
   * the vault for every letter of a word.
   */
  const looks = async (query: string) => {
    const mine = asked.ask()
    if (!query) {
      found.value = []
      working.value = false
      said.value = ''
      return
    }
    working.value = true
    said.value = ''
    await wait(HOLD)
    if (!mine.current) return
    try {
      const names = await core.names(query, EACH)
      if (!mine.current) return
      found.value = names
    } catch (error) {
      if (!mine.current) return
      found.value = []
      // The reason goes to the console; the person is told in the window's
      // own voice.
      console.error(error)
      said.value = words.notAsked
    } finally {
      if (mine.current) working.value = false
    }
  }

  /** Every vault the installation holds, asked for as the step that lists them opens. */
  const lists = async () => {
    const mine = asked.ask()
    working.value = true
    said.value = ''
    try {
      const listed = await core.vaults()
      if (!mine.current) return
      known.value = listed.vaults
      showing.value = listed.showing
    } catch (error) {
      if (!mine.current) return
      known.value = []
      // The reason goes to the console; the person is told in the window's
      // own voice.
      console.error(error)
      said.value = words.notAsked
    } finally {
      if (mine.current) working.value = false
    }
  }

  /** Something was typed. What it asks of the vault is the step's own. */
  const typing = async (text: string) => {
    typed.value = text
    if (here.value?.step === 'picking') await looks(text.trim())
  }

  /** One command as it is drawn, and nothing where the words typed leave it out. */
  const drawn = (one: Command, over: Where, word: string): PaletteItem | null => {
    const found = word === '' ? -1 : one.text.toLowerCase().indexOf(word)
    if (word !== '' && found < 0) return null
    const also = one.also ? byId.get(one.also) : undefined
    return {
      id: one.id,
      title: one.text,
      ...(found < 0 ? {} : { at: [{ from: found, to: found + word.length }] }),
      ...(one.keys ? { keys: one.keys } : {}),
      actions: [
        { id: one.id, text: one.text },
        ...(also && also.where(over, runs) ? [{ id: also.id, text: also.text }] : []),
      ],
    }
  }

  /** Why nothing over the note in front is offered. */
  const why = (over: Where): string =>
    !over.ready ? words.indexing : over.path ? words.noneFound : words.noNote

  /** Every command offered over what is in front, in the bands it holds. */
  const listed = (over: Where, text: string): readonly PaletteBand[] => {
    const word = text.trim().toLowerCase()
    const items = (band: Band): readonly PaletteItem[] =>
      commands
        .filter((one) => one.band === band && !second.has(one.id) && one.where(over, runs))
        .map((one) => drawn(one, over, word))
        .filter((item) => item !== null)

    // The runs are offered over a book and over a recording, and their band
    // stands where one of them is in front.
    const overFile = items('file')

    return [
      { id: 'note', title: words.overNote, items: items('note'), silence: why(over) },
      ...(overFile.length === 0 ? [] : [{ id: 'file', title: words.overFile, items: overFile }]),
      { id: 'window', title: words.overWindow, items: items('window'), silence: words.noneFound },
      { id: 'vault', title: words.overVault, items: items('vault'), silence: words.noneFound },
    ]
  }

  /** A name to give, as the one thing the words typed can be. */
  const naming = (text: string): PaletteBand => {
    const name = text.trim()
    return {
      id: 'naming',
      title: words.naming,
      items: name
        ? [
            {
              id: NAME,
              title: `${words.callIt} “${name}”`,
              actions: [{ id: NAME, text: words.callIt }],
            },
          ]
        : [],
      silence: words.typeName,
    }
  }

  /**
   * The notes the vault turned up. A note found by a heading is that note, and
   * a note found twice is one row.
   */
  /** Which of four the note a row of the picking step stands for is. */
  const typeOf = (id: string): NoteType | null =>
    found.value.find((one) => one.path === id)?.type ?? null

  const picking = (text: string): PaletteBand => {
    const seen = new Set<string>()
    const items: PaletteItem[] = []
    for (const one of found.value) {
      if (seen.has(one.path)) continue
      seen.add(one.path)
      items.push({
        id: one.path,
        title: one.title || one.path,
        ...(one.heading ? {} : { at: one.at }),
        actions: [{ id: PICK, text: words.travel }],
      })
    }
    return {
      id: 'picking',
      title: words.names,
      items,
      working: working.value,
      silence: said.value || (text.trim() ? words.noneFound : words.typeNote),
    }
  }

  /**
   * The bands a list the window holds is drawn in, narrowed by the words typed.
   * They are read again on every keystroke, and a row drawn as not to be chosen
   * is not chosen.
   */
  const choosing = (step: Asked, text: string): readonly PaletteBand[] => {
    const word = text.trim().toLowerCase()
    return holds.offers(step.command.id, text).map((band) => ({
      id: band.id,
      title: band.title,
      items: band.items.map((one) => offered(one, word)).filter((item) => item !== null),
      silence: band.silence ?? words.noneFound,
    }))
  }

  /** One such row, and nothing where the words typed leave it out. */
  const offered = (one: Offered, word: string): PaletteItem | null => {
    const found = word === '' ? -1 : one.title.toLowerCase().indexOf(word)
    if (word !== '' && found < 0) return null
    return {
      id: one.id,
      title: one.title,
      ...(found < 0 ? {} : { at: [{ from: found, to: found + word.length }] }),
      ...(one.detail ? { detail: one.detail } : {}),
      ...(one.disabled ? { disabled: true } : {}),
      actions: [{ id: CHOSEN, text: words.chooses }],
    }
  }

  /**
   * Why a vault the list holds is drawn and not chosen: its folder is not
   * there, or it is the one the window is showing. A vault that can be chosen
   * is marked with nothing.
   */
  const aside = (one: Known): string =>
    one.missing ? words.gone : one.name === showing.value ? words.current : ''

  /**
   * The vaults the installation holds. The two it will not take are marked
   * ahead of their folder, and cannot be chosen. The folder is what tells one
   * vault from another, so it keeps the room.
   */
  const listing = (text: string, step: Asked): PaletteBand => {
    const word = text.trim().toLowerCase()
    const items: PaletteItem[] = known.value
      .filter((one) => word === '' || one.displayName.toLowerCase().includes(word))
      .map((one) => {
        const why = aside(one)
        return {
          id: one.name,
          title: one.displayName,
          detail: why ? `${why} · ${one.path}` : one.path,
          ...(why ? { disabled: true } : {}),
          actions: [{ id: OPEN, text: step.command.text }],
        }
      })
    return {
      id: 'vaults',
      title: words.vaults,
      items,
      working: working.value,
      silence: said.value || words.noneFound,
    }
  }

  /**
   * The two answers put before a note goes to the trash. The one that changes
   * nothing is drawn first, and it is the one the keyboard opens on. Each is
   * reached by the name it is offered under, as an item of any other step is.
   */
  const asking = (step: Asked, text: string): PaletteBand => {
    const word = text.trim().toLowerCase()
    const { keeps = '', kept = '', does = '', then = '' }: Partial<Answers> =
      step.command.answers ?? {}
    const items: PaletteItem[] = [
      {
        id: NO,
        title: keeps,
        detail: kept,
        actions: [{ id: NO, text: keeps }],
      },
      {
        id: YES,
        title: `${does} ${named(step)}`,
        detail: then,
        actions: [{ id: YES, text: does }],
      },
    ]
    return {
      id: 'asking',
      title: words.asking,
      items: word
        ? items.filter((one) => (one.actions?.[0]?.text ?? '').toLowerCase().includes(word))
        : items,
      silence: words.answer,
    }
  }

  /** The name typed back, which is the one thing that reaches destroying. */
  const exactly = (step: Asked, text: string): PaletteBand => {
    const title = calling(step)
    const { does = '', then = '' }: Partial<Warns> = step.command.warns ?? {}
    return {
      id: 'exactly',
      title: words.exactly,
      items: [
        {
          id: EXACT,
          title: `${does} “${title}”`,
          detail: then,
          disabled: text.trim() !== title,
          actions: [{ id: EXACT, text: does }],
        },
      ],
    }
  }

  const bands = computed<readonly PaletteBand[]>(() => {
    const step = here.value
    if (!step) return listed(on.value, typed.value)
    if (step.step === 'naming') return [naming(typed.value)]
    if (step.step === 'picking') return [picking(typed.value)]
    if (step.step === 'choosing') return choosing(step, typed.value)
    if (step.step === 'vaults') return [listing(typed.value, step)]
    if (step.step === 'asking') return [asking(step, typed.value)]
    return [exactly(step, typed.value)]
  })

  /** What a step asks of the application as it is put in front of the person. */
  const begins = (step: Asked | null) => {
    if (step?.step === 'vaults') void lists()
  }

  /** A step opened, with whatever it wants the person to replace standing in it. */
  const puts = (step: Asked) => {
    drop()
    typed.value = step.command.filled?.(step.on) ?? ''
    steps.value = [...steps.value, step]
    begins(step)
  }

  /**
   * The item the keyboard is standing on, at a step that shows what it stands
   * on. The window is told the empty string wherever it stands on nothing.
   */
  const lights = (item: string) => {
    const step = here.value
    if (step?.step === 'choosing') holds.shows(step.command.id, item)
  }

  /** The step being asked goes, and the one under it is asked again. */
  const pops = () => {
    lights('')
    drop()
    typed.value = ''
    steps.value = steps.value.slice(0, -1)
    begins(here.value)
  }

  /**
   * A command asked for, from the palette or from a menu. One that needs
   * something opens the step that asks for it; one that needs nothing is handed
   * straight back to be carried out.
   */
  const asks = (id: string, over: Where): Deed | null => {
    const command = byId.get(id)
    if (!command || !command.where(over, runs)) return null
    if (!command.needs) return deed(command.id, over)
    if (!open.value) {
      steps.value = []
      open.value = true
    }
    puts({ step: command.needs, command, on: over })
    return null
  }

  /**
   * Why a command asked for did nothing: the vault is unread, or what it was
   * asked over is not a note. One that was taken up says nothing.
   */
  const refused = (id: string, over: Where): string => {
    const command = byId.get(id)
    if (!command || command.where(over, runs)) return ''
    return over.ready ? words.noNote : words.indexing
  }

  /** A note that moved. A step open over it is asked at the name it now has. */
  const follows = (renamed: readonly Went[] = []) => {
    if (!renamed.length) return
    steps.value = steps.value.map((step) => {
      const to = wentTo(renamed, step.on.path)
      return to ? { ...step, on: { ...step.on, path: to } } : step
    })
  }

  /** An item chosen, and what was asked of it. */
  const chose = (item: string, action: string): Deed | null => {
    const step = here.value
    if (!step) return asks(action, on.value)
    const name = typed.value.trim()
    if (step.step === 'picking') {
      const one = found.value.find((found) => found.path === item)
      if (!one) return null
      return deed(step.command.id, { ...step.on, path: one.path, title: one.title || one.path })
    }
    if (step.step === 'vaults') {
      const one = known.value.find((vault) => vault.name === item)
      // The two the list draws and does not take are the ones it says so on.
      if (!one || aside(one)) return null
      const on = { ...step.on, vault: { id: one.name, name: one.displayName } }
      if (!step.command.next) return deed(step.command.id, on)
      // The vault chosen is what the step after this one is over.
      puts({ step: step.command.next, command: step.command, on })
      return null
    }
    if (step.step === 'naming') {
      if (!name) return null
      return deed(step.command.id, step.on, name)
    }
    if (step.step === 'choosing') {
      const rows = holds.offers(step.command.id, typed.value).flatMap((band) => band.items)
      const one = rows.find((row) => row.id === item)
      if (!one || one.disabled) return null
      return deed(step.command.id, step.on, one.id)
    }
    if (step.step === 'asking') {
      // The answer that changes nothing puts the step away.
      if (action === NO) {
        pops()
        return null
      }
      if (action !== YES) return null
      return deed(step.command.id, step.on)
    }
    // The name typed back is what reaches destroying, measured against the name
    // the note carries now.
    if (action !== EXACT || name !== calling(step)) return null
    return deed(step.command.id, step.on, name)
  }

  /** The commands are opened, or put away and every step let go of. */
  const shows = (now: boolean) => {
    lights('')
    open.value = now
    drop()
    typed.value = ''
    steps.value = []
  }

  /** Escape: the step goes, and the commands go with it at the list itself. */
  const leaves = () => {
    if (here.value) return pops()
    shows(false)
  }

  /**
   * Backspace in an empty field: the step goes. At the list of commands the
   * field goes back to the search, where the character that opened them was
   * typed.
   */
  const backs = (): boolean => {
    if (here.value) {
      pops()
      return false
    }
    shows(false)
    return true
  }

  return {
    open,
    typed,
    bands,
    crumb,
    step,
    opensOn,
    placeholder,
    typing,
    lights,
    shows,
    asks,
    refused,
    follows,
    chose,
    leaves,
    backs,
    typeOf,
  }
}

/** The commands of one window, over whatever is in front of the person. */
export type Commands = ReturnType<typeof commanding>
