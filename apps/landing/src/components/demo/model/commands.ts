import { nav, reportFailure } from '@numen/ui'
import { DEMO_COMMAND } from '../lib/commands'
import { ROOT } from '../lib/entries'
import { ENTROPY_PATH } from '../lib/notes'
import { NAMING_COMMAND, type NamingRole } from '../lib/palette'
import { AGENT_TAB, FILES_TAB, PLEX_TAB } from '../lib/tabs'
import type { Agent } from './agent'
import type { Files } from './files'
import type { Layout } from './layout'
import type { Palette } from './palette'
import type { Plex } from './plex'
import type { Vault } from './vault'

export interface CommandDeps {
  readonly vault: Vault
  readonly layout: Layout
  readonly files: Files
  readonly plex: Plex
  readonly palette: Palette
  readonly agent: Agent
}

/** Where a command was started: on a row of the file tree, or on the plex and the palette. */
type Source = 'files' | 'plex'

const DEMO_COMMAND_IDS: ReadonlySet<string> = new Set(Object.values(DEMO_COMMAND))

export const isDemoCommand = (id: string): boolean => DEMO_COMMAND_IDS.has(id)

function getNamingRole(command: string): NamingRole | null {
  const found = Object.entries(NAMING_COMMAND).find(([, id]) => id === command)
  return (found?.[0] as NamingRole | undefined) ?? null
}

/** Takes a note out of the vault, closes its tabs, and moves the plex off it. */
function removeNote({ vault, layout, plex }: CommandDeps, path: string) {
  if (!vault.edit.removeNote(path)) return
  for (const [tabId, tabPath] of Object.entries(layout.state.notePaths)) {
    if (tabPath === path) layout.actions.close(tabId)
  }
  if (plex.here.value === path) plex.go(Object.keys(vault.notes)[0] ?? ENTROPY_PATH)
}

/** The commands of the menus and the palette, run over the demo vault and the windows of the demo. */
export function createCommands(deps: CommandDeps) {
  const { vault, layout, files, plex, palette, agent } = deps

  function createRelated(path: string, role: Exclude<NamingRole, 'title'>, source: Source) {
    if (source === 'files') vault.edit.createRelated('', path, role)
    else palette.actions.openNaming(role, path)
  }

  /** Makes an entry in a folder, opens the folder and asks for the name of the entry. */
  function startNaming(folder: string, create: (folder: string) => string) {
    const path = create(folder)
    files.tree.openFolder(folder)
    files.state.renaming.value = path
  }

  function renameNote(path: string, source: Source) {
    if (source === 'plex') palette.actions.openNaming('title', path)
    else if (vault.read.getNote(path)) files.state.renaming.value = path
  }

  function askAgent(path: string) {
    layout.actions.openToolTab(AGENT_TAB, 'aside')
    agent.draft.value = `${path} — `
    agent.focusCount.value += 1
  }

  const tools: Readonly<Record<string, () => void>> = {
    [DEMO_COMMAND.openPlex]: () => layout.actions.openToolTab(PLEX_TAB, 'map'),
    [DEMO_COMMAND.openFiles]: () => layout.actions.openToolTab(FILES_TAB, 'main'),
    [DEMO_COMMAND.openAgent]: () => layout.actions.openToolTab(AGENT_TAB, 'aside'),
  }

  function reveal(path: string) {
    layout.actions.openToolTab(FILES_TAB, 'main')
    files.tree.reveal(path)
  }

  function showInPlex(path: string) {
    if (!vault.read.getNote(path)) return
    plex.go(path)
    layout.actions.openToolTab(PLEX_TAB, 'map')
  }

  const overPath: Readonly<Record<string, (path: string, source: Source) => void>> = {
    [DEMO_COMMAND.renameTreeEntry]: (path) => {
      files.state.renaming.value = path === ROOT ? null : path
    },
    [DEMO_COMMAND.openNote]: (path) => layout.actions.openVaultFile(path, 'current'),
    [DEMO_COMMAND.openNoteBeside]: (path) => layout.actions.openVaultFile(path, 'split'),
    [DEMO_COMMAND.revealNote]: reveal,
    [DEMO_COMMAND.showNoteInPlex]: showInPlex,
    [DEMO_COMMAND.createNote]: () => startNaming('physics', vault.edit.createUntitled),
    [DEMO_COMMAND.renameNote]: renameNote,
    [DEMO_COMMAND.copyNotePath]: (path) => {
      nav.clipboard.writeText(path).catch(reportFailure)
    },
    [DEMO_COMMAND.removeFile]: (path) => removeNote(deps, path),
    [DEMO_COMMAND.askAgent]: askAgent,
    ...tools,
  }

  /** Runs a command over the note or file at a path. */
  function run(command: string, path: string, source: Source = 'plex') {
    const role = getNamingRole(command)
    if (role !== null && role !== 'title') createRelated(path, role, source)
    else overPath[command]?.(path, source)
  }

  const createInside: Readonly<Record<string, (folder: string) => string>> = {
    [DEMO_COMMAND.createNote]: vault.edit.createUntitled,
    [DEMO_COMMAND.createFolder]: vault.edit.createFolder,
  }

  /** Runs a command chosen in the menu of the file tree over the row the menu was opened on. */
  function runInTree(command: string, target: { path: string; others: readonly string[] }) {
    const isFolder = target.path === ROOT || vault.read.getEntryAt(target.path)?.isFolder === true
    const create = isFolder ? createInside[command] : undefined
    if (create) {
      startNaming(target.path, create)
      return
    }
    const paths = command === DEMO_COMMAND.removeFile ? [target.path, ...target.others] : [target.path]
    for (const path of paths) run(command, path, 'files')
  }

  return { run, runInTree }
}
