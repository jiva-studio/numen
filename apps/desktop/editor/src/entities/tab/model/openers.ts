/**
 * The openers that display a vault file.
 *
 * A note opens in the editor of its type. Any other file opens in the reader
 * the core selects for its content type, which a plugin registers by extension.
 * A tab kind registers how it opens a file and holds no file state.
 */
import type { Component } from 'vue'
import type { Selector } from '@numen/plugin-host'
import type { FileKind, NoteType } from '@/shared/model/fileTypes'
import type { TextRange } from '@/shared/textRange'

/** What a file the window opens is opened as: which type of note it is, or the kind a plugin registers. */
export type EditorKind = NoteType

/** Where a file opens: in the current pane or in a split. */
export type FilePlacement = 'current' | 'split'

export interface OpenPlacement {
  where?: FilePlacement | undefined
  line?: number | undefined
  /** The place the reader of a file that is no note shows, such as a search hit. */
  selector?: Selector | undefined
}

export interface OpenNewFileOptions {
  type: EditorKind
  where?: FilePlacement | undefined
  line?: number | undefined
}

/**
 * Opens a file in one editor. `line` is a position inside the file; its
 * interpretation belongs to the editor.
 */
export type FileOpener = (
  path: string,
  title: string,
  options?: FilePlacement | OpenPlacement,
) => void

import type { SourceReader } from '@/shared/model/tabKinds'
export type { SourceReader }

/** The vault queries the window uses to open a file. */
export interface FileOpenerDeps {
  fileKinds(paths: readonly string[]): Promise<ReadonlyMap<string, FileKind>>
  reconcileTab?(path: string, newKind: string): void
}

/** A reader registered for an extension, with the icon its files carry. */
interface ExtensionReader {
  readonly read: SourceReader
  readonly icon: Component | undefined
}

/** The kind assumed for a file the vault cannot be queried about. */
const ORDINARY: FileKind = { isNote: true, type: 'note' }

/** The extension of a path, lowercase and without the dot, and empty where it has none. */
const getExtension = (path: string): string => {
  // Stryker disable next-line LogicalOperator: split('/').pop() on a string always returns a string.
  const name = path.split('/').pop() ?? path
  const dot = name.lastIndexOf('.')
  // Stryker disable next-line StringLiteral: fallback when a filename has no extension matches no registered extension.
  return dot > 0 ? name.slice(dot + 1).toLowerCase() : ''
}

const fetchFileKind = async (vault: FileOpenerDeps, path: string): Promise<FileKind | null> => {
  try {
    return (await vault.fileKinds([path])).get(path) ?? null
  } catch {
    return ORDINARY
  }
}

export function fileOpeners(vault: FileOpenerDeps) {
  /** The editor for each note type, as registered by its tab kind. */
  const editors = new Map<EditorKind, FileOpener>()

  /** The reader of each extension, as the core chose it for the extension's content type. */
  const extensionReaders = new Map<string, ExtensionReader>()

  /** A tab kind registers how it opens a file. */
  const registerEditor = (type: EditorKind, open: FileOpener): (() => void) => {
    editors.set(type, open)
    return () => {
      if (editors.get(type) === open) editors.delete(type)
    }
  }

  /** A reader takes the files of an extension, which carry its icon. */
  const registerExtensionReader = (
    extension: string,
    read: SourceReader,
    icon?: Component,
  ): (() => void) => {
    const ext = (extension.startsWith('.') ? extension.slice(1) : extension).toLowerCase()
    const entry: ExtensionReader = { read, icon }
    extensionReaders.set(ext, entry)
    return () => {
      if (extensionReaders.get(ext) === entry) extensionReaders.delete(ext)
    }
  }

  /** The icon of the reader that opens a file, and nothing where none does. */
  const findFileIcon = (path: string): Component | undefined =>
    extensionReaders.get(getExtension(path))?.icon

  /**
   * A file opened in its reader at a place, with ranges of its text lit; a file no reader takes opens
   * nothing.
   */
  const readFile = (
    path: string,
    selector: Selector | null,
    ranges?: readonly TextRange[],
  ): void => {
    const reader = extensionReaders.get(getExtension(path))
    if (!reader) return
    if (ranges) reader.read(path, selector, ranges)
    else reader.read(path, selector)
  }

  /**
   * Opens a newly created file as the kind it was created as, without querying
   * the vault for its kind.
   */
  const openNewFile = (
    path: string,
    title: string,
    options: EditorKind | OpenNewFileOptions,
  ): void => {
    const type = typeof options === 'string' ? options : options.type
    const placement = typeof options === 'string' ? undefined : options
    vault.reconcileTab?.(path, type)
    const editor = editors.get(type)
    if (editor) editor(path, title, placement)
    else readFile(path, null)
  }

  /**
   * Opens a file: a note in the editor of its type, any other file in its reader
   * at the given selector.
   */
  const openFile = async (
    path: string,
    title = '',
    options?: FilePlacement | OpenPlacement,
  ): Promise<void> => {
    const placement = typeof options === 'string' ? { where: options } : options
    const kind = await fetchFileKind(vault, path)
    if (kind?.isNote) {
      const { selector: _selector, ...rest } = placement ?? {}
      openNewFile(path, title, { type: kind.type, ...rest })
      return
    }
    readFile(path, placement?.selector ?? null)
  }

  /** Opens a file at ranges of its text: a note in its editor, whole, any other file in its reader. */
  const openFileAt = async (path: string, spans: readonly TextRange[]): Promise<void> => {
    const kind = await fetchFileKind(vault, path)
    if (kind?.isNote) {
      openNewFile(path, '', kind.type)
      return
    }
    readFile(path, null, spans)
  }

  return {
    registerEditor,
    registerExtensionReader,
    findFileIcon,
    openFile,
    openFileAt,
    openNewFile,
  }
}

/** The openers the window uses to display files. */
export type FileOpeners = ReturnType<typeof fileOpeners>
