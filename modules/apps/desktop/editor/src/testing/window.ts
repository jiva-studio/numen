/**
 * The window drawn in a document, with every port it reaches mocked away.
 *
 * The application answers through four modules, and a test of the window
 * declares what each of them says here: `said` is what the vault answers,
 * `asked` is what it was asked, and `drawn` puts the window on the screen.
 */
import { afterEach, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount, type VueWrapper } from '@vue/test-utils'
import { panesOf, WorkspaceLayout, type Workspace } from '@numen/ui'
import type { Tab } from '../core'


const { said, held, asked, listed, folders, maker, stands, outside } = vi.hoisted(() => ({
  /**
   * What the vault holds at a path. A book and a note are told apart by the
   * name the file carries, the way the vault itself tells them apart, and which
   * of three a note is is what a test said.
   */
  stands: (path: string) => {
    if (/\.(epub|pdf)$/u.test(path)) return { kind: 'book' as const, type: 'note' as const }
    if (/\.(mp3|m4a|wav)$/u.test(path)) return { kind: 'recording' as const, type: 'note' as const }
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
  maker: (() => {
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
      kind: 'note' | 'book' | 'recording' | 'other'
    }[],
    /**
     * Which of three the note at each path is, as the vault answers it. A path
     * it says nothing about is the ordinary note the window reads it as.
     */
    types: {} as Record<string, 'note' | 'deck' | 'stencil'>,
    /** How many of the spans the index holds carry a vector. */
    embedded: 0,
    /** Whether the list of vaults answers at all. */
    listable: true,
    /** What the settings refuse a choice, which is a size outside its bounds. */
    refused: '',
    /** What the settings say the window is drawn as, which a test may set. */
    applied: 'preset:numen',
    mode: 'system' as 'system' | 'light' | 'dark',
    sizes: { interfaceScale: 1, textScale: 1 },
    /** The recording the vault answers with, and the words written down in it. */
    transcribed: {
      duration: 60_000,
      mediaUrl: 'numen://recording/talk.mp3',
      mediaType: 'audio/mpeg',
      cues: [{ text: 'the first thing said', from: 0, to: 4000 }] as {
        text: string
        from: number
        to: number
      }[],
      /** False while a run writing the transcript holds it. */
      editable: true,
    },
    /**
     * What the file at each path carries, as the application answers it. A path
     * it says nothing about carries nothing.
     */
    carries: {} as Record<string, Record<string, string>>,
    /** Whether the application answers what a file carries at all. */
    carrying: true,
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
    urls: [] as string[],
    /** The decks and the stencils the window asked for, in the order it asked. */
    cards: [] as string[],
    /** Each field rename the window asked the vault for. */
    renamedField: [] as string[],
    /** The cards each of those deck writes carried, by name. */
    wrote: [] as string[],
    worn: [] as string[],
    /** How often an open editor was told to take its measurements again. */
    measured: 0,
    /** The vaults the window asked to be shown, in the order it asked. */
    opened: [] as string[],
    /** The recordings the window listened to, in the order it asked. */
    listened: [] as string[],
    /** The transcripts the window wrote, as the words each carried. */
    transcribed: [] as string[],
    /** Every file the window asked what it carries, in the order it asked. */
    carried: [] as string[],
    /** Each run the window asked for, and each transcript it dropped. */
    ran: [] as string[],
    /** How often a folder was asked for, which is a vault being added. */
    chose: 0,
    /** What the window said the person has open, the last of it last. */
    attending: [] as { tabs: readonly Tab[]; front: string }[],
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

vi.mock('../vault', () => ({
  vault: {},
  vaults: {
    list: async () => listed,
    choose: async () => {
      asked.chose += 1
      return ''
    },
    add: async () => ({ vault: null, refusal: null }),
    rename: async () => ({ vault: null, refusal: null }),
    remove: async () => null,
    open: async (id: string) => {
      asked.opened.push(id)
      return null
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
      chunkCount: 0n,
      embeddedCount: BigInt(said.embedded),
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
    makeURL: async (url: string, folder: string) => {
      asked.urls.push(`${url} ${folder}`)
      return { path: folder ? `${folder}/made.url` : 'made.url', refusal: null }
    },
    changes: held,
    editing: held,
    tasks: held,
    focus: outside.stream,
    attending: async (open: { tabs: readonly Tab[]; front: string }) => {
      asked.attending.push(open)
    },
    quitting: held,
    flushed: async () => {},
    names: async () => said.names,
    search: async () => said.passages,
    fileKinds: async (paths: readonly string[]) =>
      new Map(paths.map((path) => [path, stands(path)])),
    headings: async () => new Map(),
  },
}))

vi.mock('../assets', () => ({
  documents: {
    shape: async () => ({ pages: 1, pageSizes: [{ wide: 100, high: 100 }] }),
    page: () => '',
    places: async () => [],
  },
  recordings: {
    listened: async (path: string) => {
      asked.listened.push(path)
      const { duration, mediaUrl, mediaType } = said.transcribed
      return { duration, mediaUrl, mediaType, url: '' }
    },
    cues: async () => ({ cues: said.transcribed.cues, prose: '', editable: said.transcribed.editable }),
    writes: async (path: string, cues: readonly { text: string }[]) => {
      asked.transcribed.push(`${path} ${cues.map((one) => one.text).join(' / ')}`)
    },
    plays: async (path: string, stretch: { start: number }) =>
      said.transcribed.cues.find((one) => one.from >= stretch.start)?.from ?? null,
  },
}))

vi.mock('../artifacts', () => ({
  running: {
    carries: async (path: string) => {
      asked.carried.push(path)
      if (!said.carrying) throw new Error('what the file carries cannot be asked')
      return said.carries[path] ?? {}
    },
    makes: async (path: string, of: string) => {
      asked.ran.push(`${of} ${path}`)
      return { able: true, of, made: 'queued', error: '' }
    },
    drops: async (path: string) => {
      asked.ran.push(`drop ${path}`)
      return true
    },
  },
}))

vi.mock('../cards/vault', () => ({
  cards: {
    // A card is named by the first field of the stencil it is cut by, so the
    // window is told of one.
    stencils: async () => ({
      stencils: [{ path: 'Animal.md', title: 'Animal', fields: ['Name'] }],
      held: 1,
    }),
    makeDeck: async (title: string, folder: string) => {
      asked.cards.push(`deck ${folder || '/'} ${title}`)
      return maker.makes(title, folder)
    },
    makeStencil: async (title: string, folder: string, fields: readonly string[]) => {
      asked.cards.push(`stencil ${folder || '/'} ${title} [${fields.join(', ')}]`)
      return maker.makes(title, folder)
    },
    renameField: async (path: string, from: string, to: string) => {
      asked.renamedField.push(`${path} ${from} ${to}`)
      return said.renaming
    },
    readDeck: async (path: string) => ({
      deck: {
        path,
        title: path,
        preamble: '',
        cards: [],
        sections: [],
        tail: '',
        problems: [],
      },
      refusal: null,
      at: 'a1',
      bound: 0,
    }),
    writeDeck: async (
      path: string,
      deck: { cards: readonly { values: readonly { text: string }[] }[] },
    ) => {
      asked.cards.push(`deck ${path}`)
      asked.wrote.push(deck.cards.map((card) => card.values[0]?.text ?? '').join(', '))
      return { refusal: null, changed: false, at: 'a2', bound: 0 }
    },
    readStencil: async (path: string) => ({
      stencil: { path, title: path, fields: [], faces: [], problems: [] },
      refusal: null,
      at: 'a1',
    }),
    writeStencil: async (path: string) => {
      asked.cards.push(`stencil ${path}`)
      return { refusal: null, changed: false, at: 'a2' }
    },
  },
}))

vi.mock('../agent/core', () => ({ core: { ask: held, finish: async () => {} } }))

vi.mock('../settings/theme', () => ({
  themes: {
    appearance: async () => ({
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

const App = (await import('../App.vue')).default

/** A moment for whatever the window asked the vault for to come back. */
const settles = () => new Promise((done) => setTimeout(done, 0))

/** Longer than the palette debounces a keystroke before it asks the vault. */
const DEBOUNCE = 200

/** One name the vault answers a search with, of a note of some kind. */
const nameSaid = (path: string, title: string, type: 'note' | 'deck' | 'stencil' = 'note') => ({
  path,
  title,
  heading: '',
  line: -1,
  at: [] as [],
  type,
})

/** One passage the search answers with, read out of a note of some kind. */
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
  kind: 'note' as 'note' | 'book' | 'recording' | 'other',
})

/** One passage read out of a source that is not a note. */
const sourceSaid = (path: string, kind: 'book' | 'recording') => ({
  ...passageSaid(path, ''),
  isNote: false,
  kind,
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
  said.embedded = 0
  said.listable = true
  said.refused = ''
  said.applied = 'preset:numen'
  said.mode = 'system'
  said.sizes = { interfaceScale: 1, textScale: 1 }
  said.transcribed = {
    duration: 60_000,
    mediaUrl: 'numen://recording/talk.mp3',
    mediaType: 'audio/mpeg',
    cues: [{ text: 'the first thing said', from: 0, to: 4000 }],
    editable: true,
  }
  said.carries = {}
  said.carrying = true
  said.renaming = {
    decks: [],
    cards: 0,
    notWritten: [],
    refusal: null,
    changed: false,
    at: 'a2',
  }
  maker.forget()
  outside.forget()
  asked.made = []
  asked.cards = []
  asked.renamedField = []
  asked.wrote = []
  asked.renamed = []
  asked.removed = []
  asked.moved = []
  asked.folders = []
  asked.worn = []
  asked.measured = 0
  asked.opened = []
  asked.listened = []
  asked.transcribed = []
  asked.carried = []
  asked.ran = []
  asked.chose = 0
  asked.attending = []
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
      // A stub is named by the binding the component is drawn through, and the
      // window's command palette draws the library's under `Palette`.
      stubs: {
        Plex: true,
        Editor: editor,
        Agent: true,
        Reader: reader,
        Palette: true,
        Tree: true,
      },
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
 *
 * The tab is found by the name the window addresses it by, so this harness
 * holds no screen's file.
 */
const nodeInPlex = (window: VueWrapper): string => {
  const state = window.findComponent({ name: 'PlexTab' }).props('state') as {
    picture: { value: { nodes: readonly { id: string }[] } | null }
  }
  return state.picture.value?.nodes[0]?.id ?? ''
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
const layoutOf = (window: VueWrapper): Workspace =>
  window.findComponent(WorkspaceLayout).props('modelValue') as Workspace

/** What each pane holds, by the kind each of its tabs is filed under. */
const paneKinds = (window: VueWrapper): readonly (readonly string[])[] =>
  panesOf(layoutOf(window).root).map((one) => one.tabs.map((tab) => tab.split(':')[0] ?? ''))

/** What each tab of the window is called, in the order the strip has them. */
const tabsOf = (window: VueWrapper): readonly { id: string; title: string }[] =>
  (window.findComponent(WorkspaceLayout).props('tabs') as readonly { id: string; title: string }[]) ??
  []

export {
  asked,
  cards,
  maker,
  DEBOUNCE,
  drawn,
  drawnWithPalette,
  editor,
  folders,
  layoutOf,
  listed,
  nameSaid,
  nodeInPlex,
  outside,
  paneKinds,
  passageSaid,
  reader,
  said,
  settles,
  sourceSaid,
  tabsOf,
}
