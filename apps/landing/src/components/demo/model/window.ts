import { onBeforeUnmount, reactive, watch } from 'vue'
import { DEMO_COMMAND } from '../lib/commands'
import { getKindOfTab, FILES_TAB, PLEX_TAB, PRIMARY_NOTE_TAB } from '../lib/tabs'
import { BOLTZMANN_PATH, PDF_DOCS, SHANNON_PATH } from '../lib/documents'
import { ENTROPY_PATH } from '../lib/notes'
import { getPdfDoc } from '../lib/pdfPage'
import { createAgent, type Turn } from './agent'
import { createCommands, isDemoCommand } from './commands'
import { createDeck } from './deck'
import { createFiles } from './files'
import { createLayout } from './layout'
import { createPalette } from './palette'
import { createPlayer } from './player'
import { createPlex } from './plex'
import { createTabLinks } from './tabLinks'
import { createTabMenu } from './tabMenu'
import { createVault } from './vault'

const isNotePath = (path: string): boolean => /\.md$/i.test(path)

/** The demo window: a vault in memory, and the tabs, trees, menus and palette that work on it. */
export function createDemoWindow() {
  const vault = createVault()
  const links = createTabLinks()
  const plex = createPlex(vault)
  const player = createPlayer()
  const deck = createDeck()
  const holder = { sync: (_tab: string, _path: string): boolean => false }
  const getFocusNote = () => vault.getNoteOrEntropy(plex.here.value || ENTROPY_PATH)

  const layout = createLayout({
    vault,
    links,
    getPlexTitle: () => plex.neighbourhood.value.focus.title,
    syncPath: (tab, path) => holder.sync(tab, path),
  })
  const agent = createAgent(() => getFocusNote().title)
  const palette = createPalette(vault)
  const commandsHolder: { run?: ReturnType<typeof createCommands> } = {}
  const files = createFiles({
    vault,
    openFile: (path) => layout.openVaultFile(path, 'current'),
    onSelectNote: (path) => holder.sync(FILES_TAB, path),
    remove: (path, others) => commandsHolder.run?.runInTree(DEMO_COMMAND.removeFile, { path, others }),
  })
  const commands = createCommands({ vault, layout, files, plex, palette, agent })
  commandsHolder.run = commands
  const tabMenu = createTabMenu(layout.tabs, links)

  const pdfPages = reactive<Record<string, number>>({
    [BOLTZMANN_PATH]: PDF_DOCS[BOLTZMANN_PATH].firstPage,
    [SHANNON_PATH]: PDF_DOCS[SHANNON_PATH].firstPage,
  })

  let isSyncing = false

  /** Moves the tabs linked to a tab to a path the tab moved to; false where none is linked. */
  holder.sync = (source, path) => {
    const targets = links.getLinkedTargets(source)
    if (!source || !path || isSyncing || targets.length === 0) return false
    isSyncing = true
    try {
      for (const target of targets) {
        const isNote = isNotePath(path) && Boolean(vault.getNote(path))
        if (target === PLEX_TAB && isNote) plex.go(path)
        else if (target === FILES_TAB) files.reveal(path)
        else if (getKindOfTab(target) === 'note' && isNote) layout.navigateNoteTab(target, path)
      }
    } finally {
      isSyncing = false
    }
    return true
  }

  links.link(FILES_TAB, PLEX_TAB)
  links.link(PLEX_TAB, PRIMARY_NOTE_TAB)
  watch(plex.here, (path) => holder.sync(PLEX_TAB, path))
  onBeforeUnmount(() => player.pause())

  function activateTab(tabId: string) {
    const path = layout.notePaths[tabId]
    if (getKindOfTab(tabId) === 'note' && path && vault.getNote(path)) holder.sync(tabId, path)
  }

  function chooseName(role: 'child' | 'parent' | 'jump' | 'title', path: string) {
    const name = palette.typed.value.trim()
    palette.dismiss()
    if (!name) return
    if (role === 'title') files.rename(path, name)
    else plex.go(vault.createRelated(name, path, role))
  }

  function choosePalette(itemId: string, actionId?: string) {
    const naming = palette.naming.value
    if (palette.panel.value === 'naming' && naming) {
      chooseName(naming.role, naming.path)
      return
    }
    palette.panel.value = null
    const active = plex.here.value || ENTROPY_PATH
    if (isDemoCommand(itemId)) commands.run(itemId, active, 'plex')
    else if (actionId === 'travel' && vault.getNote(itemId)) commands.run(DEMO_COMMAND.showNoteInPlex, itemId)
    else layout.openVaultFile(itemId, 'current')
  }

  function followLink(address: string, fromTab: string) {
    const cleaned = address.replace(/^\[\[|\]\]$/g, '').split('#')[0]?.trim().toLowerCase()
    const match = Object.values(vault.notes).find(
      (one) => one.path.toLowerCase() === cleaned || one.title.toLowerCase() === cleaned,
    )
    if (!match) return
    if (getKindOfTab(fromTab) === 'note') {
      layout.navigateNoteTab(fromTab, match.path)
      holder.sync(fromTab, match.path)
    } else {
      layout.openVaultFile(match.path, 'current')
    }
  }

  const writeBody = (tabId: string, body: string) => {
    const note = layout.getNoteOfTab(tabId)
    if (note) note.body = body
  }

  const openTurnSource = (turn: Turn) =>
    layout.openVaultFile(turn.subject === "Landauer's principle" ? "computation/Landauer's principle.md" : ENTROPY_PATH)

  const goToPdfPage = (path: string, page: number) => {
    pdfPages[path] = Math.min(Math.max(page, 0), getPdfDoc(path).leaves - 1)
  }

  return {
    pdfPages,
    writeBody,
    openTurnSource,
    goToPdfPage,
    vault,
    links,
    plex,
    player,
    deck,
    layout,
    agent,
    palette,
    files,
    commands,
    tabMenu,
    activateTab,
    choosePalette,
    followLink,
  }
}

export type DemoWindow = ReturnType<typeof createDemoWindow>
