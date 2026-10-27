/**
 * Command palette, keyboard shortcut dispatch, and command execution.
 */

import { computed, watch, type Component } from 'vue'
import type { CommandNote, MenuEntry } from '@numen/plugin-sdk'
import type { WelcomeAction } from '@numen/shell'
import { formatErrorMessage } from '@numen/wire'
import {
  isChord,
  findChordCommand,
  findShortcutKeys,
  registerCommand,
  runInvocation,
  useCommandPalette,
  useSearch,
  ANSWER_WORDS,
  type CommandDeps,
  type CommandTarget,
  type Notes,
  type NoteLookup,
  type PaletteLists,
  type RunSupport,
  type VaultRef,
} from '@/features/command-palette'
import { createCommandDeps } from './commandDeps'
import type { DeclaredCommands } from './declaredCommands'
import type { FileMenuCommands } from './fileMenuCommands'
import type { IndexCoverage } from '@/shared/notices/coverage'
import type { MessageLog, MessageWriter } from '@/shared/notices/messages'
import type { useWindowTabs } from '@/entities/tab'
import { WORDS } from '@/shared/words'
import { createPluginSettingCommands } from '../window/useSettings'
import type { NoteCreator } from '@/entities/note'
import type { VaultCore } from '../vault'
import type { PluginPaletteCommand } from '@/entities/plugin'

type Words = typeof WORDS

import type { SettingEdit } from '@/entities/settings'

export interface CommandsDepsOptions {
  core: VaultCore
  words: Words
  log: MessageLog
  windowTabs: ReturnType<typeof useWindowTabs>
  getTarget: () => CommandTarget
  knows: NoteLookup
  kept: PaletteLists
  runs: RunSupport
  coverage: () => IndexCoverage
  making: NoteCreator
  setVaultName: (vault: VaultRef) => void
  reload: () => void
  reached: Notes
  appearance: { chooseItem: (item: string) => Promise<void> | void }
  oneName: { choose: (item: string) => Promise<void> | void }
  rest: {
    writeSettings: (content: readonly SettingEdit[]) => Promise<void> | void
    getModelsAt?: (at: readonly string[]) => readonly import('@/entities/settings').Model[]
    plugins?: import('vue').Ref<readonly import('@/entities/settings').PluginInfo[]>
  }
  /** A note put in front of the person, where the plugin that shows notes puts it. */
  travel: (path: string) => Promise<void> | void
  /** Every tab standing on a path moved to another. */
  leavePath: (from: string, to: string) => Promise<void> | void
  getOpeningNote: () => string
  writeMessage: MessageWriter
  pluginHost?: import('@/entities/plugin').PluginHost | undefined
  /** The commands the enabled plugins declare, read as the window starts and each time the palette opens. */
  declaredCommands?: DeclaredCommands
  /** The menu the text versions of the file in front make, read each time the palette opens. */
  fileMenuCommands?: FileMenuCommands
}

export function useCommands(options: CommandsDepsOptions) {
  const { core, words, windowTabs, getTarget, knows, kept, runs, coverage, writeMessage } = options
  const { pluginHost } = options

  const getPluginCommands = () => pluginHost?.getPaletteCommands() ?? []
  /** How a plugin's command is drawn: its keystroke and its icon, where it declares them. */
  const getPluginLooks = (cmd: PluginPaletteCommand) => {
    const keys = cmd.shortcut ? findShortcutKeys(cmd.shortcut, navigator.userAgent) : undefined
    return {
      ...(keys ? { keys } : {}),
      ...(cmd.icon ? { icon: cmd.icon as Component } : {}),
    }
  }
  /** The command a plugin declares to put a note in front, and nothing where none does. */
  const findNoteDestination = () => getPluginCommands().find((cmd) => cmd.isNoteDestination)

  const palette = useSearch(core, words, {
    coverage,
    getTravel: () => findNoteDestination()?.title ?? '',
  })
  const commands = useCommandPalette(core, words, getTarget, knows, kept, runs, undefined, () => [
    ...createPluginSettingCommands(options.rest.plugins?.value ?? []),
    ...(options.declaredCommands?.paletteCommands.value ?? []),
    ...(options.fileMenuCommands?.paletteCommands.value ?? []),
    ...getPluginCommands().map((cmd) => ({
      id: cmd.id,
      text: cmd.title,
      ...getPluginLooks(cmd),
      group: cmd.scope === 'note' ? ('note' as const) : ('window' as const),
      isOffered: (target: CommandTarget) =>
        cmd.scope !== 'note' || (target.isReady && target.path !== ''),
    })),
  ])

  /**
   * The commands the plugins declare over the window with a keystroke of their
   * own, as the screen of a window with no tab offers them.
   */
  const pluginWays = computed<readonly WelcomeAction[]>(() =>
    getPluginCommands()
      .filter((cmd) => cmd.scope !== 'note' && cmd.shortcut)
      .map((cmd) => ({ id: cmd.id, text: cmd.title, ...getPluginLooks(cmd) })),
  )

  /** The commands the plugins declare over one note, in the words the palette offers them by. */
  const getNoteCommands = (): readonly MenuEntry[] =>
    getPluginCommands()
      .filter((cmd) => cmd.scope === 'note')
      .map((cmd) => ({ id: cmd.id, text: cmd.title, group: 'note' }))

  const getCommandNote = async (path: string, title: string): Promise<CommandNote> => ({
    path,
    title,
    type: path === '' ? 'note' : ((await core.fileKinds([path])).get(path)?.type ?? 'note'),
  })

  /** The note put in front by the command a plugin declares for it; false where none declares one. */
  const showNote = async (path: string): Promise<boolean> => {
    const destination = findNoteDestination()
    if (!destination || !pluginHost) return false
    return pluginHost.executeCommand(destination.id, await getCommandNote(path, ''))
  }

  if (pluginHost) {
    let unregisterAll: readonly (() => void)[] = []
    watch(
      () => pluginHost.getPaletteCommands(),
      (list) => {
        for (const unregister of unregisterAll) unregister()
        unregisterAll = list.map((cmd) =>
          registerCommand(cmd.id, async (invocation) => {
            await pluginHost.executeCommand(
              cmd.id,
              await getCommandNote(invocation.path, invocation.title),
            )
          }),
        )
      },
      { immediate: true },
    )
  }

  const { declaredCommands } = options
  if (declaredCommands) {
    const loadDeclared = () => {
      declaredCommands
        .load()
        .catch((error: unknown) => writeMessage(formatErrorMessage(error), 'error'))
    }
    loadDeclared()
    watch(commands.open, (isOpen) => {
      if (isOpen) loadDeclared()
    })
  }

  const { fileMenuCommands } = options
  if (fileMenuCommands) {
    watch(commands.open, (isOpen) => {
      if (!isOpen) return
      fileMenuCommands
        .load(getTarget().file)
        .catch((error: unknown) => writeMessage(formatErrorMessage(error), 'error'))
    })
  }

  const commandDeps: CommandDeps = createCommandDeps({
    ...options,
    search: () => {
      commands.setOpen(false)
      palette.setOpen(true)
    },
  })

  const runCommand = (id: string, target: CommandTarget) => {
    if (pluginHost?.hasCommand(id)) {
      commands.setOpen(false)
      void getCommandNote(target.path, target.title).then((note) =>
        pluginHost.executeCommand(id, note),
      )
      return
    }
    const invocation = commands.startCommand(id, target)
    if (invocation) return void runInvocation(invocation, commandDeps, ANSWER_WORDS)
    if (commands.open.value) return void palette.setOpen(false)
    writeMessage(commands.getObjection(id, target), 'error')
  }

  const onKeyDown = (event: KeyboardEvent) => {
    if (event.defaultPrevented) return
    if (windowTabs.onKeyPress(event)) return event.preventDefault()
    const getNote = () => getCommandNote(getTarget().path, getTarget().title)
    if (pluginHost?.handleShortcut(event, getNote)) return event.preventDefault()
    if (!isChord(event)) return
    const command = findChordCommand(event.key.toLowerCase(), event.shiftKey)
    if (!command) return
    event.preventDefault()
    runCommand(command, getTarget())
  }

  return {
    palette,
    commands,
    commandDeps,
    runCommand,
    onKeyDown,
    pluginWays,
    getNoteCommands,
    showNote,
  }
}
