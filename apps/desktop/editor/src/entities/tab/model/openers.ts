/**
 * The openers that display a vault file.
 *
 * A note opens in the tab kind the core chooses for it. A note no tab kind takes and
 * any other file open in the reader the core selects for their content type, which a
 * plugin registers by extension. A tab kind registers how it opens a file and holds no
 * file state.
 */
import type { Component } from 'vue'
import type { Selector } from '@numen/plugin-loader'
import type { FileKind } from '@/shared/model/fileTypes'
import type { TextRange } from '@/shared/lib/textRange'

/** Where a file opens: in the current pane or in a split. */
export type FilePlacement = 'current' | 'split'

interface OpenPlacement {
  where?: FilePlacement | undefined
  line?: number | undefined
  selector?: Selector | undefined
}

/**
 * Opens a file in one editor. `line` is a position inside the file; its
 * interpretation belongs to the editor.
 */
type FileOpener = (path: string, title: string, options?: FilePlacement | OpenPlacement) => void

import type { SourceReader } from '../lib/kinds'
export type { SourceReader }

/**
 * The vault queries the window uses to open a file, and the starts of the plugins whose tab kind or
 * reader opens it, which register them as they start.
 */
export interface FileOpenerDeps {
  fileKinds(paths: readonly string[]): Promise<ReadonlyMap<string, FileKind>>
  reconcileTab?(path: string, newKind: string): void
  startTabKind?(kind: string): Promise<void>
  startExtensionReader?(extension: string): Promise<void>
}

/** A reader registered for an extension, with the icon its files carry. */
interface ExtensionReader {
  readonly read: SourceReader
  readonly icon: Component | undefined
}

const getExtension = (path: string): string => {
  // Stryker disable next-line LogicalOperator: split('/').pop() on a string always returns a string.
  const name = path.split('/').pop() ?? path
  const dot = name.lastIndexOf('.')
  // Stryker disable next-line StringLiteral: fallback when a filename has no extension matches no registered extension.
  return dot > 0 ? name.slice(dot + 1).toLowerCase() : ''
}

const fetchFileKind = async (vault: FileOpenerDeps, path: string): Promise<FileKind | null> => {
  return (await vault.fileKinds([path])).get(path) ?? null
}

export function fileOpeners(vault: FileOpenerDeps) {
  /** How each tab kind opens a file, by the kind. */
  const editors = new Map<string, FileOpener>()

  /** The reader of each extension, as the core chose it for the extension's content type. */
  const extensionReaders = new Map<string, ExtensionReader>()

  /** A tab kind registers how it opens a file. */
  const registerEditor = (kind: string, open: FileOpener): (() => void) => {
    editors.set(kind, open)
    return () => {
      if (editors.get(kind) === open) editors.delete(kind)
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
   * Opens a file in its reader at a place, with text ranges highlighted, once the plugin of the reader
   * has started; a file with no reader is not opened.
   */
  const readFile = async (
    path: string,
    selector: Selector | null,
    ranges?: readonly TextRange[],
  ): Promise<void> => {
    const extension = getExtension(path)
    if (!extensionReaders.has(extension)) await vault.startExtensionReader?.(extension)
    const reader = extensionReaders.get(extension)
    if (!reader) return
    if (ranges) reader.read(path, selector, ranges)
    else reader.read(path, selector)
  }

  /**
   * Opens the note in the tab kind the core chose for it, once the plugin of the kind has started;
   * false where no tab kind takes it or no frontend has registered that kind.
   */
  const openNote = async (
    path: string,
    kind: FileKind,
    title: string,
    placement?: OpenPlacement,
  ): Promise<boolean> => {
    if (!kind.isNote || !kind.tabKind) return false
    if (!editors.has(kind.tabKind)) await vault.startTabKind?.(kind.tabKind)
    const editor = editors.get(kind.tabKind)
    if (!editor) return false
    vault.reconcileTab?.(path, kind.tabKind)
    const { selector: _selector, ...rest } = placement ?? {}
    editor(path, title, rest)
    return true
  }

  /**
   * Opens a file: a note in the tab kind the core chose for it, a note no tab kind takes and any
   * other file in its reader at the given selector. A failed kind query rejects and opens nothing.
   */
  const openFile = async (
    path: string,
    title = '',
    options?: FilePlacement | OpenPlacement,
  ): Promise<void> => {
    const placement = typeof options === 'string' ? { where: options } : options
    const kind = await fetchFileKind(vault, path)
    if (kind && (await openNote(path, kind, title, placement))) return
    await readFile(path, placement?.selector ?? null)
  }

  /** Opens a file at ranges of its text: a note in its editor, whole, any other file in its reader. */
  const openFileAt = async (path: string, spans: readonly TextRange[]): Promise<void> => {
    const kind = await fetchFileKind(vault, path)
    if (kind && (await openNote(path, kind, ''))) return
    await readFile(path, null, spans)
  }

  return {
    registerEditor,
    registerExtensionReader,
    findFileIcon,
    openFile,
    openFileAt,
  }
}

/** The openers the window uses to display files. */
export type FileOpeners = ReturnType<typeof fileOpeners>
