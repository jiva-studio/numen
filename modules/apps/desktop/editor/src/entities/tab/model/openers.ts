/**
 * What a file of the vault is put in front of the person with.
 *
 * Every road to a file comes through here, holding a path and no choice: what
 * the vault says it holds there decides between the editors, the reader and the
 * player. A kind of tab hands over the way it opens a file and keeps none.
 */
import type { PlexDestination } from '@numen/ui'
import type { BookFormat, DocumentFormat, FileKind, NoteType } from '@/entities/file/@x/tab'
import type { Span } from '@/shared/span'

/**
 * What a file the window opens is opened as: which kind of note it is, or the
 * preset a fourth kind of note holds.
 */
export type EditorKind = NoteType | 'preset' | 'url'

export interface OpenPlacement {
  where?: PlexDestination | undefined
  line?: number | undefined
}

export interface OpenNewFileOptions {
  type: EditorKind
  where?: PlexDestination | undefined
  line?: number | undefined
}

/**
 * A file put in front of the person in one editor. A line is somewhere inside
 * the file, and what standing there comes to is the editor's own.
 */
export type FileOpener = (
  path: string,
  title: string,
  options?: PlexDestination | OpenPlacement,
) => void

/**
 * A source put in front of the person in the reader or in the player. The
 * spans are of the source's own text, and the person is taken to the first
 * of them.
 */
export type SourceReader = (path: string, spans: readonly Span[]) => void

/** What the window asks the vault about the file it is opening. */
export interface FileOpenerDeps {
  fileKinds(paths: readonly string[]): Promise<ReadonlyMap<string, FileKind>>
  reconcileTab?(path: string, newKind: string): void
  emitEvent?(event: string): void
}

/** A source a reader is asked for, by kind and, where one format is read its own way, by format. */
export interface ReaderKey {
  readonly kind: FileKind['kind']
  readonly format?: BookFormat | DocumentFormat
}

/** What a file the vault could not be asked about at all is opened as. */
const ORDINARY: FileKind = { kind: 'note', type: 'note' }

/** The key a reader is held under: the kind, and the format where one claims it. */
const createReaderKey = (kind: FileKind['kind'], format?: BookFormat | DocumentFormat): string =>
  format ? `${kind}/${format}` : kind

const emitFileEvent = (path: string, emit?: (event: string) => void): void => {
  const dot = path.lastIndexOf('.')
  if (dot !== -1 && emit) {
    const ext = path.slice(dot + 1).toLowerCase()
    if (ext) emit(`onFile:${ext}`)
  }
}

const findExtensionReader = (
  extensionReaders: ReadonlyMap<string, SourceReader>,
  path: string,
): SourceReader | undefined => {
  const dotIndex = path.lastIndexOf('.')
  if (dotIndex === -1) return undefined
  const ext = path.slice(dotIndex + 1).toLowerCase()
  return extensionReaders.get(ext)
}

const fetchFileKind = async (vault: FileOpenerDeps, path: string): Promise<FileKind | null> => {
  try {
    return (await vault.fileKinds([path])).get(path) ?? null
  } catch {
    return ORDINARY
  }
}

const findReader = (
  readers: ReadonlyMap<string, SourceReader>,
  kind: FileKind,
): SourceReader | undefined =>
  readers.get(createReaderKey(kind.kind, kind.format)) ?? readers.get(createReaderKey(kind.kind))

export function fileOpeners(vault: FileOpenerDeps) {
  /** The editor each kind of note opens in, as its kind handed it over. */
  const editors = new Map<EditorKind, FileOpener>()

  /**
   * The reader each source opens in, under the key the kind registered: the
   * kind, and the format where the kind reads one of them its own way.
   */
  const readers = new Map<string, SourceReader>()
  const extensionReaders = new Map<string, SourceReader>()

  /** A kind of tab hands over the way it puts a file in front of the person. */
  const registerEditor = (type: EditorKind | string, open: FileOpener): (() => void) => {
    editors.set(type as EditorKind, open)
    return () => {
      if (editors.get(type as EditorKind) === open) {
        editors.delete(type as EditorKind)
      }
    }
  }

  /**
   * A kind of tab hands over the reader its sources open in. Where one format
   * of a source is read its own way, the kind that reads it that way says so.
   */
  const registerReader = (key: ReaderKey, read: SourceReader) => {
    readers.set(createReaderKey(key.kind, key.format), read)
  }

  const registerExtensionReader = (extension: string, read: SourceReader): (() => void) =>
    addExtensionReader(extensionReaders, extension, read)

  /**
   * A file just made here, put in front of the person as what it was made as.
   * The vault is not asked what stands there.
   */
  const openNewFile = (
    path: string,
    title: string,
    options: EditorKind | OpenNewFileOptions,
  ): void => dispatchNewFile(path, title, { options, vault, editors, extensionReaders })

  /**
   * A file put in front of the person in the editor made for what it is, in the
   * reader where a document stands there, or in the player where a recording
   * does. A path holding no source at all opens nothing.
   */
  const openFile = async (
    path: string,
    title = '',
    options?: PlexDestination | OpenPlacement,
  ): Promise<void> => {
    emitFileEvent(path, vault.emitEvent)
    const kind = await fetchFileKind(vault, path)
    if (kind?.kind === 'note') {
      const placement = typeof options === 'string' ? { where: options } : options
      return void openNewFile(path, title, { type: kind.type, ...placement })
    }
    dispatchReader(
      { path, kind, spans: [] },
      { readers, extensionReaders, reconcileTab: vault.reconcileTab },
    )
  }

  const openFileAt = async (path: string, spans: readonly Span[]): Promise<void> => {
    emitFileEvent(path, vault.emitEvent)
    const kind = await fetchFileKind(vault, path)
    if (kind?.kind === 'note') return void openNewFile(path, '', kind.type)
    dispatchReader(
      { path, kind, spans },
      { readers, extensionReaders, reconcileTab: vault.reconcileTab },
    )
  }

  return {
    registerEditor,
    registerReader,
    registerExtensionReader,
    openFile,
    openFileAt,
    openNewFile,
  }
}

/** The kind of tab a file opens in. */
const mapTargetKind = (kind: FileKind): string => {
  if (kind.kind === 'book') {
    return kind.format === 'epub' ? 'book' : 'document'
  }
  return kind.kind
}

/** What the window puts files in front of the person with. */
export type FileOpeners = ReturnType<typeof fileOpeners>

interface ReaderDispatchContext {
  readers: Map<string, SourceReader>
  extensionReaders: Map<string, SourceReader>
  reconcileTab?: ((path: string, kind: string) => void) | undefined
}

function dispatchReader(
  target: { path: string; kind: FileKind | null; spans: readonly Span[] },
  context: ReaderDispatchContext,
): void {
  const extReader = findExtensionReader(context.extensionReaders, target.path)
  if (extReader) return void extReader(target.path, target.spans)
  if (!target.kind) return
  context.reconcileTab?.(target.path, mapTargetKind(target.kind))
  findReader(context.readers, target.kind)?.(target.path, target.spans)
}

function addExtensionReader(
  extensionReaders: Map<string, SourceReader>,
  extension: string,
  read: SourceReader,
): () => void {
  const ext = extension.startsWith('.') ? extension.slice(1).toLowerCase() : extension.toLowerCase()
  extensionReaders.set(ext, read)
  return () => {
    if (extensionReaders.get(ext) === read) extensionReaders.delete(ext)
  }
}

function dispatchNewFile(
  path: string,
  title: string,
  context: {
    options: EditorKind | OpenNewFileOptions
    vault: FileOpenerDeps
    editors: Map<EditorKind, FileOpener>
    extensionReaders?: Map<string, SourceReader>
  },
): void {
  emitFileEvent(path, context.vault.emitEvent)
  const type = typeof context.options === 'string' ? context.options : context.options.type
  const placement = typeof context.options === 'string' ? undefined : context.options
  context.vault.reconcileTab?.(path, type)
  const editor = context.editors.get(type)
  if (editor) {
    editor(path, title, placement)
  } else if (context.extensionReaders) {
    const extReader = findExtensionReader(context.extensionReaders, path)
    if (extReader) extReader(path, [])
  }
}
