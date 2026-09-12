/**
 * What a gesture in a plex comes to, asked without a screen.
 *
 * A note made from a node other than the focus is the one to watch: it is
 * written into the vault whatever happens, and a picture one seat deep draws
 * it only from the node it was made from.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { paneById, panesOf } from '@numen/ui'
import { usePlexTab, type PlexTabState, type PlexEditor, type PlexTabDeps } from './usePlexTab'
import { plexKind } from '../kind'
import { ITEMS, NEW_NOTE } from '../lib/menu'
import { usePlexView as viewing, type PlexView } from './usePlexView'
import { WORDS as words } from '../words'
import type { NoteHeading, Neighbourhood, Seat } from '@/entities/note'
import type { NoteType } from '@/shared/file'
import { getRenamedPath, type PathRename } from '@/shared/paths'
import { useWindowTabs, type AnyTabKind } from '@/entities/tab'
import { PLEX } from '@/entities/tab'

/** A moment for whatever a gesture asked the vault for to come back. */
const settles = () => new Promise((done) => setTimeout(done, 0))

/** Which of three each note of a neighbourhood is, by the path it stands at. */
type Types = Record<string, NoteType>

/** A neighbourhood as the vault answers one: a focus, and what is around it. */
const around = (
  focus: string,
  related: readonly string[] = [],
  types: Types = {},
): Neighbourhood => ({
  focus: { path: focus, title: focus.replace(/\.md$/, '') },
  focusType: types[focus] ?? 'note',
  related: related.map((path) => ({
    seat: 'child',
    through: '',
    label: '',
    isMutual: false,
    path,
    title: path.replace(/\.md$/, ''),
    type: types[path] ?? 'note',
  })),
})

/** A plex standing on a note, which records every note it was sent to. */
const viewOn = (at: string, related: readonly string[] = [], types: Types = {}) => {
  const went: string[] = []
  const view = {
    neighbourhood: ref(around(at, related, types)),
    here: ref(at),
    error: ref(''),
    go: async (path: string) => {
      went.push(path)
      view.here.value = path
      view.neighbourhood.value = around(path, [], types)
    },
    follows: (renamed: readonly { from: string; to: string }[]) => {
      const one = renamed.find((went) => went.from === view.here.value)
      if (one) view.here.value = one.to
    },
    close: () => {},
  }
  return { view: view as unknown as PlexView, went }
}

/** A vault that takes every note it is asked to make, and records the asking. */
const createVault = (takes = true) => {
  const made: [string, string][] = []
  const joined: [string, string, string][] = []
  /** The notes this vault will write no link to, which a test names. */
  const refusedPaths = new Set<string>()
  const makes: PlexEditor = {
    make: async (from, seat) => {
      made.push([from, seat])
      return takes ? { path: 'Made.md', title: 'Made' } : null
    },
    join: async (from, to, seat) => {
      joined.push([from, to, seat])
      return takes && !refusedPaths.has(to)
    },
  }
  return { makes, made, joined, refusedPaths }
}

/** A plex tab with the window it is drawn in written down. */
const tab = (at: string, related: readonly string[] = [], takes = true, types: Types = {}) => {
  const plex = viewOn(at, related, types)
  const vault = createVault(takes)
  const opened: [string, string, string][] = []
  const asked: string[] = []
  const ran: [string, string, string][] = []
  const said: string[] = []
  /** Every note the vault was asked to make on its own. */
  const wrote: string[] = []
  /** The notes the window is dragging over this picture, which a test sets. */
  const dragging = ref<readonly string[]>([])
  /** Where a note made on its own lands, which a test empties for a vault refusing. */
  const writes = ref('Untitled note.md')
  /** Every note put in front of the person on a line of its own prose. */
  const entered: [string, number][] = []
  /** What the vault says each note is divided into, which a test sets. */
  const divides = ref<ReadonlyMap<string, readonly NoteHeading[]>>(new Map())
  /** Whether a node hangs the parts of its note, which a test turns. */
  const hangs = ref(true)
  /** Every question the vault was asked about what the notes hold. */
  const insides: (readonly string[])[] = []
  const deps: PlexTabDeps = {
    makes: vault.makes,
    ready: ref(true),
    hangs,
    parts: ref(6),
    opens: (path, title, showing, line) => {
      opened.push([path, title, showing])
      if (line !== undefined) entered.push([path, line])
    },
    // The vault answers about the notes it was asked about and no others.
    inside: async (paths) => {
      insides.push(paths)
      return new Map([...divides.value].filter(([path]) => paths.includes(path)))
    },
    asks: (text) => asked.push(text),
    runs: (id, path, title) => ran.push([id, path, title]),
    opening: ref('Opening.md'),
    first: async () => 'Opening.md',
    dragged: dragging,
    says: (text) => said.push(text),
    writes: async () => {
      wrote.push(writes.value)
      return writes.value
    },
    creatable: ['parent', 'child', 'jump'],
  }
  const state = usePlexTab(plex.view, deps)
  /** What the picture calls a note, which is what a gesture in it carries. */
  const node = (path: string) => nodeFor(state, path.replace(/\.md$/, ''))
  return {
    state,
    node,
    dragging,
    writes,
    wrote,
    went: plex.went,
    ...vault,
    opened,
    entered,
    divides,
    hangs,
    insides,
    asked,
    ran,
    said,
  }
}

/** The node of the picture drawn for the note of this title, if it draws one. */
const nodeFor = (state: PlexTabState, title: string): string =>
  state.picture.value?.nodes.find((node) => node.title === title)?.id ?? ''

describe('a note made from a node', () => {
  it('stands the plex on the node it was made from', async () => {
    const one = tab('Root.md', ['Child.md'])

    await one.state.made(one.node('Child.md'), 'child')

    expect(one.made).toEqual([['Child.md', 'child']])
    expect(one.went).toEqual(['Child.md'])
  })

  it('leaves the plex where it is when that node is the focus', async () => {
    const one = tab('Root.md')

    await one.state.made(one.node('Root.md'), 'child')

    expect(one.went).toEqual(['Root.md'])
    expect(one.state.view.here.value).toBe('Root.md')
  })

  it('travels nowhere when the vault made nothing', async () => {
    const one = tab('Root.md', ['Child.md'], false)

    await one.state.made(one.node('Child.md'), 'parent')

    expect(one.went).toEqual([])
  })
})

describe('two notes a line was drawn between', () => {
  it('stands the plex on the note the line was drawn from', async () => {
    const one = tab('Root.md', ['Child.md', 'Other.md'])

    await one.state.joined(one.node('Child.md'), one.node('Other.md'), 'jump')

    expect(one.joined).toEqual([['Child.md', 'Other.md', 'jump']])
    expect(one.went).toEqual(['Child.md'])
  })

  it('travels nowhere when nothing was written', async () => {
    const one = tab('Root.md', ['Child.md'], false)

    await one.state.joined(one.node('Child.md'), one.node('Root.md'), 'jump')

    expect(one.went).toEqual([])
  })
})

describe('the menu on a node', () => {
  const createMenuRequest = (node: string) => ({
    node,
    at: { x: 1, y: 2 },
    opening: 'pointer' as const,
  })

  it('hands the command the note it was asked for on, called what the picture calls it', () => {
    const one = tab('Root.md', ['Child.md'])
    one.state.asks(createMenuRequest(one.node('Child.md')))

    one.state.chose('read')

    expect(one.ran).toEqual([['read', 'Child.md', 'Child']])
    expect(one.state.menu.value).toBeNull()
  })

  it('hands over every command it offers, and nothing it does not', () => {
    const one = tab('Root.md', ['Child.md'])
    for (const item of ITEMS) {
      one.state.asks(createMenuRequest(one.node('Child.md')))
      one.state.chose(item.id)
    }
    one.state.asks(createMenuRequest(one.node('Child.md')))
    one.state.chose('constructor')
    one.state.asks(createMenuRequest(one.node('Child.md')))
    one.state.chose('destroy')

    expect(one.ran.map(([id]) => id)).toStrictEqual(ITEMS.map((item) => item.id))
  })

  it('does nothing when it stands on nothing', () => {
    const one = tab('Root.md')

    one.state.chose('read')

    expect(one.ran).toEqual([])
  })

  it('goes when the picture under it does', () => {
    const one = tab('Root.md', ['Child.md'])
    one.state.asks(createMenuRequest(one.node('Child.md')))

    one.state.dismiss()

    expect(one.state.menu.value).toBeNull()
  })
})

/**
 * A picture is nothing for more reasons than an empty vault, and only one of
 * them is a plex to offer a note over.
 */
describe('a plex drawing nothing', () => {
  const plex = (over: Partial<PlexTabDeps>) => {
    const view = {
      neighbourhood: ref(null),
      here: ref(''),
      error: ref(''),
      go: async () => {},
      follows: () => {},
      close: () => {},
    }
    return usePlexTab(view as unknown as PlexView, {
      makes: createVault().makes,
      ready: ref(true),
      hangs: ref(true),
      parts: ref(6),
      opens: () => {},
      inside: async () => new Map(),
      asks: () => {},
      runs: () => {},
      opening: ref(''),
      first: async () => '',
      dragged: ref([]),
      says: () => {},
      writes: async () => '',
      creatable: ['parent', 'child', 'jump'],
      ...over,
    })
  }

  it('is an empty vault where the vault is read and opens with no note', () => {
    expect(plex({}).empty.value).toBe(true)
  })

  it('is not an empty vault where the vault could not be opened', () => {
    expect(plex({ ready: ref(false) }).empty.value).toBe(false)
  })

  it('is not an empty vault while the first answer is on its way', () => {
    expect(plex({ opening: ref('Opening.md') }).empty.value).toBe(false)
  })

  it('is not an empty vault once the plex stands on a note', () => {
    expect(tab('Root.md').state.empty.value).toBe(false)
  })
})

describe('the menu off every node', () => {
  const asked = { node: null, at: { x: 1, y: 2 }, opening: 'pointer' as const }

  it('makes a note, and the plex stands on it', async () => {
    const one = tab('Root.md')
    one.state.asks(asked)

    one.state.chose(NEW_NOTE)
    await settles()

    expect(one.wrote).toStrictEqual(['Untitled note.md'])
    expect(one.went).toContain('Untitled note.md')
    expect(one.state.menu.value).toBeNull()
  })

  it('stands where it stood where the vault made none', async () => {
    const one = tab('Root.md')
    one.writes.value = ''
    one.state.asks(asked)

    one.state.chose(NEW_NOTE)
    await settles()

    expect(one.went).toStrictEqual([])
  })

  it('makes nothing of a command over a note, there being no note it is over', async () => {
    const one = tab('Root.md')
    one.state.asks(asked)

    one.state.chose('read')
    await settles()

    expect(one.wrote).toStrictEqual([])
    expect(one.ran).toStrictEqual([])
  })
})

describe('what a note in the picture is called', () => {
  it('is the title the vault gave the focus', () => {
    expect(tab('Root.md').state.nameOf('Root.md')).toBe('Root')
  })

  it('is the title of a note around it', () => {
    expect(tab('Root.md', ['Deep/Child.md']).state.nameOf('Deep/Child.md')).toBe('Deep/Child')
  })

  it('is the file it is filed under, for a note the picture does not name', () => {
    expect(tab('Root.md').state.nameOf('Deep/Elsewhere.md')).toBe('Elsewhere')
  })
})

describe('the picture', () => {
  it('is nothing while the window has nothing true to draw', () => {
    const plex = viewOn('Root.md')
    const state = usePlexTab(plex.view, {
      makes: createVault().makes,
      ready: ref(false),
      hangs: ref(true),
      parts: ref(6),
      opens: () => {},
      inside: async () => new Map(),
      asks: () => {},
      runs: () => {},
      opening: ref(''),
      first: async () => '',
      dragged: ref(['Entropy.md']),
      says: () => {},
      writes: async () => '',
      creatable: ['parent', 'child', 'jump'],
    })

    expect(state.picture.value).toBeNull()
    // Nothing is drawn, so nothing can be dragged onto it.
    expect(state.dragged.value).toStrictEqual([])
  })
})

describe('notes dragged in and let go over the picture', () => {
  it('writes the link into the note the plex stands on, naming the dragged one second', async () => {
    const one = tab('Root.md', ['Child.md'])

    await one.state.brought(['physics/Entropy.md'], 'child')

    expect(one.joined).toEqual([['Root.md', 'physics/Entropy.md', 'child']])
    expect(one.went).toEqual(['Root.md'])
  })

  it('writes one for each of them, to the one note in the one seat', async () => {
    const one = tab('Root.md')

    await one.state.brought(['Entropy.md', 'Kelvin.md', 'Heat.md'], 'child')

    expect(one.joined).toEqual([
      ['Root.md', 'Entropy.md', 'child'],
      ['Root.md', 'Kelvin.md', 'child'],
      ['Root.md', 'Heat.md', 'child'],
    ])
    expect(one.went).toEqual(['Root.md'])
  })

  it('takes the seat the drag named, whichever it was', async () => {
    const one = tab('Root.md')

    await one.state.brought(['Entropy.md'], 'parent')
    await one.state.brought(['Heat.md'], 'jump')

    expect(one.joined).toEqual([
      ['Root.md', 'Entropy.md', 'parent'],
      ['Root.md', 'Heat.md', 'jump'],
    ])
  })

  it('writes the rest where one of them was refused, and says which stayed', async () => {
    const one = tab('Root.md')
    one.refusedPaths.add('Kelvin.md')

    await one.state.brought(['Entropy.md', 'Kelvin.md', 'Heat.md'], 'child')

    expect(one.joined).toEqual([
      ['Root.md', 'Entropy.md', 'child'],
      ['Root.md', 'Kelvin.md', 'child'],
      ['Root.md', 'Heat.md', 'child'],
    ])
    expect(one.said).toEqual([`${words.refused} Kelvin`])
    expect(one.went).toEqual(['Root.md'])
  })

  it('says every one of them where the vault would write none, and travels nowhere', async () => {
    const one = tab('Root.md', [], false)

    await one.state.brought(['Entropy.md', 'Heat.md'], 'child')

    expect(one.said).toEqual([`${words.refused} Entropy, Heat`])
    expect(one.went).toEqual([])
  })

  it('joins nothing where the plex has nowhere to stand', async () => {
    const one = tab('')

    await one.state.brought(['Entropy.md'], 'child')

    expect(one.joined).toEqual([])
    expect(one.said).toEqual([])
  })

  it('leaves the note the plex stands on out, and joins the rest', async () => {
    const one = tab('Root.md')

    await one.state.brought(['Root.md', 'Entropy.md'], 'child')

    expect(one.joined).toEqual([['Root.md', 'Entropy.md', 'child']])
    expect(one.said).toEqual([])
  })
})

describe('what the picture draws a line to', () => {
  it('is what the window says it is dragging', () => {
    const one = tab('Root.md')
    one.dragging.value = ['physics/Entropy.md', 'physics/Kelvin.md']

    expect(one.state.dragged.value).toStrictEqual([
      'physics/Entropy.md',
      'physics/Kelvin.md',
    ])
  })

  it('is nothing while the window is dragging none', () => {
    expect(tab('Root.md').state.dragged.value).toStrictEqual([])
  })

  it('leaves out the note the plex stands on, and keeps the rest', () => {
    const one = tab('Root.md')
    one.dragging.value = ['Root.md', 'Entropy.md']

    expect(one.state.dragged.value).toStrictEqual(['Entropy.md'])
  })
})

describe('the parts a node hangs', () => {
  /** One heading of a note, as the vault answers one. */
  const heading = (text: string, line: number, level = 1): NoteHeading => ({ text, level, line })

  it('are what the vault said that note is divided into', async () => {
    const one = tab('Root.md', ['Child.md'])
    one.divides.value = new Map([['Child.md', [heading('Heat', 4), heading('Cold', 9, 2)]]])

    await one.state.reads()

    expect(one.state.partsOf(one.node('Child.md'))).toStrictEqual([
      { id: '4', text: 'Heat', level: 1 },
      { id: '9', text: 'Cold', level: 2 },
    ])
  })

  it('are none for a node the vault said nothing about', async () => {
    const one = tab('Root.md', ['Child.md'])
    one.divides.value = new Map([['Child.md', [heading('Heat', 4)]]])

    await one.state.reads()

    expect(one.state.partsOf(one.node('Root.md'))).toStrictEqual([])
  })

  it('are each named by the line it stands on', async () => {
    const one = tab('Root.md')
    one.divides.value = new Map([['Root.md', [heading('Heat', 12)]]])

    await one.state.reads()

    expect(one.state.partsOf(one.node('Root.md')).map((part) => part.id)).toStrictEqual(['12'])
  })

  it('are asked for again once the plex stands somewhere else', async () => {
    const one = tab('Root.md', ['Child.md'])
    one.divides.value = new Map([['Other.md', [heading('Heat', 4)]]])

    await one.state.view.go('Other.md')
    await settles()

    expect(one.state.partsOf(one.node('Other.md'))).toStrictEqual([
      { id: '4', text: 'Heat', level: 1 },
    ])
  })

  it('are what the question asked last came back with, whatever order they land in', async () => {
    // The first answer is held up until the second has settled, which is a
    // change followed while a travel is still out.
    const answers: ((held: ReadonlyMap<string, readonly NoteHeading[]>) => void)[] = []
    const one = tab('Root.md')
    const plex = usePlexTab(one.state.view, {
      makes: createVault().makes,
      ready: ref(true),
      hangs: ref(true),
      parts: ref(6),
      opens: () => {},
      inside: () => new Promise((done) => answers.push(done)),
      asks: () => {},
      runs: () => {},
      opening: ref('Root.md'),
      first: async () => 'Root.md',
      dragged: ref([]),
      says: () => {},
      writes: async () => '',
      creatable: ['parent', 'child', 'jump'],
    })
    const node = () => nodeFor(plex, 'Root')

    // The picture asks once as it is built, and that answer is left pending.
    const asked = answers.length
    void plex.reads()
    void plex.reads()
    answers[asked + 1]?.(new Map([['Root.md', [heading('Fresh', 2)]]]))
    answers[asked]?.(new Map([['Root.md', [heading('Stale', 1)]]]))
    await settles()

    expect(plex.partsOf(node())).toStrictEqual([{ id: '2', text: 'Fresh', level: 1 }])
  })

  it('are none for every node while the setting is off', async () => {
    const one = tab('Root.md', ['Child.md'])
    one.divides.value = new Map([['Child.md', [heading('Heat', 4)]]])
    await one.state.reads()
    one.hangs.value = false

    expect(one.state.partsOf(one.node('Child.md'))).toStrictEqual([])
    expect(one.state.partsOf(one.node('Root.md'))).toStrictEqual([])
  })

  it('are not asked of the vault at all while the setting is off', async () => {
    const one = tab('Root.md', ['Child.md'])
    one.hangs.value = false
    one.insides.length = 0

    await one.state.reads()

    expect(one.insides).toStrictEqual([])
  })

  it('are none for a deck, whose cards stand on no line of prose', async () => {
    const one = tab('Root.md', ['Animals.md'], true, { 'Animals.md': 'deck' })
    one.divides.value = new Map([['Animals.md', [heading('Vicuña', 4)]]])
    one.insides.length = 0

    await one.state.reads()

    expect(one.state.partsOf(one.node('Animals.md'))).toStrictEqual([])
    expect(one.insides).toStrictEqual([['Root.md']])
  })

  it('are none for a stencil, whose faces stand on no line of prose', async () => {
    const one = tab('Root.md', ['Animal.md'], true, { 'Animal.md': 'stencil' })
    one.divides.value = new Map([['Animal.md', [heading('Front', 4)]]])
    one.insides.length = 0

    await one.state.reads()

    expect(one.state.partsOf(one.node('Animal.md'))).toStrictEqual([])
    expect(one.insides).toStrictEqual([['Root.md']])
  })

  it('are the headings of an ordinary note beside them', async () => {
    const one = tab('Root.md', ['Animals.md', 'Child.md'], true, {
      'Animals.md': 'deck',
    })
    one.divides.value = new Map([['Child.md', [heading('Heat', 4)]]])

    await one.state.reads()

    expect(one.state.partsOf(one.node('Child.md'))).toStrictEqual([
      { id: '4', text: 'Heat', level: 1 },
    ])
  })

  it('are none for a deck in focus, which is where the plex is standing', async () => {
    const one = tab('Animals.md', [], true, { 'Animals.md': 'deck' })
    one.divides.value = new Map([['Animals.md', [heading('Vicuña', 4)]]])
    one.insides.length = 0

    await one.state.reads()

    expect(one.state.partsOf(one.node('Animals.md'))).toStrictEqual([])
    expect(one.insides).toStrictEqual([])
  })

  it('are hung again once the setting is turned back on', async () => {
    // Turning it is all a person does; nothing else asks for them again.
    const one = tab('Root.md', ['Child.md'])
    one.divides.value = new Map([['Child.md', [heading('Heat', 4)]]])

    one.hangs.value = false
    await settles()
    expect(one.state.partsOf(one.node('Child.md'))).toStrictEqual([])

    one.hangs.value = true
    await settles()

    expect(one.state.partsOf(one.node('Child.md'))).toStrictEqual([
      { id: '4', text: 'Heat', level: 1 },
    ])
  })
})

describe('a part of a node chosen', () => {
  it('opens the note it stands in, and puts the keyboard on its line', () => {
    const one = tab('Root.md')

    one.state.entered(one.node('Root.md'), '12')

    expect(one.opened).toStrictEqual([['Root.md', 'Root', 'here']])
    expect(one.entered).toStrictEqual([['Root.md', 12]])
  })

  it('opens the note the node stands for, the focus being another one', () => {
    const one = tab('Root.md', ['Child.md'])

    one.state.entered(one.node('Child.md'), '7')

    expect(one.opened).toStrictEqual([['Child.md', 'Child', 'here']])
    expect(one.entered).toStrictEqual([['Child.md', 7]])
  })

  it('stands the plex where it stood', () => {
    const one = tab('Root.md', ['Child.md'])

    one.state.entered(one.node('Child.md'), '7')

    expect(one.went).toStrictEqual([])
    expect(one.state.view.here.value).toBe('Root.md')
  })

  it('opens nothing for a part naming no line, which is the one standing for the rest', () => {
    const one = tab('Root.md')

    one.state.entered(one.node('Root.md'), '')

    expect(one.opened).toStrictEqual([])
    expect(one.entered).toStrictEqual([])
  })
})

/** A note beside another: where it sits, and the note it comes through. */
type NeighbourRow = readonly [path: string, seat: Seat, through?: string]

/** What a note is shown by, which a move of its file leaves alone. */
const titleOf = (path: string) => (path.split('/').pop() ?? path).replace(/\.md$/, '')

/**
 * A plex standing in a vault whose files a person can move.
 *
 * The vault answers what is around whatever note it is asked about, and no note
 * in it carries an identifier — this is the vault of somebody who writes in
 * another editor. A move renames its files and is told to the plex as a change
 * arriving tells it.
 */
const inVault = async (focus: string, beside: readonly NeighbourRow[] = []) => {
  let around = beside.map(([path, seat, through]) => ({ path, seat, through: through ?? '' }))
  /** An answer the vault is holding back, and what lets it go. */
  let holding: Promise<void> | null = null
  let release = () => {}
  /** Every note the vault was asked about, in the order it was asked. */
  const asked: string[] = []
  const vault = createVault()
  const opened: [string, string, string][] = []
  const ran: [string, string, string][] = []

  const view = viewing({
    neighbourhood: async (path) => {
      asked.push(path)
      if (holding) await holding
      return {
        focus: { path, title: titleOf(path) },
        focusType: 'note' as const,
        related: around.map((one) => ({
          path: one.path,
          title: titleOf(one.path),
          type: 'note' as const,
          seat: one.seat,
          label: '',
          through: one.through ?? '',
          isMutual: false,
        })),
      }
    },
  })
  const state = usePlexTab(view, {
    makes: vault.makes,
    ready: ref(true),
    hangs: ref(true),
    parts: ref(6),
    opens: (path, title, showing) => opened.push([path, title, showing]),
    inside: async () => new Map(),
    asks: () => {},
    runs: (id, path, title) => ran.push([id, path, title]),
    opening: ref(''),
    first: async () => '',
    dragged: ref([]),
    says: () => {},
    writes: async () => '',
    creatable: ['parent', 'child', 'jump'],
  })
  await view.go(focus)

  /** The picture as it stands, which is what a drawn window reads. */
  const picture = () => state.picture.value
  /** What the picture calls the note shown by this title. */
  const node = (title: string) => nodeFor(state, title)
  /** Every node of the picture, as the seat it sits in and what it is called. */
  const nodes = () => (picture()?.nodes ?? []).map((one) => `${one.seat} ${one.id}`)
  /** Every edge of the picture, as the two nodes it runs between. */
  const edges = () => (picture()?.edges ?? []).map((one) => `${one.from} -> ${one.to}`)

  /** Files moved in the vault, and the plex told what went where. */
  const follows = (...renamed: readonly PathRename[]) => {
    around = around.map((one) => ({
      ...one,
      path: getRenamedPath(renamed, one.path) || one.path,
      through: getRenamedPath(renamed, one.through) || one.through,
    }))
    state.follows(renamed)
  }
  /** A move, followed by the picture being asked for again. */
  const moves = async (...renamed: readonly PathRename[]) => {
    follows(...renamed)
    await view.go(view.here.value)
  }
  /** The person travels to another note, which is a picture of its own. */
  const travels = async (path: string, ...now: readonly NeighbourRow[]) => {
    around = now.map(([at, seat, through]) => ({ path: at, seat, through: through ?? '' }))
    await view.go(path)
  }

  return {
    state,
    view,
    picture,
    node,
    nodes,
    edges,
    follows,
    moves,
    travels,
    asked,
    opened,
    ran,
    made: vault.made,
    joined: vault.joined,
    /** The vault holds every answer back until it is let go of. */
    holdAnswers: () => {
      holding = new Promise<void>((wake) => (release = wake))
    },
    answers: () => release(),
  }
}

describe('a note whose file moved', () => {
  it('keeps what the picture calls it, so nothing is drawn again', async () => {
    const one = await inVault('Entropy.md')
    const before = one.nodes()

    await one.moves({ from: 'Entropy.md', to: 'physics/Entropy.md' })

    expect(one.nodes()).toStrictEqual(before)
  })

  it('keeps it where the note that moved is one around the note in focus', async () => {
    const one = await inVault('Entropy.md', [['Heat.md', 'child']])
    const before = one.node('Heat')

    await one.moves({ from: 'Heat.md', to: 'physics/Heat.md' })

    expect(one.node('Heat')).toBe(before)
    expect(one.view.here.value).toBe('Entropy.md')
  })

  it('keeps it where the note that moved is the one in focus', async () => {
    const one = await inVault('Entropy.md', [['Heat.md', 'child']])
    const before = one.node('Entropy')

    await one.moves({ from: 'Entropy.md', to: 'physics/Entropy.md' })

    expect(one.node('Entropy')).toBe(before)
    expect(one.view.here.value).toBe('physics/Entropy.md')
  })

  it('keeps it for every note of a folder that moved at once', async () => {
    const one = await inVault('physics/Entropy.md', [
      ['physics/Heat.md', 'child'],
      ['physics/Cold.md', 'child'],
    ])
    const before = one.nodes()

    await one.moves(
      { from: 'physics/Entropy.md', to: 'science/Entropy.md' },
      { from: 'physics/Heat.md', to: 'science/Heat.md' },
      { from: 'physics/Cold.md', to: 'science/Cold.md' },
    )

    expect(one.nodes()).toStrictEqual(before)
    expect(one.view.here.value).toBe('science/Entropy.md')
  })

  it('keeps it where two notes traded paths in one change', async () => {
    const one = await inVault('Root.md', [
      ['One.md', 'child'],
      ['Two.md', 'child'],
    ])
    const [, first, second] = one.picture()?.nodes ?? []

    await one.moves({ from: 'One.md', to: 'Two.md' }, { from: 'Two.md', to: 'One.md' })

    const [, now, later] = one.picture()?.nodes ?? []
    expect(now?.id).toBe(first?.id)
    expect(later?.id).toBe(second?.id)
    expect([now?.title, later?.title]).toStrictEqual(['Two', 'One'])
  })

  it('keeps it through a second move', async () => {
    const one = await inVault('Entropy.md', [['Heat.md', 'child']])
    const before = one.nodes()

    await one.moves({ from: 'Heat.md', to: 'physics/Heat.md' })
    await one.moves({ from: 'physics/Heat.md', to: 'thermo/Heat.md' })

    expect(one.nodes()).toStrictEqual(before)
  })

  it('runs the same edge between the same two nodes', async () => {
    const one = await inVault('Entropy.md', [
      ['Physics.md', 'parent'],
      ['Heat.md', 'sibling', 'Physics.md'],
    ])
    const before = one.edges()
    expect(before).toHaveLength(2)

    await one.moves(
      { from: 'Physics.md', to: 'science/Physics.md' },
      { from: 'Heat.md', to: 'science/Heat.md' },
    )

    expect(one.edges()).toStrictEqual(before)
  })

  it('is called something different from every other note of the picture', async () => {
    const one = await inVault('Entropy.md', [
      ['Heat.md', 'child'],
      ['Physics.md', 'parent'],
      ['Cold.md', 'sibling', 'Physics.md'],
    ])

    await one.moves({ from: 'Heat.md', to: 'physics/Heat.md' })

    const drawn = one.picture()?.nodes.map((node) => node.id) ?? []
    expect(new Set(drawn).size).toBe(4)
  })
})

describe('a gesture the plex reports', () => {
  it('travels to the note the node stands for', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    await one.moves({ from: 'Heat.md', to: 'physics/Heat.md' })

    one.state.activate(one.node('Heat'))

    expect(one.asked.at(-1)).toBe('physics/Heat.md')
  })

  it('opens the note the node stands for, called what the picture calls it', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    await one.moves({ from: 'Heat.md', to: 'physics/Heat.md' })

    one.state.opens(one.node('Heat'), 'beside')

    expect(one.opened).toStrictEqual([['physics/Heat.md', 'Heat', 'beside']])
  })

  it('runs a command on the note the menu stood on', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    await one.moves({ from: 'Heat.md', to: 'physics/Heat.md' })

    one.state.asks({ node: one.node('Heat'), at: { x: 1, y: 2 }, opening: 'pointer' })
    one.state.chose('read')

    expect(one.ran).toStrictEqual([['read', 'physics/Heat.md', 'Heat']])
  })

  it('makes a note in a seat of the note the node stands for', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    await one.moves({ from: 'Heat.md', to: 'physics/Heat.md' })

    await one.state.made(one.node('Heat'), 'child')

    expect(one.made).toStrictEqual([['physics/Heat.md', 'child']])
  })

  it('joins the two notes a line was drawn between', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    await one.moves({ from: 'Heat.md', to: 'physics/Heat.md' })

    await one.state.joined(one.node('Heat'), one.node('Root'), 'jump')

    expect(one.joined).toStrictEqual([['physics/Heat.md', 'Root.md', 'jump']])
  })

  it('is let go of where the picture never drew that node', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    expect(one.node('Heat')).not.toBe('')
    const asked = one.asked.length

    for (const node of ['Heat.md', 'Root.md', 'nothing']) {
      one.state.activate(node)
      one.state.opens(node, 'here')
      one.state.asks({ node, at: { x: 1, y: 2 }, opening: 'pointer' })
      one.state.chose('read')
      await one.state.made(node, 'child')
      await one.state.joined(node, one.node('Root'), 'jump')
    }

    expect(one.asked).toHaveLength(asked)
    expect(one.opened).toStrictEqual([])
    expect(one.ran).toStrictEqual([])
    expect(one.made).toStrictEqual([])
    expect(one.joined).toStrictEqual([])
  })
})

describe('a plex that travelled', () => {
  it('calls the notes it arrives at nothing it called the ones it left', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    const left = one.picture()?.nodes.map((node) => node.id) ?? []

    await one.travels('Cold.md', ['Ice.md', 'child'])

    const arrived = one.picture()?.nodes.map((node) => node.id) ?? []
    expect(arrived).toHaveLength(2)
    expect(arrived.filter((id) => left.includes(id))).toStrictEqual([])
    // A node of the picture it left stands for nothing, so a gesture dragging
    // one asks the vault about nothing.
    for (const node of left) one.state.activate(node)
    expect(one.asked.at(-1)).toBe('Cold.md')
  })

  it('calls a note that came back something other than what it called it', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    const root = one.node('Root')
    const before = one.node('Heat')

    await one.travels('Root.md')
    expect(one.node('Heat')).toBe('')
    await one.travels('Root.md', ['Heat.md', 'child'])

    expect(one.node('Heat')).not.toBe(before)
    expect(one.node('Root')).toBe(root)
  })

  it('does not cross a note that moved with an answer already on its way', async () => {
    const one = await inVault('One.md', [['Heat.md', 'child']])
    const before = one.nodes()

    // The vault is asked about the note where it was, and the file moves while
    // that answer is on its way.
    one.holdAnswers()
    const asking = one.view.go('One.md')
    one.follows({ from: 'One.md', to: 'moved/One.md' }, { from: 'Heat.md', to: 'moved/Heat.md' })
    one.answers()
    await asking
    one.picture()

    await one.view.go(one.view.here.value)

    expect(one.nodes()).toStrictEqual(before)
    expect(one.view.here.value).toBe('moved/One.md')
  })
})

/** A kind that is not a plex, for the person to be in a tab of. */
const other: AnyTabKind = {
  kind: 'other',
  opens: () => ({}),
  called: () => 'Other',
  draws: {},
}

/**
 * The plexes of a window, each standing where it was told to.
 *
 * A note is asked for somewhere the window cannot see — the palette, an agent
 * working the vault beside the person — and one of these has to take it.
 */
const window = (opening = 'Opening.md') => {
  const views: ReturnType<typeof viewOn>[] = []
  /** Every time the vault was asked where it opens, and what it answered then. */
  const asked: string[] = []
  /** Whether a node hangs the parts of its note, which a test turns. */
  const hangs = ref(true)
  /** Every question the vault was asked about what the notes hold. */
  const insides: (readonly string[])[] = []
  const first = ref(opening)

  const makes = () => {
    const view = viewOn('')
    views.push(view)
    return view.view
  }
  const held = useWindowTabs()
  const plexes = plexKind(held.handle, makes, {
    makes: createVault().makes,
    ready: ref(true),
    hangs,
    parts: ref(6),
    opens: () => {},
    inside: async (paths) => {
      insides.push(paths)
      return new Map()
    },
    asks: () => {},
    runs: () => {},
    opening: first,
    first: async () => {
      asked.push(first.value)
      return first.value
    },
    dragged: ref([]),
    says: () => {},
    writes: async () => '',
    creatable: ['parent', 'child', 'jump'],
  })
  held.declares([plexes.kind, other])

  /** A plex tab of this window, opened on what it was given. */
  const openTab = async (at = '') => {
    const id = await held.opens(PLEX, at)
    return { id, state: held.holdsIn<PlexTabState>(id, PLEX)! }
  }
  /** The person is in this tab now. */
  const enters = (id: string) => held.shown(id)
  /** The tab closes, and the window lets go of what it held. */
  const shuts = (id: string) => held.shut(id)
  /** The vault gained a note, which is what it opens with from now on. */
  const gains = (path: string) => {
    first.value = path
  }
  const onScreen = () => panesOf(held.layout.value.root).flatMap((pane) => pane.tabs)
  /** The tab the person is in, which is the active tab of the pane they are in. */
  const active = () =>
    paneById(held.layout.value.root, held.layout.value.focus)?.active ?? ''
  /** A tab holding no plex, opened in front of the person. */
  const elsewhere = () => held.opens(other.kind)
  return {
    ...plexes,
    openTab,
    enters,
    shuts,
    gains,
    onScreen,
    active,
    elsewhere,
    views,
    asked,
    hangs,
    insides,
  }
}

describe('a plex tab as it opens', () => {
  it('stands on the note the vault opens with', async () => {
    const one = window('Opening.md')

    const { state } = await one.openTab()

    expect(state.view.here.value).toBe('Opening.md')
  })

  it('stands where it was told to, whatever the vault opens with', async () => {
    const one = window('Opening.md')

    const { state } = await one.openTab('Told.md')

    expect(state.view.here.value).toBe('Told.md')
  })

  it('stands where the person is looking, when another one is open', async () => {
    const one = window('Opening.md')
    const first = await one.openTab()
    await first.state.view.go('Here.md')

    const second = await one.openTab()

    expect(second.state.view.here.value).toBe('Here.md')
  })

  it('stands nowhere while the vault opens with nothing', async () => {
    const one = window('')

    expect((await one.openTab()).state.view.here.value).toBe('')
  })
})

describe('the plex the person is looking at', () => {
  it('is the one they were last in', async () => {
    const one = window()
    const first = await one.openTab('One.md')
    const second = await one.openTab('Two.md')

    one.enters(first.id)
    expect(one.looking()).toBe('One.md')

    one.enters(second.id)
    expect(one.looking()).toBe('Two.md')
  })

  it('does not carry the error of a tab that closed to the one before it', async () => {
    const one = window()
    const first = await one.openTab('One.md')
    const second = await one.openTab('Two.md')
    second.state.view.error.value = 'Two.md is not in the vault'

    one.shuts(second.id)

    expect(one.looking()).toBe('One.md')
    expect(first.state.view.error.value).toBe('')
  })

  it('is the one before it when the tab in front closes', async () => {
    const one = window()
    const first = await one.openTab('One.md')
    const second = await one.openTab('Two.md')
    const third = await one.openTab('Three.md')
    one.enters(second.id)
    one.enters(third.id)

    one.shuts(third.id)

    expect(one.looking()).toBe('Two.md')
    expect(first.state.view.here.value).toBe('One.md')
  })

  it('is nothing at all in a window holding no plex', () => {
    const one = window()

    expect(one.looking()).toBe('')
  })
})

describe('a note put in front of the person', () => {
  it('is where the plex they are looking at travels', async () => {
    const one = window()
    const first = await one.openTab('One.md')
    const second = await one.openTab('Two.md')
    one.enters(second.id)

    await one.travel('Wanted.md')

    expect(second.state.view.here.value).toBe('Wanted.md')
    expect(first.state.view.here.value).toBe('One.md')
  })

  it('opens a plex of its own in a window holding none', async () => {
    const one = window()

    await one.travel('Wanted.md')

    expect(one.onScreen()).toHaveLength(1)
    expect(one.looking()).toBe('Wanted.md')
  })
})

describe('a note that is no longer in the vault', () => {
  it('leaves every plex standing on it somewhere else', async () => {
    const one = window()
    const first = await one.openTab('Gone.md')
    const second = await one.openTab('Gone.md')
    const third = await one.openTab('Elsewhere.md')

    await one.leaves('Gone.md', 'Root.md')

    expect(first.state.view.here.value).toBe('Root.md')
    expect(second.state.view.here.value).toBe('Root.md')
    expect(third.state.view.here.value).toBe('Elsewhere.md')
  })

  it('leaves a window holding no plex at all alone', async () => {
    const one = window()

    await expect(one.leaves('Gone.md', 'Root.md')).resolves.toBeUndefined()
  })

  it('brings the plex in front of the person, who was in another tab', async () => {
    const one = window()
    const plex = await one.openTab('One.md')
    await one.elsewhere()

    await one.travel('Wanted.md')

    expect(one.active()).toBe(plex.id)
    expect(plex.state.view.here.value).toBe('Wanted.md')
  })
})

describe('every plex asked for its picture again', () => {
  it('asks for the note it is standing on, each of its own', async () => {
    const one = window()
    await one.openTab('One.md')
    const second = await one.openTab('Two.md')

    await one.again()

    expect(one.views[0]?.went).toContain('One.md')
    expect(one.views[1]?.went).toContain('Two.md')
    expect(second.state.view.here.value).toBe('Two.md')
  })

  it('stands a plex on where the note under it went', async () => {
    const one = window()
    const plex = await one.openTab('One.md')

    await one.again([{ from: 'One.md', to: 'Renamed.md' }])

    expect(plex.state.view.here.value).toBe('Renamed.md')
    expect(one.views[0]?.went.at(-1)).toBe('Renamed.md')
  })

  it('leaves the note under it called what the picture called it', async () => {
    const one = window()
    const plex = await one.openTab('One.md')
    const before = plex.state.picture.value?.nodes.map((node) => node.id)

    await one.again([{ from: 'One.md', to: 'Renamed.md' }])

    expect(plex.state.picture.value?.nodes.map((node) => node.id)).toStrictEqual(before)
  })

  it('leaves a plex standing on a note nothing moved', async () => {
    const one = window()
    const plex = await one.openTab('One.md')

    await one.again([{ from: 'Other.md', to: 'Renamed.md' }])

    expect(plex.state.view.here.value).toBe('One.md')
  })

  it('gives one standing nowhere the note an empty vault has just gained', async () => {
    const one = window('')
    const { state } = await one.openTab()
    one.gains('First.md')

    await one.again()

    expect(state.view.here.value).toBe('First.md')
  })

  it('asks the vault where it opens once, however many stand nowhere', async () => {
    const one = window('')
    await one.openTab()
    await one.openTab()
    await one.openTab()

    await one.again()

    expect(one.asked).toHaveLength(1)
  })

  it('asks it not at all while every one of them is standing somewhere', async () => {
    const one = window()
    await one.openTab('One.md')
    await one.openTab('Two.md')

    await one.again()

    expect(one.asked).toStrictEqual([])
  })

  it('asks nothing for a plex whose tab has closed', async () => {
    const one = window()
    const first = await one.openTab('One.md')
    const second = await one.openTab('Two.md')
    one.shuts(second.id)

    await one.again()

    expect(one.views[0]?.went).toContain('One.md')
    expect(one.views[1]?.went.filter((where) => where === 'Two.md')).toHaveLength(1)
    expect(first.state.view.here.value).toBe('One.md')
  })
})

describe('a plex tab the person has closed', () => {
  it('asks the vault nothing when a setting it once answered turns', async () => {
    const one = window()
    const plex = await one.openTab('One.md')
    one.hangs.value = false
    await settles()

    one.shuts(plex.id)
    one.insides.length = 0
    one.hangs.value = true
    await settles()

    expect(one.insides).toStrictEqual([])
  })
})

describe('what a node stands for', () => {
  it('is what the vault says of the note the node draws', () => {
    const one = tab('Root.md', ['Deck.md', 'Stencil.md'], true, {
      'Deck.md': 'deck',
      'Stencil.md': 'stencil',
    })

    expect(one.state.typeOf(one.node('Deck.md'))).toBe('deck')
    expect(one.state.typeOf(one.node('Stencil.md'))).toBe('stencil')
    expect(one.state.typeOf(one.node('Root.md'))).toBe('note')
  })
})

describe('what a command asked over a plex tab is over', () => {
  it('is the note the plex is standing on, under the name the picture gives it', async () => {
    const one = window()
    const { state } = await one.openTab('physics/Ontology.md')

    expect(one.kind.over!(state)).toStrictEqual({
      path: 'physics/Ontology.md',
      title: 'physics/Ontology',
    })
  })

  it('is no note at all while the plex stands nowhere', async () => {
    const one = window('')
    const { state } = await one.openTab()

    expect(one.kind.over!(state)).toStrictEqual({ path: '', title: '' })
  })
})

describe('what a plex tab holds, as whoever answers for the person is told it', () => {
  it('is the note it is standing on', async () => {
    const one = window()
    const { state } = await one.openTab('Root.md')

    expect(one.kind.attends!(state)).toStrictEqual({ path: 'Root.md' })
  })
})

describe('what a plex tab is called', () => {
  it('is the note it stands on', async () => {
    const one = window()
    const { state } = await one.openTab('Root.md')

    expect(one.kind.called?.(state)).toBe('Root')
  })

  it('is the word for a plex while it stands nowhere', async () => {
    const one = window('')
    const { state } = await one.openTab()

    expect(one.kind.called?.(state)).toBe('Plex')
  })
})

