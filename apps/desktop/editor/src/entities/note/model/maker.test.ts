/**
 * Making a note from the picture, and joining two that are already on it.
 *
 * What is checked here is what the vault cannot answer for: which relation the new
 * note writes about the one it was made from, where it is filed, and what
 * happens when the name is already taken.
 */
import { describe, expect, it } from 'vitest'

import { CREATABLE, UNTITLED, createNoteWriter, type NoteMaker } from './maker'
import type { Link, NewNote, Role } from '@/entities/note'
import type { CreateResult } from '@/entities/file/@x/note'
import type { ErrorCode } from '@/shared/errors'
import { writer } from '@/testing/writer'

const getNotePath = (note: NewNote): string =>
  note.folder ? `${note.folder}/${note.title}.md` : `${note.title}.md`

/**
 * A core that keeps what it was asked to write and answers what a test told it
 * to, falling back on making the note.
 */
function fake(answers: CreateResult[] = [], errors: (ErrorCode | null)[] = []) {
  const request: NewNote[] = []
  const joined: { path: string; link: Link }[] = []
  const core: NoteMaker = {
    create: async (note: NewNote): Promise<CreateResult> => {
      request.push(note)
      return answers.shift() ?? { ok: true, value: { path: getNotePath(note) } }
    },
    join: async (path: string, link: Link): Promise<ErrorCode | null> => {
      joined.push({ path, link })
      return errors.shift() ?? null
    },
    removeLink: async () => null,
    replaceLinks: async () => null,
  }
  return { core, requests: request, joined, ...writer() }
}

describe('making a note in a relation of another', () => {
  it('writes the note it was made from into it, in the relation facing the one requested for', async () => {
    const { core, requests, write } = fake()
    const createdItem = await createNoteWriter(core, write).createInRelation('Ontology.md', 'child')

    expect(createdItem).toStrictEqual({ path: `${UNTITLED}.md`, title: UNTITLED })
    expect(requests).toStrictEqual([
      { title: UNTITLED, folder: '', links: [{ to: 'Ontology.md', role: 'parent' }] },
    ])
  })

  it('makes a parent the new note is the child of', async () => {
    const { core, requests, write } = fake()
    await createNoteWriter(core, write).createInRelation('Ontology.md', 'parent')

    expect(requests[0]?.links).toStrictEqual([{ to: 'Ontology.md', role: 'child' }])
  })

  it('files it in the folder the note it was made from is in', async () => {
    const { core, requests, write } = fake()
    const createdItem = await createNoteWriter(core, write).createInRelation(
      'physics/Ontology.md',
      'child',
    )

    expect(requests[0]?.folder).toBe('physics')
    expect(createdItem?.path).toBe(`physics/${UNTITLED}.md`)
  })

  it('can make a note in every relation it offers', async () => {
    for (const relation of CREATABLE) {
      const { core, requests, write } = fake()
      expect(
        await createNoteWriter(core, write).createInRelation('Ontology.md', relation),
      ).not.toBeNull()
      expect(requests[0]?.links).toHaveLength(1)
    }
  })

  it('makes nothing in a relation no link writes', async () => {
    const { core, requests, write } = fake()

    expect(
      await createNoteWriter(core, write).createInRelation('Ontology.md', 'sibling'),
    ).toBeNull()
    expect(requests).toStrictEqual([])
  })

  it('asks for the next name for as long as the vault says the last one is taken', async () => {
    const { core, requests, write, last } = fake([
      { ok: false, error: 'occupied' },
      { ok: false, error: 'occupied' },
    ])
    const making = createNoteWriter(core, write)
    const createdItem = await making.createInRelation('Ontology.md', 'child')

    expect(requests.map((note) => note.title)).toStrictEqual([
      UNTITLED,
      `${UNTITLED} 2`,
      `${UNTITLED} 3`,
    ])
    expect(createdItem).toStrictEqual({ path: `${UNTITLED} 3.md`, title: `${UNTITLED} 3` })
    expect(last()).toBe('')
  })

  it('creates an untitled note in a folder with links', async () => {
    const { core, requests, write } = fake()
    const createdItem = await createNoteWriter(core, write).createUntitled('physics', [
      { to: 'Ontology.md', role: 'parent' },
    ])

    expect(createdItem).toStrictEqual({ path: `physics/${UNTITLED}.md`, title: UNTITLED })
    expect(requests).toStrictEqual([
      { title: UNTITLED, folder: 'physics', links: [{ to: 'Ontology.md', role: 'parent' }] },
    ])
  })

  it('says an error that is not a name already taken, and asks for nothing more', async () => {
    const { core, requests, write, messages } = fake([{ ok: false, error: 'notANote' }])
    const making = createNoteWriter(core, write)

    expect(await making.createInRelation('Ontology.md', 'child')).toBeNull()
    expect(requests).toHaveLength(1)
    expect(messages.at(-1)?.text).not.toBe('')
    expect(messages.at(-1)?.kind).toBe('error')
  })

  it('says a core that could not be reached', async () => {
    const core = {
      create: async () => {
        throw new Error('the vault is out of reach')
      },
    } as unknown as NoteMaker
    const { write, last, messages } = writer()
    const making = createNoteWriter(core, write)

    expect(await making.createInRelation('Ontology.md', 'child')).toBeNull()
    expect(last()).toContain('Numen is not responding')
    expect(last()).not.toContain('out of reach')
    expect(messages.at(-1)?.kind).toBe('error')
  })
})

describe('a vault that has filed every name offered', () => {
  it('requests a hundred names and reports status', async () => {
    const { core, requests, write, messages } = fake(
      Array.from({ length: 101 }, () => ({ ok: false, error: 'occupied' }) as const),
    )

    expect(await createNoteWriter(core, write).createUntitled('', [])).toBeNull()

    expect(requests).toHaveLength(100)
    expect(requests.at(-1)?.title).toBe(`${UNTITLED} 100`)
    expect(messages.at(-1)).toStrictEqual({
      text: `every name from ${UNTITLED} onwards is taken`,
      kind: 'error',
    })
  })
})

describe('making a note under a name a person gave it', () => {
  it('makes nothing in a relation no link writes', async () => {
    const { core, requests, write } = fake()

    expect(
      await createNoteWriter(core, write).createWithTitle('Entropy', 'Ontology.md', 'sibling'),
    ).toBeNull()
    expect(requests).toStrictEqual([])
  })

  it('files it alongside the note it was made from, in that note’s relation', async () => {
    const { core, requests, write } = fake()
    const createdItem = await createNoteWriter(core, write).createWithTitle(
      'Entropy',
      'physics/Ontology.md',
      'child',
    )

    expect(createdItem).toStrictEqual({ path: 'physics/Entropy.md', title: 'Entropy' })
    expect(requests).toStrictEqual([
      {
        title: 'Entropy',
        folder: 'physics',
        links: [{ to: 'physics/Ontology.md', role: 'parent' }],
      },
    ])
  })

  it('files it at the top of the vault, joined to nothing, when it relies on no note', async () => {
    const { core, requests, write } = fake()
    const createdItem = await createNoteWriter(core, write).createWithTitle('Entropy', '', null)

    expect(createdItem).toStrictEqual({ path: 'Entropy.md', title: 'Entropy' })
    expect(requests).toStrictEqual([{ title: 'Entropy', folder: '', links: [] }])
  })
})

describe('joining two notes that are both there', () => {
  it('writes the link in the note the gesture came from, in the relation it landed in', async () => {
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

  it('writes nothing for a relation no link writes', async () => {
    const { core, joined, write } = fake()

    expect(await createNoteWriter(core, write).join('Ontology.md', 'Entropy.md', 'sibling')).toBe(
      false,
    )
    expect(joined).toStrictEqual([])
  })

  it('says why nothing was written', async () => {
    const { core, write, messages } = fake([], ['unreadable'])
    const making = createNoteWriter(core, write)

    expect(await making.join('Ontology.md', 'Entropy.md', 'jump')).toBe(false)
    expect(messages.at(-1)?.text).not.toBe('')
    expect(messages.at(-1)?.kind).toBe('error')
  })
})

describe('removing a relationship from a note', () => {
  /** A core whose removal answers as told and keeps what it was asked. */
  const removing = (answer: () => Promise<ErrorCode | null>) => {
    const request: { path: string; to: string; role: Role | undefined }[] = []
    const core = {
      removeLink: async (path: string, to: string, role?: Role) => {
        request.push({ path, to, role })
        return answer()
      },
    } as unknown as NoteMaker
    return { core, requests: request, ...writer() }
  }

  it('asks for the role of the relation it was given, and says nothing when it is done', async () => {
    const { core, requests, write, messages } = removing(async () => null)

    expect(
      await createNoteWriter(core, write).removeLink('Ontology.md', 'Entropy.md', 'child'),
    ).toBe(true)
    expect(requests).toStrictEqual([{ path: 'Ontology.md', to: 'Entropy.md', role: 'child' }])
    expect(messages).toStrictEqual([{ text: '', kind: 'info' }])
  })

  it('asks for no role where no relation was given', async () => {
    const { core, requests, write } = removing(async () => null)

    await createNoteWriter(core, write).removeLink('Ontology.md', 'Entropy.md')

    expect(requests[0]?.role).toBeUndefined()
  })

  it('says why nothing was removed', async () => {
    const { core, write, messages } = removing(async () => 'unreadable')

    expect(await createNoteWriter(core, write).removeLink('Ontology.md', 'Entropy.md')).toBe(false)
    expect(messages.at(-1)?.text).not.toBe('')
    expect(messages.at(-1)?.kind).toBe('error')
  })

  it('says a core that could not be reached', async () => {
    const { core, write, messages } = removing(async () => {
      throw new Error('the vault is out of reach')
    })

    expect(await createNoteWriter(core, write).removeLink('Ontology.md', 'Entropy.md')).toBe(false)
    expect(messages.at(-1)?.text).toContain('Numen is not responding')
    expect(messages.at(-1)?.kind).toBe('error')
  })
})

describe('joining through a core that could not be reached', () => {
  it('says so and writes nothing', async () => {
    const core = {
      join: async () => {
        throw new Error('the vault is out of reach')
      },
    } as unknown as NoteMaker
    const { write, messages } = writer()

    expect(await createNoteWriter(core, write).join('Ontology.md', 'Entropy.md', 'child')).toBe(
      false,
    )
    expect(messages.at(-1)?.text).toContain('Numen is not responding')
    expect(messages.at(-1)?.kind).toBe('error')
  })
})

describe('replacing the links between two notes', () => {
  const links = [
    { from: 'Ontology.md', to: 'Entropy.md', relation: 'child' as const, label: 'owns' },
    { from: 'Entropy.md', to: 'Ontology.md', relation: 'parent' as const, label: '' },
  ]

  /** A core whose replacement answers as told and keeps what it was asked. */
  const replacing = (answer: () => Promise<ErrorCode | null>) => {
    const request: { first: string; second: string; roles: Role[] }[] = []
    const core = {
      replaceLinks: async (first: string, second: string, pairs: readonly { role: Role }[]) => {
        request.push({ first, second, roles: pairs.map((pair) => pair.role) })
        return answer()
      },
    } as unknown as NoteMaker
    return { core, requests: request, ...writer() }
  }

  it('asks once for every link, under the role of its relation, and says nothing when done', async () => {
    const { core, requests, write, messages } = replacing(async () => null)

    expect(
      await createNoteWriter(core, write).replaceLinks('Ontology.md', 'Entropy.md', links),
    ).toBe(true)
    expect(requests).toStrictEqual([
      { first: 'Ontology.md', second: 'Entropy.md', roles: ['child', 'parent'] },
    ])
    expect(messages).toStrictEqual([{ text: '', kind: 'info' }])
  })

  it('asks for nothing where a relation is not one a link can carry', async () => {
    const { core, requests, write } = replacing(async () => null)
    const sibling = [{ ...links[0]!, relation: 'sibling' as const }]

    expect(
      await createNoteWriter(core, write).replaceLinks('Ontology.md', 'Entropy.md', sibling),
    ).toBe(false)
    expect(requests).toStrictEqual([])
  })

  it('says why the core replaced nothing', async () => {
    const { core, write, messages } = replacing(async () => 'unreadable')

    expect(
      await createNoteWriter(core, write).replaceLinks('Ontology.md', 'Entropy.md', links),
    ).toBe(false)
    expect(messages.at(-1)?.text).not.toBe('')
    expect(messages.at(-1)?.kind).toBe('error')
  })

  it('says a core that could not be reached', async () => {
    const { core, write, messages } = replacing(async () => {
      throw new Error('the vault is out of reach')
    })

    expect(
      await createNoteWriter(core, write).replaceLinks('Ontology.md', 'Entropy.md', links),
    ).toBe(false)
    expect(messages.at(-1)?.text).toContain('Numen is not responding')
  })
})
