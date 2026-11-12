/** The answer to a call that can fail: what was returned, or the sentence that says why not. */
export type Result<T> =
  { readonly ok: true; readonly value: T } | { readonly ok: false; readonly error: string }

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

/** Where the core is, and what it answers. */
export interface Core {
  readonly port: number
  readonly dir: string
  getVaultState(): Promise<VaultState>
  /** The note the vault opens on, and null while it has none. */
  getOpeningPath(): Promise<string | null>
  readNote(path: string): Promise<Result<NoteRead>>
  /** Writes against `baseline`; a null baseline writes over whatever the note now holds. */
  writeNote(path: string, prose: string, baseline: NoteBaseline | null): Promise<Result<null>>
}
