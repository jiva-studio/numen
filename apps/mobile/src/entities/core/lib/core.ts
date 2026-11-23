import type { Result } from '@numen/result'

/** The file a note was read from or written to, as the core last saw it. */
export interface NoteStamp {
  readonly path: string
  readonly size: bigint
  readonly mtime: bigint
}

/** What a caller last saw of a note: its prose and the file that prose came out of. */
export interface NoteBaseline {
  readonly prose: string
  readonly stamp: NoteStamp
}

/** A note as the core returned it; the stamp is absent when the core named no file. */
interface NoteRead {
  readonly prose: string
  readonly stamp: NoteStamp | null
}

/** The vault the core opened. */
interface VaultState {
  readonly id: string
  readonly name: string
  readonly path: string
}

/** A file or folder of a listing. An absent file is listed and its bytes are not on the phone. */
export interface FileRow {
  readonly path: string
  readonly name: string
  readonly isFolder: boolean
  readonly isAbsent: boolean
}

/** Where the core is, and what it answers. */
export interface Core {
  readonly port: number
  readonly dir: string
  getVaultState(): Promise<VaultState>
  /** The note the vault opens on, and null while it has none. */
  getOpeningPath(): Promise<string | null>
  readNote(path: string): Promise<Result<NoteRead>>
  /** Writes against `baseline`; a null baseline writes over whatever the note now holds. */
  writeNote(path: string, prose: string, baseline: NoteBaseline   /** The files and folders of a folder; the empty path is the root of the vault. */
  listFiles(folder: string): Promise<readonly FileRow[]>
  /** Downloads an absent file; the failure says why in the words the person reads. */
  fetchFile(path: string): Promise<Result<null>>
  /** Turns a downloaded file back to an absent one; a file with changes that are not sent stays. */
  dropDownload(path: string): Promise<Result<null>>
| null): Promise<Result<null>>
}
