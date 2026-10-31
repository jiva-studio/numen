/**
 * Creating a note from the window: untitled, under a title, or in a relation to another note.
 *
 * The core numbers untitled names, files a related note beside the note it is related to and
 * writes the record of the relation. The window reports what the core answers.
 */
import {
  formatErrorMessage,
  getRelationErrorWords,
  type ConnectionRole,
  type VaultRelations,
} from '@numen/plugin-sdk'
import type { NewNote } from '../lib/note'
import type { CreateResult } from '@/shared/model/fileTypes'
import type { MessageWriter } from '@/shared/notices/messages'
import { ERRORS } from '@/shared/words'

/** The operations note creation needs from the vault. */
export interface NoteMaker {
  /** A note made, named after the title it is given. An empty title makes an untitled note. */
  create(note: NewNote): Promise<CreateResult>
  readonly relations: Pick<VaultRelations, 'createRelated'>
}

/** A note that now exists: where it is filed, and what it is called. */
export interface NoteRef {
  readonly path: string
  readonly title: string
}

export function createNoteWriter(core: NoteMaker, writeMessage: MessageWriter) {
  async function createInFolder(title: string, folder: string): Promise<NoteRef | null> {
    try {
      const made = await core.create({ title, folder })
      if (!made.ok) {
        writeMessage(ERRORS[made.error], 'error')
        return null
      }
      writeMessage('')
      return { path: made.value.path, title }
    } catch (error) {
      writeMessage(formatErrorMessage(error), 'error')
      return null
    }
  }

  async function createRelated(
    title: string,
    from: string,
    role: ConnectionRole,
  ): Promise<NoteRef | null> {
    try {
      const made = await core.relations.createRelated({ from, role, title })
      if (!made.ok) {
        writeMessage(getRelationErrorWords(made.error), 'error')
        return null
      }
      writeMessage('')
      return { path: made.value, title }
    } catch (error) {
      writeMessage(formatErrorMessage(error), 'error')
      return null
    }
  }

  /** The path of an untitled note made in the folder; empty where none was made. */
  async function createUntitled(folder: string): Promise<string> {
    return (await createInFolder('', folder))?.path ?? ''
  }

  /** A note under the title, in a relation to another note, or at the root where it has no relation. */
  async function createWithTitle(
    title: string,
    from: string,
    relation: ConnectionRole | null,
  ): Promise<NoteRef | null> {
    if (relation === null) return createInFolder(title, '')
    return createRelated(title, from, relation)
  }

  return { createUntitled, createWithTitle }
}

export type NoteCreator = ReturnType<typeof createNoteWriter>
