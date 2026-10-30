import type { Result } from './types'

// TextKind is what a text version is, as the plugin that makes it names it. The core
// compares it as an opaque string.
export type TextKind = string

// Selector locates a segment in the file its text was made from (W3C Web Annotation selector). The
// core stores it as is and returns it with search hits; the plugin that made the text and the view
// that opens the file agree on its keys.
export type Selector = Readonly<Record<string, unknown>>

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

// VaultTexts is what a view reads of the text versions of the vault shown in the
// window, and how it saves a user edit.
export interface VaultTexts {
  // The live version of a file and its segments, and null where none is live.
  readLiveText(path: string): Promise<LiveText | null>
  // Records a user edit as a new version derived from `sourceVersionId`, in the
  // folder of the plugin whose view is being edited. The new version becomes the live one.
  createEdit(
    path: string,
    sourceVersionId: TextVersionId,
    segments: readonly TextSegment[],
  ): Promise<Result<TextVersion, TextErrorCode>>
}

// TextMaker is what a manifest declares of the text a plugin makes. `contentTypes`, where it names
// any, are the types of the files, such as `audio/*`, for which the host requests text from the plugin
// when they have no live version. A plugin that makes its text when a command runs names none.
export interface TextMaker {
  contentTypes?: string[]
}

// joinTextSegments is a text as one string.
export function joinTextSegments(segments: readonly TextSegment[]): string {
  return segments.map((segment) => segment.text).join('')
}
