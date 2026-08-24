/**
 * Carrying a command out, asked without a window.
 *
 * Two things are asked here that nothing else can ask: that a note is settled
 * before its file is renamed or removed, and that what a rename and a remove
 * leave behind is put to the person, because nothing repairs it and nothing
 * undoes it.
 */
import { describe, expect, it } from 'vitest'
import { commandsOf, deedOf, type Deed, type Where } from './commanding'
import { does, type Doing } from './doing'
import type { Removed, Renamed } from './core'
import { WORDS as words } from './words'

/** What is in front, which every deed is carried out over. */
const front = (over: Partial<Where> = {}): Where => ({
  tab: 'tab',
  kind: 'plex',
  path: 'physics/Ontology.md',
  title: 'Ontology',
  ready: true,
  ...over,
})

const renamed = (over: Partial<Renamed> = {}): Renamed => ({
  path: 'physics/Entropy.md',
  title: 'Entropy',
  by: 'frontmatter',
  moved: null,
  refusal: null,
  ...over,
})

const removed = (over: Partial<Removed> = {}): Removed => ({
  trashed: '.trash/Ontology.md',
  dangling: [],
  refusal: null,
  ...over,
})

/**
 * A window that writes down everything a command asked of it, in order.
 *
 * The note the window holds stands at a file the test can move, so a deed made
 * before it moved can be carried out after.
 */
const window = (
  answers: {
    renamed?: Renamed
    removed?: Removed
    made?: boolean
    /** Where the tab holding the note stands now. */
    at?: string
    /** The note is waiting on the person, so nothing may move its file. */
    asking?: boolean
  } = {},
) => {
  const done: string[] = []
  const said: string[] = []
  const at = answers.at ?? 'physics/Ontology.md'
  const on: Doing = {
    makes: async (title, from, seat) => {
      done.push(`makes ${title} ${from || '—'} ${seat ?? '—'}`)
      return answers.made === false ? null : { path: `${title}.md`, title }
    },
    renames: async (path, title) => {
      done.push(`renames ${path} ${title}`)
      return answers.renamed ?? renamed()
    },
    removes: async (path, destroy) => {
      done.push(`removes ${path} ${destroy}`)
      return answers.removed ?? removed()
    },
    notes: {
      holding: (path) => (path === at ? 'held' : null),
      where: (id) => (id === 'held' ? at : id),
      asking: () => answers.asking === true,
      settles: async (id) => void done.push(`settles ${id}`),
      shuts: (id) => void done.push(`shuts ${id}`),
      shows: (path, title, showing) => void done.push(`shows ${path} ${title} ${showing}`),
    },
    travel: async (path) => void done.push(`travel ${path}`),
    leaves: async (from, to) => void done.push(`leaves ${from} ${to}`),
    opening: () => 'Root.md',
    opens: (kind) => void done.push(`opens ${kind}`),
    closes: (tab) => void done.push(`closes ${tab}`),
    asks: (text) => void done.push(`asks ${text}`),
    copies: (path) => void done.push(`copies ${path}`),
    searches: () => void done.push('searches'),
    says: (text) => void (text ? said.push(text) : undefined),
  }
  return { on, done, said }
}

/** One command carried out over the note in front. */
const carry = async (deed: Deed, on: Doing) => does(deed, on, words)

describe('every command that is offered', () => {
  it('is carried out by something', async () => {
    for (const command of commandsOf(words)) {
      const one = window()
      await carry(deedOf(command.id, front(), 'Entropy'), one.on)

      expect(one.done, command.id).not.toStrictEqual([])
    }
  })
})

describe('a note put in front of the person', () => {
  it('opens in a tab of its own, and beside it where the second key reached it', async () => {
    const one = window()

    await carry(deedOf('read', front()), one.on)
    await carry(deedOf('beside', front()), one.on)

    expect(one.done).toStrictEqual([
      'shows physics/Ontology.md Ontology here',
      'shows physics/Ontology.md Ontology beside',
    ])
  })

  it('is travelled to in the plex, where that is what was asked', async () => {
    const one = window()

    await carry(deedOf('travel', front()), one.on)

    expect(one.done).toStrictEqual(['travel physics/Ontology.md'])
  })
})

describe('a note made', () => {
  it('writes the note it was made from into it, in the seat that was asked for', async () => {
    const one = window()

    await carry(deedOf('child', front(), 'Entropy'), one.on)

    expect(one.done[0]).toBe('makes Entropy physics/Ontology.md child')
  })

  it('stands on its own where no seat was asked for', async () => {
    const one = window()

    await carry(deedOf('note', front(), 'Entropy'), one.on)

    expect(one.done[0]).toBe('makes Entropy — —')
  })

  it('is travelled to in the plex the person is looking at', async () => {
    const one = window()

    await carry(deedOf('child', front(), 'Entropy'), one.on)

    expect(one.done.at(-1)).toBe('travel Entropy.md')
  })

  it('opens in a tab beside the note the person is in', async () => {
    const one = window()

    await carry(deedOf('child', front({ kind: 'note' }), 'Entropy'), one.on)

    expect(one.done.at(-1)).toBe('shows Entropy.md Entropy beside')
  })

  it('takes the person nowhere where the vault would not make it', async () => {
    const one = window({ made: false })

    await carry(deedOf('child', front(), 'Entropy'), one.on)

    expect(one.done).toStrictEqual(['makes Entropy physics/Ontology.md child'])
  })

  it('is nothing at all where nothing was typed', async () => {
    const one = window()

    await carry(deedOf('child', front(), ''), one.on)

    expect(one.done).toStrictEqual([])
  })
})

describe('a note renamed', () => {
  it('has nothing on its way to its file before the file moves', async () => {
    const one = window()

    await carry(deedOf('title', front(), 'Entropy'), one.on)

    expect(one.done).toStrictEqual(['settles held', 'renames physics/Ontology.md Entropy'])
  })

  it('is refused while the note is waiting on the person', async () => {
    const one = window({ asking: true })

    await carry(deedOf('title', front(), 'Entropy'), one.on)

    expect(one.done).toStrictEqual([])
    expect(one.said).toStrictEqual([words.unanswered])
  })

  it('is renamed where no tab of the window holds it', async () => {
    const one = window()

    await carry(deedOf('title', front({ path: 'Elsewhere.md' }), 'Entropy'), one.on)

    expect(one.done).toStrictEqual(['renames Elsewhere.md Entropy'])
  })

  it('is left alone where the name it was given is the name it has', async () => {
    const one = window()

    await carry(deedOf('title', front(), 'Ontology'), one.on)

    expect(one.done).toStrictEqual([])
  })

  it('says the links that mean another note now, which nothing repairs', async () => {
    const one = window({
      renamed: renamed({
        moved: {
          from: 'physics/Ontology.md',
          to: 'physics/Entropy.md',
          repaired: ['Notes.md'],
          retargeted: [
            { in: 'Order.md', target: 'Entropy', now: 'other/Entropy.md' },
            { in: 'Order.md', target: 'Entropy', now: 'other/Entropy.md' },
          ],
        },
      }),
    })

    await carry(deedOf('title', front(), 'Entropy'), one.on)

    expect(one.said).toStrictEqual([`${words.retargeted} Order.md`])
  })

  it('says nothing where the rename left every link meaning what it meant', async () => {
    const one = window()

    await carry(deedOf('title', front(), 'Entropy'), one.on)

    expect(one.said).toStrictEqual([])
  })

  it('says the name was taken, and that the note carries the new one', async () => {
    const one = window({ renamed: renamed({ refusal: 'occupied' }) })

    await carry(deedOf('title', front(), 'Entropy'), one.on)

    expect(one.said).toStrictEqual([words.refused.occupied])
  })

  it('says a note whose frontmatter cannot be read cannot be renamed', async () => {
    const one = window({ renamed: renamed({ refusal: 'unreadable' }) })

    await carry(deedOf('title', front(), 'Entropy'), one.on)

    expect(one.said).toStrictEqual([words.refused.unreadable])
  })

  it('says a name no file can be named', async () => {
    const one = window({ renamed: renamed({ refusal: 'unnameable' }) })

    await carry(deedOf('title', front(), '...'), one.on)

    expect(one.said).toStrictEqual([words.refused.unnameable])
  })
})

describe('a note removed', () => {
  it('has nothing on its way to its file before the file goes', async () => {
    const one = window()

    await carry(deedOf('remove', front()), one.on)

    expect(one.done.slice(0, 2)).toStrictEqual(['settles held', 'removes physics/Ontology.md false'])
  })

  it('goes off the disk where destroying was what was asked', async () => {
    const one = window()

    await carry(deedOf('destroy', front(), 'Ontology'), one.on)

    expect(one.done[1]).toBe('removes physics/Ontology.md true')
  })

  it('says the notes that link to nothing now, which nothing repairs', async () => {
    const one = window({ removed: removed({ dangling: ['Order.md', 'Notes.md'] }) })

    await carry(deedOf('remove', front()), one.on)

    expect(one.said).toStrictEqual([`${words.dangling} Order.md, Notes.md`])
  })

  it('leaves every plex standing on it at the note the vault opens with', async () => {
    const one = window()

    await carry(deedOf('remove', front()), one.on)

    expect(one.done.at(-1)).toBe('leaves physics/Ontology.md Root.md')
  })

  it('leaves the plexes alone where the vault opens with no note at all', async () => {
    const one = window()
    const nowhere: Doing = { ...one.on, opening: () => '' }

    await carry(deedOf('remove', front()), nowhere)

    expect(one.done.some((step) => step.startsWith('leaves'))).toBe(false)
  })

  it('lets go of the tab that was reading it', async () => {
    const one = window()

    await carry(deedOf('remove', front(), '', 'held'), one.on)

    expect(one.done).toContain('shuts held')
  })

  it('keeps the tab of a note the vault would not remove', async () => {
    const one = window({ removed: removed({ refusal: 'missing' }) })

    await carry(deedOf('remove', front(), '', 'held'), one.on)

    expect(one.done).not.toContain('shuts held')
  })

  it('says a note that is not in the vault, and takes the plex nowhere', async () => {
    const one = window({ removed: removed({ refusal: 'missing' }) })

    await carry(deedOf('remove', front()), one.on)

    expect(one.said).toStrictEqual([words.refused.missing])
    expect(one.done.at(-1)).toBe('removes physics/Ontology.md false')
  })

  it('is refused while the note is waiting on the person', async () => {
    const one = window({ asking: true })

    await carry(deedOf('remove', front()), one.on)

    expect(one.done).toStrictEqual([])
    expect(one.said).toStrictEqual([words.unanswered])
  })
})

/**
 * A deed is made when a person answers and carried out a moment later, and the
 * vault moves in between. The tab holding the note is what says where it is.
 */
describe('a note that moved between the answer and the deed', () => {
  it('is renamed where it stands now, not at the name the deed was made over', async () => {
    const one = window({ at: 'physics/Being.md' })

    await carry(deedOf('title', front(), 'Substance', 'held'), one.on)

    expect(one.done).toStrictEqual(['settles held', 'renames physics/Being.md Substance'])
  })

  it('is removed where it stands now', async () => {
    const one = window({ at: 'physics/Being.md' })

    await carry(deedOf('remove', front(), '', 'held'), one.on)

    expect(one.done.slice(0, 2)).toStrictEqual(['settles held', 'removes physics/Being.md false'])
  })

  it('is left at the name it was made over where no tab holds it', async () => {
    const one = window({ at: 'physics/Being.md' })

    await carry(deedOf('remove', front()), one.on)

    expect(one.done[0]).toBe('removes physics/Ontology.md false')
  })
})

describe('a command over the window', () => {
  it('opens a tab of the kind asked for, and closes the one in front', async () => {
    const one = window()

    await carry(deedOf('plex', front()), one.on)
    await carry(deedOf('agent', front()), one.on)
    await carry(deedOf('close', front()), one.on)

    expect(one.done).toStrictEqual(['opens plex', 'opens agent', 'closes tab'])
  })

  it('hands the field back to the search', async () => {
    const one = window()

    await carry(deedOf('find', front()), one.on)

    expect(one.done).toStrictEqual(['searches'])
  })
})

describe('a command over the vault', () => {
  it('travels to the note the vault opens with', async () => {
    const one = window()

    await carry(deedOf('first', front()), one.on)

    expect(one.done).toStrictEqual(['travel Root.md'])
  })

  it('says a vault that opens with no note at all', async () => {
    const one = window()
    const empty: Doing = { ...one.on, opening: () => '' }

    await does(deedOf('first', front()), empty, words)

    expect(one.said).toStrictEqual([words.nowhere])
  })
})

describe('what the window is asked about a note', () => {
  it('is put to the agent, and its path put on the clipboard', async () => {
    const one = window()

    await carry(deedOf('ask', front()), one.on)
    await carry(deedOf('copy', front()), one.on)

    expect(one.done).toStrictEqual(['asks physics/Ontology.md — ', 'copies physics/Ontology.md'])
  })
})

describe('nothing to carry out', () => {
  it('does nothing at all', async () => {
    const one = window()

    await does(null, one.on, words)
    await carry(deedOf('constructor', front()), one.on)
    await carry(deedOf('', front()), one.on)

    expect(one.done).toStrictEqual([])
  })

  it('says what the vault could not be asked, and asks no further', async () => {
    const one = window()
    const broken: Doing = { ...one.on, removes: async () => Promise.reject(new Error('gone')) }

    await does(deedOf('remove', front()), broken, words)

    expect(one.said).toStrictEqual(['Error: gone'])
  })
})
