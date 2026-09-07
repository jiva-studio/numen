/**
 * The palette a person types into: which step it stands on, what it draws
 * there, and the invocation an answer makes.
 *
 * A command that needs nothing is carried out the moment it is chosen. One that
 * needs a name, a note, a vault or an answer opens a step of its own, and the
 * step it stands on is what the field means. What a step draws is read again
 * on every keystroke, so a vault moving under an open step is drawn as it is.
 */
import { computed, ref, shallowRef } from 'vue'
import type { PaletteGroup, PaletteItem } from '@numen/ui'
import { answerGuard as latest } from '../questions'
import { movedTo, type Move, type NoteType, type Vault } from '../core'
import {
  commandsOf,
  invocationOf,
  inGroup,
  type Command,
  type CommandGroup,
  type CommandsDeps,
  type CommandTarget,
  type ConfirmWords,
  type CommandInvocation,
  type PromptStep,
  type RetypeWords,
  type RunSupport,
  type Words,
} from './commands'
import type { NoteLookup, PaletteLists, StepRow } from './lists'
import type { NameMatch } from './search'

/** The one item of a step that asks for one thing. */
const NAME = 'name'
const PICK = 'pick'
const OPEN = 'open'
const EXACT = 'exactly'

/** The one thing every item of a list the window holds can be asked. */
const CHOSEN = 'chosen'

/**
 * Whether these words are somewhere a browser would go, which is what decides
 * that the step offers a row at all. The vault reads the address again and is
 * what refuses one nothing can be fetched from.
 */
const fetchable = (typed: string): boolean => {
  try {
    const address = new URL(typed.trim())
    return (address.protocol === 'http:' || address.protocol === 'https:') && address.hostname !== ''
  } catch {
    return false
  }
}

/** The two answers of the step that confirms. */
const NO = 'no'
const YES = 'yes'

/** How many notes the step that picks one asks for. */
const EACH = 8

/** How long a keystroke waits before the vault is asked. */
const HOLD = 120

/** One step of a command: what it asks for, and what it is over. */
interface PendingStep {
  readonly step: PromptStep
  readonly command: Command
  readonly on: CommandTarget
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))

export function commandPalette(
  core: CommandsDeps,
  words: Words,
  at: () => CommandTarget,
  knows: NoteLookup,
  holds: PaletteLists,
  runs: RunSupport,
  wait: (ms: number) => Promise<unknown> = sleep,
) {
  /** Whether the commands are drawn at all. */
  const open = ref(false)
  const typed = ref('')

  const commands = commandsOf(words)
  const byId = new Map(commands.map((one) => [one.id, one]))

  /**
   * The steps a command asked for, the last of them the one being asked now.
   * None of them is the list of commands itself.
   */
  const steps = shallowRef<readonly PendingStep[]>([])

  /** The names the vault answered the step that picks a note with. */
  const found = shallowRef<readonly NameMatch[]>([])
  /** The vaults the installation answered the step that lists them with. */
  const known = shallowRef<readonly Vault[]>([])
  /** Which of them that answer said this window is showing. */
  const showing = ref('')
  const working = ref(false)
  /** What the vault could not be asked, in words a person reads. */
  const said = ref('')

  /** A keystroke takes the question over, and only the newest is drawn. */
  const asked = latest()

  /** What the commands are over, as the window stands now. */
  const on = computed<CommandTarget>(() => at())

  /** One command as it is carried out, over the note the window holds it by. */
  const invocation = (id: string, over: CommandTarget, name = ''): CommandInvocation =>
    invocationOf(id, over, name, over.path ? knows.holding(over.path) : null)

  /**
   * What the thing a step is over is called now. A command over the vault is
   * over the one the window shows; one over a note is over the note at the name
   * it is filed under now.
   */
  const calling = (step: PendingStep): string =>
    step.command.group === 'vault'
      ? step.on.vault.name
      : knows.called(step.on.path) || step.on.title

  /**
   * What a step is over, as its answer names it: the one thing by the name it
   * carries, or how many things there are.
   */
  const named = (step: PendingStep): string => {
    const others = step.on.others?.length ?? 0
    return others > 0 ? words.several(others + 1) : `“${calling(step)}”`
  }

  /** The step being asked, and nothing at the list of commands. */
  const here = computed<PendingStep | null>(() => steps.value.at(-1) ?? null)

  /** Which step this is, in a few words, drawn beside the field. */
  const crumb = computed(() => here.value?.command.text ?? words.command)

  /** What the words typed here will mean, drawn in place of them. */
  const placeholder = computed(() => {
    const step = here.value
    switch (step?.step) {
      case 'naming':
        return words.typeName
      case 'address':
        return words.typeAddress
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
    const rows = holds.offers(step.command.id, typed.value).flatMap((group) => group.items)
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
  const drawn = (one: Command, over: CommandTarget, word: string): PaletteItem | null => {
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
  const why = (over: CommandTarget): string =>
    !over.ready ? words.indexing : over.path ? words.noneFound : words.noNote

  /** Every command offered over what is in front, in the groups it holds. */
  const listed = (over: CommandTarget, text: string): readonly PaletteGroup[] => {
    const word = text.trim().toLowerCase()
    const items = (group: CommandGroup): readonly PaletteItem[] =>
      inGroup(commands, group)
        .filter((one) => one.where(over, runs))
        .map((one) => drawn(one, over, word))
        .filter((item) => item !== null)

    // The runs are offered over a book and over a recording, and their group
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
  const naming = (text: string): PaletteGroup => {
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
   * An address to point at, as the one thing the words typed can be. What is
   * offered is what a browser would go to; the vault reads it again and is what
   * refuses one it cannot fetch.
   */
  const address = (text: string): PaletteGroup => {
    const raw = text.trim()
    return {
      id: 'address',
      title: words.address,
      items: fetchable(raw)
        ? [
            {
              id: NAME,
              title: `${words.importIt} “${raw}”`,
              actions: [{ id: NAME, text: words.importIt }],
            },
          ]
        : [],
      silence: raw ? words.notAnAddress : words.typeAddress,
    }
  }

  /**
   * The notes the vault turned up. A note found by a heading is that note, and
   * a note found twice is one row.
   */
  /** Which of four the note a row of the picking step stands for is. */
  const typeOf = (id: string): NoteType | null =>
    found.value.find((one) => one.path === id)?.type ?? null

  const picking = (text: string): PaletteGroup => {
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
   * The groups a list the window holds is drawn in, narrowed by the words typed.
   * They are read again on every keystroke, and a row drawn as not to be chosen
   * is not chosen.
   */
  const choosing = (step: PendingStep, text: string): readonly PaletteGroup[] => {
    const word = text.trim().toLowerCase()
    return holds.offers(step.command.id, text).map((group) => ({
      id: group.id,
      title: group.title,
      items: group.items.map((one) => offered(one, word)).filter((item) => item !== null),
      silence: group.silence ?? words.noneFound,
    }))
  }

  /** One such row, and nothing where the words typed leave it out. */
  const offered = (one: StepRow, word: string): PaletteItem | null => {
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
  const aside = (one: Vault): string =>
    one.missing ? words.gone : one.name === showing.value ? words.current : ''

  /**
   * The vaults the installation holds. The two it will not take are marked
   * ahead of their folder, and cannot be chosen. The folder is what tells one
   * vault from another, so it keeps the room.
   */
  const listing = (text: string, step: PendingStep): PaletteGroup => {
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
  const asking = (step: PendingStep, text: string): PaletteGroup => {
    const word = text.trim().toLowerCase()
    const { keeps = '', kept = '', does = '', then = '' }: Partial<ConfirmWords> =
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
  const exactly = (step: PendingStep, text: string): PaletteGroup => {
    const title = calling(step)
    const { does = '', then = '' }: Partial<RetypeWords> = step.command.warns ?? {}
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

  const groups = computed<readonly PaletteGroup[]>(() => {
    const step = here.value
    if (!step) return listed(on.value, typed.value)
    if (step.step === 'naming') return [naming(typed.value)]
    if (step.step === 'address') return [address(typed.value)]
    if (step.step === 'picking') return [picking(typed.value)]
    if (step.step === 'choosing') return choosing(step, typed.value)
    if (step.step === 'vaults') return [listing(typed.value, step)]
    if (step.step === 'asking') return [asking(step, typed.value)]
    return [exactly(step, typed.value)]
  })

  /** What a step asks of the application as it is put in front of the person. */
  const begins = (step: PendingStep | null) => {
    if (step?.step === 'vaults') void lists()
  }

  /** A step opened, with whatever it wants the person to replace standing in it. */
  const puts = (step: PendingStep) => {
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
  const asks = (id: string, over: CommandTarget): CommandInvocation | null => {
    const command = byId.get(id)
    if (!command || !command.where(over, runs)) return null
    if (!command.needs) return invocation(command.id, over)
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
  const refused = (id: string, over: CommandTarget): string => {
    const command = byId.get(id)
    if (!command || command.where(over, runs)) return ''
    return over.ready ? words.noNote : words.indexing
  }

  /** A note that moved. A step open over it is asked at the name it now has. */
  const follows = (renamed: readonly Move[] = []) => {
    if (!renamed.length) return
    steps.value = steps.value.map((step) => {
      const to = movedTo(renamed, step.on.path)
      return to ? { ...step, on: { ...step.on, path: to } } : step
    })
  }

  /** An item chosen, and what was asked of it. */
  const chose = (item: string, action: string): CommandInvocation | null => {
    const step = here.value
    if (!step) return asks(action, on.value)
    const name = typed.value.trim()
    if (step.step === 'picking') {
      const one = found.value.find((found) => found.path === item)
      if (!one) return null
      return invocation(step.command.id, { ...step.on, path: one.path, title: one.title || one.path })
    }
    if (step.step === 'vaults') {
      const one = known.value.find((vault) => vault.name === item)
      // The two the list draws and does not take are the ones it says so on.
      if (!one || aside(one)) return null
      const on = { ...step.on, vault: { id: one.name, name: one.displayName } }
      if (!step.command.next) return invocation(step.command.id, on)
      // The vault chosen is what the step after this one is over.
      puts({ step: step.command.next, command: step.command, on })
      return null
    }
    if (step.step === 'naming') {
      if (!name) return null
      return invocation(step.command.id, step.on, name)
    }
    if (step.step === 'address') {
      if (!fetchable(name)) return null
      return invocation(step.command.id, step.on, name)
    }
    if (step.step === 'choosing') {
      const rows = holds.offers(step.command.id, typed.value).flatMap((group) => group.items)
      const one = rows.find((row) => row.id === item)
      if (!one || one.disabled) return null
      return invocation(step.command.id, step.on, one.id)
    }
    if (step.step === 'asking') {
      // The answer that changes nothing puts the step away.
      if (action === NO) {
        pops()
        return null
      }
      if (action !== YES) return null
      return invocation(step.command.id, step.on)
    }
    // The name typed back is what reaches destroying, measured against the name
    // the note carries now.
    if (action !== EXACT || name !== calling(step)) return null
    return invocation(step.command.id, step.on, name)
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
    groups,
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
export type Commands = ReturnType<typeof commandPalette>
