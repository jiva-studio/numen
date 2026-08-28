/**
 * The window drawn, in a document, with both ports mocked away.
 *
 * Two things are asked here: that every kind the window declares is drawn when
 * a tab holds one, and that a vault which could not be read is not shown as an
 * empty one.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount, type VueWrapper } from '@vue/test-utils'
import {
  Agent,
  branch,
  Editor,
  Notices,
  pane,
  Palette,
  panesOf,
  Plex,
  Reader,
  Tree,
  Workspace,
  type PaletteBand,
  type WorkspaceLayout,
} from '@numen/ui'
import AgentTab from './agent/AgentTab.vue'
import { linkOf } from './agent/places'
import DeckTab from './cards/DeckTab.vue'
import DocumentTab from './document/DocumentTab.vue'
import FilesTab from './files/FilesTab.vue'
import NoteTab from './note/NoteTab.vue'
import PlexTab from './plex/PlexTab.vue'
import Welcome from './welcome/Welcome.vue'
import { WORDS as plexWords } from './plex/words'
import { REFUSED } from './words'
import { plexCalled } from './workspace'

const { said, held, asked, listed, folders, cuts, stands, outside } = vi.hoisted(() => ({
  /**
   * What the vault holds at a path. A book and a note are told apart by the
   * name the file carries, the way the vault itself tells them apart, and which
   * of three a note is is what a test said.
   */
  stands: (path: string) => {
    if (/\.(epub|pdf)$/u.test(path)) return { kind: 'book' as const, type: 'note' as const }
    if (!/\.(md|note)$/u.test(path)) return { kind: 'other' as const, type: 'note' as const }
    return { kind: 'note' as const, type: said.types[path] ?? ('note' as const) }
  },
  /**
   * The places something outside the window asks to be put in front of the
   * person, and the stream the window hears them on.
   */
  outside: (() => {
    const queue: { path: string; start: number; length: number }[] = []
    let wake: (() => void) | null = null
    return {
      asks: (at: { path: string; start: number; length: number }) => {
        queue.push(at)
        wake?.()
        wake = null
      },
      forget: () => queue.splice(0),
      stream: async function* () {
        for (;;) {
          while (queue.length > 0) yield queue.shift()!
          await new Promise<void>((woken) => {
            wake = woken
          })
        }
      },
    }
  })(),
  /**
   * The vault as it makes a deck or a stencil: the file is named after the
   * title, and the extension is the vault's own and no caller's. It is not
   * markdown here, so a window building the path for itself reaches nothing.
   */
  cuts: (() => {
    const filed = new Set<string>()
    /** Whether the vault answers what it is asked at all. */
    let reached = true
    return {
      forget: () => {
        filed.clear()
        reached = true
      },
      /** The vault is out of reach, so asking it for one reaches nothing. */
      breaks: () => {
        reached = false
      },
      makes: (title: string, folder: string) => {
        if (!reached) throw new Error('the vault could not be reached')
        const path = `${folder ? `${folder}/` : ''}${title}.note`
        if (filed.has(path)) return { path: '', refusal: 'occupied' as const }
        filed.add(path)
        return { path, refusal: null }
      },
    }
  })(),
  /** What the mocked vault answers about itself, set before the window draws. */
  said: {
    ready: true,
    failed: '',
    opening: 'Root.md' as string | null,
    names: [] as {
      path: string
      title: string
      heading: string
      line: number
      at: []
      type: 'note' | 'deck' | 'stencil'
    }[],
    /** The passages the search answers with. */
    passages: [] as {
      path: string
      title: string
      isNote: boolean
      text: string
      start: number
      length: number
      line: number
      at: []
      type: 'note' | 'deck' | 'stencil'
    }[],
    /**
     * Which of three the note at each path is, as the vault answers it. A path
     * it says nothing about is the ordinary note the window reads it as.
     */
    types: {} as Record<string, 'note' | 'deck' | 'stencil'>,
    /** Whether the list of vaults answers at all. */
    listable: true,
    /** What the settings refuse a choice, which is a size outside its bounds. */
    refused: '',
    /** What the settings say the window is drawn as, which a test may set. */
    applied: 'preset:numen',
    mode: 'system' as 'system' | 'light' | 'dark',
    sizes: { interfaceScale: 1, textScale: 1 },
    /** What renaming a field of a stencil comes back with. */
    renaming: {
      decks: [] as string[],
      cards: 0,
      notWritten: [] as { path: string; text: string }[],
      refusal: null as null | string,
      changed: false,
      at: 'a2',
    },
  },
  /** A stream that stays open, so nothing the window follows ever ends. */
  async *held(): AsyncGenerator<never> {
    await new Promise<never>(() => {})
  },
  /** What the window asked the application for, in the order it asked. */
  asked: {
    made: [] as string[],
    renamed: [] as string[],
    removed: [] as string[],
    moved: [] as string[],
    folders: [] as string[],
    /** The decks and the stencils the window asked for, in the order it asked. */
    cut: [] as string[],
    /** Each field rename the window asked the vault for. */
    renamedField: [] as string[],
    /** The cards each of those deck writes carried, by name. */
    wrote: [] as string[],
    worn: [] as string[],
    /** How often an open editor was told to take its measurements again. */
    measured: 0,
  },
  /** The vaults this installation holds, and the one the window is showing. */
  listed: {
    vaults: [{ id: 'physics', name: 'Physics', path: '/vaults/Physics', missing: false }],
    showing: 'physics',
  },
  /** What each folder of the vault holds, as a listing answers it. */
  folders: {
    '': [
      { path: 'physics', name: 'physics', folder: true, kind: 'other' },
      { path: 'Root.md', name: 'Root.md', folder: false, kind: 'note' },
      { path: 'Cover.png', name: 'Cover.png', folder: false, kind: 'other' },
    ],
    physics: [
      { path: 'physics/Entropy.md', name: 'Entropy.md', folder: false, kind: 'note' },
      { path: 'physics/Kelvin.md', name: 'Kelvin.md', folder: false, kind: 'note' },
    ],
  } as Record<string, readonly Record<string, unknown>[]>,
}))

vi.mock('./vault', () => ({
  vault: {},
  vaults: {
    list: async () => listed,
    choose: async () => '',
    add: async () => ({ vault: null, refusal: null }),
    rename: async () => ({ vault: null, refusal: null }),
    forget: async () => null,
    erase: async () => null,
    open: async () => null,
  },
  documents: {
    shape: async () => ({ pages: 1, sheets: [{ wide: 100, high: 100 }] }),
    page: () => '',
    places: async () => [],
  },
  cards: {
    // A card is named by the first field of the stencil it is cut by, so the
    // window is told of one.
    stencils: async () => ({
      stencils: [{ path: 'Animal.md', title: 'Animal', fields: ['Name'] }],
      held: 1,
    }),
    makeDeck: async (title: string, folder: string) => {
      asked.cut.push(`deck ${folder || '/'} ${title}`)
      return cuts.makes(title, folder)
    },
    makeStencil: async (title: string, folder: string, fields: readonly string[]) => {
      asked.cut.push(`stencil ${folder || '/'} ${title} [${fields.join(', ')}]`)
      return cuts.makes(title, folder)
    },
    renameField: async (path: string, from: string, to: string) => {
      asked.renamedField.push(`${path} ${from} ${to}`)
      return said.renaming
    },
    readDeck: async (path: string) => ({
      deck: { path, title: path, preamble: '', cards: [], tail: '', problems: [] },
      refusal: null,
      at: 'a1',
      bound: 0,
    }),
    writeDeck: async (path: string, deck: { cards: readonly { name: string }[] }) => {
      asked.cut.push(`deck ${path}`)
      asked.wrote.push(deck.cards.map((card) => card.name).join(', '))
      return { refusal: null, changed: false, at: 'a2', bound: 0 }
    },
    readStencil: async (path: string) => ({
      stencil: { path, title: path, fields: [], faces: [], problems: [] },
      refusal: null,
      at: 'a1',
    }),
    writeStencil: async (path: string) => {
      asked.cut.push(`stencil ${path}`)
      return { refusal: null, changed: false, at: 'a2' }
    },
  },
  core: {
    vaults: async () => {
      if (!said.listable) throw new Error('the vaults are not there')
      return listed
    },
    state: async () => ({
      name: 'Vault',
      path: '/vaults/Physics',
      ready: said.ready,
      failed: said.failed,
      unwatched: '',
      unreachable: '',
      chunks: 0n,
      embedded: 0n,
      embedding: false,
    }),
    opening: async () => (said.opening ? { path: said.opening } : null),
    neighbourhood: async (path: string) => ({
      focus: { path, title: path.replace(/\.md$/, '') },
      related: [],
    }),
    read: async () => ({ body: 'what is written', at: 'a1' }),
    write: async () => ({ at: 'a2' }),
    create: async ({ title }: { title: string }) => {
      asked.made.push(title)
      return { path: `${title}.md`, refusal: null }
    },
    rename: async (path: string, title: string) => {
      asked.renamed.push(`${path} ${title}`)
      return { path, title, by: 'frontmatter', moved: null, refusal: null }
    },
    remove: async (path: string, destroy?: boolean) => {
      asked.removed.push(`${path} ${destroy ?? false}`)
      return { trashed: `.trash/${path}`, dangling: [], refusal: null }
    },
    list: async (folder: string) => folders[folder] ?? [],
    move: async (from: string, to: string) => {
      asked.moved.push(`${from} ${to}`)
      return { moved: null, refusal: null }
    },
    makeFolder: async (path: string) => {
      asked.folders.push(path)
      return null
    },
    changes: held,
    editing: held,
    tasks: held,
    focus: outside.stream,
    quitting: held,
    flushed: async () => {},
    names: async () => said.names,
    search: async () => said.passages,
    standing: async (paths: readonly string[]) =>
      new Map(paths.map((path) => [path, stands(path)])),
    headings: async () => new Map(),
  },
}))

vi.mock('./agent/core', () => ({ core: { ask: held, finish: async () => {} } }))

vi.mock('./theme', () => ({
  themes: {
    catalogue: async () => ({
      themes: [
        { name: 'preset:numen', title: 'numen', shipped: true, pinned: false },
        { name: 'mine:sea', title: 'sea', shipped: false, pinned: false },
      ],
      applied: said.applied,
      mode: said.mode,
      sizes: said.sizes,
      bounds: { interfaceScale: { least: 0.8, most: 2 }, textScale: { least: 0.8, most: 1.75 } },
    }),
    text: async (name: string) => `:root { --numen-surface: ${name} }`,
    chooses: async (
      name: string,
      mode: string,
      sizes: { interfaceScale: number; textScale: number },
    ) => {
      asked.worn.push(`${name} ${mode} ${sizes.interfaceScale}/${sizes.textScale}`)
      return said.refused
    },
    changed: held,
  },
}))

const App = (await import('./App.vue')).default

/** A moment for whatever the window asked the vault for to come back. */
const settles = () => new Promise((done) => setTimeout(done, 0))

/** Longer than the palette holds a keystroke before it asks the vault. */
const HELD = 200

/** One name the vault answers a search with, of a note of one of three kinds. */
const nameSaid = (path: string, title: string, type: 'note' | 'deck' | 'stencil' = 'note') => ({
  path,
  title,
  heading: '',
  line: -1,
  at: [] as [],
  type,
})

/** One passage the search answers with, read out of a note of one of three kinds. */
const passageSaid = (path: string, title: string, type: 'note' | 'deck' | 'stencil' = 'note') => ({
  path,
  title,
  isNote: true,
  text: 'what it says',
  start: 0,
  length: 4,
  line: 3,
  at: [] as [],
  type,
})

/** Something drawn in a pane that answers what the window asks of it. */
const answers = (name: string, drawn: Record<string, unknown>) =>
  defineComponent({
    name,
    setup: (_, { expose }) => {
      expose(drawn)
      return () => h('div')
    },
  })

/** An editor answers the three things a note asks of one; a page, the one. */
const editor = answers('Editor', {
  focus: () => true,
  measure: () => (asked.measured += 1),
  reveal: () => true,
})
const reader = answers('Reader', { measure: () => {} })

/**
 * Every window a test drew. A window listens for the keystrokes that open the
 * palette for as long as it is mounted, and the next test draws its own.
 */
const windows: { unmount(): void }[] = []

afterEach(() => {
  for (const window of windows.splice(0)) window.unmount()
  said.ready = true
  said.opening = 'Root.md'
  said.names = []
  said.passages = []
  said.types = {}
  said.listable = true
  said.refused = ''
  said.applied = 'preset:numen'
  said.mode = 'system'
  said.sizes = { interfaceScale: 1, textScale: 1 }
  said.renaming = {
    decks: [],
    cards: 0,
    notWritten: [],
    refusal: null,
    changed: false,
    at: 'a2',
  }
  cuts.forget()
  outside.forget()
  asked.made = []
  asked.cut = []
  asked.renamedField = []
  asked.wrote = []
  asked.renamed = []
  asked.removed = []
  asked.moved = []
  asked.folders = []
  asked.worn = []
  asked.measured = 0
  listed.vaults = [{ id: 'physics', name: 'Physics', path: '/vaults/Physics', missing: false }]
  listed.showing = 'physics'
})

/**
 * The window drawn, with what each kind draws inside it stubbed. The tabs
 * themselves are the window's own, and they are what is asked about here.
 */
async function drawn() {
  const window = mount(App, {
    global: {
      stubs: { Plex: true, Editor: editor, Agent: true, Reader: reader, Palette: true, Tree: true },
    },
  })
  windows.push(window)
  await settles()
  await settles()
  await settles()
  return window
}

/** What the corner of the window is saying, one string per card. */
const cards = (window: VueWrapper): readonly string[] =>
  window.findAll('article.notice').map((card) => card.text())

/**
 * What the plex calls the note it is standing on, which is what every gesture
 * it reports carries. The vault is asked about a path, and this is not one.
 */
const nodeInPlex = (window: VueWrapper): string => {
  const held = window.findComponent(PlexTab).props('held') as {
    picture: { value: { nodes: readonly { id: string }[] } | null }
  }
  return held.picture.value?.nodes[0]?.id ?? ''
}

/**
 * The window with a palette a person can type into. The palette draws itself at
 * the end of the document, and is read off the document.
 */
async function drawnWithPalette() {
  const window = mount(App, {
    global: { stubs: { Plex: true, Editor: editor, Agent: true, Reader: reader, Tree: true } },
    attachTo: document.body,
  })
  windows.push(window)
  await settles()
  await settles()
  await settles()
  return window
}

/** How the window is split, as the workspace it draws has it. */
const layoutOf = (window: VueWrapper): WorkspaceLayout =>
  window.findComponent(Workspace).props('modelValue') as WorkspaceLayout

/** What each pane holds, by the kind each of its tabs is filed under. */
const paneKinds = (window: VueWrapper): readonly (readonly string[])[] =>
  panesOf(layoutOf(window).root).map((one) => one.tabs.map((tab) => tab.split(':')[0] ?? ''))

/** What each tab of the window is called, in the order the strip has them. */
const tabsOf = (window: VueWrapper): readonly { id: string; title: string }[] =>
  (window.findComponent(Workspace).props('tabs') as readonly { id: string; title: string }[]) ?? []

describe('the window as it opens', () => {
  it('draws a plex in the room, and an agent in front of the files beside it', async () => {
    const window = await drawn()

    expect(paneKinds(window)).toStrictEqual([['plex'], ['agent', 'files']])
    expect(panesOf(layoutOf(window).root)[1]?.active).toMatch(/^agent:/)
    expect(window.findComponent(PlexTab).exists()).toBe(true)
    expect(window.findComponent(AgentTab).exists()).toBe(true)
    expect(window.findComponent(FilesTab).exists()).toBe(true)
  })

  it('leaves the person in the plex, which holds the greater share', async () => {
    const window = await drawn()
    const root = layoutOf(window).root

    expect(layoutOf(window).focus).toBe('main')
    expect(root.kind === 'branch' ? root.sizes : []).toStrictEqual([0.72, 0.28])
  })

  it('hands the tree what the root of the vault holds', async () => {
    const window = await drawn()

    expect(
      (window.findComponent(Tree).props('rows') as readonly { id: string }[]).map((one) => one.id),
    ).toStrictEqual(['physics', 'Root.md', 'Cover.png'])
  })
})

describe('the window holding no tab', () => {
  /** Every tab closed, by the keystroke that closes the one in front. */
  const closesEvery = async (window: VueWrapper) => {
    for (let each = tabsOf(window).length; each > 0; each -= 1) {
      globalThis.dispatchEvent(
        new KeyboardEvent('keydown', { key: 'W', ctrlKey: true, shiftKey: true, cancelable: true }),
      )
      await settles()
    }
  }

  it('opens on the welcome screen, holding nothing, while the list shows no vault', async () => {
    listed.showing = ''

    const window = await drawn()

    expect(tabsOf(window)).toStrictEqual([])
    expect(window.findComponent(Welcome).exists()).toBe(true)
  })

  it('comes to the same screen once every tab it opened with is closed', async () => {
    const window = await drawn()

    await closesEvery(window)

    expect(tabsOf(window)).toStrictEqual([])
    expect(window.findComponent(Welcome).exists()).toBe(true)
  })

  it('draws the vault the list is showing on it, said to be the one in front', async () => {
    const window = await drawn()

    await closesEvery(window)

    expect(window.findComponent(Welcome).props('vaults')).toStrictEqual([
      { id: 'physics', name: 'Physics', path: '/vaults/Physics', detail: 'Current' },
    ])
  })

  it('says in the corner that the vaults could not be listed, and draws none', async () => {
    said.listable = false

    const window = await drawn()

    expect(cards(window).join(' ')).toContain('The vaults could not be listed')
    expect(window.findComponent(Welcome).props('vaults')).toStrictEqual([])
  })
})

describe('a row activated in the files', () => {
  const activated = async (path: string) => {
    const window = await drawn()
    window.findComponent(Tree).vm.$emit('activate', path)
    await settles()
    await settles()
    return window
  }

  it('opens a note in a tab of its own', async () => {
    const window = await activated('Root.md')

    expect(window.findComponent(NoteTab).exists()).toBe(true)
  })

  it('opens nothing at all for a file the vault holds no source for', async () => {
    const window = await activated('Cover.png')

    expect(window.findComponent(NoteTab).exists()).toBe(false)
    expect(window.findComponent(DocumentTab).exists()).toBe(false)
  })

  it('opens nothing for a folder, which turns where it stands', async () => {
    const window = await activated('physics')

    expect(window.findComponent(NoteTab).exists()).toBe(false)
  })
})

describe('a file carried out of the tree', () => {
  /** The window with the physics folder open, and a row carried out of it. */
  const carrying = async (rows: readonly string[]) => {
    const window = await drawn()
    const tree = window.findComponent(Tree)
    tree.vm.$emit('open', 'physics')
    await settles()
    tree.vm.$emit('carry', rows)
    await settles()
    return window
  }

  const carried = (window: VueWrapper) => window.findComponent(Plex).props('carried')

  it('is what the plex draws a line to, though neither knows the other is there', async () => {
    expect(carried(await carrying(['physics/Entropy.md']))).toStrictEqual([
      'physics/Entropy.md',
    ])
  })

  it('is every note of the selection, all of them at once', async () => {
    const window = await carrying(['physics/Entropy.md', 'physics/Kelvin.md'])

    expect(carried(window)).toStrictEqual(['physics/Entropy.md', 'physics/Kelvin.md'])
  })

  it('leaves out the note the plex is standing on, and keeps the rest', async () => {
    const window = await carrying(['Root.md', 'physics/Entropy.md'])

    expect(carried(window)).toStrictEqual(['physics/Entropy.md'])
  })

  it('is nothing once it has been let go of, wherever that was', async () => {
    const window = await carrying(['physics/Entropy.md'])

    window.findComponent(Tree).vm.$emit('drop')
    await settles()

    expect(carried(window)).toStrictEqual([])
  })

  it('leaves out a file the vault holds no note for, and a folder', async () => {
    const window = await carrying(['physics', 'physics/Entropy.md', 'Cover.png'])

    expect(carried(window)).toStrictEqual(['physics/Entropy.md'])
  })

  it('is nothing where the rows hold no note at all', async () => {
    expect(carried(await carrying(['Cover.png', 'physics']))).toStrictEqual([])
  })
})

describe('a note asked for in the plex', () => {
  it('is drawn in a tab of its own', async () => {
    const window = await drawn()

    window.findComponent(Plex).vm.$emit('show', nodeInPlex(window), 'here')
    await settles()

    expect(window.findComponent(NoteTab).exists()).toBe(true)
    expect(window.findComponent(NoteTab).findComponent(Editor).exists()).toBe(true)
  })

  it('is asked for by the node, and a path opens nothing', async () => {
    const window = await drawn()

    window.findComponent(Plex).vm.$emit('show', 'Root.md', 'here')
    await settles()

    expect(window.findComponent(NoteTab).exists()).toBe(false)
  })
})

describe('a place an answer names', () => {
  it('is drawn in the document it stands in', async () => {
    const window = await drawn()

    window
      .findComponent(Agent)
      .vm.$emit(
        'follow',
        { id: 'said', voice: 'said', text: '[here](numen:Source.pdf?start=0&length=4)' },
        'numen:Source.pdf?start=0&length=4',
        { preventDefault: () => {} },
      )
    await settles()

    expect(window.findComponent(DocumentTab).exists()).toBe(true)
    expect(window.findComponent(DocumentTab).findComponent(Reader).exists()).toBe(true)
  })
})

describe('the palette', () => {
  /** A keystroke taken on the window, and whether the window took it. */
  const pressed = (key: string) => {
    const event = new KeyboardEvent('keydown', { key, ctrlKey: true, cancelable: true })
    globalThis.dispatchEvent(event)
    return event
  }

  const bandsOf = (window: Awaited<ReturnType<typeof drawn>>) =>
    (window.findComponent(Palette).props('bands') as readonly { id: string }[]).map((one) => one.id)

  it('opens on the commands for what is in front, and prints nothing', async () => {
    const window = await drawn()

    const event = pressed('p')
    await settles()

    expect(event.defaultPrevented).toBe(true)
    expect(window.findComponent(Palette).props('open')).toBe(true)
    expect(bandsOf(window)).toStrictEqual(['note', 'window', 'vault'])
  })

  it('opens on the search under its own keystroke', async () => {
    const window = await drawn()

    pressed('k')
    await settles()

    expect(window.findComponent(Palette).props('open')).toBe(true)
    expect(bandsOf(window)).toStrictEqual([])
  })

  /** A name the search turned up, chosen to be read. */
  const reads = async (window: Awaited<ReturnType<typeof drawn>>, path: string) => {
    pressed('k')
    await settles()
    window.findComponent(Palette).vm.$emit('update:modelValue', 'ani')
    await new Promise((done) => setTimeout(done, HELD))
    window.findComponent(Palette).vm.$emit('choose', path, 'note')
    await settles()
    await settles()
  }

  it('opens a deck it turned up in the editor of its cards', async () => {
    said.names = [nameSaid('Animals.md', 'Animals', 'deck')]
    said.types = { 'Animals.md': 'deck' }
    const window = await drawn()

    await reads(window, 'Animals.md')

    expect(paneKinds(window).flat()).toContain('deck')
    expect(window.findComponent(NoteTab).exists()).toBe(false)
  })

  it('opens a stencil it turned up in the editor of its fields and faces', async () => {
    said.names = [nameSaid('Animal.md', 'Animal', 'stencil')]
    said.types = { 'Animal.md': 'stencil' }
    const window = await drawn()

    await reads(window, 'Animal.md')

    expect(paneKinds(window).flat()).toContain('stencil')
    expect(window.findComponent(NoteTab).exists()).toBe(false)
  })

  it('opens an ordinary note the same search turned up in a note tab', async () => {
    said.names = [nameSaid('Animals.md', 'Animals')]
    const window = await drawn()

    await reads(window, 'Animals.md')

    expect(paneKinds(window).flat()).not.toContain('deck')
    expect(window.findComponent(NoteTab).exists()).toBe(true)
  })

  it('turns from the search to the commands on the character that means them', async () => {
    const window = await drawn()
    pressed('k')
    await settles()

    window.findComponent(Palette).vm.$emit('update:modelValue', '>')
    await settles()

    expect(bandsOf(window)).toStrictEqual(['note', 'window', 'vault'])
  })

  /** The note the commands are over, which the step that renames one opens on. */
  const overNote = async (window: Awaited<ReturnType<typeof drawn>>) => {
    window.findComponent(Palette).vm.$emit('choose', 'title', 'title')
    await settles()
    return window.findComponent(Palette).props('modelValue')
  }

  /** The tab of the plex standing on that note, as the window calls it. */
  const plexTab = (window: Awaited<ReturnType<typeof drawn>>, note: string): string =>
    (window.findComponent(Workspace).props('tabs') as readonly { id: string; title: string }[])
      .find((one) => one.title === plexCalled(plexWords.plex, note))
      ?.id ?? ''

  it('is over the plex in the tab in front, not the plex last put in front', async () => {
    said.names = [nameSaid('physics/Entropy.md', 'Entropy')]
    const window = await drawn()

    // A second plex, standing on a note of its own, put in front last.
    pressed('p')
    await settles()
    window.findComponent(Palette).vm.$emit('choose', 'plex', 'plex')
    await settles()
    pressed('k')
    await settles()
    window.findComponent(Palette).vm.$emit('update:modelValue', 'en')
    await new Promise((done) => setTimeout(done, HELD))
    window.findComponent(Palette).vm.$emit('choose', 'physics/Entropy.md', 'plex')
    await settles()

    // Each of the two dragged into a pane of its own. The pane the person is in
    // is the one holding the plex on the note the vault opens with.
    window.findComponent(Workspace).vm.$emit('update:modelValue', {
      root: branch(
        'root',
        [
          pane('main', [plexTab(window, 'Root')]),
          pane('aside', [plexTab(window, 'physics/Entropy')]),
        ],
        [0.5, 0.5],
      ),
      axis: 'horizontal',
      focus: 'main',
    })
    await settles()

    pressed('p')
    await settles()

    expect(await overNote(window)).toBe('Root')
  })
})

/**
 * The keystrokes drawn on a command's row. Each is asked for on the window, as
 * a person presses it with the palette nowhere in sight.
 */
describe('a command reached by its own keystroke', () => {
  const field = () => document.body.querySelector<HTMLInputElement>('.palette__field')

  /** A keystroke taken on the window, and whether the window took it. */
  const pressed = (key: string, over: Partial<KeyboardEventInit> = {}) => {
    const event = new KeyboardEvent('keydown', { key, ctrlKey: true, cancelable: true, ...over })
    globalThis.dispatchEvent(event)
    return event
  }

  const type = async (text: string) => {
    const into = field()
    if (!into) return
    into.value = text
    into.dispatchEvent(new Event('input'))
    await settles()
  }

  const press = async (key: string) => {
    field()?.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
    await settles()
  }

  /** The row of a command in the list of commands, by the identity it is drawn under. */
  const rowOf = (window: Awaited<ReturnType<typeof drawn>>, id: string) => {
    const bands = window.findComponent(Palette).props('bands') as readonly PaletteBand[]
    return bands.flatMap((band) => band.items).find((one) => one.id === id)
  }

  it('draws the keystroke on its row, written for the keyboard in hand', async () => {
    const window = await drawn()

    pressed('p')
    await settles()

    expect(rowOf(window, 'note')?.keys).toEqual({ marks: ['control'], letter: 'N' })
    expect(rowOf(window, 'goto')?.keys).toEqual({ marks: ['control'], letter: 'G' })
  })

  it('draws no keystroke on the rows no keystroke reaches', async () => {
    const window = await drawn()

    pressed('p')
    await settles()

    expect(rowOf(window, 'destroy')?.keys).toBeUndefined()
    expect(rowOf(window, 'eraseVault')?.keys).toBeUndefined()
  })

  it('makes a note under the name typed, on the keystroke the new note draws', async () => {
    await drawnWithPalette()

    const event = pressed('n')
    await settles()
    await type('Entropy')
    await press('Enter')
    await settles()

    expect(event.defaultPrevented).toBe(true)
    expect(asked.made).toStrictEqual(['Entropy'])
  })

  it('makes a deck under the name typed, and opens it in the editor of its cards', async () => {
    const window = await drawnWithPalette()

    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
    await settles()
    await type('New deck')
    await press('Enter')
    await type('Animals')
    await press('Enter')
    await settles()

    expect(asked.cut).toStrictEqual(['deck / Animals'])
    expect(paneKinds(window).flat()).toContain('deck')
  })

  it('opens the deck where the vault filed it, and at no path of its own making', async () => {
    const window = await drawnWithPalette()

    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
    await settles()
    await type('New deck')
    await press('Enter')
    await type('Animals')
    await press('Enter')
    await settles()

    const deck = window.findComponent(DeckTab).props('held') as { shown(): { path: string } }
    expect(deck.shown().path).toBe('Animals.note')
    expect(deck.shown().path).not.toBe('Animals.md')
  })

  it('says the refusal and opens nothing where the name is taken already', async () => {
    const window = await drawnWithPalette()

    const makes = async () => {
      globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
      await settles()
      await type('New deck')
      await press('Enter')
      await type('Animals')
      await press('Enter')
      await settles()
    }
    await makes()
    await makes()

    expect(paneKinds(window).flat().filter((kind) => kind === 'deck')).toHaveLength(1)
    expect(window.text()).toContain(REFUSED.occupied)
  })

  it('makes a stencil the same way, and opens no deck', async () => {
    const window = await drawnWithPalette()

    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
    await settles()
    await type('New stencil')
    await press('Enter')
    await type('Animal')
    await press('Enter')
    await settles()

    expect(asked.cut).toStrictEqual(['stencil / Animal [Field 1]'])
    expect(paneKinds(window).flat()).not.toContain('deck')
  })

  it('makes the stencil carrying the field its cards are named by, and not none', async () => {
    await drawnWithPalette()

    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
    await settles()
    await type('New stencil')
    await press('Enter')
    await type('Animal')
    await press('Enter')
    await settles()

    expect(asked.cut).not.toStrictEqual(['stencil / Animal []'])
  })

  it('opens the step that picks a note, on the keystroke going to one draws', async () => {
    said.names = [nameSaid('physics/Entropy.md', 'Entropy')]
    const window = await drawnWithPalette()

    const event = pressed('g')
    await settles()

    expect(event.defaultPrevented).toBe(true)
    expect(window.findComponent(Palette).props('crumb')).toBe('Go to a note')
  })

  it('travels to the note picked on that step', async () => {
    said.names = [nameSaid('physics/Entropy.md', 'Entropy')]
    const window = await drawnWithPalette()

    pressed('g')
    await settles()
    await type('en')
    await new Promise((done) => setTimeout(done, HELD))
    await press('Enter')
    await settles()

    const plex = window.findComponent(PlexTab).props('held') as {
      view: { here: { value: string } }
    }
    expect(plex.view.here.value).toBe('physics/Entropy.md')
  })

  it('leaves a keystroke alone while Alt is held with it', async () => {
    const window = await drawn()

    const event = pressed('n', { altKey: true })
    await settles()

    expect(event.defaultPrevented).toBe(false)
    expect(window.findComponent(Palette).props('open')).toBe(false)
    expect(asked.made).toStrictEqual([])
  })

  it('leaves a keystroke a pane has already answered alone', async () => {
    const window = await drawn()

    const event = new KeyboardEvent('keydown', { key: 'n', ctrlKey: true, cancelable: true })
    event.preventDefault()
    globalThis.dispatchEvent(event)
    await settles()

    expect(window.findComponent(Palette).props('open')).toBe(false)
  })
})

/**
 * The keystrokes that hold Shift. The letter each holds is spoken for on its
 * own, so what the window does with it turns on Shift alone.
 */
describe('a command reached by a keystroke holding Shift', () => {
  /** A keystroke taken on the window, and whether the window took it. */
  const pressed = (key: string, over: Partial<KeyboardEventInit> = {}) => {
    const event = new KeyboardEvent('keydown', { key, ctrlKey: true, cancelable: true, ...over })
    globalThis.dispatchEvent(event)
    return event
  }

  const tabs = (window: Awaited<ReturnType<typeof drawn>>) =>
    (window.findComponent(Workspace).props('tabs') as readonly { id: string }[]) ?? []

  it('no longer puts the palette up on the letter that puts it up alone', async () => {
    const window = await drawn()

    const event = pressed('K', { shiftKey: true })
    await settles()

    expect(event.defaultPrevented).toBe(false)
    expect(window.findComponent(Palette).props('open')).toBe(false)
  })

  it('no longer puts the commands up on the letter that puts them up alone', async () => {
    const window = await drawn()

    pressed('P', { shiftKey: true })
    await settles()

    // The plex the window opened with is the one it is standing on, which is
    // what showing the note in the plex leaves in front.
    expect(window.findComponent(Palette).props('bands')).toStrictEqual([])
  })

  it('opens an agent in a tab of its own', async () => {
    const window = await drawn()
    const before = window.findAllComponents(AgentTab).length

    const event = pressed('A', { shiftKey: true })
    await settles()

    expect(event.defaultPrevented).toBe(true)
    expect(window.findAllComponents(AgentTab).length).toBe(before + 1)
  })

  it('closes the tab in front', async () => {
    const window = await drawn()
    const before = tabs(window).length

    const event = pressed('W', { shiftKey: true })
    await settles()

    expect(event.defaultPrevented).toBe(true)
    expect(tabs(window).length).toBe(before - 1)
  })

  it('makes a child note of the note in front, under the name typed', async () => {
    const window = await drawnWithPalette()

    const event = pressed('C', { shiftKey: true })
    await settles()
    expect(window.findComponent(Palette).props('crumb')).toBe('New child note')

    const field = document.body.querySelector<HTMLInputElement>('.palette__field')
    if (field) {
      field.value = 'Entropy'
      field.dispatchEvent(new Event('input'))
      await settles()
      field.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
      await settles()
    }

    expect(event.defaultPrevented).toBe(true)
    expect(asked.made).toStrictEqual(['Entropy'])
  })

  it('shows the note in front in the plex', async () => {
    const window = await drawn()

    const event = pressed('P', { shiftKey: true })
    await settles()

    expect(event.defaultPrevented).toBe(true)
    const plex = window.findComponent(PlexTab).props('held') as {
      view: { here: { value: string } }
    }
    expect(plex.view.here.value).toBe('Root.md')
  })

  it('draws every keystroke that holds Shift on the row that names it', async () => {
    const window = await drawn()

    pressed('p')
    await settles()

    const bands = window.findComponent(Palette).props('bands') as readonly PaletteBand[]
    const drawnKeys = Object.fromEntries(
      bands.flatMap((band) => band.items).map((one) => [one.id, one.keys]),
    )
    expect(drawnKeys['travel']).toEqual({ marks: ['control', 'shift'], letter: 'P' })
    expect(drawnKeys['child']).toEqual({ marks: ['control', 'shift'], letter: 'C' })
    expect(drawnKeys['agent']).toEqual({ marks: ['control', 'shift'], letter: 'A' })
    expect(drawnKeys['close']).toEqual({ marks: ['control', 'shift'], letter: 'W' })
  })
})

describe('a command asked for on a node of the plex', () => {
  /** The menu on a node, and an item of it chosen. */
  const chose = async (window: Awaited<ReturnType<typeof drawn>>, id: string) => {
    const plex = window.findComponent(PlexTab).props('held') as {
      asks: (one: unknown) => void
      chose: (id: string) => void
    }
    plex.asks({ node: nodeInPlex(window), at: { x: 0, y: 0 }, from: null, opening: 'below' })
    plex.chose(id)
    await settles()
  }

  it('says in the tab what that tab could not show', async () => {
    const window = await drawn()
    const tab = window.findComponent(PlexTab)
    const held = tab.props('held') as { view: { trouble: { value: string } } }

    held.view.trouble.value = 'Gone.md is not in the vault'
    await settles()

    expect(tab.find('.warning').text()).toBe('Gone.md is not in the vault')
    expect(cards(window)).toStrictEqual([])
  })

  it('says nothing where it was taken up', async () => {
    const window = await drawn()

    await chose(window, 'title')

    expect(cards(window)).toStrictEqual([])
    expect(window.findComponent(Palette).props('crumb')).toBe('Change title')
  })
})

/**
 * A vault still being read draws no picture, so a command over it is reached by
 * the keyboard: this is the window a person meets while a vault is opening.
 */
describe('a command asked for while the vault is being read', () => {
  /** The keystroke for a new note, which is a command over the window. */
  const askedFor = async () => {
    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'n', ctrlKey: true }))
    await settles()
  }

  it('says the vault is still being read, and carries no command out', async () => {
    said.ready = false
    said.opening = null
    const window = await drawn()

    await askedFor()

    expect(cards(window)).toStrictEqual(['reading the vault…', 'The vault is still being read'])
    expect(asked.made).toStrictEqual([])
  })

  it('lets a person put away what it told them, and forgets it', async () => {
    said.ready = false
    said.opening = null
    const window = await drawn()

    await askedFor()
    await window.findAll('article.notice button')[1]!.trigger('click')
    await settles()

    expect(cards(window)).toStrictEqual(['reading the vault…'])
    // Put away is put away for good: the window stops handing the corner a card
    // it has been told the person is finished with.
    expect(window.findComponent(Notices).props('notices')).toHaveLength(1)
  })
})

/** The palette drawn as a person meets it, with nothing of it stubbed. */
/** The note goes to the vault's .trash folder, so nothing is asked over it. */
describe('the keyboard on the command that removes a note', () => {
  const field = () => document.body.querySelector<HTMLInputElement>('.palette__field')

  const press = async (key: string) => {
    field()?.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
    await settles()
  }

  const type = async (text: string) => {
    const into = field()
    if (!into) return
    into.value = text
    into.dispatchEvent(new Event('input'))
    await settles()
  }

  /** The commands open, with the one that removes a note lit and taken. */
  const overRemove = async () => {
    const window = await drawnWithPalette()
    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
    await settles()
    await type('remove')
    await press('Enter')
    return window
  }

  it('removes the note the moment the command is chosen', async () => {
    await overRemove()

    expect(asked.removed).toStrictEqual(['Root.md false'])
  })

  it('stands on no step, and the palette goes with the choice', async () => {
    const window = await overRemove()

    expect(document.body.textContent).not.toContain('Keep the note')
    expect(window.findComponent(Palette).props('open')).toBe(false)
  })
})

/** A deck and a stencil answer a removal the way a note does. */
describe('a file the window has open in an editor of cards, removed from the tree', () => {
  const field = () => document.body.querySelector<HTMLInputElement>('.palette__field')

  const press = async (key: string) => {
    field()?.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
    await settles()
  }

  const type = async (text: string) => {
    const into = field()
    if (!into) return
    into.value = text
    into.dispatchEvent(new Event('input'))
    await settles()
  }

  /** The window with one deck or one stencil made and put in front. */
  const holding = async (command: string, name: string) => {
    const window = await drawnWithPalette()
    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
    await settles()
    await type(command)
    await press('Enter')
    await type(name)
    await press('Enter')
    await settles()
    return window
  }

  /** A row taken out of the vault, as the tree asks for it. */
  const removes = async (window: VueWrapper, path: string) => {
    window.findComponent(Tree).vm.$emit('remove', [path])
    await settles()
    await settles()
  }

  it('lets go of the tab holding a deck', async () => {
    const window = await holding('New deck', 'Animals')
    expect(paneKinds(window).flat()).toContain('deck')

    await removes(window, 'Animals.note')

    expect(asked.removed).toStrictEqual(['Animals.note false'])
    expect(paneKinds(window).flat()).not.toContain('deck')
  })

  it('lets go of the tab holding a stencil', async () => {
    const window = await holding('New stencil', 'Animal')
    expect(paneKinds(window).flat()).toContain('stencil')

    await removes(window, 'Animal.note')

    expect(asked.removed).toStrictEqual(['Animal.note false'])
    expect(paneKinds(window).flat()).not.toContain('stencil')
  })

  it('writes the card nobody had saved before the file goes', async () => {
    const window = await holding('New deck', 'Animals')
    const held = window.findComponent(DeckTab).props('held') as {
      adds(name: string, stencil: string, values: readonly { field: string; text: string }[]): void
    }

    held.adds('Vicuña', 'Animal', [])
    await removes(window, 'Animals.note')

    // Making the deck is no write, so the only one is what the person added.
    expect(asked.wrote).toStrictEqual(['Vicuña'])
  })
})

/** The tree makes one where the row stands, and the vault may answer nothing. */
describe('a deck or a stencil the file tree asked the vault for', () => {
  /** What a row of the tree asks for, on the row the menu was opened on. */
  const asksFor = async (stencil: boolean) => {
    const window = await drawn()
    const tree = window.findComponent(FilesTab).props('held') as {
      cuts(path: string | null, stencil: boolean): Promise<void>
    }
    cuts.breaks()
    await tree.cuts(null, stencil)
    await settles()
    return window
  }

  it('says why no deck was made, where the vault could not be reached', async () => {
    const window = await asksFor(false)

    expect(cards(window).join(' ')).toContain('could not be reached')
  })

  it('says why no stencil was made, the same way', async () => {
    const window = await asksFor(true)

    expect(cards(window).join(' ')).toContain('could not be reached')
  })
})

/** How the window is drawn, walked as a person walks it, with nothing stubbed. */
describe('the four commands over how the window is drawn', () => {
  /** What the mode's element holds while the tokens are read as a pair. */
  const PAIR = ':root { color-scheme: light dark; }'
  /** What the page was served wearing, which is the applied theme's file. */
  const SERVED = ':root { --numen-surface: #101014 }'
  /** What the page was served drawn at, which is as it is designed. */
  const SIZED = ':root { --numen-interface-scale: 1; --numen-text-scale: 1; }'

  const styled = (css: string) => {
    const one = document.createElement('style')
    one.textContent = css
    return one
  }

  /** What the head is wearing, in the order the elements stand in it. */
  const dressed = () =>
    [...document.head.querySelectorAll('style')].map((one) => one.textContent)

  const field = () => document.body.querySelector<HTMLInputElement>('.palette__field')

  const press = async (key: string) => {
    field()?.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
    await settles()
  }

  const type = async (text: string) => {
    const into = field()
    if (!into) return
    into.value = text
    into.dispatchEvent(new Event('input'))
    await settles()
  }

  /** The page as the window's handler serves it, before the window is drawn. */
  beforeEach(() => {
    for (const one of document.head.querySelectorAll('style')) one.remove()
    document.head.append(styled(PAIR), styled(SERVED), styled(SIZED))
  })

  /** The keyboard still walking, and the keyboard stood still on a row. */
  const walking = () => new Promise((done) => setTimeout(done, 60))
  const stands = () => new Promise((done) => setTimeout(done, 200))

  /** Every tenth the interface goes between, as a person reads them. */
  const TENTHS = [
    '80%',
    '90%',
    '100%',
    '110%',
    '120%',
    '130%',
    '140%',
    '150%',
    '160%',
    '170%',
    '180%',
    '190%',
    '200%',
  ]

  /** The commands open, and the one the words typed name taken up. */
  const over = async (typed: string) => {
    const window = await drawnWithPalette()
    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
    await settles()
    await type(typed)
    await press('Enter')
    return window
  }

  /** Every row the words typed leave, in the order they are drawn. */
  const left = () =>
    [...document.body.querySelectorAll('.palette__item')].map((one) =>
      one.querySelector('.palette__name')?.textContent?.trim(),
    )

  /** The bands standing, by the name each carries. */
  const bands = () =>
    [...document.body.querySelectorAll('.palette__title')].map((one) => one.textContent?.trim())

  /** The second line of every row drawn, and nothing for a row carrying none. */
  const beside = () =>
    [...document.body.querySelectorAll('.palette__item')].map((one) =>
      one.querySelector('.palette__detail')?.textContent?.trim(),
    )

  describe('the words a person types for them', () => {
    it('find the theme by “theme”, and light and dark by either word', async () => {
      await drawnWithPalette()
      globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
      await settles()

      await type('theme')
      expect(left()).toStrictEqual(['Change the theme'])

      await type('light')
      expect(left()).toStrictEqual(['Light or dark'])

      await type('dark')
      expect(left()).toStrictEqual(['Light or dark'])
    })

    it('find the two sizes by “interface”, by “font” and by “reading”', async () => {
      await drawnWithPalette()
      globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
      await settles()

      await type('interface')
      expect(left()).toStrictEqual(['Interface size'])

      await type('font')
      expect(left()).toStrictEqual(['Reading font size'])

      await type('reading')
      expect(left()).toStrictEqual(['Reading font size'])
    })

    it('turn up both of them, and nothing else, for the word they share', async () => {
      await drawnWithPalette()
      globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
      await settles()

      await type('size')

      expect(left()).toStrictEqual(['Interface size', 'Reading font size'])
    })
  })

  describe('the step that offers the themes', () => {
    it('draws the shelves as bands, and opens on the theme the window wears', async () => {
      await over('theme')

      expect(bands()).toStrictEqual(['Ships with numen', 'Your own themes'])
      expect(document.body.querySelector('[data-here]')?.textContent).toContain('numen')
      expect(dressed()).toStrictEqual([PAIR, SERVED, SIZED])
    })

    it('wears the theme the keyboard walks onto', async () => {
      await over('theme')

      await press('ArrowDown')

      expect(dressed()).toStrictEqual([PAIR, ':root { --numen-surface: mine:sea }', SIZED])
      expect(asked.worn).toStrictEqual([])
    })

    it('puts back the theme the settings name when the step is left', async () => {
      await over('theme')
      await press('ArrowDown')

      await press('Escape')

      expect(dressed()).toStrictEqual([PAIR, SERVED, SIZED])
      expect(asked.worn).toStrictEqual([])
    })

    it('keeps wearing the theme that was chosen, and writes it down', async () => {
      await over('theme')
      await press('ArrowDown')

      await press('Enter')

      expect(asked.worn).toStrictEqual(['mine:sea system 1/1'])
      expect(dressed()).toStrictEqual([PAIR, ':root { --numen-surface: mine:sea }', SIZED])
    })
  })

  describe('the step that offers light and dark', () => {
    it('opens on the half the tokens are read as, in a band of its own', async () => {
      await over('light')

      expect(bands()).toStrictEqual(['Light and dark'])
      expect(left()).toStrictEqual(['Follow the system', 'Light', 'Dark'])
      expect(dressed()).toStrictEqual([PAIR, SERVED, SIZED])
    })

    it('reads the tokens as the half the keyboard walks onto', async () => {
      await over('light')

      await press('ArrowDown')

      expect(dressed()).toStrictEqual([':root { color-scheme: light; }', SERVED, SIZED])
      expect(asked.worn).toStrictEqual([])
    })

    it('puts back the half the settings name when the step is left', async () => {
      await over('light')
      await press('ArrowDown')

      await press('Escape')

      expect(dressed()).toStrictEqual([PAIR, SERVED, SIZED])
      expect(asked.worn).toStrictEqual([])
    })

    it('writes the half that was chosen, leaving the theme where it was', async () => {
      await over('dark')
      await press('End')

      await press('Enter')

      expect(asked.worn).toStrictEqual(['preset:numen dark 1/1'])
      expect(dressed()).toStrictEqual([':root { color-scheme: dark; }', SERVED, SIZED])
    })
  })

  /**
   * The row the keyboard is standing on, read off the document rather than out
   * of the list the window handed the palette.
   */
  const standingOn = () =>
    document.body.querySelector('[data-here] .palette__name')?.textContent?.trim()

  describe('a step opened over a setting', () => {
    /** The page as the handler serves it, dressed as the settings say. */
    const serves = (mode = PAIR, sizes = SIZED) => {
      for (const one of document.head.querySelectorAll('style')) one.remove()
      document.head.append(styled(mode), styled(SERVED), styled(sizes))
      return [mode, SERVED, sizes]
    }

    it('stands on the theme the settings name, and goes on wearing it', async () => {
      said.applied = 'mine:sea'
      const was = serves()

      await over('theme')

      expect(standingOn()).toBe('sea')
      expect(dressed()).toStrictEqual(was)
      expect(asked.worn).toStrictEqual([])
    })

    it('stands on the half the tokens are read as, and goes on reading them so', async () => {
      said.mode = 'dark'
      const was = serves(':root { color-scheme: dark; }')

      await over('light')

      expect(standingOn()).toBe('Dark')
      expect(dressed()).toStrictEqual(was)
    })

    it('stands on the size the interface is drawn at, and leaves it there', async () => {
      said.sizes = { interfaceScale: 1.5, textScale: 1 }
      const was = serves(PAIR, ':root { --numen-interface-scale: 1.5; --numen-text-scale: 1; }')

      await over('interface')
      await stands()

      expect(standingOn()).toBe('150%')
      expect(dressed()).toStrictEqual(was)
    })

    it('stands on a size between two steps, which is the row put in for it', async () => {
      said.sizes = { interfaceScale: 1, textScale: 1.17 }
      const was = serves(PAIR, ':root { --numen-interface-scale: 1; --numen-text-scale: 1.17; }')

      await over('reading')
      await stands()

      expect(standingOn()).toBe('117%')
      expect(dressed()).toStrictEqual(was)
    })

    it('leaves the keyboard where typing puts it, and does not walk it back', async () => {
      said.sizes = { interfaceScale: 1.5, textScale: 1 }
      serves(PAIR, ':root { --numen-interface-scale: 1.5; --numen-text-scale: 1; }')
      await over('interface')

      await type('137')

      expect(standingOn()).toBe('137%')
    })
  })

  describe('the step that offers how large the interface is drawn', () => {
    it('opens on the size the window is drawn at, in a band of its own', async () => {
      await over('interface')

      expect(bands()).toStrictEqual(['How large the interface is drawn'])
      expect(left()).toStrictEqual(TENTHS)
      expect(dressed()).toStrictEqual([PAIR, SERVED, SIZED])
    })

    it('draws a second line on the one row the window is drawn at, and no other', async () => {
      await over('interface')

      expect(beside()).toStrictEqual(
        TENTHS.map((title) => (title === '100%' ? 'Current' : undefined)),
      )
    })

    it('offers a number typed into the field, and narrows to it alone', async () => {
      await over('interface')

      await type('137')

      expect(left()).toStrictEqual(['137%'])
    })

    it('draws and writes a number typed, the way it does a step', async () => {
      await over('interface')
      await type('137')

      await press('Enter')

      expect(asked.worn).toStrictEqual(['preset:numen system 1.37/1'])
      expect(dressed().at(-1)).toBe(':root { --numen-interface-scale: 1.37; --numen-text-scale: 1; }')
    })

    it('offers no row for a number the range does not reach, and says nothing', async () => {
      await over('interface')

      await type('250')

      expect(left()).toStrictEqual([])
      expect(document.body.querySelector('.palette__silence')?.textContent?.trim()).toBe('Nothing')
    })

    it('narrows the steps, and offers nothing of its own, for digits inside one', async () => {
      await over('interface')

      await type('15')

      expect(left()).toStrictEqual(['150%'])
    })

    it('holds the size until the keyboard has stood on the row it walked to', async () => {
      await over('interface')
      await press('End')

      await walking()
      expect(dressed()).toStrictEqual([PAIR, SERVED, SIZED])

      await stands()
      expect(dressed().at(-1)).toBe(':root { --numen-interface-scale: 2; --numen-text-scale: 1; }')
      expect(asked.worn).toStrictEqual([])
    })

    it('puts back the size the settings name when the step is left', async () => {
      await over('interface')
      await press('End')
      await stands()

      await press('Escape')

      expect(dressed()).toStrictEqual([PAIR, SERVED, SIZED])
      expect(asked.worn).toStrictEqual([])
    })

    it('keeps drawing at the size that was chosen, and writes it down', async () => {
      await over('interface')
      await press('End')

      await press('Enter')

      expect(asked.worn).toStrictEqual(['preset:numen system 2/1'])
      expect(dressed().at(-1)).toBe(':root { --numen-interface-scale: 2; --numen-text-scale: 1; }')
    })

    it('says what the settings refused, where the window says what it could not do', async () => {
      said.refused = 'appearance.interface_scale is 2, which is outside 0.8 to 1.5'
      const window = await over('interface')
      await press('End')

      await press('Enter')
      await settles()

      expect(cards(window).join(' ')).toContain('outside 0.8 to 1.5')
      expect(dressed()).toStrictEqual([PAIR, SERVED, SIZED])
    })
  })

  describe('the editor of an open note', () => {
    /** A note in a tab of its own, with the editor's measurements taken. */
    const opened = async () => {
      const window = await drawnWithPalette()
      window.findComponent(Plex).vm.$emit('show', nodeInPlex(window), 'here')
      await settles()
      asked.measured = 0
      globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
      await settles()
      return window
    }

    it('takes its measurements again at the size the keyboard is held on', async () => {
      await opened()
      await type('interface')
      await press('Enter')

      await press('End')
      await stands()

      expect(asked.measured).toBe(1)
    })

    it('takes them again at the size that was chosen', async () => {
      await opened()
      await type('reading')
      await press('Enter')

      await press('End')
      await press('Enter')
      await settles()

      expect(asked.measured).toBe(1)
    })

    it('is left alone while the keyboard is walking rows, and by a theme', async () => {
      await opened()
      await type('theme')
      await press('Enter')

      await press('ArrowDown')
      await walking()

      expect(asked.measured).toBe(0)
    })
  })

  describe('the step that offers how large the text is set', () => {
    it('opens on the sizes the reading text goes between', async () => {
      await over('reading')

      expect(bands()).toStrictEqual(['How large the text is set'])
      // The far end of this one falls between two steps, and is offered there.
      expect(left()).toStrictEqual([...TENTHS.slice(0, 10), '175%'])
    })

    it('writes the size that was chosen beside the interface’s, which stands', async () => {
      await over('reading')
      await press('End')

      await press('Enter')

      expect(asked.worn).toStrictEqual(['preset:numen system 1/1.75'])
      expect(dressed().at(-1)).toBe(':root { --numen-interface-scale: 1; --numen-text-scale: 1.75; }')
    })
  })
})

describe('the window with no note to show', () => {
  it('says the vault could not be read, and that nothing was read from it', async () => {
    said.opening = null
    said.failed = 'the vault folder is not there'

    const window = await drawn()

    const corner = cards(window)
    expect(corner[0]).toContain('the vault folder is not there')
    expect(corner[1]).toBe('nothing was read')
  })

  it('says nothing when the vault was read and holds none', async () => {
    said.opening = null
    said.failed = ''

    const window = await drawn()

    expect(cards(window)).toStrictEqual([])
  })
})

describe('every road to a file', () => {
  /**
   * Each road, by what it is called, as a gesture on a window standing on one
   * file. They are listed here so that every one of them is asked the same
   * four questions, and a road opening a file some other way is a road missing
   * from this list.
   */
  const ROADS: Record<string, (window: VueWrapper, path: string) => Promise<void>> = {
    'the plex': async (window) => {
      window.findComponent(Plex).vm.$emit('show', nodeInPlex(window), 'here')
    },
    'the tree': async (window, path) => {
      window.findComponent(Tree).vm.$emit('activate', path)
    },
    'the palette': async (window, path) => {
      await typedIn(window, 'ani')
      window.findComponent(Palette).vm.$emit('choose', path, 'note')
    },
    'the search': async (window, path) => {
      await typedIn(window, 'ani')
      window.findComponent(Palette).vm.$emit('choose', `text:${path}:0`, 'note')
    },
    'a command': async (window) => {
      pressing('p')
      await settles()
      window.findComponent(Palette).vm.$emit('choose', 'read', 'read')
    },
    // The two roads that arrive naming a stretch of a source's own text: a link
    // in an answer the agent wrote, and a place asked for from outside the
    // window altogether.
    'a link in an answer': async (window, path) => {
      const turn = { id: 'a', voice: 'answered' as const, text: '' }
      const link = linkOf({ path, start: 0, length: 4 })
      window
        .findComponent(Agent)
        .vm.$emit('follow', turn, link, new MouseEvent('click', { cancelable: true }))
    },
    'a place asked for from outside': async (_, path) => {
      outside.asks({ path, start: 0, length: 4 })
    },
  }

  /** A keystroke the window answers, which the palette and the commands are. */
  const pressing = (key: string) =>
    globalThis.dispatchEvent(
      new KeyboardEvent('keydown', { key, ctrlKey: true, cancelable: true }),
    )

  /** The search open, with words typed into it and the answers back. */
  const typedIn = async (window: VueWrapper, typed: string) => {
    pressing('k')
    await settles()
    window.findComponent(Palette).vm.$emit('update:modelValue', typed)
    await new Promise((done) => setTimeout(done, HELD))
  }

  /** The root of the vault holds one file of each of four. */
  const root = folders['']
  beforeEach(() => {
    folders[''] = [
      { path: 'Animals.md', name: 'Animals.md', folder: false, kind: 'note', type: 'deck' },
      { path: 'Animal.md', name: 'Animal.md', folder: false, kind: 'note', type: 'stencil' },
      { path: 'Ants.md', name: 'Ants.md', folder: false, kind: 'note', type: 'note' },
      { path: 'Ants.epub', name: 'Ants.epub', folder: false, kind: 'book', type: 'note' },
    ]
  })
  afterEach(() => {
    folders[''] = root ?? []
  })

  /** What the window drew, having been taken to one file by one road. */
  const taken = async (
    road: (window: VueWrapper, path: string) => Promise<void>,
    path: string,
    type: 'note' | 'deck' | 'stencil',
  ): Promise<readonly string[]> => {
    said.types = { [path]: type }
    said.opening = path
    said.names = [nameSaid(path, path, type)]
    said.passages = [passageSaid(path, path, type)]

    const window = await drawn()
    await road(window, path)
    await settles()
    await settles()
    return paneKinds(window).flat()
  }

  for (const [name, road] of Object.entries(ROADS)) {
    it(`opens a deck in the editor of its cards, reached by ${name}`, async () => {
      expect(await taken(road, 'Animals.md', 'deck')).toContain('deck')
    })

    it(`opens a stencil in the editor of its fields and faces, reached by ${name}`, async () => {
      expect(await taken(road, 'Animal.md', 'stencil')).toContain('stencil')
    })

    it(`opens an ordinary note in the editor of its prose, reached by ${name}`, async () => {
      const drew = await taken(road, 'Ants.md', 'note')

      expect(drew).toContain('note')
      expect(drew).not.toContain('deck')
      expect(drew).not.toContain('stencil')
    })

    // The vault is never asked about the book by anything but the window, and
    // the palette is told it turned up a note: the path alone has to be enough.
    it(`opens a book in the reader, reached by ${name}`, async () => {
      const drew = await taken(road, 'Ants.epub', 'note')

      expect(drew).toContain('document')
      expect(drew).not.toContain('note')
    })
  }
})
