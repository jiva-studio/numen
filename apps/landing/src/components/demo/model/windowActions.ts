import { DEMO_COMMAND } from '../lib/commands'
import { ENTROPY_PATH } from '../lib/notes'
import { getKindOfTab } from '../lib/tabs'
import type { Turn } from './agent'
import { isDemoCommand, type createCommands } from './commands'
import type { Files } from './files'
import type { Layout } from './layout'
import type { Palette } from './palette'
import type { Plex } from './plex'
import type { TabLinks } from './tabLinks'
import type { Vault } from './vault'

interface WindowActionDeps {
  readonly vault: Vault
  readonly plex: Plex
  readonly layout: Layout
  readonly files: Files
  readonly palette: Palette
  readonly commands: ReturnType<typeof createCommands>
  readonly links: TabLinks
  /** Moves the tabs linked to a tab to a path; false where none is linked. */
  readonly sync: (tab: string, path: string) => boolean
}

/** What a person does in the demo window that touches several of its parts: choosing in the palette, following a link, editing. */
export function createWindowActions(deps: WindowActionDeps) {
  const { vault, plex, layout, files, palette, commands, links, sync } = deps

  function activateTab(tabId: string) {
    const path = layout.state.notePaths[tabId]
    if (getKindOfTab(tabId) === 'note' && path && vault.read.getNote(path)) sync(tabId, path)
  }

  function chooseName(role: 'child' | 'parent' | 'jump' | 'title', path: string) {
    const name = palette.state.typed.value.trim()
    palette.actions.dismiss()
    if (!name) return
    if (role === 'title') files.edits.rename(path, name)
    else plex.go(vault.edit.createRelated(name, path, role))
  }

  function choosePalette(itemId: string, actionId?: string) {
    const naming = palette.state.naming.value
    if (palette.state.panel.value === 'naming' && naming) {
      chooseName(naming.role, naming.path)
      return
    }
    palette.state.panel.value = null
    const active = plex.here.value || ENTROPY_PATH
    if (isDemoCommand(itemId)) commands.run(itemId, active, 'plex')
    else if (actionId === 'travel' && vault.read.getNote(itemId)) commands.run(DEMO_COMMAND.showNoteInPlex, itemId)
    else layout.actions.openVaultFile(itemId, 'current')
  }

  function followLink(address: string, fromTab: string) {
    const cleaned = address.replace(/^\[\[|\]\]$/g, '').split('#')[0]?.trim().toLowerCase()
    const match = Object.values(vault.notes).find(
      (one) => one.path.toLowerCase() === cleaned || one.title.toLowerCase() === cleaned,
    )
    if (!match) return
    if (getKindOfTab(fromTab) === 'note') {
      layout.actions.navigateNoteTab(fromTab, match.path)
      sync(fromTab, match.path)
    } else {
      layout.actions.openVaultFile(match.path, 'current')
    }
  }

  const writeBody = (tabId: string, body: string) => {
    const note = layout.actions.getNoteOfTab(tabId)
    if (note) note.body = body
  }

  const openTurnSource = (turn: Turn) =>
    layout.actions.openVaultFile(turn.subject === "Landauer's principle" ? "computation/Landauer's principle.md" : ENTROPY_PATH)

  const getFocusTitle = (): string => vault.read.getNoteOrEntropy(plex.here.value || ENTROPY_PATH).title

  return {
    writeBody,
    openTurnSource,
    activateTab,
    choosePalette,
    followLink,
    getLinkGroup: links.getGroupOf,
    getFocusTitle,
  }
}
