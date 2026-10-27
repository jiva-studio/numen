/**
 * What a file of the vault is put in front of the person with.
 *
 * A note opens in the editor of its type. Any other file opens in the reader
 * the core chose for its content type, which a plugin registers by extension.
 * A kind of tab hands over the way it opens a file and keeps none.
 */
import type { Component } from 'vue'
import type { Selector } from '@numen/plugin-sdk'
import type { FileKind, NoteType } from '@/entities/file/@x/tab'
import type { TextRange } from '@/shared/textRange'

/** What a file the window opens is opened as: which type of note it is, or the kind a plugin registers. */
export type EditorKind = NoteType

/** Where a file opens: in the pane the person is in, or beside it. */
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
 * A file put in front of the person in one editor. A line is somewhere inside
 * the file, and what standing there evaluates to is the editor's own.
 */
export type FileOpener = (
  path: string,
  title: string,
  options?: FilePlacement | OpenPlacement,
) => void

/** A file put in front of the person in its reader, at the place the selector names. */
export type SourceReader = (path: string, selector: Selector | null) => void

/** What the window asks the vault about the file it is opening. */
export interface FileOpenerDeps {
  fileKinds(paths: readonly string[]): Promise<ReadonlyMap<string, FileKind>>
  reconcileTab?(path: string, newKind: string): void
}

/** A reader registered for an extension, with the icon its files carry. */
interface ExtensionReader {
  readonly read: SourceReader
  readonly icon: Component | undefined
}

/** What a file the vault could not be asked about at all is opened as. */
const ORDINARY: FileKind = { isNote: true, type: 'note' }

/** The extension of a path, lowercase and without the dot, and empty where it has none. */
const getExtension = (path: string): string => {
  const name = path.split('/').pop() ?? path
  const dot = name.lastIndexOf('.')
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
  /** The editor each kind of note opens in, as its kind handed it over. */
  const editors = new Map<EditorKind, FileOpener>()

  /** The reader of each extension, as the core chose it for the extension's content type. */
  const extensionReaders = new Map<string, ExtensionReader>()

  /** A kind of tab hands over the way it puts a file in front of the person. */
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

  /** A file opened in its reader at a place; a file no reader takes opens nothing. */
  const readFile = (path: string, selector: Selector | null): void =>
    extensionReaders.get(getExtension(path))?.read(path, selector)

  /**
   * A file just made here, put in front of the person as what it was made as.
   * The vault is not asked what stands there.
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
   * A file put in front of the person: a note in the editor of its type, any
   * other file in its reader at the selector given.
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
      return void openNewFile(path, title, { type: kind.type, ...rest })
    }
    readFile(path, placement?.selector ?? null)
  }

  /** A file put in front of the person with runs of a note's text marked. */
  const openFileAt = async (path: string, _spans: readonly TextRange[]): Promise<void> => {
    const kind = await fetchFileKind(vault, path)
    if (kind?.isNote) return void openNewFile(path, '', kind.type)
    readFile(path, null)
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

/** What the window puts files in front of the person with. */
export type FileOpeners = ReturnType<typeof fileOpeners>
