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
import { isWebAddress } from './address'
import { answerGuard as latest } from '../questions'
import { movedTo, type Move } from '../note'
import type { Vault } from '../vaults'
import { commandsOf } from './commands'
import { view } from './view'
import type { RunSupport } from './runs'
import { EXACT, NO, YES, type PendingStep } from './step'
import {
  invocationOf,
  type CommandsDeps,
  type CommandTarget,
  type CommandInvocation,
  type Words,
} from './target'
import type { NoteLookup, PaletteLists } from './lists'
import type { NameMatch } from './search'

/** How many notes the step that picks one asks for. */
const EACH = 8

/** How long a keystroke waits before the vault is asked. */
const HOLD = 120

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))

export function useCommandPalette(
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

  const draws = view({
    words,
    commands,
    byId,
    runs,
    holds,
    found,
    known,
    showing,
    working,
    said,
    calling,
    named,
  })

  const groups = computed(() => draws.groupsOf(here.value, on.value, typed.value))


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
   * Why a command asked for did nothing: the vault never opened, or what it was
   * asked over is not a note. One that was taken up says nothing.
   */
  const refused = (id: string, over: CommandTarget): string => {
    const command = byId.get(id)
    if (!command || command.where(over, runs)) return ''
    return over.ready ? words.noNote : words.noVault
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
      const one = known.value.find((vault) => vault.id === item)
      // The two the list draws and does not take are the ones it says so on.
      if (!one || draws.aside(one)) return null
      const on = { ...step.on, vault: { id: one.id, name: one.name } }
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
      if (!isWebAddress(name)) return null
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
  const setOpen = (now: boolean) => {
    lights('')
    open.value = now
    drop()
    typed.value = ''
    steps.value = []
  }
  const shows = setOpen

  /** Escape: the step goes, and the commands go with it at the list itself. */
  const leaves = () => {
    if (here.value) return pops()
    setOpen(false)
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
    setOpen(false)
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
    setOpen,
    shows,
    asks,
    refused,
    follows,
    chose,
    leaves,
    backs,
    typeOf: draws.typeOf,
  }
}

/** The commands of one window, over whatever is in front of the person. */
export type Commands = ReturnType<typeof useCommandPalette>
