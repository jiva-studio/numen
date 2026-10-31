/**
 * Creating a note from the window.
 *
 * The core numbers untitled names and decides where a related note is filed and
 * what it writes. The tests cover what the window sends and what it says about
 * the answer.
 */
import { describe, expect, it } from 'vitest'
import { asFailure, asValue, type RelatedNoteRequest, type RelationError } from '@numen/plugin-sdk'

import { createNoteWriter, type NoteMaker } from './maker'
import type { NewNote } from '@/entities/note'
import type { CreateResult } from '@/shared/model/fileTypes'
import { writer } from '@/testing/writer'

/** A core double that records the requests it receives and answers as the test configures. */
function fake(
  created: CreateResult = { ok: true, value: { path: 'Untitled note.md' } },
  related: { ok: true; value: string } | { ok: false; error: RelationError } = {
    ok: true,
    value: 'physics/Untitled note.md',
  },
) {
  const notes: NewNote[] = []
  const requests: RelatedNoteRequest[] = []
  const core: NoteMaker = {
    create: async (note) => {
      notes.push(note)
      return created
    },
    relations: {
      createRelated: async (request) => {
        requests.push(request)
        return related.ok ? asValue(related.value) : asFailure(related.error)
      },
    },
  }
  return { core, notes, requests, ...writer() }
}

describe('making an untitled note', () => {
  it('asks the core for a note with no title, in the folder', async () => {
    const { core, notes, write } = fake()

    expect(await createNoteWriter(core, write).createUntitled('physics')).toBe('Untitled note.md')
    expect(notes).toStrictEqual([{ title: '', folder: 'physics' }])
  })

  it('answers with no path and says why when the core made nothing', async () => {
    const { core, write, messages } = fake({ ok: false, error: 'alreadyExists' })

    expect(await createNoteWriter(core, write).createUntitled('')).toBe('')
    expect(messages.at(-1)?.kind).toBe('error')
    expect(messages.at(-1)?.text).not.toBe('')
  })

  it('says a core that could not be reached', async () => {
    const core: NoteMaker = {
      create: async () => {
        throw new Error('the vault is out of reach')
      },
      relations: { createRelated: async () => asFailure('unreachable') },
    }
    const { write, last, messages } = writer()

    expect(await createNoteWriter(core, write).createUntitled('')).toBe('')
    expect(last()).toContain('Numen is not responding')
    expect(last()).not.toContain('out of reach')
    expect(messages.at(-1)?.kind).toBe('error')
  })
})

describe('making a note under a name a person gave it', () => {
  it('files a note with no relation at the root under the title', async () => {
    const { core, notes, write } = fake({ ok: true, value: { path: 'Entropy.md' } })

    const made = await createNoteWriter(core, write).createWithTitle('Entropy', '', null)

    expect(made).toStrictEqual({ path: 'Entropy.md', title: 'Entropy' })
    expect(notes).toStrictEqual([{ title: 'Entropy', folder: '' }])
  })

  it('leaves the relation of a related note to the core', async () => {
    const { core, requests, notes, write, last } = fake(undefined, {
      ok: true,
      value: 'physics/Entropy.md',
    })

    const made = await createNoteWriter(core, write).createWithTitle(
      'Entropy',
      'physics/Ontology.md',
      'child',
    )

    expect(made).toStrictEqual({ path: 'physics/Entropy.md', title: 'Entropy' })
    expect(requests).toStrictEqual([
      { from: 'physics/Ontology.md', role: 'child', title: 'Entropy' },
    ])
    expect(notes).toStrictEqual([])
    expect(last()).toBe('')
  })

  it('says why the core made no related note', async () => {
    const { core, write, messages } = fake(undefined, { ok: false, error: 'cycle' })

    expect(
      await createNoteWriter(core, write).createWithTitle('Entropy', 'A.md', 'parent'),
    ).toBeNull()
    expect(messages.at(-1)).toStrictEqual({
      text: 'a note cannot sit above itself in the hierarchy',
      kind: 'error',
    })
  })

  it('says a related note the core could not be asked for', async () => {
    const core: NoteMaker = {
      create: async () => ({ ok: true, value: { path: '' } }),
      relations: {
        createRelated: async () => {
          throw new Error('the vault is out of reach')
        },
      },
    }
    const { write, last, messages } = writer()

    expect(
      await createNoteWriter(core, write).createWithTitle('Entropy', 'A.md', 'jump'),
    ).toBeNull()
    expect(last()).toContain('Numen is not responding')
    expect(messages.at(-1)?.kind).toBe('error')
  })
})
