/**
 * The palette the user types into: opening it, the text typed, and starting or choosing a command.
 *
 * A command that needs nothing runs as soon as it is chosen. One that needs a name, a note, a
 * vault or an answer opens a step of its own.
 */
import type { Command } from '@/entities/command'
import type { CommandInvocation, CommandTarget } from '@/shared/model/command'
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
    steps.stack.reset()
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
    steps.stack.pushStep({ step: 'arguments', command, on: target, promptIndex, values })
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

  const startStep = (
    command: Command,
    step: NonNullable<Command['input']>,
    target: CommandTarget,
    input: string | undefined,
  ): CommandInvocation | null => {
    const name = input?.trim() ?? ''
    if (step === 'name' && name !== '') return invocation(command.id, target, name)
    if (isNameOmittable(command, target)) return invocation(command.id, target)
    openSteps()
    steps.stack.pushStep({ step, command, on: target })
    return null
  }

  /** Starts a command; the text typed for it, where given, answers a step that asks for a name. */
  const startCommand = (
    id: string,
    target: CommandTarget,
    input?: string,
    values?: Readonly<Record<string, string>>,
  ): CommandInvocation | null => {
    const command = rows.byId.value.get(id)
    if (!command || !isStartable(command, target, values)) return null
    if (!command.input) return invocation(command.id, target)
    if (command.input === 'arguments' && values) return startArguments(command, target, values)
    return startStep(command, command.input, target, input)
  }

  const getObjection = (id: string, target: CommandTarget): string => {
    const command = rows.byId.value.get(id)
    if (!command || command.isAvailable(target, deps.runs)) return ''
    return target.isReady ? deps.words.noNote : deps.words.noVault
  }

  return { startCommand, getObjection }
}

export function createPalette(session: PaletteSession, view: ReturnType<typeof usePaletteView>) {
  const { state, steps, on, choose } = session
  const { previewItem, setOpen, searchNames } = session.handlers
  const { startCommand, getObjection } = createPaletteCommands(session)

  const setTyped = async (text: string) => {
    state.typed.value = text
    // Stryker disable next-line MethodExpression: debounced.ask trims the query.
    if (steps.here.value?.step === 'note') await searchNames(text.trim())
  }

  const chooseItem = (item: string, action: string): CommandInvocation | null => {
    const step = steps.here.value
    if (!step) return startCommand(action, on.value)
    return choose(
      step,
      item,
      action,
      state.results.found.value,
      state.results.known.value,
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
