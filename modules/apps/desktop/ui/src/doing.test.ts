/**
 * Carrying a command out, asked without a window.
 *
 * Two things are asked here: that a note is settled before its file is renamed
 * or removed, and that what a rename and a remove leave behind is put to the
 * person.
 */
import { describe, expect, it } from 'vitest'
import { commandsOf, deedOf, type Deed, type Where } from './commanding'
import { does, type Doing } from './doing'
import type { Added, Known, Movement, Refused, Removed, Renamed, VaultRefused } from './core'
import { WORDS as words } from './words'

/** What is in front, which every deed is carried out over. */
const front = (over: Partial<Where> = {}): Where => ({
  tab: 'tab',
  kind: 'plex',
  path: 'physics/Ontology.md',
  title: 'Ontology',
  vault: { id: 'physics', name: 'Physics' },
  ready: true,
  ...over,
})

/** One vault as the list answers one. */
const known = (id: string, name: string): Known => ({
  id,
  name,
  path: `/vaults/${name}`,
  missing: false,
})

const renamed = (over: Partial<Renamed> = {}): Renamed => ({
  path: 'physics/Entropy.md',
  title: 'Entropy',
  frontmatter: false,
  moved: null,
  refusal: null,
  changed: false,
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
    /** The folder the person chose in the machine's own picker. */
    chose?: string
    /** What the list of vaults answered adding or renaming one. */
    added?: Added
    /** What the list of vaults refused forgetting, erasing or opening one. */
    turnedDown?: VaultRefused
    /** What moving a file came back with. */
    movement?: Movement
    /** What making a folder was refused with. */
    folderRefused?: Refused
  } = {},
) => {
  const done: string[] = []
  const said: string[] = []
  const at = answers.at ?? 'physics/Ontology.md'
  const refusal = answers.turnedDown ?? null
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
    moves: async (from, to) => {
      done.push(`moves ${from} ${to}`)
      return answers.movement ?? { moved: null, refusal: null }
    },
    makesFolder: async (path) => {
      done.push(`makes folder ${path}`)
      return answers.folderRefused ?? null
    },
    reveals: (path) => void done.push(`reveals ${path}`),
    notes: {
      holding: (path) => (path === at ? 'held' : null),
      where: (id) => (id === 'held' ? at : id),
      asking: () => answers.asking === true,
      settles: async (id) => void done.push(`settles ${id}`),
      shuts: (id) => void done.push(`shuts ${id}`),
      shows: (path, title, showing) => void done.push(`shows ${path} ${title} ${showing}`),
    },
    vaults: {
      list: async () => ({ vaults: [known('physics', 'Physics')], showing: 'physics' }),
      choose: async (title) => {
        done.push(`choose ${title}`)
        return answers.chose ?? '/vaults/Heat'
      },
      add: async (path, name) => {
        done.push(`add ${path} ${name || '—'}`)
        return answers.added ?? { vault: known('heat', 'Heat'), refusal: null }
      },
      rename: async (id, name) => {
        done.push(`renames vault ${id} ${name}`)
        return answers.added ?? { vault: known(id, name), refusal: null }
      },
      forget: async (id) => {
        done.push(`forgets ${id}`)
        return refusal
      },
      erase: async (id) => {
        done.push(`erases ${id}`)
        return refusal
      },
      open: async (id) => {
        done.push(`opens vault ${id}`)
        return refusal
      },
    },
    calls: (vault) => void done.push(`calls ${vault.id} ${vault.name}`),
    reloads: () => void done.push('reloads'),
    travel: async (path) => void done.push(`travel ${path}`),
    leaves: async (from, to) => void done.push(`leaves ${from} ${to}`),
    opening: () => 'Root.md',
    opens: (kind) => void done.push(`opens ${kind}`),
    closes: (tab) => void done.push(`closes ${tab}`),
    asks: (text) => void done.push(`asks ${text}`),
    copies: (path) => void done.push(`copies ${path}`),
    searches: () => void done.push('searches'),
    appearance: async (chosen) => void done.push(`appearance ${chosen}`),
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

  it('says the note was written elsewhere while this was asked', async () => {
    const one = window({ renamed: renamed({ changed: true }) })

    await carry(deedOf('title', front(), 'Entropy'), one.on)

    expect(one.said).toStrictEqual([words.overtaken])
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
          repaired: [],
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

  it('says nothing of the notes whose links it wrote again', async () => {
    const one = window({
      renamed: renamed({
        moved: {
          from: 'physics/Ontology.md',
          to: 'physics/Entropy.md',
          repaired: ['Notes.md', 'Order.md'],
          retargeted: [],
        },
      }),
    })

    await carry(deedOf('title', front(), 'Entropy'), one.on)

    expect(one.said).toStrictEqual([])
  })

  it('says nothing of the title it wrote into the frontmatter', async () => {
    const one = window({ renamed: renamed({ frontmatter: true }) })

    await carry(deedOf('title', front(), 'Entropy'), one.on)

    expect(one.said).toStrictEqual([])
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

  it('says where in the trash it landed, which is the way back to it', async () => {
    const one = window()

    await carry(deedOf('remove', front()), one.on)

    expect(one.said).toStrictEqual([`${words.trashedAt} .trash/Ontology.md`])
  })

  it('says nothing about the trash where the note was destroyed', async () => {
    const one = window({ removed: removed({ trashed: '' }) })

    await carry(deedOf('destroy', front(), 'Ontology'), one.on)

    expect(one.said).toStrictEqual([])
  })

  it('says where it went, and the notes that link to nothing now', async () => {
    const one = window({ removed: removed({ dangling: ['Order.md', 'Notes.md'] }) })

    await carry(deedOf('remove', front()), one.on)

    expect(one.said).toStrictEqual([
      `${words.trashedAt} .trash/Ontology.md. ${words.dangling} Order.md, Notes.md`,
    ])
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

describe('a file filed somewhere else', () => {
  /** The destination is the whole path, so a name changed in one folder is a move. */
  const moved = (to: string) => deedOf('move', front(), to)

  it('is asked of the vault under the path it is filed at from now on', async () => {
    const one = window()

    await carry(moved('notes/Ontology.md'), one.on)

    expect(one.done).toStrictEqual(['settles held', 'moves physics/Ontology.md notes/Ontology.md'])
  })

  it('settles the tab holding it before its file goes anywhere', async () => {
    const one = window()

    await carry(moved('notes/Ontology.md'), one.on)

    expect(one.done.indexOf('settles held')).toBeLessThan(
      one.done.indexOf('moves physics/Ontology.md notes/Ontology.md'),
    )
  })

  it('stays where it is while its tab is waiting on the person', async () => {
    const one = window({ asking: true })

    await carry(moved('notes/Ontology.md'), one.on)

    expect(one.done).toStrictEqual([])
    expect(one.said).toStrictEqual([words.unanswered])
  })

  it('stays where it is where something of that name is filed there', async () => {
    const one = window({ movement: { moved: null, refusal: 'occupied' } })

    await carry(moved('notes/Ontology.md'), one.on)

    expect(one.said).toStrictEqual([words.occupied])
  })

  it('says nothing of a note renamed, which is what a move is not', async () => {
    const one = window({ movement: { moved: null, refusal: 'occupied' } })

    await carry(moved('notes/Ontology.md'), one.on)

    expect(one.said).not.toContain(words.refused.occupied)
  })

  it('names the notes whose links mean another note now', async () => {
    const one = window({
      movement: {
        moved: {
          from: 'physics/Ontology.md',
          to: 'notes/Ontology.md',
          repaired: [],
          retargeted: [{ in: 'physics/Being.md', target: 'Ontology', now: 'notes/Ontology.md' }],
        },
        refusal: null,
      },
    })

    await carry(moved('notes/Ontology.md'), one.on)

    expect(one.said.at(-1)).toContain('physics/Being.md')
  })

  it('asks the vault for nothing where it landed where it already was', async () => {
    const one = window()

    await carry(moved('physics/Ontology.md'), one.on)

    expect(one.done).toStrictEqual([])
  })
})

describe('a folder made', () => {
  it('is asked of the vault under the path it goes at', async () => {
    const one = window()

    await carry(deedOf('makeFolder', front(), 'physics/heat'), one.on)

    expect(one.done).toStrictEqual(['makes folder physics/heat'])
  })

  it('is not made where something of that name is filed there', async () => {
    const one = window({ folderRefused: 'occupied' })

    await carry(deedOf('makeFolder', front(), 'physics/heat'), one.on)

    expect(one.said).toStrictEqual([words.occupied])
  })

  it('asks the vault for nothing where no path was given', async () => {
    const one = window()

    await carry(deedOf('makeFolder', front()), one.on)

    expect(one.done).toStrictEqual([])
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

  it('hands over the row that was chosen, and nothing about the note in front', async () => {
    const one = window()

    await carry(deedOf('appearance', front(), 'mine:sea'), one.on)
    await carry(deedOf('appearance', front(), 'mode:dark'), one.on)

    expect(one.done).toStrictEqual(['appearance mine:sea', 'appearance mode:dark'])
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

describe('another vault under this window', () => {
  const heat = () => front({ vault: { id: 'heat', name: 'Heat' } })

  it('is opened, and the page drawn again on it', async () => {
    const one = window()

    await carry(deedOf('openVault', heat()), one.on)

    expect(one.done).toStrictEqual(['opens vault heat', 'reloads'])
  })

  it('leaves the page where it stands where the vault would not open', async () => {
    const one = window({ turnedDown: 'showing' })

    await carry(deedOf('openVault', heat()), one.on)

    expect(one.done).toStrictEqual(['opens vault heat'])
    expect(one.said).toStrictEqual([words.unvaulted.showing])
  })
})

describe('a vault made', () => {
  it('is the folder chosen in the machine’s own picker, and is opened', async () => {
    const one = window()

    await carry(deedOf('newVault', front()), one.on)

    expect(one.done).toStrictEqual([
      `choose ${words.folder}`,
      'add /vaults/Heat —',
      'opens vault heat',
      'reloads',
    ])
  })

  it('is nothing at all where the person closed the picker', async () => {
    const one = window({ chose: '' })

    await carry(deedOf('newVault', front()), one.on)

    expect(one.done).toStrictEqual([`choose ${words.folder}`])
    expect(one.said).toStrictEqual([])
  })

  it('says a folder that lies inside a vault already added', async () => {
    const one = window({ added: { vault: null, refusal: 'overlaps' } })

    await carry(deedOf('newVault', front()), one.on)

    expect(one.said).toStrictEqual([words.unvaulted.overlaps])
  })
})

describe('a vault renamed', () => {
  it('is called what was typed, and the window calls it that from now on', async () => {
    const one = window()

    await carry(deedOf('renameVault', front(), 'Heat'), one.on)

    expect(one.done).toStrictEqual(['renames vault physics Heat', 'calls physics Heat'])
  })

  it('is left alone where the name it was given is the name it has', async () => {
    const one = window()

    await carry(deedOf('renameVault', front(), 'Physics'), one.on)

    expect(one.done).toStrictEqual([])
  })

  it('says a name another vault is already called', async () => {
    const one = window({ added: { vault: null, refusal: 'nameTaken' } })

    await carry(deedOf('renameVault', front(), 'Heat'), one.on)

    expect(one.said).toStrictEqual([words.unvaulted.nameTaken])
  })
})

/** The vault taken off the list is the one that was chosen, never the one in front. */
describe('a vault taken off the list', () => {
  const heat = () => front({ vault: { id: 'heat', name: 'Heat' } })

  it('is forgotten, and its folder left where it is', async () => {
    const one = window()

    await carry(deedOf('forgetVault', heat()), one.on)

    expect(one.done).toStrictEqual(['forgets heat'])
  })

  it('says the only vault this installation has stays on it', async () => {
    const one = window({ turnedDown: 'lastVault' })

    await carry(deedOf('forgetVault', heat()), one.on)

    expect(one.said).toStrictEqual([words.unvaulted.lastVault])
  })

  it('is erased where erasing was what was asked', async () => {
    const one = window()

    await carry(deedOf('eraseVault', heat(), 'Heat'), one.on)

    expect(one.done).toStrictEqual(['erases heat'])
  })

  it('says a machine with nowhere to put what is deleted', async () => {
    const one = window({ turnedDown: 'noTrash' })

    await carry(deedOf('eraseVault', heat(), 'Heat'), one.on)

    expect(one.said).toStrictEqual([words.unvaulted.noTrash])
  })

  it('says the vault in front of the person, which the window stands on', async () => {
    const one = window({ turnedDown: 'showing' })

    await carry(deedOf('forgetVault', front()), one.on)

    expect(one.said).toStrictEqual([words.unvaulted.showing])
  })
})

describe('what the list of vaults refused', () => {
  it('reaches the person in the window’s own words, whichever it was', async () => {
    for (const refusal of Object.keys(words.unvaulted) as VaultRefused[]) {
      const one = window({ turnedDown: refusal })

      await carry(deedOf('openVault', front()), one.on)

      expect(words.unvaulted[refusal], refusal).not.toBe('')
      expect(one.said, refusal).toStrictEqual([words.unvaulted[refusal]])
    }
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
