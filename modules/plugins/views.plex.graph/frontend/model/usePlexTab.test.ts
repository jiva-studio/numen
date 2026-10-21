/**
 * What a gesture in a plex evaluates to, asked without a screen.
 *
 * A note made from a node other than the focus is the one to watch: it is
 * written into the vault whatever happens, and a picture one relation deep draws
 * it only from the node it was made from.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { usePlexTab, type PlexTabState, type PlexTabDeps } from './usePlexTab'
import { ITEMS, NEW_NOTE } from '../lib/menu'
import { usePlexView as viewing, type PlexView } from './usePlexView'
import { WORDS as words } from '../words'
import { createVault, settle, viewOn, type Types, type NeighbourSpec } from '../fixtures'
import type { NoteHeading, NoteRelation, PathRename } from '../types'
import { getRenamedPath } from '@numen/plugin-sdk'

/** A plex tab with the window it is drawn in written down. */
const tab = (
  at: string,
  neighbours: readonly NeighbourSpec[] = [],
  takes = true,
  types: Types = {},
) => {
  const plex = viewOn(at, neighbours, types)
  const vault = createVault(takes)
  const opened: [string, string, string][] = []
  const asked: string[] = []
  const ran: [string, string, string][] = []
  const messages: string[] = []
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
  /** Whether a node nests the headings of its note, which a test turns. */
  const isNesting = ref(true)
  /** Every question the vault was asked about what the notes hold. */
  const headingsAsked: (readonly string[])[] = []
  const deps: PlexTabDeps = {
    editor: vault.editor,
    isReady: ref(true),
    isNesting,
    headings: ref(6),
    openNote: (path, options) => {
      opened.push([path, options?.title ?? '', options?.showing ?? 'here'])
      if (options?.line !== undefined) entered.push([path, options.line])
    },
    // The vault answers about the notes it was asked about and no others.
    readHeadings: async (paths) => {
      headingsAsked.push(paths)
      return new Map([...divides.value].filter(([path]) => paths.includes(path)))
    },
    askAgent: (text) => asked.push(text),
    runCommand: (id, path, title) => ran.push([id, path, title]),
    openingPath: ref('Opening.md'),
    readOpeningPath: async () => 'Opening.md',
    dragged: dragging,
    showMessage: (text) => messages.push(text),
    createUntitledNote: async () => {
      wrote.push(writes.value)
      return writes.value
    },
    creatable: ['parent', 'child', 'jump'],
  }
  const state = usePlexTab(plex.view, deps)
  /** What the picture calls a note, which is what a gesture in it carries. */
  const node = (path: string) => getNode(state, path.replace(/\.md$/, ''))
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
    isNesting,
    headingsAsked,
    asked,
    ran,
    messages,
  }
}

/** The node of the picture drawn for the note of this title, if it draws one. */
const getNode = (state: PlexTabState, title: string): string =>
  state.picture.value?.nodes.find((node) => node.title === title)?.id ?? ''

describe('a note made from a node', () => {
  it('remains the plex on the node it was made from', async () => {
    const one = tab('Root.md', ['Child.md'])

    await one.state.createNode(one.node('Child.md'), 'child')

    expect(one.made).toEqual([['Child.md', 'child']])
    expect(one.went).toEqual(['Child.md'])
  })

  it('leaves the plex where it is when that node is the focus', async () => {
    const one = tab('Root.md')

    await one.state.createNode(one.node('Root.md'), 'child')

    expect(one.went).toEqual(['Root.md'])
    expect(one.state.view.here.value).toBe('Root.md')
  })

  it('travels nowhere when the vault made nothing', async () => {
    const one = tab('Root.md', ['Child.md'], false)

    await one.state.createNode(one.node('Child.md'), 'parent')

    expect(one.went).toEqual([])
  })
})

describe('two notes a line was drawn between', () => {
  it('remains the plex on the note the line was drawn from', async () => {
    const one = tab('Root.md', ['Child.md', 'Other.md'])

    await one.state.joinNodes(one.node('Child.md'), one.node('Other.md'), 'jump')

    expect(one.joined).toEqual([['Child.md', 'Other.md', 'jump']])
    expect(one.went).toEqual(['Child.md'])
  })

  it('travels nowhere when nothing was written', async () => {
    const one = tab('Root.md', ['Child.md'], false)

    await one.state.joinNodes(one.node('Child.md'), one.node('Root.md'), 'jump')

    expect(one.went).toEqual([])
  })
})

describe('the menu on a node', () => {
  const createMenuRequest = (node: string) => ({
    node,
    at: { x: 1, y: 2 },
    opening: 'pointer' as const,
  })

  it('passes the command the note it was requested for on, called what the picture calls it', () => {
    const one = tab('Root.md', ['Child.md'])
    one.state.openMenu(createMenuRequest(one.node('Child.md')))

    one.state.chooseMenuItem('read')

    expect(one.ran).toEqual([['read', 'Child.md', 'Child']])
    expect(one.state.menu.value).toBeNull()
  })

  it('passes over every command it offers, and nothing it does not', () => {
    const one = tab('Root.md', ['Child.md'])
    for (const item of ITEMS) {
      one.state.openMenu(createMenuRequest(one.node('Child.md')))
      one.state.chooseMenuItem(item.id)
    }
    one.state.openMenu(createMenuRequest(one.node('Child.md')))
    one.state.chooseMenuItem('constructor')
    one.state.openMenu(createMenuRequest(one.node('Child.md')))
    one.state.chooseMenuItem('destroy')

    expect(one.ran.map(([id]) => id)).toStrictEqual(ITEMS.map((item) => item.id))
  })

  it('does nothing when it relies on nothing', () => {
    const one = tab('Root.md')

    one.state.chooseMenuItem('read')

    expect(one.ran).toEqual([])
  })

  it('goes when the picture under it does', () => {
    const one = tab('Root.md', ['Child.md'])
    one.state.openMenu(createMenuRequest(one.node('Child.md')))

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
      followMoves: () => {},
      close: () => {},
    }
    return usePlexTab(view as unknown as PlexView, {
      editor: createVault().editor,
      isReady: ref(true),
      isNesting: ref(true),
      headings: ref(6),
      openNote: () => {},
      readHeadings: async () => new Map(),
      askAgent: () => {},
      runCommand: () => {},
      openingPath: ref(''),
      readOpeningPath: async () => '',
      dragged: ref([]),
      showMessage: () => {},
      createUntitledNote: async () => '',
      creatable: ['parent', 'child', 'jump'],
      ...over,
    })
  }

  it('is an empty vault where the vault is read and opens with no note', () => {
    expect(plex({}).empty.value).toBe(true)
  })

  it('is not an empty vault where the vault could not be opened', () => {
    expect(plex({ isReady: ref(false) }).empty.value).toBe(false)
  })

  it('is not an empty vault while the first answer is on its way', () => {
    expect(plex({ openingPath: ref('Opening.md') }).empty.value).toBe(false)
  })

  it('is not an empty vault once the plex relies on a note', () => {
    expect(tab('Root.md').state.empty.value).toBe(false)
  })
})

describe('the menu off every node', () => {
  const asked = { node: null, at: { x: 1, y: 2 }, opening: 'pointer' as const }

  it('makes a note, and the plex relies on it', async () => {
    const one = tab('Root.md')
    one.state.openMenu(asked)

    one.state.chooseMenuItem(NEW_NOTE)
    await settle()

    expect(one.wrote).toStrictEqual(['Untitled note.md'])
    expect(one.went).toContain('Untitled note.md')
    expect(one.state.menu.value).toBeNull()
  })

  it('remains where it stood where the vault made none', async () => {
    const one = tab('Root.md')
    one.writes.value = ''
    one.state.openMenu(asked)

    one.state.chooseMenuItem(NEW_NOTE)
    await settle()

    expect(one.went).toStrictEqual([])
  })

  it('makes nothing of a command over a note, there being no note it is over', async () => {
    const one = tab('Root.md')
    one.state.openMenu(asked)

    one.state.chooseMenuItem('read')
    await settle()

    expect(one.wrote).toStrictEqual([])
    expect(one.ran).toStrictEqual([])
  })
})

describe('what a note in the picture is called', () => {
  it('is the title the vault gave the focus', () => {
    expect(tab('Root.md').state.getName('Root.md')).toBe('Root')
  })

  it('is the title of a note around it', () => {
    expect(tab('Root.md', ['Deep/Child.md']).state.getName('Deep/Child.md')).toBe('Deep/Child')
  })

  it('is the file it is filed under, for a note the picture does not name', () => {
    expect(tab('Root.md').state.getName('Deep/Elsewhere.md')).toBe('Elsewhere')
  })
})

describe('the picture', () => {
  it('is nothing while the window has nothing true to draw', () => {
    const plex = viewOn('Root.md')
    const state = usePlexTab(plex.view, {
      editor: createVault().editor,
      isReady: ref(false),
      isNesting: ref(true),
      headings: ref(6),
      openNote: () => {},
      readHeadings: async () => new Map(),
      askAgent: () => {},
      runCommand: () => {},
      openingPath: ref(''),
      readOpeningPath: async () => '',
      dragged: ref(['Entropy.md']),
      showMessage: () => {},
      createUntitledNote: async () => '',
      creatable: ['parent', 'child', 'jump'],
    })

    expect(state.picture.value).toBeNull()
    // Nothing is drawn, so nothing can be dragged onto it.
    expect(state.dragged.value).toStrictEqual([])
  })
})

describe('notes dragged in and let go over the picture', () => {
  it('writes the link into the note the plex relies on, naming the dragged one second', async () => {
    const one = tab('Root.md', ['Child.md'])

    await one.state.dropNodes(['physics/Entropy.md'], 'child')

    expect(one.joined).toEqual([['Root.md', 'physics/Entropy.md', 'child']])
    expect(one.went).toEqual(['Root.md'])
  })

  it('writes one for each of them, to the one note in the one relation', async () => {
    const one = tab('Root.md')

    await one.state.dropNodes(['Entropy.md', 'Kelvin.md', 'Heat.md'], 'child')

    expect(one.joined).toEqual([
      ['Root.md', 'Entropy.md', 'child'],
      ['Root.md', 'Kelvin.md', 'child'],
      ['Root.md', 'Heat.md', 'child'],
    ])
    expect(one.went).toEqual(['Root.md'])
  })

  it('takes the relation the drag named, whichever it was', async () => {
    const one = tab('Root.md')

    await one.state.dropNodes(['Entropy.md'], 'parent')
    await one.state.dropNodes(['Heat.md'], 'jump')

    expect(one.joined).toEqual([
      ['Root.md', 'Entropy.md', 'parent'],
      ['Root.md', 'Heat.md', 'jump'],
    ])
  })

  it('writes the rest where one of them failed, and says which stayed', async () => {
    const one = tab('Root.md')
    one.notJoinedPaths.add('Kelvin.md')

    await one.state.dropNodes(['Entropy.md', 'Kelvin.md', 'Heat.md'], 'child')

    expect(one.joined).toEqual([
      ['Root.md', 'Entropy.md', 'child'],
      ['Root.md', 'Kelvin.md', 'child'],
      ['Root.md', 'Heat.md', 'child'],
    ])
    expect(one.messages).toEqual([`${words.notJoined} Kelvin`])
    expect(one.went).toEqual(['Root.md'])
  })

  it('says every one of them where the vault would write none, and travels nowhere', async () => {
    const one = tab('Root.md', [], false)

    await one.state.dropNodes(['Entropy.md', 'Heat.md'], 'child')

    expect(one.messages).toEqual([`${words.notJoined} Entropy, Heat`])
    expect(one.went).toEqual([])
  })

  it('joins nothing where the plex has nowhere to stand', async () => {
    const one = tab('')

    await one.state.dropNodes(['Entropy.md'], 'child')

    expect(one.joined).toEqual([])
    expect(one.messages).toEqual([])
  })

  it('leaves the note the plex relies on out, and joins the rest', async () => {
    const one = tab('Root.md')

    await one.state.dropNodes(['Root.md', 'Entropy.md'], 'child')

    expect(one.joined).toEqual([['Root.md', 'Entropy.md', 'child']])
    expect(one.messages).toEqual([])
  })
})

describe('what the picture draws a line to', () => {
  it('is what the window says it is dragging', () => {
    const one = tab('Root.md')
    one.dragging.value = ['physics/Entropy.md', 'physics/Kelvin.md']

    expect(one.state.dragged.value).toStrictEqual(['physics/Entropy.md', 'physics/Kelvin.md'])
  })

  it('is nothing while the window is dragging none', () => {
    expect(tab('Root.md').state.dragged.value).toStrictEqual([])
  })

  it('leaves out the note the plex relies on, and keeps the rest', () => {
    const one = tab('Root.md')
    one.dragging.value = ['Root.md', 'Entropy.md']

    expect(one.state.dragged.value).toStrictEqual(['Entropy.md'])
  })
})

describe('the headings a node nests', () => {
  /** One heading of a note, as the vault answers one. */
  const heading = (text: string, line: number, level = 1): NoteHeading => ({ text, level, line })

  it('are what the vault returned that note is divided into', async () => {
    const one = tab('Root.md', ['Child.md'])
    one.divides.value = new Map([['Child.md', [heading('Heat', 4), heading('Cold', 9, 2)]]])

    await one.state.readHeadings()

    expect(one.state.getHeadings(one.node('Child.md'))).toStrictEqual([
      { id: '4', text: 'Heat', level: 1 },
      { id: '9', text: 'Cold', level: 2 },
    ])
  })

  it('are none for a node the vault returned nothing about', async () => {
    const one = tab('Root.md', ['Child.md'])
    one.divides.value = new Map([['Child.md', [heading('Heat', 4)]]])

    await one.state.readHeadings()

    expect(one.state.getHeadings(one.node('Root.md'))).toStrictEqual([])
  })

  it('are each named by the line it relies on', async () => {
    const one = tab('Root.md')
    one.divides.value = new Map([['Root.md', [heading('Heat', 12)]]])

    await one.state.readHeadings()

    expect(one.state.getHeadings(one.node('Root.md')).map((heading) => heading.id)).toStrictEqual([
      '12',
    ])
  })

  it('are requested for again once the plex remains somewhere else', async () => {
    const one = tab('Root.md', ['Child.md'])
    one.divides.value = new Map([['Other.md', [heading('Heat', 4)]]])

    await one.state.view.go('Other.md')
    await settle()

    expect(one.state.getHeadings(one.node('Other.md'))).toStrictEqual([
      { id: '4', text: 'Heat', level: 1 },
    ])
  })

  it('are what the question requested last came back with, whatever order they land in', async () => {
    // The first answer is held up until the second has settled, which is a
    // change followed while a travel is still out.
    const answers: ((held: ReadonlyMap<string, readonly NoteHeading[]>) => void)[] = []
    const one = tab('Root.md')
    const plex = usePlexTab(one.state.view, {
      editor: createVault().editor,
      isReady: ref(true),
      isNesting: ref(true),
      headings: ref(6),
      openNote: () => {},
      readHeadings: () => new Promise((done) => answers.push(done)),
      askAgent: () => {},
      runCommand: () => {},
      openingPath: ref('Root.md'),
      readOpeningPath: async () => 'Root.md',
      dragged: ref([]),
      showMessage: () => {},
      createUntitledNote: async () => '',
      creatable: ['parent', 'child', 'jump'],
    })
    const node = () => getNode(plex, 'Root')

    // The picture asks once as it is built, and that answer is left pending.
    const asked = answers.length
    void plex.readHeadings()
    void plex.readHeadings()
    answers[asked + 1]?.(new Map([['Root.md', [heading('Fresh', 2)]]]))
    answers[asked]?.(new Map([['Root.md', [heading('Stale', 1)]]]))
    await settle()

    expect(plex.getHeadings(node())).toStrictEqual([{ id: '2', text: 'Fresh', level: 1 }])
  })

  it('are none for every node while the setting is off', async () => {
    const one = tab('Root.md', ['Child.md'])
    one.divides.value = new Map([['Child.md', [heading('Heat', 4)]]])
    await one.state.readHeadings()
    one.isNesting.value = false

    expect(one.state.getHeadings(one.node('Child.md'))).toStrictEqual([])
    expect(one.state.getHeadings(one.node('Root.md'))).toStrictEqual([])
  })

  it('are not requested of the vault at all while the setting is off', async () => {
    const one = tab('Root.md', ['Child.md'])
    one.isNesting.value = false
    one.headingsAsked.length = 0

    await one.state.readHeadings()

    expect(one.headingsAsked).toStrictEqual([])
  })

  it('are none for a deck, whose cards stand on no line of prose', async () => {
    const one = tab('Root.md', ['Animals.md'], true, { 'Animals.md': 'deck' })
    one.divides.value = new Map([['Animals.md', [heading('Vicuña', 4)]]])
    one.headingsAsked.length = 0

    await one.state.readHeadings()

    expect(one.state.getHeadings(one.node('Animals.md'))).toStrictEqual([])
    expect(one.headingsAsked).toStrictEqual([['Root.md']])
  })

  it('are none for a stencil, whose faces stand on no line of prose', async () => {
    const one = tab('Root.md', ['Animal.md'], true, { 'Animal.md': 'stencil' })
    one.divides.value = new Map([['Animal.md', [heading('Front', 4)]]])
    one.headingsAsked.length = 0

    await one.state.readHeadings()

    expect(one.state.getHeadings(one.node('Animal.md'))).toStrictEqual([])
    expect(one.headingsAsked).toStrictEqual([['Root.md']])
  })

  it('are the headings of an ordinary note alongside them', async () => {
    const one = tab('Root.md', ['Animals.md', 'Child.md'], true, {
      'Animals.md': 'deck',
    })
    one.divides.value = new Map([['Child.md', [heading('Heat', 4)]]])

    await one.state.readHeadings()

    expect(one.state.getHeadings(one.node('Child.md'))).toStrictEqual([
      { id: '4', text: 'Heat', level: 1 },
    ])
  })

  it('are none for a deck in focus, which is where the plex is standing', async () => {
    const one = tab('Animals.md', [], true, { 'Animals.md': 'deck' })
    one.divides.value = new Map([['Animals.md', [heading('Vicuña', 4)]]])
    one.headingsAsked.length = 0

    await one.state.readHeadings()

    expect(one.state.getHeadings(one.node('Animals.md'))).toStrictEqual([])
    expect(one.headingsAsked).toStrictEqual([])
  })

  it('are hung again once the setting is turned back on', async () => {
    // Turning it is all a person does; nothing else asks for them again.
    const one = tab('Root.md', ['Child.md'])
    one.divides.value = new Map([['Child.md', [heading('Heat', 4)]]])

    one.isNesting.value = false
    await settle()
    expect(one.state.getHeadings(one.node('Child.md'))).toStrictEqual([])

    one.isNesting.value = true
    await settle()

    expect(one.state.getHeadings(one.node('Child.md'))).toStrictEqual([
      { id: '4', text: 'Heat', level: 1 },
    ])
  })
})

describe('a heading of a node chosen', () => {
  it('opens the note it remains in, and puts the keyboard on its line', () => {
    const one = tab('Root.md')

    one.state.openHeading(one.node('Root.md'), '12')

    expect(one.opened).toStrictEqual([['Root.md', 'Root', 'here']])
    expect(one.entered).toStrictEqual([['Root.md', 12]])
  })

  it('opens the note the node remains for, the focus being another one', () => {
    const one = tab('Root.md', ['Child.md'])

    one.state.openHeading(one.node('Child.md'), '7')

    expect(one.opened).toStrictEqual([['Child.md', 'Child', 'here']])
    expect(one.entered).toStrictEqual([['Child.md', 7]])
  })

  it('remains the plex where it stood', () => {
    const one = tab('Root.md', ['Child.md'])

    one.state.openHeading(one.node('Child.md'), '7')

    expect(one.went).toStrictEqual([])
    expect(one.state.view.here.value).toBe('Root.md')
  })

  it('opens nothing for a heading naming no line, which is the one standing for the rest', () => {
    const one = tab('Root.md')

    one.state.openHeading(one.node('Root.md'), '')

    expect(one.opened).toStrictEqual([])
    expect(one.entered).toStrictEqual([])
  })
})

/** A note beside another: where it is located, and the note it comes through. */
type NeighbourRow = readonly [path: string, relation: NoteRelation, through?: string]

/** What a note is shown by, which a move of its file leaves alone. */
const getNoteTitle = (path: string) => (path.split('/').pop() ?? path).replace(/\.md$/, '')

/**
 * A plex standing in a vault whose files a person can move.
 *
 * The vault answers what is around whatever note it is asked about, and no note
 * in it carries an identifier — this is the vault of somebody who writes in
 * another editor. A move renames its files and is told to the plex as a change
 * arriving tells it.
 */
const inVault = async (focus: string, beside: readonly NeighbourRow[] = []) => {
  let aroundVal = beside.map(([path, relation, through]) => ({
    path,
    relation,
    through: through ?? '',
  }))
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
        focus: { path, title: getNoteTitle(path) },
        focusType: 'note' as const,
        related: around.map((one) => ({
          path: one.path,
          title: getNoteTitle(one.path),
          type: 'note' as const,
          relation: one.relation,
          label: '',
          through: one.through ?? '',
          isMutual: false,
        })),
      }
    },
  })
  const state = usePlexTab(view, {
    editor: vault.editor,
    isReady: ref(true),
    isNesting: ref(true),
    headings: ref(6),
    openNote: (path, options) =>
      opened.push([path, options?.title ?? '', options?.showing ?? 'here']),
    readHeadings: async () => new Map(),
    askAgent: () => {},
    runCommand: (id, path, title) => ran.push([id, path, title]),
    openingPath: ref(''),
    readOpeningPath: async () => '',
    dragged: ref([]),
    showMessage: () => {},
    createUntitledNote: async () => '',
    creatable: ['parent', 'child', 'jump'],
  })
  await view.go(focus)

  /** The picture currently, which is what a drawn window reads. */
  const picture = () => state.picture.value
  /** What the picture calls the note shown by this title. */
  const node = (title: string) => getNode(state, title)
  /** Every node of the picture, as the relation it sits in and what it is called. */
  const nodes = () => (picture()?.nodes ?? []).map((one) => `${one.relation} ${one.id}`)
  /** Every edge of the picture, as the two nodes it runs between. */
  const edges = () => (picture()?.edges ?? []).map((one) => `${one.from} -> ${one.to}`)

  /** Files moved in the vault, and the plex told what went where. */
  const moveFiles = (...renamed: readonly PathRename[]) => {
    around = around.map((one) => ({
      ...one,
      path: getRenamedPath(renamed, one.path) || one.path,
      through: getRenamedPath(renamed, one.through) || one.through,
    }))
    state.followMoves(renamed)
  }
  /** A move, followed by the picture being asked for again. */
  const moveFilesAndReload = async (...renamed: readonly PathRename[]) => {
    moveFiles(...renamed)
    await view.go(view.here.value)
  }
  /** The person travels to another note, which is a picture of its own. */
  const goToNote = async (path: string, ...now: readonly NeighbourRow[]) => {
    around = now.map(([at, relation, through]) => ({ path: at, relation, through: through ?? '' }))
    await view.go(path)
  }

  return {
    state,
    view,
    picture,
    node,
    nodes,
    edges,
    moveFiles,
    moveFilesAndReload,
    goToNote,
    asked,
    opened,
    ran,
    made: vault.made,
    joined: vault.joined,
    /** The vault holds every answer back until it is release. */
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

    await one.moveFilesAndReload({ from: 'Entropy.md', to: 'physics/Entropy.md' })

    expect(one.nodes()).toStrictEqual(before)
  })

  it('keeps it where the note that moved is one around the note in focus', async () => {
    const one = await inVault('Entropy.md', [['Heat.md', 'child']])
    const before = one.node('Heat')

    await one.moveFilesAndReload({ from: 'Heat.md', to: 'physics/Heat.md' })

    expect(one.node('Heat')).toBe(before)
    expect(one.view.here.value).toBe('Entropy.md')
  })

  it('keeps it where the note that moved is the one in focus', async () => {
    const one = await inVault('Entropy.md', [['Heat.md', 'child']])
    const before = one.node('Entropy')

    await one.moveFilesAndReload({ from: 'Entropy.md', to: 'physics/Entropy.md' })

    expect(one.node('Entropy')).toBe(before)
    expect(one.view.here.value).toBe('physics/Entropy.md')
  })

  it('keeps it for every note of a folder that moved at once', async () => {
    const one = await inVault('physics/Entropy.md', [
      ['physics/Heat.md', 'child'],
      ['physics/Cold.md', 'child'],
    ])
    const before = one.nodes()

    await one.moveFilesAndReload(
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

    await one.moveFilesAndReload({ from: 'One.md', to: 'Two.md' }, { from: 'Two.md', to: 'One.md' })

    const [, now, later] = one.picture()?.nodes ?? []
    expect(now?.id).toBe(first?.id)
    expect(later?.id).toBe(second?.id)
    expect([now?.title, later?.title]).toStrictEqual(['Two', 'One'])
  })

  it('keeps it through a second move', async () => {
    const one = await inVault('Entropy.md', [['Heat.md', 'child']])
    const before = one.nodes()

    await one.moveFilesAndReload({ from: 'Heat.md', to: 'physics/Heat.md' })
    await one.moveFilesAndReload({ from: 'physics/Heat.md', to: 'thermo/Heat.md' })

    expect(one.nodes()).toStrictEqual(before)
  })

  it('runs the same edge between the same two nodes', async () => {
    const one = await inVault('Entropy.md', [
      ['Physics.md', 'parent'],
      ['Heat.md', 'sibling', 'Physics.md'],
    ])
    const before = one.edges()
    expect(before).toHaveLength(2)

    await one.moveFilesAndReload(
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

    await one.moveFilesAndReload({ from: 'Heat.md', to: 'physics/Heat.md' })

    const drawn = one.picture()?.nodes.map((node) => node.id) ?? []
    expect(new Set(drawn).size).toBe(4)
  })
})

describe('a gesture the plex reports', () => {
  it('travels to the note the node remains for', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    await one.moveFilesAndReload({ from: 'Heat.md', to: 'physics/Heat.md' })

    one.state.activate(one.node('Heat'))

    expect(one.asked.at(-1)).toBe('physics/Heat.md')
  })

  it('opens the note the node remains for, called what the picture calls it', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    await one.moveFilesAndReload({ from: 'Heat.md', to: 'physics/Heat.md' })

    one.state.openNode(one.node('Heat'), 'beside')

    expect(one.opened).toStrictEqual([['physics/Heat.md', 'Heat', 'beside']])
  })

  it('runs a command on the note the menu stood on', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    await one.moveFilesAndReload({ from: 'Heat.md', to: 'physics/Heat.md' })

    one.state.openMenu({ node: one.node('Heat'), at: { x: 1, y: 2 }, opening: 'pointer' })
    one.state.chooseMenuItem('read')

    expect(one.ran).toStrictEqual([['read', 'physics/Heat.md', 'Heat']])
  })

  it('makes a note in a relation of the note the node remains for', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    await one.moveFilesAndReload({ from: 'Heat.md', to: 'physics/Heat.md' })

    await one.state.createNode(one.node('Heat'), 'child')

    expect(one.made).toStrictEqual([['physics/Heat.md', 'child']])
  })

  it('joins the two notes a line was drawn between', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    await one.moveFilesAndReload({ from: 'Heat.md', to: 'physics/Heat.md' })

    await one.state.joinNodes(one.node('Heat'), one.node('Root'), 'jump')

    expect(one.joined).toStrictEqual([['physics/Heat.md', 'Root.md', 'jump']])
  })

  it('is release where the picture never drew that node', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    expect(one.node('Heat')).not.toBe('')
    const asked = one.asked.length

    for (const node of ['Heat.md', 'Root.md', 'nothing']) {
      one.state.activate(node)
      one.state.openNode(node, 'here')
      one.state.openMenu({ node, at: { x: 1, y: 2 }, opening: 'pointer' })
      one.state.chooseMenuItem('read')
      await one.state.createNode(node, 'child')
      await one.state.joinNodes(node, one.node('Root'), 'jump')
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

    await one.goToNote('Cold.md', ['Ice.md', 'child'])

    const arrived = one.picture()?.nodes.map((node) => node.id) ?? []
    expect(arrived).toHaveLength(2)
    expect(arrived.filter((id) => left.includes(id))).toStrictEqual([])

    // A node stands for nothing once a navigation has passed it by twice, so
    // a gesture dragging one asks the vault about nothing.
    await one.goToNote('Warm.md', ['Fire.md', 'child'])
    for (const node of left) one.state.activate(node)
    expect(one.asked.at(-1)).toBe('Warm.md')
  })

  it('calls a note that arrives back the navigation right after leaving what it always called it', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    const root = one.node('Root')
    const before = one.node('Heat')

    await one.goToNote('Root.md')
    expect(one.node('Heat')).toBe('')
    await one.goToNote('Root.md', ['Heat.md', 'child'])

    expect(one.node('Heat')).toBe(before)
    expect(one.node('Root')).toBe(root)
  })

  it('calls a note that stayed away two navigations something other than what it called it', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    const before = one.node('Heat')

    await one.goToNote('Root.md')
    await one.goToNote('Cold.md', ['Ice.md', 'child'])
    await one.goToNote('Root.md', ['Heat.md', 'child'])

    expect(one.node('Heat')).not.toBe(before)
  })

  it('does not cross a note that moved with an answer already on its way', async () => {
    const one = await inVault('One.md', [['Heat.md', 'child']])
    const before = one.nodes()

    // The vault is asked about the note where it was, and the file moves while
    // that answer is on its way.
    one.holdAnswers()
    const asking = one.view.go('One.md')
    one.moveFiles({ from: 'One.md', to: 'moved/One.md' }, { from: 'Heat.md', to: 'moved/Heat.md' })
    one.answers()
    await asking
    one.picture()

    await one.view.go(one.view.here.value)

    expect(one.nodes()).toStrictEqual(before)
    expect(one.view.here.value).toBe('moved/One.md')
  })
})

describe('what a node remains for', () => {
  it('is what the vault says of the note the node draws', () => {
    const one = tab('Root.md', ['Deck.md', 'Stencil.md'], true, {
      'Deck.md': 'deck',
      'Stencil.md': 'stencil',
    })

    expect(one.state.getType(one.node('Deck.md'))).toBe('deck')
    expect(one.state.getType(one.node('Stencil.md'))).toBe('stencil')
    expect(one.state.getType(one.node('Root.md'))).toBe('note')
  })
})

describe('quick-link popover from a node handle', () => {
  it('opens quick link request with drop coordinates', async () => {
    const one = tab('Root.md', ['Child.md'])
    const childNode = one.node('Child.md')

    await one.state.createNode(childNode, 'child', { x: 450, y: 320 })

    expect(one.state.quickLink.value).toStrictEqual({
      from: childNode,
      relation: 'child',
      at: { x: 450, y: 320 },
    })
  })

  it('dismisses quick link popover cleanly on dismiss', async () => {
    const one = tab('Root.md')
    one.state.openQuickLink({
      from: one.node('Root.md'),
      relation: 'parent',
      at: { x: 100, y: 200 },
    })

    one.state.dismissQuickLink()

    expect(one.state.quickLink.value).toBeNull()
  })

  it('creates new note and navigates to it when confirmed with title and navigation enabled', async () => {
    const one = tab('Root.md')
    const rootNode = one.node('Root.md')
    one.state.openQuickLink({ from: rootNode, relation: 'child', at: { x: 100, y: 200 } })

    await one.state.confirmQuickLink('SubTopic', false)

    expect(one.state.quickLink.value).toBeNull()
    expect(one.made).toEqual([['Root.md', 'child']])
    expect(one.went).toEqual(['SubTopic.md'])
  })

  it('joins existing note and navigates to it when confirmed as existing note', async () => {
    const one = tab('Root.md', ['Other.md'])
    const rootNode = one.node('Root.md')
    one.state.openQuickLink({ from: rootNode, relation: 'jump', at: { x: 100, y: 200 } })

    await one.state.confirmQuickLink('Other.md', true)

    expect(one.state.quickLink.value).toBeNull()
    expect(one.joined).toEqual([['Root.md', 'Other.md', 'jump']])
    expect(one.went).toEqual(['Other.md'])
  })

  it('remains on source note when navigation on create is disabled', async () => {
    const one = tab('Root.md')
    one.state.isNavigatingOnCreate.value = false
    const rootNode = one.node('Root.md')
    one.state.openQuickLink({ from: rootNode, relation: 'child', at: { x: 100, y: 200 } })

    await one.state.confirmQuickLink('SubTopic', false)

    expect(one.made).toEqual([['Root.md', 'child']])
    expect(one.went).toEqual(['Root.md'])
  })

  it('joins existing note and stays on source note when navigation is disabled', async () => {
    const one = tab('Root.md', ['Other.md'])
    one.state.isNavigatingOnCreate.value = false
    const rootNode = one.node('Root.md')
    one.state.openQuickLink({ from: rootNode, relation: 'jump', at: { x: 100, y: 200 } })

    await one.state.confirmQuickLink('Other.md', true)

    expect(one.joined).toEqual([['Root.md', 'Other.md', 'jump']])
    expect(one.went).toEqual(['Root.md'])
  })

  it('ignores confirmQuickLink when title is empty or quick link is inactive', async () => {
    const one = tab('Root.md')
    await one.state.confirmQuickLink('', false)
    expect(one.made).toEqual([])

    const rootNode = one.node('Root.md')
    one.state.openQuickLink({ from: rootNode, relation: 'child', at: { x: 100, y: 200 } })
    await one.state.confirmQuickLink('   ', false)
    expect(one.made).toEqual([])
  })

  it('searches notes via searchNames dependency', async () => {
    const searchMock = vi.fn().mockResolvedValue([{ path: 'Note.md', title: 'Note' }])
    const plex = viewOn('Root.md', [])
    const vault = createVault()
    const stateWithSearch = usePlexTab(plex.view, {
      editor: vault.editor,
      isReady: ref(true),
      isNesting: ref(true),
      headings: ref(6),
      openNote: () => {},
      readHeadings: async () => new Map(),
      askAgent: () => {},
      runCommand: () => {},
      openingPath: ref('Opening.md'),
      readOpeningPath: async () => 'Opening.md',
      dragged: ref([]),
      showMessage: () => {},
      createUntitledNote: async () => '',
      creatable: ['parent', 'child', 'jump'],
      searchNames: searchMock,
    })

    const results = await stateWithSearch.searchNotes('test')
    expect(searchMock).toHaveBeenCalledWith('test')
    expect(results).toEqual([{ path: 'Note.md', title: 'Note' }])

    const stateNoSearch = usePlexTab(plex.view, {
      editor: vault.editor,
      isReady: ref(true),
      isNesting: ref(true),
      headings: ref(6),
      openNote: () => {},
      readHeadings: async () => new Map(),
      askAgent: () => {},
      runCommand: () => {},
      openingPath: ref('Opening.md'),
      readOpeningPath: async () => 'Opening.md',
      dragged: ref([]),
      showMessage: () => {},
      createUntitledNote: async () => '',
      creatable: ['parent', 'child', 'jump'],
    })
    expect(await stateNoSearch.searchNotes('test')).toEqual([])
  })

  it('handles openLinkInspector with invalid pair formats gracefully', async () => {
    const { state } = tab('Focus.md', ['Child.md'])
    await settle()

    state.openLinkInspector('invalidSingleToken', { x: 0, y: 0 })
    expect(state.linkInspector.value).toBeNull()

    state.openLinkInspector('unknown1 unknown2', { x: 0, y: 0 })
    expect(state.linkInspector.value).toBeNull()
  })

  it('searches notes via searchNotes method', async () => {
    const { state } = tab('Focus.md', ['Child.md'])
    const emptyResults = await state.searchNotes('query')
    expect(emptyResults).toEqual([])
  })

  it('ignores invalid headings in openHeading', () => {
    const { state, node, opened, entered } = tab('Focus.md', ['Child.md'])
    const focusId = node('Focus')

    state.openHeading(focusId, 'not-a-number')
    expect(entered).toEqual([])
    expect(opened).toEqual([])

    state.openHeading('nonexistent', '1')
    expect(entered).toEqual([])
  })

  it('confirms quick link with whitespace title or inactive state without error', async () => {
    const { state } = tab('Focus.md', ['Child.md'])
    await state.confirmQuickLink('  ', false)
    expect(state.quickLink.value).toBeNull()
  })
})

describe('usePlexTab edge cases and helpers', () => {
  it('covers openHeading edge cases, dismissals and menu choices', async () => {
    const one = tab('Root.md', ['Child.md'])
    one.state.openHeading('unknown-node', '12')
    one.state.openHeading(one.node('Root.md'), 'not-a-number')

    one.state.openQuickLink({ from: one.node('Root.md'), relation: 'child', at: { x: 10, y: 10 } })
    expect(one.state.quickLink.value).not.toBeNull()
    const pair = `${one.node('Root.md')} ${one.node('Child.md')}`
    one.state.openLinkInspector(pair, { x: 10, y: 10 })
    expect(one.state.linkInspector.value).not.toBeNull()

    one.state.openMenu({ node: null, at: { x: 10, y: 10 }, opening: 'pointer' })
    one.state.dismiss()
    expect(one.state.menu.value).toBeNull()
    expect(one.state.quickLink.value).toBeNull()
    expect(one.state.linkInspector.value).toBeNull()

    one.state.openMenu({ node: null, at: { x: 10, y: 10 }, opening: 'pointer' })
    one.state.chooseMenuItem('other-id')
    expect(one.state.menu.value).toBeNull()

    expect(one.state.empty.value).toBe(false)
    expect(one.state.dragged.value).toEqual([])
  })
})

describe('what a plex keeps calling a note', () => {
  it('keeps the name of a note the plex left for one navigation, from the first picture on', async () => {
    const one = tab('Root.md', ['Child.md', 'Other.md'])
    const other = one.node('Other.md')

    await one.state.view.go('Child.md')
    await settle()
    await one.state.view.go('Root.md')
    await settle()

    expect(one.node('Other.md')).toBe(other)
  })
})

describe('the notes a plex asks the vault for the headings of', () => {
  it('are none while the plex draws nothing', async () => {
    const one = tab('Root.md')

    one.state.view.neighbourhood.value = null
    await settle()

    expect(one.headingsAsked).toStrictEqual([['Root.md']])
  })

  it('leave out a note around the focus that has no path', () => {
    const one = tab('Root.md', [{ path: '' }])

    expect(one.headingsAsked).toStrictEqual([['Root.md']])
  })
})

describe('a quick link confirmed', () => {
  it('writes nothing from a node the picture never drew', async () => {
    const one = tab('Root.md', ['Other.md'])
    one.state.openQuickLink({ from: 'nowhere', relation: 'jump', at: { x: 1, y: 2 } })

    await one.state.confirmQuickLink('Other.md', true)

    expect(one.joined).toStrictEqual([])
    expect(one.went).toStrictEqual([])
  })

  it('remains where it stood when the vault wrote no link to the note', async () => {
    const one = tab('Root.md', ['Other.md'], false)
    one.state.openQuickLink({ from: one.node('Root.md'), relation: 'jump', at: { x: 1, y: 2 } })

    await one.state.confirmQuickLink('Other.md', true)

    expect(one.joined).toStrictEqual([['Root.md', 'Other.md', 'jump']])
    expect(one.went).toStrictEqual([])
  })

  it('remains where it stood when the vault made no note', async () => {
    const one = tab('Root.md', [], false)
    one.state.openQuickLink({ from: one.node('Root.md'), relation: 'child', at: { x: 1, y: 2 } })

    await one.state.confirmQuickLink('Compost', false)

    expect(one.made).toStrictEqual([['Root.md', 'child']])
    expect(one.went).toStrictEqual([])
  })
})

describe('what a note is called where the picture names it nowhere', () => {
  it('is the file it is filed under while the plex draws nothing', () => {
    const one = tab('Root.md')
    one.state.view.neighbourhood.value = null

    expect(one.state.getName('Deep/Elsewhere.md')).toBe('Elsewhere')
  })

  it('is the file it is filed under alongside the notes the picture does name', () => {
    expect(tab('Root.md', ['Child.md']).state.getName('Deep/Elsewhere.md')).toBe('Elsewhere')
  })

  it('loses only the extension at the end of the file name', () => {
    expect(tab('Root.md').state.getName('Deep/Readme.md copy.md')).toBe('Readme.md copy')
  })
})

describe('what a node the picture never drew remains for', () => {
  it('is a note', () => {
    expect(tab('Root.md', [], true, { 'Root.md': 'deck' }).state.getType('nowhere')).toBe('note')
  })
})

describe('a node opened', () => {
  it('opens its note here unless told where', () => {
    const one = tab('Root.md', ['Child.md'])

    one.state.openNode(one.node('Child.md'))

    expect(one.opened).toStrictEqual([['Child.md', 'Child', 'here']])
  })

  it('opens nothing for a heading whose name is not wholly a line number', () => {
    const one = tab('Root.md', ['Child.md'])

    one.state.openHeading(one.node('Child.md'), 'a12')
    one.state.openHeading(one.node('Child.md'), '12a')

    expect(one.opened).toStrictEqual([])
    expect(one.entered).toStrictEqual([])
  })
})
