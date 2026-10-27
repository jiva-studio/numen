import type { Result } from './types'

// TextKind is what a text version is, as the plugin that makes it names it: 'transcript',
// 'recognition'. The core compares it as an opaque string.
export type TextKind = string

// Selector locates a segment in the file its text was made from (W3C Web Annotation selector). The
// core stores it as is and returns it with search hits; the view that opens the file reads it. Each
// key and its unit come from a standard:
//   t      [start, end] in seconds from the start of a recording (W3C Media Fragments)
//   page   a page of a document, counted from 1 (RFC 8118)
//   cfi    an EPUB Canonical Fragment Identifier
//   boxes  rectangles on the page, each [minX, minY, maxX, maxY] in fractions of the page
// A plugin may add keys of its own, named with its plugin id as a prefix.
export interface Selector {
  readonly t?: readonly [number, number]
  readonly page?: number
  readonly cfi?: string
  readonly boxes?: readonly (readonly [number, number, number, number])[]
  readonly [key: string]: unknown
}

// TextSegment is one segment of a version's text. `heading` is the heading the segment opens, and
// absent for a segment inside one.
export interface TextSegment {
  readonly text: string
  readonly heading?: string
  readonly selector?: Selector
}

// TextVersionId names a version among the versions of one file: the plugin id and its key, joined
// by "/".
export type TextVersionId = string

// TextVersion is the host's record of one text of a file. `sourceVersionId` is empty for a version
// made from the file itself. `file` is the version's file in the folder of the plugin that made it.
export interface TextVersion {
  readonly id: TextVersionId
  readonly path: string
  readonly pluginId: string
  readonly key: string
  readonly sourceVersionId: TextVersionId
  readonly kind: TextKind
  readonly file: string
  readonly createTime: Date
}

// LiveText is the live version of a file and its segments.
export interface LiveText {
  readonly version: TextVersion
  readonly segments: readonly TextSegment[]
}

// TextErrorCode is why an edit was not recorded: the source version is not there, or the segments
// are not valid.
export type TextErrorCode = 'missing' | 'invalid'

// VaultTexts is what a view reads of the text versions of the vault the window shows, and how it
// saves a person's edit.
export interface VaultTexts {
  // The live version of a file and its segments, and null where none is live.
  readLiveText(path: string): Promise<LiveText | null>
  // Records a person's edit as a new version made from `sourceVersionId`, in the folder of the
  // plugin whose view the person edits in. The new version becomes the live one.
  createEdit(
    path: string,
    sourceVersionId: TextVersionId,
    segments: readonly TextSegment[],
  ): Promise<Result<TextVersion, TextErrorCode>>
}

// TextInput is what a manifest says a plugin makes text from: files of the content types
// `contentTypes` names, such as `audio/*`, and versions of the kinds `kinds` names. `autoRun` asks
// to be run on a file of its types that has no live version. `menuEntry` is the plugin's entry in
// the menu of a file its input takes; a plugin without one is not offered there.
export interface TextInput {
  contentTypes?: string[]
  kinds?: TextKind[]
  autoRun?: boolean
  menuEntry?: TextMenuEntry
}

// TextMenuEntry is what a plugin's entry in a file's menu reads and is drawn with. `icon` is a
// Lucide icon name, such as `scan-text`.
export interface TextMenuEntry {
  title: string
  icon?: string
}

// joinTextSegments is a text as one string.
export function joinTextSegments(segments: readonly TextSegment[]): string {
  return segments.map((segment) => segment.text).join('')
}
