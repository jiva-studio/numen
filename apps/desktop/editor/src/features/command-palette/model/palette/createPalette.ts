/**
 * The palette the user types into: opening it, the text typed, and starting or choosing a command.
 *
 * A command that needs nothing runs as soon as it is chosen. One that needs a name, a note, a
 * vault or an answer opens a step of its own.
 */
import type { Command, CommandInvocation, CommandTarget } from '@/entities/command'
import { findUnsuppliedPrompt } from './navigation'
import type { PaletteSession } from './session'
import type { usePaletteView } from './usePaletteView'

function isNameOmittable(command: Command, target: CommandTarget): boolean {
  const isNamedFolder = target.isTargetNamed === true && target.isFolder === true
  return command.canOmitNameOverFolder === true && isNamedFolder
}

function createPaletteCommands(session: PaletteSession) {
  const { deps, state, steps, rows, invocation } = session

  const openSteps = (): void => {
    if (state.open.value) return
    steps.reset()
    state.open.value = true
  }

  const startArguments = (
    command: Command,
    target: CommandTarget,
    values: Readonly<Record<string, string>>,
  ): CommandInvocation | null => {
    const promptIndex = findUnsuppliedPrompt(command, 0, values)
    if (promptIndex < 0) return invocation(command.id, target)
    openSteps()
    steps.pushStep({ step: 'arguments', command, on: target, promptIndex, values })
    return null
  }

  // A command with no input that comes with values was offered by a row or slot entry that carries them.
  const isStartable = (
    command: Command,
    target: CommandTarget,
    values: Readonly<Record<string, string>> | undefined,
  ): boolean => {
    const isOffered = !command.input && values !== undefined && Object.keys(values).length > 0
    return isOffered || command.isAvailable(target, deps.runs)
  }

  /** Starts a command; the text typed for it, where given, answers a step that asks for a name. */
  const startCommand = (
    id: string,
    target: CommandTarget,
    input?: string,
    values?: Readonly<Record<string, string>>,
  ): CommandInvocation | null => {
    const command = rows.byId.value.get(id)
    if (!command) return null
    if (!isStartable(command, target, values)) return null
    if (!command.input) return invocation(command.id, target)
    if (command.input === 'arguments' && values) return startArguments(command, target, values)
    const name = input?.trim() ?? ''
    if (command.input === 'name' && name !== '') return invocation(command.id, target, name)
    if (isNameOmittable(command, target)) return invocation(command.id, target)
    openSteps()
    steps.pushStep({ step: command.input, command, on: target })
    return null
  }

  const getObjection = (id: string, target: CommandTarget): string => {
    const command = rows.byId.value.get(id)
    if (!command || command.isAvailable(target, deps.runs)) return ''
    return target.isReady ? deps.words.noNote : deps.words.noVault
  }

  return { startCommand, getObjection }
}

export function createPalette(session: PaletteSession, view: ReturnType<typeof usePaletteView>) {
  const { state, steps, on, previewItem, setOpen, searchNames } = session
  const { startCommand, getObjection } = createPaletteCommands(session)

  const setTyped = async (text: string) => {
    state.typed.value = text
    // Stryker disable next-line MethodExpression: debounced.ask trims the query.
    if (steps.here.value?.step === 'note') await searchNames(text.trim())
  }

  const chooseItem = (item: string, action: string): CommandInvocation | null => {
    const step = steps.here.value
    if (!step) return startCommand(action, on.value)
    return steps.chooseInStep(
      step,
      item,
      action,
      state.found.value,
      state.known.value,
      view.isVaultAside,
    )
  }

  return {
    open: state.open,
    typed: state.typed,
    setTyped,
    previewItem,
    setOpen,
    startCommand,
    getObjection,
    chooseItem,
  }
}
