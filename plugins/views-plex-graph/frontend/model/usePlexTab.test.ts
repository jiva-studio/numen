/**
 * Gesture evaluation in a plex, without a rendered screen.
 *
 * A note created from a node other than the focus is the case under test: it
 * is written into the vault regardless, and a graph one relation deep renders
 * it only from the node it was created from.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref, type Component } from 'vue'
import { createPlexTab, type PlexTabState, type PlexTabDeps } from './usePlexTab'
import type { CommandTarget, SlotCommand, SlotItem } from '@numen/plugin-sdk'
import { usePlexView as viewing, type PlexView } from './usePlexView'
import { WORDS as words } from '../words'
import {
  createLine,
  createVault,
  settle,
  viewOn,
  type Types,
  type RelatedNoteInput,
} from '../fixtures'
import type { NoteHeading, NoteRelation, PathRename } from '../types'
import { getRenamedPath } from '@numen/plugin-sdk'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { getDefaultLook, readLook } from './look'

/** The icons the window draws a declared command and a note of a declared type with. */
const SHARE_ICON: Component = { name: 'ShareIcon', render: () => null }
const SAMPLE_ICON: Component = { name: 'SampleIcon', render: () => null }

const createEntry = (command: string, extra: Partial<SlotCommand> = {}): SlotCommand => ({
  kind: 'command',
  key: `org.example.sample/command/${command}`,
  pluginId: 'org.example.sample',
  command,
  title: command,
  shortcut: '',
  hasItems: false,
  ...extra,
})

/** The row of the entry that opens the note. */
const OPEN_ROW = 'org.example.sample/command/app:file.open'

/** The row of the entry the slot holds over the canvas. */
const NEW_NOTE_ROW = 'org.example.sample/command/app:note.create'

/** The entries the slot of the canvas menu holds in these tests. */
const CANVAS_MENU: readonly SlotItem[] = [createEntry('app:note.create')]

/** The entries the slot of the node menu holds in these tests. */
const NODE_MENU: readonly SlotItem[] = [
  createEntry('app:file.open', { icon: SHARE_ICON }),
  { kind: 'separator', key: 'separator/1' },
  createEntry('org.example.sample:share.run', { icon: SHARE_ICON }),
  { kind: 'separator', key: 'separator/2' },
  createEntry('app:file.remove'),
]

/** What a test replaces of the deps, a part at a time. */
interface DepsOverrides {
  readonly window?: Partial<PlexTabDeps['window']>
  readonly menu?: Partial<PlexTabDeps['menu']>
  readonly notes?: Partial<PlexTabDeps['notes']>
}

const createDeps = (editor: PlexTabDeps['editor'], overrides: DepsOverrides = {}): PlexTabDeps => ({
  editor,
  tabId: 'plex:1',
  look: getDefaultLook(),
  creatable: ['parent', 'child', 'jump'],
  window: {
    isReady: ref(true),
    openNote: () => undefined,
    getNoteIcon: () => null,
    openingPath: ref(''),
    readOpeningPath: async () => '',
    dragged: ref([]),
    showMessage: () => undefined,
    ...overrides.window,
  },
  menu: {
    getMenuEntries: () => [],
    listItems: async () => [],
    runCommand: async () => '',
    ...overrides.menu,
  },
  notes: {
    isNesting: ref(true),
    headings: ref(6),
    readHeadings: async () => new Map(),
    ...overrides.notes,
  },
})

const tab = (
  targetPath: string,
  relatedNotes: readonly RelatedNoteInput[] = [],
  takes = true,
  types: Types = {},
) => {
  const plex = viewOn(targetPath, relatedNotes, types)
  const vault = createVault(takes)
  const openedCalls: [string, string, string][] = []
  const request: string[] = []
  const ran: [string, string, string][] = []
  const menuTargets: CommandTarget[] = []
  const messages: string[] = []
  /** The notes the window is dragging over this picture, which a test sets. */
  const dragging = ref<readonly string[]>([])
  /** The path the window answers a run of the command that makes a note with, which a test empties. */
  const created = ref('Untitled note 1.md')
  /** Every note shown on a line of its own prose. */
  const entered: [string, number][] = []
  /** The sections each note is divided into, set per test. */
  const divides = ref<ReadonlyMap<string, readonly NoteHeading[]>>(new Map())
  /** Whether a node nests the headings of its note, which a test turns. */
  const isNesting = ref(true)
  /** Every query the vault received about note contents. */
  const requestedHeadings: (readonly string[])[] = []
  const deps: PlexTabDeps = {
    editor: vault.editor,
    tabId: 'plex:1',
    look: getDefaultLook(),
    creatable: ['parent', 'child', 'jump'],
    window: {
      isReady: ref(true),
      openNote: (path, options) => {
        openedCalls.push([path, options?.title ?? '', options?.showing ?? 'current'])
        if (options?.line !== undefined) entered.push([path, options.line])
      },
      getNoteIcon: (tabKind) => (tabKind === 'sample' ? SAMPLE_ICON : null),
      openingPath: ref('Opening.md'),
      readOpeningPath: async () => 'Opening.md',
      dragged: dragging,
      showMessage: (text) => messages.push(text),
    },
    menu: {
      getMenuEntries: (target) => {
        menuTargets.push(target)
        return target.isFolder ? CANVAS_MENU : NODE_MENU
      },
      listItems: async () => [],
      runCommand: async (id, target) => {
        ran.push([id, target.path, target.title])
        return id === 'app:note.create' ? created.value : ''
      },
    },
    notes: {
      isNesting,
      headings: ref(6),
      // The vault answers only for the notes it was queried about.
      readHeadings: async (paths) => {
        requestedHeadings.push(paths)
        return new Map([...divides.value].filter(([path]) => paths.includes(path)))
      },
    },
  }
  const state = createPlexTab(plex.view, deps)
  /** The label of a note in the graph, which a gesture on it carries. */
  const node = (path: string) => getNode(state, path.replace(/\.md$/, ''))
  return {
    state,
    node,
    dragging,
    created,
    went: plex.went,
    ...vault,
    opened: openedCalls,
    entered,
    divides,
    isNesting,
    requestedHeadings,
    requested: request,
    requestLog: request,
    ran,
    menuTargets,
    messages,
  }
}

const getNode = (state: PlexTabState, title: string): string =>
  state.canvas.picture.value?.nodes.find((node) => node.title === title)?.id ?? ''

describe('a note made from a node', () => {
  it('remains the plex on the node it was made from', async () => {
    const one = tab('Root.md', ['Child.md'])

    await one.state.nodes.addNode(one.node('Child.md'), 'child')

    expect(one.made).toEqual([['Child.md', 'child']])
    expect(one.went).toEqual(['Child.md'])
  })

  it('leaves the plex where it is when that node is the focus', async () => {
    const one = tab('Root.md')

    await one.state.nodes.addNode(one.node('Root.md'), 'child')

    expect(one.went).toEqual(['Root.md'])
    expect(one.state.canvas.view.here.value).toBe('Root.md')
  })

  it('travels nowhere when the vault made nothing', async () => {
    const one = tab('Root.md', ['Child.md'], false)

    await one.state.nodes.addNode(one.node('Child.md'), 'parent')

    expect(one.went).toEqual([])
  })
})

describe('two notes a line was drawn between', () => {
  it('remains the plex on the note the line was drawn from', async () => {
    const one = tab('Root.md', ['Child.md', 'Other.md'])

    await one.state.nodes.joinNodes(one.node('Child.md'), one.node('Other.md'), 'jump')

    expect(one.joined).toEqual([['Child.md', 'Other.md', 'jump']])
    expect(one.went).toEqual(['Child.md'])
  })

  it('travels nowhere when nothing was written', async () => {
    const one = tab('Root.md', ['Child.md'], false)

    await one.state.nodes.joinNodes(one.node('Child.md'), one.node('Root.md'), 'jump')

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
    one.state.menu.openMenu(createMenuRequest(one.node('Child.md')))

    one.state.menu.chooseMenuItem(OPEN_ROW)

    expect(one.ran).toEqual([['app:file.open', 'Child.md', 'Child']])
    expect(one.state.menu.request.value).toBeNull()
  })

  it('draws the entries of the slot over the note of the node, and runs nothing it does not offer', () => {
    const one = tab('Root.md', ['Child.md'])
    one.state.menu.openMenu(createMenuRequest(one.node('Child.md')))

    expect(one.state.menu.menuItems.value.map((item) => [item.id, item.group])).toStrictEqual([
      ['org.example.sample/command/app:file.open', '0'],
      ['org.example.sample/command/org.example.sample:share.run', '1'],
      ['org.example.sample/command/app:file.remove', '2'],
    ])
    expect(one.menuTargets).toStrictEqual([
      { path: 'Child.md', title: 'Child', isFolder: false, others: [], tab: '' },
    ])
    one.state.menu.chooseMenuItem('constructor')
    one.state.menu.openMenu(createMenuRequest(one.node('Child.md')))
    one.state.menu.chooseMenuItem('org.example.sample/command/org.example.sample:share.run')

    expect(one.ran).toEqual([['org.example.sample:share.run', 'Child.md', 'Child']])
  })

  it('does not offer a new note over a node', () => {
    const one = tab('Root.md', ['Child.md'])
    one.state.menu.openMenu(createMenuRequest(one.node('Child.md')))

    expect(one.state.menu.menuItems.value.map((item) => item.id)).not.toContain(NEW_NOTE_ROW)
  })

  it('passes the note type of a node that is not a plain note', () => {
    const one = tab('Root.md', ['Sample.md'], true, { 'Sample.md': 'sample' })
    one.state.menu.openMenu(createMenuRequest(one.node('Sample.md')))

    expect(one.menuTargets.at(-1)).toMatchObject({ path: 'Sample.md', noteType: 'sample' })
  })

  it('draws a row with the icon of its entry, and none without one', () => {
    const one = tab('Root.md', ['Child.md'])
    one.state.menu.openMenu(createMenuRequest(one.node('Child.md')))

    expect(one.state.menu.getMenuIcon('org.example.sample/command/app:file.open')).toBe(SHARE_ICON)
    expect(one.state.menu.getMenuIcon('org.example.sample/command/app:file.remove')).toBeNull()
  })

  it('does nothing when it relies on nothing', () => {
    const one = tab('Root.md')

    one.state.menu.chooseMenuItem(OPEN_ROW)

    expect(one.ran).toEqual([])
  })

  it('goes when the picture under it does', () => {
    const one = tab('Root.md', ['Child.md'])
    one.state.menu.openMenu(createMenuRequest(one.node('Child.md')))

    one.state.menu.dismiss()

    expect(one.state.menu.request.value).toBeNull()
  })
})

/**
 * A picture is nothing for more reasons than an empty vault, and only one of
 * them is a plex to offer a note over.
 */
describe('a plex drawing nothing', () => {
  const plex = (overrides: DepsOverrides) => {
    const view = {
      noteGraph: ref(null),
      here: ref(''),
      error: ref(''),
      go: async () => undefined,
      followMoves: () => undefined,
      close: () => undefined,
    }
    return createPlexTab(view as unknown as PlexView, createDeps(createVault().editor, overrides))
  }

  it('is an empty vault where the vault is read and opens with no note', () => {
    expect(plex({}).canvas.empty.value).toBe(true)
  })

  it('is not an empty vault where the vault could not be opened', () => {
    expect(
      plex({
        window: {
          isReady: ref(false),
        },
      }).canvas.empty.value,
    ).toBe(false)
  })

  it('is not an empty vault while the first answer is on its way', () => {
    expect(
      plex({
        window: {
          openingPath: ref('Opening.md'),
        },
      }).canvas.empty.value,
    ).toBe(false)
  })

  it('is not an empty vault once the plex relies on a note', () => {
    expect(tab('Root.md').state.canvas.empty.value).toBe(false)
  })
})

describe('the menu of a place with no node', () => {
  const request = { node: null, at: { x: 1, y: 2 }, opening: 'pointer' as const }

  it('asks for the entries over the vault root of its own tab', () => {
    const one = tab('Root.md')

    one.state.menu.openMenu(request)

    expect(one.menuTargets).toStrictEqual([
      { path: '', title: '', isFolder: true, others: [], tab: 'plex:1' },
    ])
  })

  it('draws the entries the slot holds over the canvas', () => {
    const one = tab('Root.md')
    one.state.menu.openMenu(request)

    expect(one.state.menu.menuItems.value.map((item) => item.id)).toStrictEqual([NEW_NOTE_ROW])
  })

  it('runs the command of the row over the canvas target, and closes the menu', async () => {
    const one = tab('Root.md')
    one.state.menu.openMenu(request)

    one.state.menu.chooseMenuItem(NEW_NOTE_ROW)
    await settle()

    expect(one.ran).toStrictEqual([['app:note.create', '', '']])
    expect(one.state.menu.request.value).toBeNull()
  })

  it('shows the note the run made', async () => {
    const one = tab('Root.md')
    one.state.menu.openMenu(request)

    one.state.menu.chooseMenuItem(NEW_NOTE_ROW)
    await settle()

    expect(one.went).toStrictEqual(['Untitled note 1.md'])
    expect(one.state.canvas.view.here.value).toBe('Untitled note 1.md')
  })

  it('shows nothing where the run made no note', async () => {
    const one = tab('Root.md')
    one.created.value = ''
    one.state.menu.openMenu(request)

    one.state.menu.chooseMenuItem(NEW_NOTE_ROW)
    await settle()

    expect(one.went).toStrictEqual([])
  })
})

describe('what a note in the picture is called', () => {
  it('is the title the vault gave the focus', () => {
    expect(tab('Root.md').state.nodes.getName('Root.md')).toBe('Root')
  })

  it('is the title of a note around it', () => {
    expect(tab('Root.md', ['Deep/Child.md']).state.nodes.getName('Deep/Child.md')).toBe(
      'Deep/Child',
    )
  })

  it('is the file it is filed under, for a note the picture does not name', () => {
    expect(tab('Root.md').state.nodes.getName('Deep/Elsewhere.md')).toBe('Elsewhere')
  })
})

describe('the picture', () => {
  it('is nothing while the window has nothing true to draw', () => {
    const plex = viewOn('Root.md')
    const state = createPlexTab(
      plex.view,
      createDeps(createVault().editor, {
        window: {
          isReady: ref(false),
          dragged: ref(['Entropy.md']),
        },
      }),
    )

    expect(state.canvas.picture.value).toBeNull()
    // Nothing is drawn, so nothing can be dragged onto it.
    expect(state.canvas.dragged.value).toStrictEqual([])
  })
})

describe('notes dragged in and dropped over the picture', () => {
  it('writes the link into the note the plex relies on, naming the dragged one second', async () => {
    const one = tab('Root.md', ['Child.md'])

    await one.state.nodes.dropNodes(['physics/Entropy.md'], 'child')

    expect(one.joined).toEqual([['Root.md', 'physics/Entropy.md', 'child']])
    expect(one.went).toEqual(['Root.md'])
  })

  it('writes one for each of them, to the one note in the one relation', async () => {
    const one = tab('Root.md')

    await one.state.nodes.dropNodes(['Entropy.md', 'Kelvin.md', 'Heat.md'], 'child')

    expect(one.joined).toEqual([
      ['Root.md', 'Entropy.md', 'child'],
      ['Root.md', 'Kelvin.md', 'child'],
      ['Root.md', 'Heat.md', 'child'],
    ])
    expect(one.went).toEqual(['Root.md'])
  })

  it('takes the relation the drag named, whichever it was', async () => {
    const one = tab('Root.md')

    await one.state.nodes.dropNodes(['Entropy.md'], 'parent')
    await one.state.nodes.dropNodes(['Heat.md'], 'jump')

    expect(one.joined).toEqual([
      ['Root.md', 'Entropy.md', 'parent'],
      ['Root.md', 'Heat.md', 'jump'],
    ])
  })

  it('writes the rest where one of them failed, and says which stayed', async () => {
    const one = tab('Root.md')
    one.notJoinedPaths.add('Kelvin.md')

    await one.state.nodes.dropNodes(['Entropy.md', 'Kelvin.md', 'Heat.md'], 'child')

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

    await one.state.nodes.dropNodes(['Entropy.md', 'Heat.md'], 'child')

    expect(one.messages).toEqual([`${words.notJoined} Entropy, Heat`])
    expect(one.went).toEqual([])
  })

  it('joins nothing where the plex has nowhere to stand', async () => {
    const one = tab('')

    await one.state.nodes.dropNodes(['Entropy.md'], 'child')

    expect(one.joined).toEqual([])
    expect(one.messages).toEqual([])
  })

  it('leaves the note the plex relies on out, and joins the rest', async () => {
    const one = tab('Root.md')

    await one.state.nodes.dropNodes(['Root.md', 'Entropy.md'], 'child')

    expect(one.joined).toEqual([['Root.md', 'Entropy.md', 'child']])
    expect(one.messages).toEqual([])
  })
})

describe('what the picture draws a line to', () => {
  it('is what the window says it is dragging', () => {
    const one = tab('Root.md')
    one.dragging.value = ['physics/Entropy.md', 'physics/Kelvin.md']

    expect(one.state.canvas.dragged.value).toStrictEqual([
      'physics/Entropy.md',
      'physics/Kelvin.md',
    ])
  })

  it('is nothing while the window is dragging none', () => {
    expect(tab('Root.md').state.canvas.dragged.value).toStrictEqual([])
  })

  it('leaves out the note the plex relies on, and keeps the rest', () => {
    const one = tab('Root.md')
    one.dragging.value = ['Root.md', 'Entropy.md']

    expect(one.state.canvas.dragged.value).toStrictEqual(['Entropy.md'])
  })
})

describe('the headings a node nests', () => {
  /** One heading of a note, as the vault answers one. */
  const heading = (text: string, line: number, level = 1): NoteHeading => ({ text, level, line })

  it('are what the vault returned that note is divided into', async () => {
    const one = tab('Root.md', ['Child.md'])
    one.divides.value = new Map([['Child.md', [heading('Heat', 4), heading('Cold', 9, 2)]]])

    await one.state.headings.readHeadings()

    expect(one.state.headings.getHeadings(one.node('Child.md'))).toStrictEqual([
      { id: '4', text: 'Heat', level: 1 },
      { id: '9', text: 'Cold', level: 2 },
    ])
  })

  it('are none for a node the vault returned nothing about', async () => {
    const one = tab('Root.md', ['Child.md'])
    one.divides.value = new Map([['Child.md', [heading('Heat', 4)]]])

    await one.state.headings.readHeadings()

    expect(one.state.headings.getHeadings(one.node('Root.md'))).toStrictEqual([])
  })

  it('are each named by the line it relies on', async () => {
    const one = tab('Root.md')
    one.divides.value = new Map([['Root.md', [heading('Heat', 12)]]])

    await one.state.headings.readHeadings()

    expect(
      one.state.headings.getHeadings(one.node('Root.md')).map((heading) => heading.id),
    ).toStrictEqual(['12'])
  })

  it('are requested for again once the plex remains somewhere else', async () => {
    const one = tab('Root.md', ['Child.md'])
    one.divides.value = new Map([['Other.md', [heading('Heat', 4)]]])

    await one.state.canvas.view.go('Other.md')
    await settle()

    expect(one.state.headings.getHeadings(one.node('Other.md'))).toStrictEqual([
      { id: '4', text: 'Heat', level: 1 },
    ])
  })

  it('are what the question requested last returned, whatever order they land in', async () => {
    // The first answer is held up until the second has settled, which is a
    // change followed while a travel is still out.
    const answers: ((target: ReadonlyMap<string, readonly NoteHeading[]>) => void)[] = []
    const one = tab('Root.md')
    const plex = createPlexTab(
      one.state.canvas.view,
      createDeps(createVault().editor, {
        window: {
          openingPath: ref('Root.md'),
          readOpeningPath: async () => 'Root.md',
        },
        notes: {
          readHeadings: () => new Promise((isDone) => answers.push(isDone)),
        },
      }),
    )
    const node = () => getNode(plex, 'Root')

    // The graph queries once when built, and that response stays pending.
    const request = answers.length
    const reads = [plex.headings.readHeadings(), plex.headings.readHeadings()]
    answers[request + 1]?.(new Map([['Root.md', [heading('Fresh', 2)]]]))
    answers[request]?.(new Map([['Root.md', [heading('Stale', 1)]]]))
    await Promise.all(reads)
    await settle()

    expect(plex.headings.getHeadings(node())).toStrictEqual([{ id: '2', text: 'Fresh', level: 1 }])
  })

  it('are none for every node while the setting is off', async () => {
    const one = tab('Root.md', ['Child.md'])
    one.divides.value = new Map([['Child.md', [heading('Heat', 4)]]])
    await one.state.headings.readHeadings()
    one.isNesting.value = false

    expect(one.state.headings.getHeadings(one.node('Child.md'))).toStrictEqual([])
    expect(one.state.headings.getHeadings(one.node('Root.md'))).toStrictEqual([])
  })

  it('are not requested of the vault at all while the setting is off', async () => {
    const one = tab('Root.md', ['Child.md'])
    one.isNesting.value = false
    one.requestedHeadings.length = 0

    await one.state.headings.readHeadings()

    expect(one.requestedHeadings).toStrictEqual([])
  })

  it('are none for a note of a plugin type, whose headings are its own', async () => {
    const one = tab('Root.md', ['Animals.md'], true, { 'Animals.md': 'sample' })
    one.divides.value = new Map([['Animals.md', [heading('Vicuña', 4)]]])
    one.requestedHeadings.length = 0

    await one.state.headings.readHeadings()

    expect(one.state.headings.getHeadings(one.node('Animals.md'))).toStrictEqual([])
    expect(one.requestedHeadings).toStrictEqual([['Root.md']])
  })

  it('are none for a note of another plugin type', async () => {
    const one = tab('Root.md', ['Animal.md'], true, { 'Animal.md': 'form' })
    one.divides.value = new Map([['Animal.md', [heading('Front', 4)]]])
    one.requestedHeadings.length = 0

    await one.state.headings.readHeadings()

    expect(one.state.headings.getHeadings(one.node('Animal.md'))).toStrictEqual([])
    expect(one.requestedHeadings).toStrictEqual([['Root.md']])
  })

  it('are the headings of an ordinary note alongside them', async () => {
    const one = tab('Root.md', ['Animals.md', 'Child.md'], true, {
      'Animals.md': 'sample',
    })
    one.divides.value = new Map([['Child.md', [heading('Heat', 4)]]])

    await one.state.headings.readHeadings()

    expect(one.state.headings.getHeadings(one.node('Child.md'))).toStrictEqual([
      { id: '4', text: 'Heat', level: 1 },
    ])
  })

  it('are none for a note of a plugin type in focus, which is where the plex is standing', async () => {
    const one = tab('Animals.md', [], true, { 'Animals.md': 'sample' })
    one.divides.value = new Map([['Animals.md', [heading('Vicuña', 4)]]])
    one.requestedHeadings.length = 0

    await one.state.headings.readHeadings()

    expect(one.state.headings.getHeadings(one.node('Animals.md'))).toStrictEqual([])
    expect(one.requestedHeadings).toStrictEqual([])
  })

  it('are hung again once the setting is turned back on', async () => {
    // Turning is the only user action; nothing else requests them again.
    const one = tab('Root.md', ['Child.md'])
    one.divides.value = new Map([['Child.md', [heading('Heat', 4)]]])

    one.isNesting.value = false
    await settle()
    expect(one.state.headings.getHeadings(one.node('Child.md'))).toStrictEqual([])

    one.isNesting.value = true
    await settle()

    expect(one.state.headings.getHeadings(one.node('Child.md'))).toStrictEqual([
      { id: '4', text: 'Heat', level: 1 },
    ])
  })
})

describe('a heading of a node chosen', () => {
  it('opens the note it remains in, and puts the keyboard on its line', () => {
    const one = tab('Root.md')

    one.state.nodes.openHeading(one.node('Root.md'), '12')

    expect(one.opened).toStrictEqual([['Root.md', 'Root', 'current']])
    expect(one.entered).toStrictEqual([['Root.md', 12]])
  })

  it('opens the note the node remains for, the focus being another one', () => {
    const one = tab('Root.md', ['Child.md'])

    one.state.nodes.openHeading(one.node('Child.md'), '7')

    expect(one.opened).toStrictEqual([['Child.md', 'Child', 'current']])
    expect(one.entered).toStrictEqual([['Child.md', 7]])
  })

  it('remains the plex where it stood', () => {
    const one = tab('Root.md', ['Child.md'])

    one.state.nodes.openHeading(one.node('Child.md'), '7')

    expect(one.went).toStrictEqual([])
    expect(one.state.canvas.view.here.value).toBe('Root.md')
  })

  it('opens nothing for a heading naming no line, which is the one standing for the rest', () => {
    const one = tab('Root.md')

    one.state.nodes.openHeading(one.node('Root.md'), '')

    expect(one.opened).toStrictEqual([])
    expect(one.entered).toStrictEqual([])
  })
})

/** A note beside another: where it is located, and the note it comes through. */
type RelatedNoteRow = readonly [path: string, relation: NoteRelation, through?: string]

const getNoteTitle = (path: string) => (path.split('/').pop() ?? path).replace(/\.md$/, '')

const inVault = async (focus: string, initialRelatedNotes: readonly RelatedNoteRow[] = []) => {
  let relatedItems = initialRelatedNotes.map(([path, relation, through]) => ({
    path,
    relation,
    through: through ?? '',
  }))
  /** An answer the vault is holding back, and what releases it. */
  let pendingGate: Promise<void> | null = null
  let release: () => void = () => undefined
  /** Every note the vault was queried about, in query order. */
  const request: string[] = []
  const vault = createVault()
  const openedCalls: [string, string, string][] = []
  const ran: [string, string, string][] = []

  const view = viewing({
    listRelatedNotes: async (path) => {
      request.push(path)
      if (pendingGate) await pendingGate
      return {
        focus: { path, title: getNoteTitle(path) },
        focusType: 'note' as const,
        related: relatedItems.map((one) => ({
          path: one.path,
          title: getNoteTitle(one.path),
          type: 'note' as const,
          relation: one.relation,
        })),
        connections: relatedItems.map((one) => createLine(path, one)),
      }
    },
  })
  const state = createPlexTab(
    view,
    createDeps(vault.editor, {
      window: {
        openNote: (path, options) =>
          openedCalls.push([path, options?.title ?? '', options?.showing ?? 'current']),
      },
      menu: {
        getMenuEntries: () => NODE_MENU,
        runCommand: async (id, target) => {
          ran.push([id, target.path, target.title])
          return ''
        },
      },
    }),
  )
  await view.go(focus)

  /** The current graph, which a rendered window reads. */
  const picture = () => state.canvas.picture.value
  /** The label of the note shown by this title. */
  const node = (title: string) => getNode(state, title)
  const nodes = () => (picture()?.nodes ?? []).map((one) => `${one.relation} ${one.id}`)
  /** Every edge of the graph, as the pair of nodes it connects. */
  const edges = () => (picture()?.edges ?? []).map((one) => `${one.from} -> ${one.to}`)

  /** Files moved in the vault, and the plex notified of the old and new paths. */
  const moveFiles = (...renamed: readonly PathRename[]) => {
    relatedItems = relatedItems.map((one) => ({
      ...one,
      path: getRenamedPath(renamed, one.path) || one.path,
      through: getRenamedPath(renamed, one.through) || one.through,
    }))
    state.nodes.followMoves(renamed)
  }
  /** A move, followed by a new graph request. */
  const moveFilesAndReload = async (...renamed: readonly PathRename[]) => {
    moveFiles(...renamed)
    await view.go(view.here.value)
  }
  /** The user navigates to another note, which is a graph of its own. */
  const goToNote = async (path: string, ...now: readonly RelatedNoteRow[]) => {
    relatedItems = now.map(([at, relation, through]) => ({
      path: at,
      relation,
      through: through ?? '',
    }))
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
    requested: request,
    requestLog: request,
    opened: openedCalls,
    ran,
    made: vault.made,
    joined: vault.joined,
    /** The vault holds every answer back until it is release. */
    holdAnswers: () => {
      pendingGate = new Promise<void>((resolve) => (release = resolve))
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

    const nodeIds = one.picture()?.nodes.map((node) => node.id) ?? []
    expect(new Set(nodeIds).size).toBe(4)
  })
})

describe('a gesture the plex reports', () => {
  it('travels to the note the node remains for', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    await one.moveFilesAndReload({ from: 'Heat.md', to: 'physics/Heat.md' })

    one.state.nodes.activate(one.node('Heat'))

    expect(one.requested.at(-1)).toBe('physics/Heat.md')
  })

  it('opens the note the node remains for, called what the picture calls it', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    await one.moveFilesAndReload({ from: 'Heat.md', to: 'physics/Heat.md' })

    one.state.nodes.openNode(one.node('Heat'), 'split')

    expect(one.opened).toStrictEqual([['physics/Heat.md', 'Heat', 'split']])
  })

  it('runs a command on the note the menu stood on', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    await one.moveFilesAndReload({ from: 'Heat.md', to: 'physics/Heat.md' })

    one.state.menu.openMenu({ node: one.node('Heat'), at: { x: 1, y: 2 }, opening: 'pointer' })
    one.state.menu.chooseMenuItem(OPEN_ROW)

    expect(one.ran).toStrictEqual([['app:file.open', 'physics/Heat.md', 'Heat']])
  })

  it('makes a note in a relation of the note the node remains for', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    await one.moveFilesAndReload({ from: 'Heat.md', to: 'physics/Heat.md' })

    await one.state.nodes.addNode(one.node('Heat'), 'child')

    expect(one.made).toStrictEqual([['physics/Heat.md', 'child']])
  })

  it('joins the two notes a line was drawn between', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    await one.moveFilesAndReload({ from: 'Heat.md', to: 'physics/Heat.md' })

    await one.state.nodes.joinNodes(one.node('Heat'), one.node('Root'), 'jump')

    expect(one.joined).toStrictEqual([['physics/Heat.md', 'Root.md', 'jump']])
  })

  it('is release where the picture never drew that node', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    expect(one.node('Heat')).not.toBe('')
    const request = one.requested.length

    for (const node of ['Heat.md', 'Root.md', 'nothing']) {
      one.state.nodes.activate(node)
      one.state.nodes.openNode(node, 'current')
      one.state.menu.openMenu({ node, at: { x: 1, y: 2 }, opening: 'pointer' })
      one.state.menu.chooseMenuItem(OPEN_ROW)
      await one.state.nodes.addNode(node, 'child')
      await one.state.nodes.joinNodes(node, one.node('Root'), 'jump')
    }

    expect(one.requested).toHaveLength(request)
    expect(one.opened).toStrictEqual([])
    expect(one.ran).toStrictEqual([])
    expect(one.made).toStrictEqual([])
    expect(one.joined).toStrictEqual([])
  })
})

describe('a plex that travelled', () => {
  it('calls the notes it arrives at nothing it called the ones it left', async () => {
    const one = await inVault('Root.md', [['Heat.md', 'child']])
    const initialNodeIds = one.picture()?.nodes.map((node) => node.id) ?? []

    await one.goToNote('Cold.md', ['Ice.md', 'child'])

    const nextNodeIds = one.picture()?.nodes.map((node) => node.id) ?? []
    expect(nextNodeIds).toHaveLength(2)
    expect(nextNodeIds.filter((id) => initialNodeIds.includes(id))).toStrictEqual([])

    // A node is invalid after two navigations, so a drag on it issues no vault
    // query.
    await one.goToNote('Warm.md', ['Fire.md', 'child'])
    for (const node of initialNodeIds) one.state.nodes.activate(node)
    expect(one.requested.at(-1)).toBe('Warm.md')
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

    // The vault is queried for the note at its old path, and the file moves while
    // that query is pending.
    one.holdAnswers()
    const nav = one.view.go('One.md')
    one.moveFiles({ from: 'One.md', to: 'moved/One.md' }, { from: 'Heat.md', to: 'moved/Heat.md' })
    one.answers()
    await nav
    one.picture()

    await one.view.go(one.view.here.value)

    expect(one.nodes()).toStrictEqual(before)
    expect(one.view.here.value).toBe('moved/One.md')
  })
})

describe('what a node remains for', () => {
  it('is what the vault says of the note the node draws', () => {
    const one = tab('Root.md', ['Sample.md', 'Form.md'], true, {
      'Sample.md': 'sample',
      'Form.md': 'form',
    })

    expect(one.state.headings.getType(one.node('Sample.md'))).toBe('sample')
    expect(one.state.headings.getType(one.node('Form.md'))).toBe('form')
    expect(one.state.headings.getType(one.node('Root.md'))).toBe('note')
  })

  it('is drawn with the icon of the tab kind the core chose for it, and a note it reports none for with none', () => {
    const one = tab('Root.md', ['Sample.md'], true, { 'Sample.md': 'sample' })

    expect(one.state.menu.getNodeIcon(one.node('Sample.md'))).toBe(SAMPLE_ICON)
    expect(one.state.menu.getNodeIcon(one.node('Root.md'))).toBeNull()
  })
})

describe('quick-link popover from a node handle', () => {
  it('opens quick link request with drop coordinates', async () => {
    const one = tab('Root.md', ['Child.md'])
    const childNode = one.node('Child.md')

    await one.state.nodes.addNode(childNode, 'child', { x: 450, y: 320 })

    expect(one.state.quickLink.quickLink.value).toStrictEqual({
      from: childNode,
      relation: 'child',
      at: { x: 450, y: 320 },
    })
  })

  it('dismisses quick link popover cleanly on dismiss', async () => {
    const one = tab('Root.md')
    one.state.quickLink.openQuickLink({
      from: one.node('Root.md'),
      relation: 'parent',
      at: { x: 100, y: 200 },
    })

    one.state.quickLink.dismissQuickLink()

    expect(one.state.quickLink.quickLink.value).toBeNull()
  })

  it('creates new note and navigates to it when confirmed with title and navigation enabled', async () => {
    const one = tab('Root.md')
    const rootNode = one.node('Root.md')
    one.state.quickLink.openQuickLink({ from: rootNode, relation: 'child', at: { x: 100, y: 200 } })

    await one.state.quickLink.confirmQuickLink('SubTopic', false)

    expect(one.state.quickLink.quickLink.value).toBeNull()
    expect(one.made).toEqual([['Root.md', 'child']])
    expect(one.went).toEqual(['SubTopic.md'])
  })

  it('joins existing note and navigates to it when confirmed as existing note', async () => {
    const one = tab('Root.md', ['Other.md'])
    const rootNode = one.node('Root.md')
    one.state.quickLink.openQuickLink({ from: rootNode, relation: 'jump', at: { x: 100, y: 200 } })

    await one.state.quickLink.confirmQuickLink('Other.md', true)

    expect(one.state.quickLink.quickLink.value).toBeNull()
    expect(one.joined).toEqual([['Root.md', 'Other.md', 'jump']])
    expect(one.went).toEqual(['Other.md'])
  })

  it('remains on source note when navigation on create is disabled', async () => {
    const one = tab('Root.md')
    one.state.quickLink.isNavigatingOnCreate.value = false
    const rootNode = one.node('Root.md')
    one.state.quickLink.openQuickLink({ from: rootNode, relation: 'child', at: { x: 100, y: 200 } })

    await one.state.quickLink.confirmQuickLink('SubTopic', false)

    expect(one.made).toEqual([['Root.md', 'child']])
    expect(one.went).toEqual(['Root.md'])
  })

  it('joins existing note and stays on source note when navigation is disabled', async () => {
    const one = tab('Root.md', ['Other.md'])
    one.state.quickLink.isNavigatingOnCreate.value = false
    const rootNode = one.node('Root.md')
    one.state.quickLink.openQuickLink({ from: rootNode, relation: 'jump', at: { x: 100, y: 200 } })

    await one.state.quickLink.confirmQuickLink('Other.md', true)

    expect(one.joined).toEqual([['Root.md', 'Other.md', 'jump']])
    expect(one.went).toEqual(['Root.md'])
  })

  it('ignores confirmQuickLink when title is empty or quick link is inactive', async () => {
    const one = tab('Root.md')
    await one.state.quickLink.confirmQuickLink('', false)
    expect(one.made).toEqual([])

    const rootNode = one.node('Root.md')
    one.state.quickLink.openQuickLink({ from: rootNode, relation: 'child', at: { x: 100, y: 200 } })
    await one.state.quickLink.confirmQuickLink('   ', false)
    expect(one.made).toEqual([])
  })

  it('searches notes via searchNames dependency', async () => {
    const searchMock = vi.fn().mockResolvedValue([{ path: 'Note.md', title: 'Note' }])
    const plex = viewOn('Root.md', [])
    const vault = createVault()
    const stateWithSearch = createPlexTab(
      plex.view,
      createDeps(vault.editor, {
        window: {
          openingPath: ref('Opening.md'),
          readOpeningPath: async () => 'Opening.md',
        },
        notes: {
          searchNames: searchMock,
        },
      }),
    )

    const results = await stateWithSearch.quickLink.searchNotes('test')
    expect(searchMock).toHaveBeenCalledWith('test')
    expect(results).toEqual([{ path: 'Note.md', title: 'Note' }])

    const stateNoSearch = createPlexTab(
      plex.view,
      createDeps(vault.editor, {
        window: {
          openingPath: ref('Opening.md'),
          readOpeningPath: async () => 'Opening.md',
        },
      }),
    )
    expect(await stateNoSearch.quickLink.searchNotes('test')).toEqual([])
  })

  it('handles openLinkInspector with invalid pair formats gracefully', async () => {
    const { state } = tab('Focus.md', ['Child.md'])
    await settle()

    await state.inspector.openLinkInspector('invalidSingleToken', { x: 0, y: 0 })
    expect(state.inspector.linkInspector.value).toBeNull()

    await state.inspector.openLinkInspector('unknown1 unknown2', { x: 0, y: 0 })
    expect(state.inspector.linkInspector.value).toBeNull()
  })

  it('searches notes via searchNotes method', async () => {
    const { state } = tab('Focus.md', ['Child.md'])
    const emptyResults = await state.quickLink.searchNotes('query')
    expect(emptyResults).toEqual([])
  })

  it('ignores invalid headings in openHeading', () => {
    const { state, node, opened, entered } = tab('Focus.md', ['Child.md'])
    const focusId = node('Focus')

    state.nodes.openHeading(focusId, 'not-a-number')
    expect(entered).toEqual([])
    expect(opened).toEqual([])

    state.nodes.openHeading('nonexistent', '1')
    expect(entered).toEqual([])
  })

  it('confirms quick link with whitespace title or inactive state without error', async () => {
    const { state } = tab('Focus.md', ['Child.md'])
    await state.quickLink.confirmQuickLink('  ', false)
    expect(state.quickLink.quickLink.value).toBeNull()
  })
})

describe('createPlexTab edge cases and helpers', () => {
  it('covers openHeading edge cases, dismissals and menu choices', async () => {
    const one = tab('Root.md', ['Child.md'])
    one.state.nodes.openHeading('unknown-node', '12')
    one.state.nodes.openHeading(one.node('Root.md'), 'not-a-number')

    one.state.quickLink.openQuickLink({
      from: one.node('Root.md'),
      relation: 'child',
      at: { x: 10, y: 10 },
    })
    expect(one.state.quickLink.quickLink.value).not.toBeNull()
    const pair = `${one.node('Root.md')} ${one.node('Child.md')}`
    await one.state.inspector.openLinkInspector(pair, { x: 10, y: 10 })
    expect(one.state.inspector.linkInspector.value).not.toBeNull()

    one.state.menu.openMenu({ node: null, at: { x: 10, y: 10 }, opening: 'pointer' })
    one.state.menu.dismiss()
    expect(one.state.menu.request.value).toBeNull()
    expect(one.state.quickLink.quickLink.value).toBeNull()
    expect(one.state.inspector.linkInspector.value).toBeNull()

    one.state.menu.openMenu({ node: null, at: { x: 10, y: 10 }, opening: 'pointer' })
    one.state.menu.chooseMenuItem('other-id')
    expect(one.state.menu.request.value).toBeNull()

    expect(one.state.canvas.empty.value).toBe(false)
    expect(one.state.canvas.dragged.value).toEqual([])
  })
})

describe('what a plex keeps calling a note', () => {
  it('keeps the name of a note the plex left for one navigation, from the first picture on', async () => {
    const one = tab('Root.md', ['Child.md', 'Other.md'])
    const other = one.node('Other.md')

    await one.state.canvas.view.go('Child.md')
    await settle()
    await one.state.canvas.view.go('Root.md')
    await settle()

    expect(one.node('Other.md')).toBe(other)
  })
})

describe('the notes a plex asks the vault for the headings of', () => {
  it('are none while the plex draws nothing', async () => {
    const one = tab('Root.md')

    one.state.canvas.view.noteGraph.value = null
    await settle()

    expect(one.requestedHeadings).toStrictEqual([['Root.md']])
  })

  it('leave out a note around the focus that has no path', () => {
    const one = tab('Root.md', [{ path: '' }])

    expect(one.requestedHeadings).toStrictEqual([['Root.md']])
  })
})

describe('a quick link confirmed', () => {
  it('writes nothing from a node the picture never drew', async () => {
    const one = tab('Root.md', ['Other.md'])
    one.state.quickLink.openQuickLink({ from: 'nowhere', relation: 'jump', at: { x: 1, y: 2 } })

    await one.state.quickLink.confirmQuickLink('Other.md', true)

    expect(one.joined).toStrictEqual([])
    expect(one.went).toStrictEqual([])
  })

  it('remains where it stood when the vault wrote no link to the note', async () => {
    const one = tab('Root.md', ['Other.md'], false)
    one.state.quickLink.openQuickLink({
      from: one.node('Root.md'),
      relation: 'jump',
      at: { x: 1, y: 2 },
    })

    await one.state.quickLink.confirmQuickLink('Other.md', true)

    expect(one.joined).toStrictEqual([['Root.md', 'Other.md', 'jump']])
    expect(one.went).toStrictEqual([])
  })

  it('remains where it stood when the vault made no note', async () => {
    const one = tab('Root.md', [], false)
    one.state.quickLink.openQuickLink({
      from: one.node('Root.md'),
      relation: 'child',
      at: { x: 1, y: 2 },
    })

    await one.state.quickLink.confirmQuickLink('Compost', false)

    expect(one.made).toStrictEqual([['Root.md', 'child']])
    expect(one.went).toStrictEqual([])
  })
})

describe('what a note is called where the picture names it nowhere', () => {
  it('is the file it is filed under while the plex draws nothing', () => {
    const one = tab('Root.md')
    one.state.canvas.view.noteGraph.value = null

    expect(one.state.nodes.getName('Deep/Elsewhere.md')).toBe('Elsewhere')
  })

  it('is the file it is filed under alongside the notes the picture does name', () => {
    expect(tab('Root.md', ['Child.md']).state.nodes.getName('Deep/Elsewhere.md')).toBe('Elsewhere')
  })

  it('loses only the extension at the end of the file name', () => {
    expect(tab('Root.md').state.nodes.getName('Deep/Readme.md copy.md')).toBe('Readme.md copy')
  })
})

describe('what a node the picture never drew remains for', () => {
  it('is a note', () => {
    expect(
      tab('Root.md', [], true, { 'Root.md': 'sample' }).state.headings.getType('nowhere'),
    ).toBe('note')
  })
})

describe('a node opened', () => {
  it('opens its note here unless told where', () => {
    const one = tab('Root.md', ['Child.md'])

    one.state.nodes.openNode(one.node('Child.md'))

    expect(one.opened).toStrictEqual([['Child.md', 'Child', 'current']])
  })

  it('opens nothing for a heading whose name is not wholly a line number', () => {
    const one = tab('Root.md', ['Child.md'])

    one.state.nodes.openHeading(one.node('Child.md'), 'a12')
    one.state.nodes.openHeading(one.node('Child.md'), '12a')

    expect(one.opened).toStrictEqual([])
    expect(one.entered).toStrictEqual([])
  })
})

describe('the References setting on an open tab', () => {
  const open = (initial?: string) => {
    const context = createMockPluginContext({
      pluginId: 'md.numen.views.plex.graph',
      initialSettings: initial ? { references: initial } : {},
    })
    const plex = viewOn('Root.md', [
      'Child.md',
      { path: 'Cites.md', relation: 'reference', direction: 'outgoing' },
      { path: 'CitedBy.md', relation: 'reference', direction: 'incoming' },
    ])
    const state = createPlexTab(plex.view, {
      ...createDeps(createVault(true).editor),
      look: readLook(context),
    })
    const titles = () => state.canvas.picture.value?.nodes.map((node) => node.title)
    return { context, plex, titles }
  }

  it('shows the references of both directions by default', () => {
    expect(open().titles()).toEqual(['Root', 'Child', 'Cites', 'CitedBy'])
  })

  it('loses the references when the setting is turned off, without asking the vault again', () => {
    const one = open()

    one.context.setSetting('references', 'off')

    expect(one.titles()).toEqual(['Root', 'Child'])
    expect(one.plex.went).toEqual([])
  })

  it('follows the setting from one direction to the other and back', () => {
    const one = open('outgoing')
    expect(one.titles()).toEqual(['Root', 'Child', 'Cites'])

    one.context.setSetting('references', 'incoming')
    expect(one.titles()).toEqual(['Root', 'Child', 'CitedBy'])

    one.context.setSetting('references', 'both')
    expect(one.titles()).toEqual(['Root', 'Child', 'Cites', 'CitedBy'])
  })
})
