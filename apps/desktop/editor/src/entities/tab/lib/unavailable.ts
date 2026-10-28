import type { Component } from 'vue'
import UnavailablePane from './UnavailablePane.vue'
import type { AnyTabKind } from './kinds'

/** What a tab of an unavailable kind holds: the kind it was made for, its path, and the state its plugin saved. */
export interface UnavailableState {
  readonly kind: string
  readonly targetPath: string
  readonly saved: unknown
  readonly title?: string | undefined
}

export interface UnavailableKindOptions {
  readonly title?: string | undefined
  readonly icon?: Component | undefined
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
  'reader:markdown': 'Note',
  'reader:pdf': 'Document',
  'reader:reader-pdf': 'Document',
  'reader:epub': 'Book',
  'reader:reader-epub': 'Book',
  'reader:media-player': 'Recording',
}

const UNAVAILABLE_FEATURE_NAMES: Readonly<Record<string, string>> = {
  'reader:markdown': 'Markdown notes',
  note: 'Markdown notes',
  'reader:pdf': 'PDF documents',
  'reader:reader-pdf': 'PDF documents',
  document: 'PDF documents',
  'reader:epub': 'EPUB books',
  'reader:reader-epub': 'EPUB books',
  book: 'EPUB books',
  'reader:media-player': 'audio playback and recording',
  recording: 'audio playback and recording',
  files: 'file tree',
  plex: 'plex graph',
  agent: 'agent chat',
  cards: 'flashcards',
  preset: 'flashcard presets',
}

const UNAVAILABLE_PLUGIN_FILTERS: Readonly<Record<string, string>> = {
  'reader:markdown': 'reader:markdown',
  note: 'reader:markdown',
  'reader:pdf': 'reader:pdf',
  'reader:reader-pdf': 'reader:pdf',
  document: 'reader:pdf',
  'reader:epub': 'reader:epub',
  'reader:reader-epub': 'reader:epub',
  book: 'reader:epub',
  'reader:media-player': 'reader:media-player',
  recording: 'feature:recording',
  files: 'feature:files',
  plex: 'feature:plex',
  agent: 'feature:agent',
  cards: 'feature:flashcards',
  preset: 'feature:flashcards',
}

/** Formats the tab title for an unavailable kind when the tab is not backed by a file path. */
export function formatUnavailableKindTitle(kind: string): string {
  const mapped = UNAVAILABLE_TAB_TITLES[kind]
  if (mapped) return mapped
  const stripped = kind.startsWith('reader:') ? kind.slice('reader:'.length) : kind
  if (!stripped) return 'Tab'
  return stripped.charAt(0).toUpperCase() + stripped.slice(1)
}

/** Formats the human-readable feature name for an unavailable tab kind. */
export function formatUnavailableFeature(kind: string): string {
  const mapped = UNAVAILABLE_FEATURE_NAMES[kind]
  if (mapped) return mapped
  if (kind.startsWith('reader:')) {
    const readerName = kind.slice('reader:'.length).replace(/^reader-/, '')
    return readerName ? `${readerName} files` : kind
  }
  return kind
}

/** Returns the settings plugin search filter for an unavailable tab kind. */
export function getUnavailablePluginFilter(kind: string): string {
  const mapped = UNAVAILABLE_PLUGIN_FILTERS[kind]
  if (mapped) return mapped
  if (kind.startsWith('reader:')) return kind
  return `feature:${kind}`
}

/** The placeholder kinds made here. */
const placeholders = new WeakSet<AnyTabKind>()

/** Whether a kind is the placeholder of a kind that is not registered. */
export const isUnavailableKind = (kind: AnyTabKind): boolean => placeholders.has(kind)

/** The last segment of a path, which names the tab of a file. */
const getFileName = (path: string): string => path.split('/').pop() ?? path

/**
 * The kind of a tab whose own kind is not registered, because its plugin is disabled or missing.
 * It keeps the kind, the path and the saved state, so the open tabs keep the tab as it was and
 * the plugin opens it again once it is enabled.
 */
export const createUnavailableKind = (
  kind: string,
  options?: UnavailableKindOptions,
): AnyTabKind => {
  const placeholder: AnyTabKind = {
    kind,
    ...(options?.icon ? { icon: options.icon } : {}),
    open: (targetPath: string, _id: string, saved?: unknown): UnavailableState => ({
      kind,
      targetPath,
      saved,
      ...(options?.title ? { title: options.title } : {}),
    }),
    getTitle: (state: unknown) => {
      const unavailable = state as UnavailableState
      return (
        unavailable.title ||
        getFileName(unavailable.targetPath) ||
        options?.title ||
        formatUnavailableKindTitle(kind)
      )
    },
    getOpenTab: (state: unknown) => ({ path: (state as UnavailableState).targetPath }),
    getState: (state: unknown) => (state as UnavailableState).saved,
    pane: UnavailablePane,
    identity: (targetPath: string) => targetPath,
  }
  placeholders.add(placeholder)
  return placeholder
}
