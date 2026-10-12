/**
 * Making a note from the picture, and joining two that are already on it.
 *
 * What is checked here is what the vault cannot answer for: which seat the new
 * note writes about the one it was made from, where it is filed, and what
 * happens when the name is already taken.
 */
import { describe, expect, it } from 'vitest'

import { CREATABLE, UNTITLED, createNoteWriter, type NoteMaker } from './maker'
import type { Link, NewNote, Role } from '@/entities/note'
import type { CreateResult } from '@/entities/file'
import type { ErrorCode } from '@/shared/errors'
import { writer } from '@/testing/writer'

const getNotePath = (note: NewNote): string =>
  note.folder ? `${note.folder}/${note.title}.md` : `${note.title}.md`

/**
 * A core that keeps what it was asked to write and answers what a test told it
 * to, falling back on making the note.
 */
function fake(answers: CreateResult[] = [], errors: (ErrorCode | null)[] = []) {
  const asked: NewNote[] = []
  const joined: { path: string; link: Link }[] = []
  const core: NoteMaker = {
    create: async (note: NewNote): Promise<CreateResult> => {
      asked.push(note)
      return answers.shift() ?? { ok: true, value: { path: getNotePath(note) } }
    },
    join: async (path: string, link: Link): Promise<ErrorCode | null> => {
      joined.push({ path, link })
      return errors.shift() ?? null
    },
    removeLink: async () => null,
  }
  return { core, asked, joined, ...writer() }
}

describe('making a note in a seat of another', () => {
  it('writes the note it was made from into it, in the seat facing the one asked for', async () => {
    const { core, asked, write } = fake()
    const made = await createNoteWriter(core, write).createInSeat('Ontology.md', 'child')

    expect(made).toStrictEqual({ path: `${UNTITLED}.md`, title: UNTITLED })
    expect(asked).toStrictEqual([
      { title: UNTITLED, folder: '', links: [{ to: 'Ontology.md', role: 'parent' }] },
    ])
  })

  it('makes a parent the new note is the child of', async () => {
    const { core, asked, write } = fake()
    await createNoteWriter(core, write).createInSeat('Ontology.md', 'parent')

    expect(asked[0]?.links).toStrictEqual([{ to: 'Ontology.md', role: 'child' }])
  })

  it('files it in the folder the note it was made from is in', async () => {
    const { core, asked, write } = fake()
    const made = await createNoteWriter(core, write).createInSeat('physics/Ontology.md', 'child')

    expect(asked[0]?.folder).toBe('physics')
    expect(made?.path).toBe(`physics/${UNTITLED}.md`)
  })

  it('can make a note in every seat it offers', async () => {
    for (const seat of CREATABLE) {
      const { core, asked, write } = fake()
      expect(await createNoteWriter(core, write).createInSeat('Ontology.md', seat)).not.toBeNull()
      expect(asked[0]?.links).toHaveLength(1)
    }
  })

  it('makes nothing in a seat no link writes', async () => {
    const { core, asked, write } = fake()

    expect(await createNoteWriter(core, write).createInSeat('Ontology.md', 'sibling')).toBeNull()
    expect(asked).toStrictEqual([])
  })

  it('asks for the next name for as long as the vault says the last one is taken', async () => {
    const { core, asked, write, last } = fake([
      { ok: false, error: 'occupied' },
      { ok: false, error: 'occupied' },
    ])
    const making = createNoteWriter(core, write)
    const made = await making.createInSeat('Ontology.md', 'child')

    expect(asked.map((note) => note.title)).toStrictEqual([
      UNTITLED,
      `${UNTITLED} 2`,
      `${UNTITLED} 3`,
    ])
    expect(made).toStrictEqual({ path: `${UNTITLED} 3.md`, title: `${UNTITLED} 3` })
    expect(last()).toBe('')
  })

  it('creates an untitled note in a folder with links', async () => {
    const { core, asked, write } = fake()
    const made = await createNoteWriter(core, write).createUntitled('physics', [
      { to: 'Ontology.md', role: 'parent' },
    ])

    expect(made).toStrictEqual({ path: `physics/${UNTITLED}.md`, title: UNTITLED })
    expect(asked).toStrictEqual([
      { title: UNTITLED, folder: 'physics', links: [{ to: 'Ontology.md', role: 'parent' }] },
    ])
  })

  it('says an error that is not a name already taken, and asks for nothing more', async () => {
    const { core, asked, write, told } = fake([{ ok: false, error: 'notANote' }])
    const making = createNoteWriter(core, write)

    expect(await making.createInSeat('Ontology.md', 'child')).toBeNull()
    expect(asked).toHaveLength(1)
    expect(told.at(-1)?.text).not.toBe('')
    expect(told.at(-1)?.kind).toBe('error')
  })

  it('says a core that could not be reached', async () => {
    const core = {
      create: async () => {
        throw new Error('the vault is out of reach')
      },
    } as unknown as NoteMaker
    const { write, last, told } = writer()
    const making = createNoteWriter(core, write)

    expect(await making.createInSeat('Ontology.md', 'child')).toBeNull()
    expect(last()).toContain('numen did not answer')
    expect(last()).not.toContain('out of reach')
    expect(told.at(-1)?.kind).toBe('error')
  })
})

describe('a vault that has filed every name offered', () => {
  it('is asked for a hundred names, and then the person is told', async () => {
    const { core, asked, write, told } = fake(
      Array.from({ length: 101 }, () => ({ ok: false, error: 'occupied' }) as const),
    )

    expect(await createNoteWriter(core, write).createUntitled('', [])).toBeNull()

    expect(asked).toHaveLength(100)
    expect(asked.at(-1)?.title).toBe(`${UNTITLED} 100`)
    expect(told.at(-1)).toStrictEqual({
      text: `every name from ${UNTITLED} onwards is taken`,
      kind: 'error',
    })
  })
})

describe('making a note under a name a person gave it', () => {
  it('makes nothing in a seat no link writes', async () => {
    const { core, asked, write } = fake()

    expect(
      await createNoteWriter(core, write).createWithTitle('Entropy', 'Ontology.md', 'sibling'),
    ).toBeNull()
    expect(asked).toStrictEqual([])
  })

  it('files it beside the note it was made from, in that note’s seat', async () => {
    const { core, asked, write } = fake()
    const made = await createNoteWriter(core, write).createWithTitle(
      'Entropy',
      'physics/Ontology.md',
      'child',
    )

    expect(made).toStrictEqual({ path: 'physics/Entropy.md', title: 'Entropy' })
    expect(asked).toStrictEqual([
      {
        title: 'Entropy',
        folder: 'physics',
        links: [{ to: 'physics/Ontology.md', role: 'parent' }],
      },
    ])
  })

  it('files it at the top of the vault, joined to nothing, when it stands on no note', async () => {
    const { core, asked, write } = fake()
    const made = await createNoteWriter(core, write).createWithTitle('Entropy', '', null)

    expect(made).toStrictEqual({ path: 'Entropy.md', title: 'Entropy' })
    expect(asked).toStrictEqual([{ title: 'Entropy', folder: '', links: [] }])
  })
})

describe('joining two notes that are both there', () => {
  it('writes the link in the note the gesture came from, in the seat it landed in', async () => {
    const { core, joined, write, last } = fake()
    const making = createNoteWriter(core, write)

    expect(await making.join('Ontology.md', 'Entropy.md', 'child')).toBe(true)
    expect(joined).toStrictEqual([
      { path: 'Ontology.md', link: { to: 'Entropy.md', role: 'child' } },
    ])
    expect(last()).toBe('')
  })

  it('writes the label it was given into the link', async () => {
    const { core, joined, write } = fake()

    await createNoteWriter(core, write).join('Ontology.md', 'Entropy.md', 'jump', 'see also')

    expect(joined).toStrictEqual([
      { path: 'Ontology.md', link: { to: 'Entropy.md', role: 'jump', label: 'see also' } },
    ])
  })

  it('writes nothing for a seat no link writes', async () => {
    const { core, joined, write } = fake()

    expect(await createNoteWriter(core, write).join('Ontology.md', 'Entropy.md', 'sibling')).toBe(
      false,
    )
    expect(joined).toStrictEqual([])
  })

  it('says why nothing was written', async () => {
    const { core, write, told } = fake([], ['unreadable'])
    const making = createNoteWriter(core, write)

    expect(await making.join('Ontology.md', 'Entropy.md', 'jump')).toBe(false)
    expect(told.at(-1)?.text).not.toBe('')
    expect(told.at(-1)?.kind).toBe('error')
  })
})

describe('removing a relationship from a note', () => {
  /** A core whose removal answers as told and keeps what it was asked. */
  const removing = (answer: () => Promise<ErrorCode | null>) => {
    const asked: { path: string; to: string; role: Role | undefined }[] = []
    const core = {
      removeLink: async (path: string, to: string, role?: Role) => {
        asked.push({ path, to, role })
        return answer()
      },
    } as unknown as NoteMaker
    return { core, asked, ...writer() }
  }

  it('asks for the role of the seat it was given, and says nothing when it is done', async () => {
    const { core, asked, write, told } = removing(async () => null)

    expect(
      await createNoteWriter(core, write).removeLink('Ontology.md', 'Entropy.md', 'child'),
    ).toBe(true)
    expect(asked).toStrictEqual([{ path: 'Ontology.md', to: 'Entropy.md', role: 'child' }])
    expect(told).toStrictEqual([{ text: '', kind: 'report' }])
  })

  it('asks for no role where no seat was given', async () => {
    const { core, asked, write } = removing(async () => null)

    await createNoteWriter(core, write).removeLink('Ontology.md', 'Entropy.md')

    expect(asked[0]?.role).toBeUndefined()
  })

  it('says why nothing was removed', async () => {
    const { core, write, told } = removing(async () => 'unreadable')

    expect(await createNoteWriter(core, write).removeLink('Ontology.md', 'Entropy.md')).toBe(false)
    expect(told.at(-1)?.text).not.toBe('')
    expect(told.at(-1)?.kind).toBe('error')
  })

  it('says a core that could not be reached', async () => {
    const { core, write, told } = removing(async () => {
      throw new Error('the vault is out of reach')
    })

    expect(await createNoteWriter(core, write).removeLink('Ontology.md', 'Entropy.md')).toBe(false)
    expect(told.at(-1)?.text).toContain('numen did not answer')
    expect(told.at(-1)?.kind).toBe('error')
  })
})

describe('joining through a core that could not be reached', () => {
  it('says so and writes nothing', async () => {
    const core = {
      join: async () => {
        throw new Error('the vault is out of reach')
      },
    } as unknown as NoteMaker
    const { write, told } = writer()

    expect(await createNoteWriter(core, write).join('Ontology.md', 'Entropy.md', 'child')).toBe(
      false,
    )
    expect(told.at(-1)?.text).toContain('numen did not answer')
    expect(told.at(-1)?.kind).toBe('error')
  })
})
