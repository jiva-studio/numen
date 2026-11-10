/**
 * The palette the user types into, put together from its three parts: the palette itself, what it
 * draws, and the steps it moves between.
 *
 * What a step renders is recomputed on every keystroke, so a vault changing under an open step is
 * rendered as it is.
 */
import type { Command, CommandTarget, RunSupport } from '@/entities/command'
import type { Wait } from '../search/debounced'
import type { CommandsDeps, NoteLookup, PaletteLists } from '../../types'
import type { Words } from '../../words'
import { createPaletteSession, type CommandPaletteDeps } from './session'
import { createPalette } from './createPalette'
import { createPaletteNavigation } from './createPaletteNavigation'
import { usePaletteView } from './usePaletteView'

type PaletteArgs =
  | [CommandPaletteDeps]
  | [
      CommandsDeps,
      Words,
      () => CommandTarget,
      NoteLookup,
      PaletteLists,
      RunSupport,
      (Wait | undefined)?,
      (() => readonly Command[])?,
    ]

function parsePaletteDeps(args: PaletteArgs): CommandPaletteDeps {
  if (args.length === 1) return args[0]
  const [core, words, targetPoint, knows, lists, runs, wait, getCommands] = args
  return {
    core,
    words,
    at: targetPoint,
    knows,
    lists,
    runs,
    // Stryker disable next-line ConditionalExpression: Setting optional property to undefined is equivalent when read.
    ...(wait !== undefined ? { wait } : {}),
    // Stryker disable next-line ConditionalExpression: Setting optional property to undefined is equivalent when read.
    ...(getCommands !== undefined ? { getCommands } : {}),
  }
}

export function createCommandPalette(...args: PaletteArgs) {
  const session = createPaletteSession(parsePaletteDeps(args))
  const navigation = createPaletteNavigation(session)
  const view = usePaletteView(session)
  const palette = createPalette(session, view)

  return {
    open: palette.open,
    typed: palette.typed,
    groups: view.groups,
    crumb: navigation.crumb,
    step: navigation.step,
    opensOn: navigation.opensOn,
    placeholder: navigation.placeholder,
    setTyped: palette.setTyped,
    previewItem: palette.previewItem,
    setOpen: palette.setOpen,
    startCommand: palette.startCommand,
    getObjection: palette.getObjection,
    applyRenames: navigation.applyRenames,
    chooseItem: palette.chooseItem,
    leaveStep: navigation.leaveStep,
    goBack: navigation.goBack,
    findTabKind: view.findTabKind,
    findCommandIcon: view.findCommandIcon,
  }
}

export type Commands = ReturnType<typeof createCommandPalette>
