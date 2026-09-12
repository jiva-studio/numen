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
import { answerGuard as latest } from '@/shared/questions'
import type { Vault } from '@/shared/vaults'
import { commandsOf } from './commands'
import { view } from './view'
import type { RunSupport } from './runs'
import {
  invocationOf,
  type CommandsDeps,
  type CommandTarget,
  type CommandInvocation,
  type Words,
} from './target'
import type { NoteLookup, PaletteLists } from './rows'
import type { NameMatch } from './search'
import { createPaletteSteps } from './navigation'

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
  const open = ref(false)
  const typed = ref('')

  const commands = commandsOf(words)
  const byId = new Map(commands.map((one) => [one.id, one]))

  const found = shallowRef<readonly NameMatch[]>([])
  const known = shallowRef<readonly Vault[]>([])
  const showing = ref('')
  const isWorking = ref(false)
  const said = ref('')

  const asked = latest()

  const on = computed<CommandTarget>(() => at())

  const invocation = (id: string, over: CommandTarget, name = ''): CommandInvocation =>
    invocationOf(id, over, name, over.path ? knows.holding(over.path) : null)

  const drop = () => {
    asked.drop()
    found.value = []
    known.value = []
    showing.value = ''
    isWorking.value = false
    said.value = ''
  }

  const setOpen = (now: boolean) => {
    lights('')
    open.value = now
    drop()
    typed.value = ''
    steps.reset()
  }

  const lights = (item: string) => {
    const step = steps.here.value
    if (step?.step === 'choosing') holds.shows(step.command.id, item)
  }

  const lists = async () => {
    const mine = asked.ask()
    isWorking.value = true
    said.value = ''
    try {
      const listed = await core.vaults()
      if (!mine.current) return
      known.value = listed.vaults
      showing.value = listed.showing
    } catch (error) {
      if (!mine.current) return
      known.value = []
      console.error(error)
      said.value = words.notAsked
    } finally {
      if (mine.current) isWorking.value = false
    }
  }

  const steps = createPaletteSteps(
    words,
    holds,
    knows,
    typed,
    drop,
    lights,
    () => void lists(),
    () => setOpen(false),
    invocation,
  )

  const looks = async (query: string) => {
    const mine = asked.ask()
    if (!query) {
      found.value = []
      isWorking.value = false
      said.value = ''
      return
    }
    isWorking.value = true
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
      console.error(error)
      said.value = words.notAsked
    } finally {
      if (mine.current) isWorking.value = false
    }
  }

  const typing = async (text: string) => {
    typed.value = text
    if (steps.here.value?.step === 'picking') await looks(text.trim())
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
    working: isWorking,
    said,
    calling: steps.calling,
    named: steps.named,
  })

  const groups = computed(() => draws.groupsOf(steps.here.value, on.value, typed.value))

  const asks = (id: string, over: CommandTarget): CommandInvocation | null => {
    const command = byId.get(id)
    if (!command || !command.where(over, runs)) return null
    if (!command.needs) return invocation(command.id, over)
    if (!open.value) {
      steps.reset()
      open.value = true
    }
    steps.puts({ step: command.needs, command, on: over })
    return null
  }

  const refused = (id: string, over: CommandTarget): string => {
    const command = byId.get(id)
    if (!command || command.where(over, runs)) return ''
    return over.ready ? words.noNote : words.noVault
  }

  const chose = (item: string, action: string): CommandInvocation | null => {
    const step = steps.here.value
    if (!step) return asks(action, on.value)
    return steps.chooseInStep(step, item, action, found.value, known.value, (one) => Boolean(draws.aside(one)))
  }

  return {
    open,
    typed,
    groups,
    crumb: steps.crumb,
    step: steps.step,
    opensOn: steps.opensOn,
    placeholder: steps.placeholder,
    typing,
    lights,
    setOpen,
    shows: setOpen,
    asks,
    refused,
    follows: steps.follows,
    chose,
    leaves: steps.leaves,
    backs: steps.backs,
    typeOf: draws.typeOf,
  }
}

export type Commands = ReturnType<typeof useCommandPalette>
