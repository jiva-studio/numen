import type { Disposable, Result } from './types'

// TextFormat is the shape of a text version's places: plain text has none, timed text is placed at
// moments of a recording, positioned text in rectangles on pages.
export type TextFormat = 'plain' | 'timed' | 'positioned'

// TextRect is a rectangle on a page, in fractions of the page.
export interface TextRect {
  readonly minX: number
  readonly minY: number
  readonly maxX: number
  readonly maxY: number
}

// TextPlace is where a part stands: a page, counted from zero, and rectangles on it for positioned
// text; moments in milliseconds for timed text.
export interface TextPlace {
  readonly page: number
  readonly rects: readonly TextRect[]
  readonly startMs: number
  readonly endMs: number
}

// TextPart is one run of a text and its place. `section` names the section the part opens, and is
// empty for a part inside one.
export interface TextPart {
  readonly text: string
  readonly place: TextPlace
  readonly section: string
}

// TextVersionId names a version among the versions of one file: the plugin id and its key, joined
// by "/".
export type TextVersionId = string

// TextVersion is the host's record of one text of a file. `inputVersionId` is empty for a version
// made from the file itself. `inputHash` is the SHA-256, in hex, of the file's bytes for such a
// version and the hash of the input version otherwise; `hash` is the version's own text now.
export interface TextVersion {
  readonly id: TextVersionId
  readonly path: string
  readonly pluginId: string
  readonly key: string
  readonly inputVersionId: TextVersionId
  readonly format: TextFormat
  readonly inputHash: string
  readonly hash: string
  readonly createTime: Date
}

// TextListing is the versions of a file, and the live one the window shows: empty where none is.
export interface TextListing {
  readonly versions: readonly TextVersion[]
  readonly liveVersionId: TextVersionId
}

// TextPage is one page of a text. An empty `nextCursor` is the last page.
export interface TextPage {
  readonly parts: readonly TextPart[]
  readonly nextCursor: string
}

// TextErrorCode is why a text could not be read.
export type TextErrorCode = 'missing' | 'repeatedCursor'

// VaultTexts is what a plugin reads of the text versions of the vault the window shows. Text is
// read through the host from the plugin that made it.
export interface VaultTexts {
  listVersions(path: string): Promise<TextListing>
  readText(
    path: string,
    versionId: TextVersionId,
    cursor?: string,
  ): Promise<Result<TextPage, TextErrorCode>>
  // The handler is told the path of every file whose versions change.
  onVersionsChange(handler: (path: string) => void): Disposable
}

// TextRequest names one page of a text a plugin keeps.
export interface TextRequest {
  readonly vaultId: string
  readonly path: string
  readonly key: string
  readonly cursor: string
}

// TextCreation asks for text of a file, or of the version `inputVersionId` names.
export interface TextCreation {
  readonly vaultId: string
  readonly path: string
  readonly inputVersionId: TextVersionId
}

// TextProvider is what a plugin that declares a text input answers the host with.
export interface TextProvider {
  getText(request: TextRequest): Promise<Result<TextPage, TextErrorCode>>
  // Starts making text; the plugin reports the version once there is text.
  createText(request: TextCreation): Promise<void>
  deleteText(vaultId: string, path: string, key: string): Promise<void>
}

// TextInput is what a manifest says a plugin makes text from: files of the types `files` names, by
// extension with its dot, and versions in the formats `formats` names. `autoRun` asks to be run on
// a file of its types that has no live version.
export interface TextInput {
  files?: string[]
  formats?: TextFormat[]
  autoRun?: boolean
}

// readWholeText reads every page of a version's text, in order.
export async function readWholeText(
  texts: VaultTexts,
  path: string,
  versionId: TextVersionId,
): Promise<Result<readonly TextPart[], TextErrorCode>> {
  const parts: TextPart[] = []
  let cursor = ''
  for (;;) {
    const page = await texts.readText(path, versionId, cursor)
    if (!page.ok) {
      return page
    }
    parts.push(...page.value.parts)
    if (page.value.nextCursor === '') {
      return { ok: true, value: parts }
    }
    if (page.value.nextCursor === cursor) {
      return { ok: false, error: 'repeatedCursor' }
    }
    cursor = page.value.nextCursor
  }
}

// joinTextParts is a text as one string.
export function joinTextParts(parts: readonly TextPart[]): string {
  return parts.map((part) => part.text).join('')
}
