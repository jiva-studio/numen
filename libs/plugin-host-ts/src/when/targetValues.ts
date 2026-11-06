/**
 * The target variables of when conditions, derived in this one place from the path a place passes,
 * as VS Code derives its resource keys from the URI. The core derives the same values for a run.
 */
import type { CommandTarget } from '@numen/plugin-sdk'

/** What the derivation reads of a target. */
export type TargetPath = Pick<CommandTarget, 'path' | 'isFolder' | 'others'>

/** What a place read of a note before it asks: its type and its frontmatter properties. */
export interface TargetNote {
  readonly type?: string
  readonly properties?: Readonly<Record<string, unknown>>
}

const NO_NOTE = { 'target.noteType': '', 'target.properties': {} } as const

const getNoteValues = (note: TargetNote) => ({
  'target.noteType': note.type ?? '',
  'target.properties': note.properties ?? {},
})

/** The extension that makes a file a note. */
const NOTE_EXTENSION = 'md'

const NO_TARGET = {
  'target.isRoot': false,
  'target.isFolder': false,
  'target.isMultiple': false,
  'target.isNote': false,
  'target.isFile': false,
  'target.contentType': '',
  'target.noteType': '',
  'target.properties': {},
} as const

const getExtension = (path: string): string => {
  const name = path.slice(path.lastIndexOf('/') + 1)
  const dot = name.lastIndexOf('.')
  return dot > 0 ? name.slice(dot + 1).toLowerCase() : ''
}

/**
 * The target variables over the target: its content type by its extension from the table the core
 * sends, and its note type and frontmatter properties where the place read them. All are false or empty where there is no target.
 */
export function getTargetValues(
  target: TargetPath | undefined,
  contentTypes: Readonly<Record<string, string>>,
  note: TargetNote = {},
): Readonly<Record<string, unknown>> {
  if (!target) return NO_TARGET
  const { path, isFolder, others } = target
  const extension = isFolder ? '' : getExtension(path)
  const isNote = extension === NOTE_EXTENSION
  return {
    'target.isRoot': isFolder && path === '',
    'target.isFolder': isFolder,
    'target.isMultiple': others.length > 0,
    'target.isNote': isNote,
    'target.isFile': !isFolder && !isNote && path !== '',
    'target.contentType': contentTypes[extension] ?? '',
    ...(isNote ? getNoteValues(note) : NO_NOTE),
  }
}
