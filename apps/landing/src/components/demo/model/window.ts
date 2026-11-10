import { watch } from 'vue'
import { DEMO_COMMAND } from '../lib/commands'
import { getKindOfTab, FILES_TAB, PLEX_TAB, PRIMARY_NOTE_TAB } from '../lib/tabs'
import type { Agent } from './agent'
import { createCommands } from './commands'
import { createFiles } from './files'
import { createLayout } from './layout'
import { createPalette } from './palette'
import { createPlex } from './plex'
import { createTabLinks } from './tabLinks'
import { createTabMenu } from './tabMenu'
import { createVault } from './vault'
import { createWindowActions } from './windowActions'

const isNotePath = (path: string): boolean => /\.md$/i.test(path)

/** The demo window: a vault in memory, and the tabs, trees, menus and palette that work on it. */
export function createDemoWindow(agent: Agent) {
  const vault = createVault()
  const links = createTabLinks()
  const plex = createPlex(vault)
  const holder = { sync: (_tab: string, _path: string): boolean => false }

  const layout = createLayout({
    vault,
    links,
    getPlexTitle: () => plex.neighborhood.value.focus.title,
    syncPath: (tab, path) => holder.sync(tab, path),
  })
  const palette = createPalette(vault)
  const commandsHolder: { run?: ReturnType<typeof createCommands> } = {}
  const files = createFiles({
    vault,
    openFile: (path) => layout.actions.openVaultFile(path, 'current'),
    onSelectNote: (path) => holder.sync(FILES_TAB, path),
    remove: (path, others) => commandsHolder.run?.runInTree(DEMO_COMMAND.removeFile, { path, others }),
  })
  const commands = createCommands({ vault, layout, files, plex, palette, agent })
  commandsHolder.run = commands
  const tabMenu = createTabMenu(layout.state.tabs, links)

  let isSyncing = false

  /** Moves the tabs linked to a tab to a path the tab moved to; false where none is linked. */
  holder.sync = (source, path) => {
    const targets = links.getLinkedTargets(source)
    if (!source || !path || isSyncing || targets.length === 0) return false
    isSyncing = true
    try {
      for (const target of targets) moveTarget(target, path)
    } finally {
      isSyncing = false
    }
    return true
  }

  /** Moves one linked tab to a path. */
  function moveTarget(target: string, path: string) {
    const isNote = isNotePath(path) && Boolean(vault.read.getNote(path))
    if (target === PLEX_TAB && isNote) plex.go(path)
    else if (target === FILES_TAB) files.tree.reveal(path)
    else if (getKindOfTab(target) === 'note' && isNote) layout.actions.navigateNoteTab(target, path)
  }

  links.link(FILES_TAB, PLEX_TAB)
  links.link(PLEX_TAB, PRIMARY_NOTE_TAB)
  watch(plex.here, (path) => holder.sync(PLEX_TAB, path))

  const actions = createWindowActions({ vault, plex, layout, files, palette, commands, links, sync: holder.sync })

  return { vault, plex, layout, files, commands, palette, tabMenu, actions }
}
