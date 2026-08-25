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
  Plex,
  Reader,
  Workspace,
  type PaletteBand,
} from '@numen/ui'
import AgentTab from './agent/AgentTab.vue'
import DocumentTab from './document/DocumentTab.vue'
import NoteTab from './note/NoteTab.vue'
import PlexTab from './plex/PlexTab.vue'
import { WORDS as plexWords } from './plex/words'
import { plexCalled } from './workspace'

const { said, held, asked, listed } = vi.hoisted(() => ({
  /** What the mocked vault answers about itself, set before the window draws. */
  said: {
    ready: true,
    failed: '',
    opening: 'Root.md' as string | null,
    names: [] as { path: string; title: string; heading: string; line: number; at: [] }[],
    /** What the settings refuse a choice, which is a size outside its bounds. */
    refused: '',
    /** What the settings say the window is drawn as, which a test may set. */
    applied: 'preset:numen',
    mode: 'system' as 'system' | 'light' | 'dark',
    sizes: { interfaceScale: 1, textScale: 1 },
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
    worn: [] as string[],
    /** How often an open editor was told to take its measurements again. */
    measured: 0,
  },
  /** The vaults this installation holds, and the one the window is showing. */
  listed: {
    vaults: [{ id: 'physics', name: 'Physics', path: '/vaults/Physics', missing: false }],
    showing: 'physics',
  },
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
  core: {
    vaults: async () => listed,
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
    changes: held,
    editing: held,
    tasks: held,
    focus: held,
    quitting: held,
    flushed: async () => {},
    names: async () => said.names,
    search: async () => [],
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
  said.refused = ''
  said.applied = 'preset:numen'
  said.mode = 'system'
  said.sizes = { interfaceScale: 1, textScale: 1 }
  asked.made = []
  asked.renamed = []
  asked.removed = []
  asked.worn = []
  asked.measured = 0
})

/**
 * The window drawn, with what each kind draws inside it stubbed. The tabs
 * themselves are the window's own, and they are what is asked about here.
 */
async function drawn() {
  const window = mount(App, {
    global: { stubs: { Plex: true, Editor: editor, Agent: true, Reader: reader, Palette: true } },
  })
  windows.push(window)
  await settles()
  await settles()
  return window
}

/** What the corner of the window is saying, one string per card. */
const cards = (window: VueWrapper): readonly string[] =>
  window.findAll('article.notice').map((card) => card.text())

/**
 * The window with a palette a person can type into. The palette draws itself at
 * the end of the document, and is read off the document.
 */
async function drawnWithPalette() {
  const window = mount(App, {
    global: { stubs: { Plex: true, Editor: editor, Agent: true, Reader: reader } },
    attachTo: document.body,
  })
  windows.push(window)
  await settles()
  await settles()
  return window
}

describe('the window as it opens', () => {
  it('draws a plex and an agent, each in a tab of its own', async () => {
    const window = await drawn()

    expect(window.findComponent(PlexTab).exists()).toBe(true)
    expect(window.findComponent(AgentTab).exists()).toBe(true)
  })

  it('offers a note, a plex and an agent to a tab with nothing in it', async () => {
    const window = await drawn()

    await window.find('[data-workspace-new]').trigger('click')

    expect(window.findAll('.blank__choice').map((one) => one.text())).toEqual([
      'New note',
      'New plex',
      'New agent',
    ])
  })
})

describe('a note asked for in the plex', () => {
  it('is drawn in a tab of its own', async () => {
    const window = await drawn()

    window.findComponent(Plex).vm.$emit('show', 'Root.md', 'here')
    await settles()

    expect(window.findComponent(NoteTab).exists()).toBe(true)
    expect(window.findComponent(NoteTab).findComponent(Editor).exists()).toBe(true)
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
    said.names = [{ path: 'physics/Entropy.md', title: 'Entropy', heading: '', line: -1, at: [] }]
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

  it('opens the step that picks a note, on the keystroke going to one draws', async () => {
    said.names = [{ path: 'physics/Entropy.md', title: 'Entropy', heading: '', line: -1, at: [] }]
    const window = await drawnWithPalette()

    const event = pressed('g')
    await settles()

    expect(event.defaultPrevented).toBe(true)
    expect(window.findComponent(Palette).props('crumb')).toBe('Go to a note')
  })

  it('travels to the note picked on that step', async () => {
    said.names = [{ path: 'physics/Entropy.md', title: 'Entropy', heading: '', line: -1, at: [] }]
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
    plex.asks({ path: 'Root.md', at: { x: 0, y: 0 }, from: null, opening: 'below' })
    plex.chose(id)
    await settles()
  }

  it('says the vault is still being read, and carries no command out', async () => {
    said.ready = false
    said.opening = null
    const window = await drawn()

    await chose(window, 'title')

    expect(cards(window)).toStrictEqual(['reading the vault…', 'The vault is still being read'])
  })

  it('lets a person put away what it told them, and forgets it', async () => {
    said.ready = false
    said.opening = null
    const window = await drawn()

    await chose(window, 'title')
    await window.findAll('article.notice button')[1]!.trigger('click')
    await settles()

    expect(cards(window)).toStrictEqual(['reading the vault…'])
    // Put away is put away for good: the window stops handing the corner a card
    // it has been told the person is finished with.
    expect(window.findComponent(Notices).props('notices')).toHaveLength(1)
  })

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

/** The palette drawn as a person meets it, with nothing of it stubbed. */
describe('the keyboard on a step that confirms', () => {
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

  /** The commands open, with the one that removes a note lit. */
  const overRemove = async () => {
    const window = await drawnWithPalette()
    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
    await settles()
    await type('remove')
    await press('Enter')
    return window
  }

  it('opens the confirmation on the answer that changes nothing', async () => {
    await overRemove()

    expect(document.body.querySelector('[data-here]')?.textContent).toContain('Keep the note')
  })

  it('removes nothing when the keystroke that opened it lands twice', async () => {
    await overRemove()

    await press('Enter')

    expect(asked.removed).toStrictEqual([])
  })

  it('removes the note when the answer that removes it is the one chosen', async () => {
    await overRemove()

    await press('ArrowDown')
    await press('Enter')

    expect(asked.removed).toStrictEqual(['Root.md false'])
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
      window.findComponent(Plex).vm.$emit('show', 'Root.md', 'here')
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
