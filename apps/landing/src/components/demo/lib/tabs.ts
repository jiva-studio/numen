export const FILES_TAB = 'files'
export const PLEX_TAB = 'plex:main'
export const AGENT_TAB = 'agent:main'
export const PRIMARY_NOTE_TAB = 'note:1'

export const RECORDING_PATH = 'lectures/Lecture 4.mp3'
export const YOUTUBE_LINK_PATH = 'lectures/Feynman 1964.md'
export const DECK_PATH = 'decks/Thermodynamics.md'

export type TabKind = 'files' | 'plex' | 'agent' | 'note' | 'reader:pdf' | 'recording' | 'link' | 'deck'

const PREFIX_KINDS: readonly (readonly [string, TabKind])[] = [
  ['note:', 'note'],
  ['reader:pdf:', 'reader:pdf'],
  ['recording:', 'recording'],
  ['link:', 'link'],
  ['deck:', 'deck'],
]

/** The kind of a tab, read off its id: the three tools have an id each, the rest start with their kind. */
export function getKindOfTab(id: string): TabKind | undefined {
  if (id === FILES_TAB) return 'files'
  if (id === PLEX_TAB) return 'plex'
  if (id === AGENT_TAB) return 'agent'
  return PREFIX_KINDS.find(([prefix]) => id.startsWith(prefix))?.[1]
}

/** Whether the tab follows a vault path another tab moves to. */
export function isTabLinkable(id: string): boolean {
  const kind = getKindOfTab(id)
  return kind === 'files' || kind === 'plex' || kind === 'note'
}

/** Whether the tab shows a file and not a tool. */
export function isContentTab(id: string): boolean {
  const kind = getKindOfTab(id)
  return kind !== undefined && kind !== 'files' && kind !== 'plex' && kind !== 'agent'
}

export const getPdfTabId = (path: string): string => `reader:pdf:${path}`
export const getRecordingTabId = (path: string): string => `recording:${path}`
export const getLinkTabId = (path: string): string => `link:${path}`
export const getDeckTabId = (path: string): string => `deck:${path}`
export const getPdfPathOfTab = (id: string): string => id.slice('reader:pdf:'.length)
