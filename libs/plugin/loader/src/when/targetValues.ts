/**
 * The target variables of when conditions, derived in this one place from the path a place passes,
 * as VS Code derives its resource keys from the URI. The core derives the same values for a run.
 */
import { isNotePath, type CommandTarget, type JsonObject, type JsonValue } from '@numen/plugin-sdk'

/** What a place read of the note of a target: its type and its frontmatter. */
export interface TargetNote {
  readonly type?: string | undefined
  readonly properties?: JsonObject | undefined
}

/** What the derivation reads of a target. */
export type TargetPath = Pick<CommandTarget, 'path' | 'isFolder' | 'others'>

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

const getNoteValues = ({ type, properties }: TargetNote) => ({
  'target.noteType': type ?? '',
  'target.properties': properties ?? {},
})

/**
 * The target variables over the target: its content type by its extension from the table the core
 * sends, and its note type and properties where the place read them. All are false or empty where
 * there is no target.
 */
export function getTargetValues(
  target: TargetPath | undefined,
  contentTypes: Readonly<Record<string, string>>,
  note: TargetNote = {},
): Readonly<Record<string, JsonValue>> {
  if (!target) return NO_TARGET
  const { path, isFolder, others } = target
  const extension = isFolder ? '' : getExtension(path)
  const isNote = !isFolder && isNotePath(path)
  return {
    'target.isRoot': isFolder && path === '',
    'target.isFolder': isFolder,
    'target.isMultiple': others.length > 0,
    'target.isNote': isNote,
    'target.isFile': !isFolder && !isNote && path !== '',
    'target.contentType': contentTypes[extension] ?? '',
    ...getNoteValues(isNote ? note : {}),
  }
}
