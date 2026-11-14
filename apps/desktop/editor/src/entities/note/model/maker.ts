/**
 * Creating a note from the window: untitled, under a title, or in a relation to another note.
 *
 * The core numbers untitled names, files a related note beside the note it is related to and
 * writes the record of the relation. The window reports what the core answers.
 */
import { formatError } from '@/shared/api/errors'
import type { NoteApi } from '../api/notes'
import {
  getRelationErrorWords,
  type ConnectionRole,
  type VaultRelations,
} from '@numen/plugin-loader'
import type { MessageWriter } from '@/shared/model/notices/messages'
import { ERRORS } from '@/shared/ui/words'

/** A note that now exists: where it is filed, and what it is called. */
interface NoteRef {
  readonly path: string
  readonly title: string
}

/** The calls a note is made through. */
export interface NoteMakingApi {
  readonly createNote: NoteApi['createNote']
  readonly relations: Pick<VaultRelations, 'createRelated'>
}

export function createNoteWriter(api: NoteMakingApi, writeMessage: MessageWriter) {
  async function createInFolder(title: string, folder: string): Promise<NoteRef | null> {
    try {
      const made = await api.createNote({ title, folder })
      if (!made.ok) {
        writeMessage(ERRORS[made.error], 'error')
        return null
      }
      writeMessage('')
      return { path: made.value.path, title }
    } catch (error) {
      writeMessage(formatError(error), 'error')
      return null
    }
  }

  async function createRelatedNote(
    title: string,
    from: string,
    role: ConnectionRole,
  ): Promise<NoteRef | null> {
    try {
      const made = await api.relations.createRelated({ from, role, title })
      if (!made.ok) {
        writeMessage(getRelationErrorWords(made.error), 'error')
        return null
      }
      writeMessage('')
      return { path: made.value, title }
    } catch (error) {
      writeMessage(formatError(error), 'error')
      return null
    }
  }

  /** The path of an untitled note made in the folder; empty where none was made. */
  async function createUntitledNote(folder: string): Promise<string> {
    return (await createInFolder('', folder))?.path ?? ''
  }

  /** A note under the title, in a relation to another note, or at the root where it has no relation. */
  async function createWithTitle(
    title: string,
    from: string,
    relation: ConnectionRole | null,
  ): Promise<NoteRef | null> {
    if (relation === null) return createInFolder(title, '')
    return createRelatedNote(title, from, relation)
  }

  return { createUntitledNote, createWithTitle }
}

export type NoteCreator = ReturnType<typeof createNoteWriter>
