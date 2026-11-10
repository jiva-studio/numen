import type { Ref } from 'vue'
import type { FileConflict } from './buffers'

// WikilinkChoice is a note a wikilink can name.
export interface WikilinkChoice {
  readonly title: string
  readonly type: 'note'
}

// EditorHandle is what a note editor lets its tab call: moving the keyboard into it, laying it out
// again, and scrolling to a line. Each answers false where the editor cannot.
export interface EditorHandle {
  focus?(): boolean
  measure?(): void
  reveal?(line: number): boolean
}

// NoteTabState is the `state` prop the window passes the component of a note tab. The refs
// are the host's store of the note: the component reads them and calls the methods to write.
export interface NoteTabState {
  readonly path: string
  readonly body: Readonly<Ref<string>>
  readonly isLoading: Readonly<Ref<boolean>>
  readonly errorMessage: Readonly<Ref<string>>
  readonly fileConflict?: Readonly<Ref<FileConflict>>
  readonly wikilinks?: {
    search(query: string): Promise<readonly WikilinkChoice[]>
    headings(noteTitle: string): Promise<readonly string[]>
  }
  updateBody(text: string): void
  save(): void
  // Keeps the text on screen over the file that changed on disk.
  keepMine?(): void
  // Reads the file that changed on disk, dropping the text on screen.
  takeFile?(): void
  followLink?(address: string): void
  // Hands the window the editor, which it lays out again when the tab is shown.
  attachEditor?(editor: EditorHandle | null): void
}
