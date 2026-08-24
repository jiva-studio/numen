/**
 * Making a note from the picture, and joining two that are already on it.
 *
 * What is checked here is what the vault cannot answer for: which seat the new
 * note writes about the one it was made from, where it is filed, and what
 * happens when the name is already taken.
 */
import { describe, expect, it } from 'vitest'

import { CREATABLE, UNTITLED, creating } from './creating'
import type { Core, Made, NewLink, NewNote } from './core'

const pathOf = (note: NewNote): string =>
  note.folder ? `${note.folder}/${note.title}.md` : `${note.title}.md`

/**
 * A core that keeps what it was asked to write and answers what a test told it
 * to, falling back on making the note.
 */
function fake(answers: Made[] = [], refusals: Made['refusal'][] = []) {
  const asked: NewNote[] = []
  const joined: { path: string; link: NewLink }[] = []
  const core = {
    create: async (note: NewNote): Promise<Made> => {
      asked.push(note)
      return answers.shift() ?? { path: pathOf(note), refusal: null }
    },
    join: async (path: string, link: NewLink): Promise<Made['refusal']> => {
      joined.push({ path, link })
      return refusals.shift() ?? null
    },
  } as unknown as Core
  return { core, asked, joined }
}

describe('making a note in a seat of another', () => {
  it('writes the note it was made from into it, in the seat facing the one asked for', async () => {
    const { core, asked } = fake()
    const made = await creating(core).make('Ontology.md', 'child')

    expect(made).toStrictEqual({ path: `${UNTITLED}.md`, title: UNTITLED })
    expect(asked).toStrictEqual([
      { title: UNTITLED, folder: '', links: [{ to: 'Ontology.md', seat: 'parent' }] },
    ])
  })

  it('makes a parent the new note is the child of', async () => {
    const { core, asked } = fake()
    await creating(core).make('Ontology.md', 'parent')

    expect(asked[0]?.links).toStrictEqual([{ to: 'Ontology.md', seat: 'child' }])
  })

  it('files it in the folder the note it was made from is in', async () => {
    const { core, asked } = fake()
    const made = await creating(core).make('physics/Ontology.md', 'child')

    expect(asked[0]?.folder).toBe('physics')
    expect(made?.path).toBe(`physics/${UNTITLED}.md`)
  })

  it('can make a note in every seat it offers', async () => {
    for (const seat of CREATABLE) {
      const { core, asked } = fake()
      expect(await creating(core).make('Ontology.md', seat)).not.toBeNull()
      expect(asked[0]?.links).toHaveLength(1)
    }
  })

  it('makes nothing in a seat no link writes', async () => {
    const { core, asked } = fake()

    expect(await creating(core).make('Ontology.md', 'sibling')).toBeNull()
    expect(asked).toStrictEqual([])
  })

  it('asks for the next name for as long as the vault says the last one is taken', async () => {
    const { core, asked } = fake([
      { path: '', refusal: 'occupied' },
      { path: '', refusal: 'occupied' },
    ])
    const making = creating(core)
    const made = await making.make('Ontology.md', 'child')

    expect(asked.map((note) => note.title)).toStrictEqual([
      UNTITLED,
      `${UNTITLED} 2`,
      `${UNTITLED} 3`,
    ])
    expect(made).toStrictEqual({ path: `${UNTITLED} 3.md`, title: `${UNTITLED} 3` })
    expect(making.said.value).toBe('')
  })

  it('says a refusal that is not a name already taken, and asks for nothing more', async () => {
    const { core, asked } = fake([{ path: '', refusal: 'notANote' }])
    const making = creating(core)

    expect(await making.make('Ontology.md', 'child')).toBeNull()
    expect(asked).toHaveLength(1)
    expect(making.said.value).not.toBe('')
  })

  it('says a core that could not be reached', async () => {
    const core = {
      create: async () => {
        throw new Error('the vault is out of reach')
      },
    } as unknown as Core
    const making = creating(core)

    expect(await making.make('Ontology.md', 'child')).toBeNull()
    expect(making.said.value).toContain('out of reach')
  })
})

describe('a note made on its own', () => {
  it('is filed at the top of the vault, joined to nothing', async () => {
    const { core, asked } = fake()
    const made = await creating(core).start()

    expect(made).toStrictEqual({ path: `${UNTITLED}.md`, title: UNTITLED })
    expect(asked).toStrictEqual([{ title: UNTITLED, folder: '', links: [] }])
  })

  it('takes the next free name, as a note made in a seat does', async () => {
    const { core } = fake([{ path: '', refusal: 'occupied' }])

    expect(await creating(core).start()).toStrictEqual({
      path: `${UNTITLED} 2.md`,
      title: `${UNTITLED} 2`,
    })
  })

  it('is nothing when the vault refused, and the refusal is said', async () => {
    const { core } = fake([{ path: '', refusal: 'unreadable' }])
    const making = creating(core)

    expect(await making.start()).toBeNull()
    expect(making.said.value).not.toBe('')
  })
})

describe('joining two notes that are both there', () => {
  it('writes the link in the note the gesture came from, in the seat it landed in', async () => {
    const { core, joined } = fake()
    const making = creating(core)

    expect(await making.join('Ontology.md', 'Entropy.md', 'child')).toBe(true)
    expect(joined).toStrictEqual([
      { path: 'Ontology.md', link: { to: 'Entropy.md', seat: 'child' } },
    ])
    expect(making.said.value).toBe('')
  })

  it('says why nothing was written', async () => {
    const { core } = fake([], ['unreadable'])
    const making = creating(core)

    expect(await making.join('Ontology.md', 'Entropy.md', 'jump')).toBe(false)
    expect(making.said.value).not.toBe('')
  })
})
