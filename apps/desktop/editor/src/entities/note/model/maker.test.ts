/**
 * Creating a note from the window.
 *
 * The core numbers untitled names and decides where a related note is filed and
 * what it writes. The tests cover what the window sends and what it says about
 * the answer.
 */
import { describe, expect, it } from 'vitest'
import { NOTE_ERROR_CODE, NOTE_ROLE, type NoteErrorCodeValue } from '@numen/core-client'
import { createLostConnection, type CoreAnswers } from '@numen/core-client/testing'
import { getRelationErrorWords } from '@numen/plugin-loader'

import { createNoteWriter, type NoteMakingApi } from './maker'
import { getNoteIcon, registerTabKindIcon } from './icons'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { writer } from '@/testing/writer'

/** The calls a note is made through, over the core the test answered last. */
let making: NoteMakingApi

/** The core answers the creation of notes as `answers` say. */
const answerMaking = (answers: CoreAnswers): void => {
  const api = createAnsweredApi(answers)
  making = { createNote: api.note.createNote, relations: api.relations }
}

/** What the core answers to the creation of a note, and to the creation of a related one. */
interface Answers {
  readonly created?: { path?: string; error?: NoteErrorCodeValue }
  readonly related?: { path?: string; error?: NoteErrorCodeValue }
}

function fake({
  created = { path: 'Untitled note.md' },
  related = { path: 'physics/Untitled note.md' },
}: Answers = {}) {
  const notes: { title: string; folder: string }[] = []
  const requests: { from: string; role: number; title: string }[] = []
  answerMaking({
    notes: {
      createNote: (request) => {
        notes.push({ title: request.title, folder: request.path })
        return created
      },
      createRelatedNote: (request) => {
        requests.push({ from: request.from, role: request.role, title: request.title })
        return related
      },
    },
  })
  return { notes, requests, ...writer() }
}

describe('making an untitled note', () => {
  it('asks the core for a note with no title, in the folder', async () => {
    const { notes, write } = fake()

    expect(await createNoteWriter(making, write).createUntitledNote('physics')).toBe(
      'Untitled note.md',
    )
    expect(notes).toStrictEqual([{ title: '', folder: 'physics' }])
  })

  it('answers with no path and says why when the core made nothing', async () => {
    const { write, messages } = fake({ created: { error: NOTE_ERROR_CODE.ALREADY_EXISTS } })

    expect(await createNoteWriter(making, write).createUntitledNote('')).toBe('')
    expect(messages.at(-1)).toStrictEqual({
      text: 'a file with this name exists',
      kind: 'error',
    })
  })

  it('says a core that could not be reached', async () => {
    answerMaking({
      notes: {
        createNote: () => {
          throw createLostConnection('the vault is out of reach')
        },
      },
    })
    const { write, last, messages } = writer()

    expect(await createNoteWriter(making, write).createUntitledNote('')).toBe('')
    expect(last()).toContain('Numen is not responding')
    expect(last()).not.toContain('out of reach')
    expect(messages.at(-1)?.kind).toBe('error')
  })
})

describe('making a note under a name a person gave it', () => {
  it('files a note with no relation at the root under the title', async () => {
    const { notes, write, last } = fake({ created: { path: 'Entropy.md' } })

    const made = await createNoteWriter(making, write).createWithTitle('Entropy', '', null)

    expect(made).toStrictEqual({ path: 'Entropy.md', title: 'Entropy' })
    expect(notes).toStrictEqual([{ title: 'Entropy', folder: '' }])
    expect(last()).toBe('')
  })

  it('leaves the relation of a related note to the core', async () => {
    const { requests, notes, write, last } = fake({ related: { path: 'physics/Entropy.md' } })

    const made = await createNoteWriter(making, write).createWithTitle(
      'Entropy',
      'physics/Ontology.md',
      'child',
    )

    expect(made).toStrictEqual({ path: 'physics/Entropy.md', title: 'Entropy' })
    expect(requests).toStrictEqual([
      { from: 'physics/Ontology.md', role: NOTE_ROLE.CHILD, title: 'Entropy' },
    ])
    expect(notes).toStrictEqual([])
    expect(last()).toBe('')
  })

  it('says why the core made no related note', async () => {
    const { write, messages } = fake({ related: { error: NOTE_ERROR_CODE.NOT_FOUND } })

    expect(
      await createNoteWriter(making, write).createWithTitle('Entropy', 'A.md', 'parent'),
    ).toBeNull()
    expect(messages.at(-1)).toStrictEqual({
      text: getRelationErrorWords('notFound'),
      kind: 'error',
    })
  })

  it('fails with a message when the core cannot be reached for a related note', async () => {
    answerMaking({
      notes: {
        createRelatedNote: () => {
          throw createLostConnection('the vault is out of reach')
        },
      },
    })
    const { write, last, messages } = writer()

    expect(
      await createNoteWriter(making, write).createWithTitle('Entropy', 'A.md', 'jump'),
    ).toBeNull()
    expect(last()).toContain('Numen is not responding')
    expect(messages.at(-1)?.kind).toBe('error')
  })
})

describe('registerTabKindIcon', () => {
  it('keeps the newer icon when an older disposer runs late', () => {
    const first = { name: 'FirstIcon' }
    const second = { name: 'SecondIcon' }
    const disposeFirst = registerTabKindIcon('sample', first)
    const disposeSecond = registerTabKindIcon('sample', second)

    disposeFirst()
    expect(getNoteIcon('sample')).toBe(second)

    disposeSecond()
    expect(getNoteIcon('sample')).not.toBe(second)
  })
})
