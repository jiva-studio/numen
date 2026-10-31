/** Names and plugin-search filters for the tab kinds whose plugin is unavailable. */

export const READER_PREFIX = 'reader:'

function stripReaderKind(kind: string): string {
  return kind.startsWith(READER_PREFIX)
    ? kind.slice(READER_PREFIX.length).replace(/^reader-/, '')
    : kind
}

const UNAVAILABLE_TAB_TITLES: Readonly<Record<string, string>> = {
  files: 'Files',
  plex: 'Plex',
  agent: 'Agent',
  cards: 'Flashcards',
  preset: 'Preset',
  recording: 'Recording',
  note: 'Note',
  document: 'Document',
  book: 'Book',
  markdown: 'Note',
  pdf: 'Document',
  epub: 'Book',
  'media-player': 'Recording',
}

const UNAVAILABLE_FEATURE_NAMES: Readonly<Record<string, string>> = {
  markdown: 'Markdown notes',
  note: 'Markdown notes',
  pdf: 'PDF documents',
  document: 'PDF documents',
  epub: 'EPUB books',
  book: 'EPUB books',
  'media-player': 'audio playback and recording',
  recording: 'audio playback and recording',
  files: 'file tree',
  plex: 'plex graph',
  agent: 'agent chat',
  cards: 'flashcards',
  preset: 'flashcard presets',
}

const UNAVAILABLE_READER_TARGETS: Readonly<Record<string, string>> = {
  markdown: 'markdown',
  note: 'markdown',
  'media-player': 'media-player',
}

const UNAVAILABLE_FEATURE_FILTERS: Readonly<Record<string, string>> = {
  recording: 'feature:recording',
  files: 'feature:files',
  plex: 'feature:plex',
  agent: 'feature:agent',
  cards: 'feature:flashcards',
  preset: 'feature:flashcards',
}

/** Formats the tab title for an unavailable kind when the tab is not backed by a file path. */
export function formatUnavailableKindTitle(kind: string): string {
  const normalized = stripReaderKind(kind)
  const mapped = UNAVAILABLE_TAB_TITLES[normalized]
  if (mapped) return mapped
  const stripped = kind.startsWith(READER_PREFIX) ? kind.slice(READER_PREFIX.length) : kind
  if (!stripped) return 'Tab'
  return stripped.charAt(0).toUpperCase() + stripped.slice(1)
}

/** Formats the human-readable feature name for an unavailable tab kind. */
export function formatUnavailableFeature(kind: string): string {
  const normalized = stripReaderKind(kind)
  const mapped = UNAVAILABLE_FEATURE_NAMES[normalized]
  if (mapped) return mapped
  if (kind.startsWith(READER_PREFIX)) {
    return normalized ? `${normalized} files` : kind
  }
  return kind
}

/** Returns the settings plugin search filter for an unavailable tab kind. */
export function getUnavailablePluginFilter(kind: string): string {
  const normalized = stripReaderKind(kind)
  const readerTarget = UNAVAILABLE_READER_TARGETS[normalized]
  if (readerTarget) return `${READER_PREFIX}${readerTarget}`
  const featureFilter = UNAVAILABLE_FEATURE_FILTERS[normalized]
  if (featureFilter) return featureFilter
  if (kind.startsWith(READER_PREFIX)) return `${READER_PREFIX}${normalized}`
  return `feature:${kind}`
}
