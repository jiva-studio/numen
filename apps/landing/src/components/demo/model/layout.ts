import { computed, reactive, ref } from 'vue'
import {
  branch,
  closeTab,
  getPanes,
  openTab,
  openTabAdjacent,
  pane,
  paneById,
  type Tab,
  type Workspace,
} from '@numen/workbench'
import { getPdfDoc, hasPdfDoc } from '../lib/pdfPage'
import {
  AGENT_TAB,
  DECK_PATH,
  FILES_TAB,
  PLEX_TAB,
  PRIMARY_NOTE_TAB,
  RECORDING_PATH,
  YOUTUBE_LINK_PATH,
  getDeckTabId,
  getLinkTabId,
  getPdfTabId,
  getRecordingTabId,
  getKindOfTab,
  type TabKind,
  getPdfPathOfTab,
  isContentTab,
} from '../lib/tabs'
import { ENTROPY_PATH } from '../lib/notes'
import type { TabLinks } from './tabLinks'
import type { Vault } from './vault'

export type Placement = 'current' | 'split'

const BOLTZMANN_TAB = getPdfTabId('reading/Boltzmann 1877.pdf')

function createInitialLayout(): Workspace {
  const content = [PRIMARY_NOTE_TAB, BOLTZMANN_TAB, getRecordingTabId(RECORDING_PATH), getLinkTabId(YOUTUBE_LINK_PATH), getDeckTabId(DECK_PATH)]
  return {
    root: branch(
      'root',
      [
        pane('main', [FILES_TAB]),
        branch('middle', [pane('map', [PLEX_TAB]), pane('note', content, PRIMARY_NOTE_TAB)], [0.52, 0.48]),
        pane('aside', [AGENT_TAB]),
      ],
      [0.22, 0.5, 0.28],
    ),
    axis: 'horizontal',
    focus: 'map',
  }
}

export interface LayoutDeps {
  readonly vault: Vault
  readonly links: TabLinks
  readonly getPlexTitle: () => string | undefined
  readonly syncPath: (sourceTab: string, path: string) => boolean
}

const FIXED_TITLES: Readonly<Partial<Record<TabKind, string>>> = {
  files: 'Files',
  agent: 'Agent',
  recording: 'Lecture 4',
  link: 'Feynman 1964',
  deck: 'Thermodynamics',
}

function getTitle(deps: LayoutDeps, notePaths: Record<string, string>, id: string): string {
  const kind = getKindOfTab(id)
  if (kind === 'plex') return deps.getPlexTitle() || 'Plex'
  const fixed = kind === undefined ? undefined : FIXED_TITLES[kind]
  if (fixed !== undefined) return fixed
  if (kind === 'reader:pdf') return getPdfDoc(getPdfPathOfTab(id)).title
  const path = notePaths[id] ?? ENTROPY_PATH
  return deps.vault.read.getNote(path)?.title ?? path
}

function isContentPane(one: { id: string; tabs: readonly string[] }): boolean {
  return one.tabs.some(isContentTab) || (!['main', 'map', 'aside'].includes(one.id) && one.tabs.length === 0)
}

function findContentPaneId(workspace: Workspace): string | null {
  const focused = paneById(workspace.root, workspace.focus)
  if (focused && isContentPane(focused)) return focused.id
  return getPanes(workspace.root).find(isContentPane)?.id ?? null
}

function getShownWorkspace(
  workspace: Workspace,
  tabId: string,
  placement: Placement,
  createId: () => string,
): Workspace {
  const contentPane = findContentPaneId(workspace)
  if (placement === 'split' && contentPane) {
    return openTabAdjacent(workspace, tabId, { side: 'right', onto: contentPane, createId })
  }
  if (contentPane) return openTab(workspace, tabId, contentPane)
  const anchor = paneById(workspace.root, 'map')?.id ?? getPanes(workspace.root)[0]?.id
  return anchor
    ? openTabAdjacent(workspace, tabId, { side: 'bottom', onto: anchor, createId })
    : openTab(workspace, tabId)
}

/** The open tabs and where the panes of the window hold them. */
export function createLayout(deps: LayoutDeps) {
  const held = ref<Workspace>(createInitialLayout())
  const openTabIds = ref<string[]>([
    FILES_TAB, PLEX_TAB, AGENT_TAB, PRIMARY_NOTE_TAB, BOLTZMANN_TAB,
    getRecordingTabId(RECORDING_PATH), getLinkTabId(YOUTUBE_LINK_PATH), getDeckTabId(DECK_PATH),
  ])
  const notePaths = reactive<Record<string, string>>({ [PRIMARY_NOTE_TAB]: ENTROPY_PATH })
  let paneCount = 1
  let noteTabCount = 1

  deps.vault.onRename((from, to) => {
    for (const [id, path] of Object.entries(notePaths)) {
      if (path === from) notePaths[id] = to
    }
  })

  const createPaneId = (): string => `pane-${(paneCount += 1)}`
  const tabs = computed<readonly Tab[]>(() =>
    openTabIds.value.map((id) => ({ id, title: getTitle(deps, notePaths, id) })),
  )

  function showTab(tabId: string, placement: Placement): void {
    if (!openTabIds.value.includes(tabId)) openTabIds.value = [...openTabIds.value, tabId]
    held.value = getShownWorkspace(held.value, tabId, placement, createPaneId)
  }

  function navigateNoteTab(tabId: string, path: string): void {
    notePaths[tabId] = path
  }

  function openNote(path: string, placement: Placement): string {
    const existing = openTabIds.value.find((id) => getKindOfTab(id) === 'note' && notePaths[id] === path)
    const linked = openTabIds.value.find((id) => getKindOfTab(id) === 'note' && deps.links.getGroupOf(id) !== null)
    let tabId = existing
    if (placement === 'current' && !tabId && linked) {
      navigateNoteTab(linked, path)
      tabId = linked
    }
    if (placement !== 'current' || !tabId) {
      tabId = `note:${(noteTabCount += 1)}`
      notePaths[tabId] = path
    }
    showTab(tabId, placement)
    deps.syncPath(tabId, path)
    return tabId
  }

  /** Opens the tab a vault file is read in, and returns its id; null for a path the vault does not have. */
  function openVaultFile(path: string, placement: Placement = 'current'): string | null {
    const fixed: Readonly<Record<string, string>> = {
      [RECORDING_PATH]: getRecordingTabId(RECORDING_PATH),
      [YOUTUBE_LINK_PATH]: getLinkTabId(YOUTUBE_LINK_PATH),
      [DECK_PATH]: getDeckTabId(DECK_PATH),
    }
    const tabId = fixed[path] ?? (hasPdfDoc(path) ? getPdfTabId(path) : undefined)
    if (tabId) {
      showTab(tabId, placement)
      return tabId
    }
    return deps.vault.read.getNote(path) ? openNote(path, placement) : null
  }

  function openToolTab(tabId: string, preferredPane: string): void {
    if (!openTabIds.value.includes(tabId)) openTabIds.value = [...openTabIds.value, tabId]
    const target = paneById(held.value.root, preferredPane) ? preferredPane : undefined
    held.value = openTab(held.value, tabId, target)
  }

  function close(tabId: string): void {
    deps.links.unlink(tabId)
    held.value = closeTab(held.value, tabId)
    openTabIds.value = openTabIds.value.filter((id) => id !== tabId)
    delete notePaths[tabId]
  }

  return {
    state: { held, tabs, notePaths },
    actions: {
      createPaneId,
      openVaultFile,
      openToolTab,
      navigateNoteTab,
      close,
      getNoteOfTab: (tabId: string) => {
        const path = notePaths[tabId]
        return path === undefined ? undefined : deps.vault.read.getNote(path)
      },
    },
  }
}

export type Layout = ReturnType<typeof createLayout>
