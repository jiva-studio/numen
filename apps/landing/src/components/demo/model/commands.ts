import { reportFailure } from '@numen/ui'
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

/** The commands of the menus and the palette, run over the demo vault and the windows of the demo. */
export function createCommands(deps: CommandDeps) {
  const { vault, layout, files, plex, palette, agent } = deps

  function removeNote(path: string) {
    if (!vault.removeNote(path)) return
    for (const [tabId, tabPath] of Object.entries(layout.notePaths)) {
      if (tabPath === path) layout.close(tabId)
    }
    if (plex.here.value === path) plex.go(Object.keys(vault.notes)[0] ?? ENTROPY_PATH)
  }

  function createRelated(path: string, role: Exclude<NamingRole, 'title'>, source: Source) {
    if (source === 'files') vault.createRelated('', path, role)
    else palette.openNaming(role, path)
  }

  function createNote() {
    const path = vault.createUntitled('physics')
    files.openFolder('physics')
    files.renaming.value = path
  }

  function renameNote(path: string, source: Source) {
    if (source === 'plex') palette.openNaming('title', path)
    else if (vault.getNote(path)) files.renaming.value = path
  }

  function askAgent(path: string) {
    layout.openToolTab(AGENT_TAB, 'aside')
    agent.draft.value = `${path} — `
    agent.focusCount.value += 1
  }

  const tools: Readonly<Record<string, () => void>> = {
    [DEMO_COMMAND.openPlex]: () => layout.openToolTab(PLEX_TAB, 'map'),
    [DEMO_COMMAND.openFiles]: () => layout.openToolTab(FILES_TAB, 'main'),
    [DEMO_COMMAND.openAgent]: () => layout.openToolTab(AGENT_TAB, 'aside'),
  }

  function reveal(path: string) {
    layout.openToolTab(FILES_TAB, 'main')
    files.reveal(path)
  }

  function showInPlex(path: string) {
    if (!vault.getNote(path)) return
    plex.go(path)
    layout.openToolTab(PLEX_TAB, 'map')
  }

  /** Runs a command over the note or file at a path. */
  function run(command: string, path: string, source: Source = 'plex') {
    const role = getNamingRole(command)
    if (command === DEMO_COMMAND.renameTreeEntry) files.renaming.value = path === ROOT ? null : path
    else if (command === DEMO_COMMAND.openNote) layout.openVaultFile(path, 'current')
    else if (command === DEMO_COMMAND.openNoteBeside) layout.openVaultFile(path, 'split')
    else if (command === DEMO_COMMAND.revealNote) reveal(path)
    else if (command === DEMO_COMMAND.showNoteInPlex) showInPlex(path)
    else if (role !== null && role !== 'title') createRelated(path, role, source)
    else if (command === DEMO_COMMAND.createNote) createNote()
    else if (command === DEMO_COMMAND.renameNote) renameNote(path, source)
    else if (command === DEMO_COMMAND.copyNotePath) navigator.clipboard.writeText(path).catch(reportFailure)
    else if (command === DEMO_COMMAND.removeFile) removeNote(path)
    else if (command === DEMO_COMMAND.askAgent) askAgent(path)
    else tools[command]?.()
  }

  /** Runs a command chosen in the menu of the file tree over the row the menu was opened on. */
  function runInTree(command: string, target: { path: string; others: readonly string[] }) {
    const isFolder = target.path === ROOT || vault.getEntryAt(target.path)?.isFolder === true
    if (command === DEMO_COMMAND.createNote && isFolder) {
      const path = vault.createUntitled(target.path)
      files.openFolder(target.path)
      files.renaming.value = path
      return
    }
    if (command === DEMO_COMMAND.createFolder && isFolder) {
      const path = vault.createFolder(target.path)
      files.openFolder(target.path)
      files.renaming.value = path
      return
    }
    const paths = command === DEMO_COMMAND.removeFile ? [target.path, ...target.others] : [target.path]
    for (const path of paths) run(command, path, 'files')
  }

  return { run, runInTree }
}
