/**
 * What a person can ask for, and the steps a command asks them for first.
 *
 * The palette draws bands of items and knows nothing of vaults or windows.
 * This is the vocabulary: what each command is called, where it is offered,
 * and what it wants typed before it can happen. Carrying one out is `doing.ts`.
 *
 * A command that needs nothing is a deed the moment it is chosen. One that
 * needs a name, a note or an answer puts the palette on a step of its own, and
 * the step it is on is what the field means.
 */
import { computed, ref, shallowRef } from 'vue'
import type { PaletteBand, PaletteItem } from '@numen/ui'
import { wentTo, type Went } from './core'
import type { Named, Silences } from './finding'

/** The one question the commands ask of the vault: the names in it that match. */
export interface Asking {
  names(query: string, limit: number): Promise<readonly Named[]>
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
export type Band = 'note' | 'window' | 'vault'

/** Which step the palette is on: one being asked for, or the list of commands. */
export type Step = 'commands' | 'naming' | 'picking' | 'asking' | 'exactly'

/** What a command wants before it can happen, which is the step that asks. */
export type Needed = Exclude<Step, 'commands'>

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
  /** Whether the vault has been read and can be asked to do anything. */
  readonly ready: boolean
}

/** One thing a person can ask for. */
export interface Command {
  readonly id: string
  readonly text: string
  /** The keystroke that reaches it away from the palette. */
  readonly keys?: string
  /** What it asks for before it happens. */
  readonly needs?: Needed
  /** The band it is offered in. */
  readonly band: Band
  /** Whether it is offered at all over what is in front. */
  where(at: Where): boolean
  /** What stands in the field when its step opens, for the person to replace. */
  filled?(at: Where): string
  /** The command Shift and Enter reach on the same row. */
  readonly also?: string
}

/** One command as it is carried out: what it is over, and what was typed for it. */
export interface Deed {
  readonly id: string
  /** The note it is over. Empty for a command over the window or the vault. */
  readonly path: string
  /**
   * The identity of the tab holding that note, and nothing where none holds it.
   * A note that moves is at another name by the time the deed is carried out.
   */
  readonly note: string | null
  readonly title: string
  /** What was typed for it: a name to give, or a name typed back. */
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
  readonly newNote: string
  readonly newPlex: string
  readonly newAgent: string
  readonly close: string
  readonly find: string
  /** The keystroke the search answers to away from the palette. */
  readonly findKeys: string
  readonly first: string
  readonly goto: string
  /** The three bands the commands are drawn in. */
  readonly overNote: string
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
  /** The two answers to the confirmation: the one that changes nothing, first. */
  readonly asking: string
  readonly answer: string
  readonly keeps: string
  readonly kept: string
  readonly removes: string
  readonly trashed: string
  /** The name typed back, which is what destroying asks for. */
  readonly exactly: string
  readonly typeBack: string
  readonly destroys: string
  readonly forever: string
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
const EXACT = 'exactly'

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

const always = (): boolean => true

/**
 * Every command, in the order it is drawn.
 *
 * A command reached by Shift and Enter on another one's row is offered here
 * too and drawn nowhere: the row it belongs to is the one that names it.
 */
export const commandsOf = (words: Words): readonly Command[] => [
  { id: 'read', text: words.read, band: 'note', where: onNote, also: 'beside' },
  { id: 'beside', text: words.beside, band: 'note', where: onNote },
  { id: 'travel', text: words.travel, band: 'note', where: onNote },
  { id: 'child', text: words.child, band: 'note', needs: 'naming', where: onNote },
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
  {
    id: 'remove',
    text: words.remove,
    band: 'note',
    needs: 'asking',
    where: onNote,
    also: 'destroy',
  },
  { id: 'destroy', text: words.destroy, band: 'note', needs: 'exactly', where: onNote },
  { id: 'ask', text: words.ask, band: 'note', where: onNote },
  { id: 'copy', text: words.copy, band: 'note', where: onNote },
  { id: 'note', text: words.newNote, band: 'window', needs: 'naming', where: (at) => at.ready },
  { id: 'plex', text: words.newPlex, band: 'window', where: always },
  { id: 'agent', text: words.newAgent, band: 'window', where: always },
  { id: 'close', text: words.close, band: 'window', where: (at) => at.tab !== '' },
  { id: 'find', text: words.find, keys: words.findKeys, band: 'window', where: always },
  { id: 'first', text: words.first, band: 'vault', where: (at) => at.ready },
  { id: 'goto', text: words.goto, band: 'vault', needs: 'picking', where: (at) => at.ready },
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
  note,
  title: at.title,
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
  const waiting = ref(false)
  /** What the vault could not be asked, in words a person reads. */
  const said = ref('')

  /**
   * Which question is the current one. A keystroke and every answer to what was
   * asked before it are measured against this, and only the newest is drawn.
   */
  let asked = 0

  /** What the commands are over, as the window stands now. */
  const on = computed<Where>(() => at())

  /** One command as it is carried out, over the note the window holds it by. */
  const deed = (id: string, over: Where, name = ''): Deed =>
    deedOf(id, over, name, over.path ? knows.holding(over.path) : null)

  /** What the note a step is over is called now. */
  const calling = (step: Asked): string => knows.called(step.on.path) || step.on.title

  /** The step being asked, and nothing at the list of commands. */
  const here = computed<Asked | null>(() => steps.value.at(-1) ?? null)

  /** Which step this is, in a few words, drawn beside the field. */
  const crumb = computed(() => here.value?.command.text ?? words.command)

  /** What the words typed here will mean, drawn in place of them. */
  const placeholder = computed(() => {
    switch (here.value?.step) {
      case 'naming':
        return words.typeName
      case 'picking':
        return words.typeNote
      case 'asking':
        return words.answer
      case 'exactly':
        return words.typeBack
      default:
        return words.typeCommand
    }
  })

  /** Which step the palette is on, as one word it hands back. */
  const step = computed(
    () => `${steps.value.length}:${here.value?.command.id ?? ''}:${here.value?.step ?? 'commands'}`,
  )

  /** Nothing is being asked, and nothing already asked for will be drawn. */
  const drop = () => {
    asked += 1
    found.value = []
    waiting.value = false
    said.value = ''
  }

  /**
   * The names the vault holds that match. The hold is what keeps a question off
   * the vault for every letter of a word.
   */
  const looks = async (query: string) => {
    const mine = ++asked
    if (!query) {
      found.value = []
      waiting.value = false
      said.value = ''
      return
    }
    waiting.value = true
    said.value = ''
    await wait(HOLD)
    if (mine !== asked) return
    try {
      const names = await core.names(query, EACH)
      if (mine !== asked) return
      found.value = names
    } catch (error) {
      if (mine !== asked) return
      found.value = []
      // The reason goes to the console; the person is told in the window's
      // own voice.
      console.error(error)
      said.value = words.notAsked
    } finally {
      if (mine === asked) waiting.value = false
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
    // A command over a note is drawn with the note it is over.
    const detail = one.band === 'note' ? over.title : ''
    const also = one.also ? byId.get(one.also) : undefined
    return {
      id: one.id,
      title: one.text,
      ...(found < 0 ? {} : { at: [{ from: found, to: found + word.length }] }),
      ...(detail ? { detail } : {}),
      ...(one.keys ? { keys: one.keys } : {}),
      actions: [
        { id: one.id, text: one.text },
        ...(also && also.where(over) ? [{ id: also.id, text: also.text }] : []),
      ],
    }
  }

  /** Why nothing over the note in front is offered. */
  const why = (over: Where): string =>
    !over.ready ? words.indexing : over.path ? words.noneFound : words.noNote

  /** Every command offered over what is in front, in the three bands. */
  const listed = (over: Where, text: string): readonly PaletteBand[] => {
    const word = text.trim().toLowerCase()
    const items = (band: Band): readonly PaletteItem[] =>
      commands
        .filter((one) => one.band === band && !second.has(one.id) && one.where(over))
        .map((one) => drawn(one, over, word))
        .filter((item) => item !== null)

    return [
      { id: 'note', title: words.overNote, items: items('note'), silence: why(over) },
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
      working: waiting.value,
      silence: said.value || (text.trim() ? words.noneFound : words.typeNote),
    }
  }

  /**
   * The two answers put before a note goes to the trash. The one that changes
   * nothing is drawn first, and it is the one the keyboard opens on. Each is
   * reached by the name it is offered under, as an item of any other step is.
   */
  const asking = (step: Asked, text: string): PaletteBand => {
    const word = text.trim().toLowerCase()
    const items: PaletteItem[] = [
      {
        id: NO,
        title: words.keeps,
        detail: words.kept,
        actions: [{ id: NO, text: words.keeps }],
      },
      {
        id: YES,
        title: `${words.removes} “${calling(step)}”`,
        detail: words.trashed,
        actions: [{ id: YES, text: words.removes }],
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
    return {
      id: 'exactly',
      title: words.exactly,
      items: [
        {
          id: EXACT,
          title: `${words.destroys} “${title}”`,
          detail: words.forever,
          disabled: text.trim() !== title,
          actions: [{ id: EXACT, text: words.destroys }],
        },
      ],
    }
  }

  const bands = computed<readonly PaletteBand[]>(() => {
    const step = here.value
    if (!step) return listed(on.value, typed.value)
    if (step.step === 'naming') return [naming(typed.value)]
    if (step.step === 'picking') return [picking(typed.value)]
    if (step.step === 'asking') return [asking(step, typed.value)]
    return [exactly(step, typed.value)]
  })

  /** A step opened, with whatever it wants the person to replace standing in it. */
  const puts = (step: Asked) => {
    drop()
    typed.value = step.command.filled?.(step.on) ?? ''
    steps.value = [...steps.value, step]
  }

  /** The step being asked goes, and the one under it is asked again. */
  const pops = () => {
    drop()
    typed.value = ''
    steps.value = steps.value.slice(0, -1)
  }

  /**
   * A command asked for, from the palette or from a menu. One that needs
   * something opens the step that asks for it; one that needs nothing is handed
   * straight back to be carried out.
   */
  const asks = (id: string, over: Where): Deed | null => {
    const command = byId.get(id)
    if (!command || !command.where(over)) return null
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
    if (!command || command.where(over)) return ''
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
    if (step.step === 'naming') {
      if (!name) return null
      return deed(step.command.id, step.on, name)
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
    placeholder,
    typing,
    shows,
    asks,
    refused,
    follows,
    chose,
    leaves,
    backs,
  }
}
