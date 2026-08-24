/**
 * The window drawn, in a document, with both ports mocked away.
 *
 * Two things are asked here that nothing else can ask: that every kind the
 * window declares is drawn when a tab holds one, and that a vault which could
 * not be read is not shown as an empty one. The failure of the second is
 * silent — one wrong line and an unreadable vault reads as an empty one.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { Agent, branch, Editor, pane, Palette, Plex, Reader, Workspace } from '@numen/ui'
import AgentTab from './agent/AgentTab.vue'
import DocumentTab from './document/DocumentTab.vue'
import NoteTab from './note/NoteTab.vue'
import PlexTab from './plex/PlexTab.vue'

const { said, held, asked } = vi.hoisted(() => ({
  /** What the mocked vault answers about itself, set before the window draws. */
  said: {
    ready: true,
    failed: '',
    opening: 'Root.md' as string | null,
    names: [] as { path: string; title: string; heading: string; line: number; at: [] }[],
  },
  /** A stream that stays open, so nothing the window follows ever ends. */
  async *held(): AsyncGenerator<never> {
    await new Promise<never>(() => {})
  },
  /** What the window asked the vault to do to a note, in the order it asked. */
  asked: { renamed: [] as string[], removed: [] as string[] },
}))

vi.mock('./vault', () => ({
  vault: {},
  documents: {
    shape: async () => ({ pages: 1, sheets: [{ wide: 100, high: 100 }] }),
    page: () => '',
    places: async () => [],
  },
  core: {
    state: async () => ({
      name: 'Vault',
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

const App = (await import('./App.vue')).default

/** A moment for whatever the window asked the vault for to come back. */
const settles = () => new Promise((done) => setTimeout(done, 0))

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
const editor = answers('Editor', { focus: () => true, measure: () => {}, reveal: () => true })
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
  asked.renamed = []
  asked.removed = []
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

/**
 * The window with a palette a person can type into. The palette draws itself at
 * the end of the document, so it is read off the document rather than off here.
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

  /** What the note band says the commands in it are over. */
  const overNote = (window: Awaited<ReturnType<typeof drawn>>) => {
    const bands = window.findComponent(Palette).props('bands') as readonly {
      id: string
      items: readonly { detail?: string }[]
    }[]
    return bands.find((one) => one.id === 'note')?.items[0]?.detail
  }

  /** The tab of each plex the window holds, under the note it is standing on. */
  const plexTabs = (window: Awaited<ReturnType<typeof drawn>>) =>
    new Map(
      (window.findComponent(Workspace).props('tabs') as readonly { id: string; title: string }[])
        .filter((one) => one.title.startsWith('Plex · '))
        .map((one) => [one.title.replace('Plex · ', ''), one.id]),
    )

  /**
   * Two plexes, in panes of their own, standing on notes of their own. The one
   * in front is the one the person is in; the other was put in front last.
   */
  const split = async () => {
    said.names = [{ path: 'physics/Entropy.md', title: 'Entropy', heading: '', line: -1, at: [] }]
    const window = await drawn()

    pressed('p')
    await settles()
    window.findComponent(Palette).vm.$emit('choose', 'plex', 'plex')
    await settles()

    pressed('k')
    await settles()
    window.findComponent(Palette).vm.$emit('update:modelValue', 'en')
    await new Promise((done) => setTimeout(done, 200))
    window.findComponent(Palette).vm.$emit('choose', 'physics/Entropy.md', 'plex')
    await settles()

    const tabs = plexTabs(window)
    expect([...tabs.keys()].sort()).toStrictEqual(['Root', 'physics/Entropy'])
    // A tab dragged into a pane of its own. The pane the person is in is the
    // one it was dragged out of, which no tab of it changed.
    window.findComponent(Workspace).vm.$emit('update:modelValue', {
      root: branch(
        'root',
        [pane('main', [tabs.get('Root')!]), pane('aside', [tabs.get('physics/Entropy')!])],
        [0.5, 0.5],
      ),
      axis: 'horizontal',
      focus: 'main',
    })
    await settles()
    return window
  }

  it('is over the plex in the tab in front, not the plex last put in front', async () => {
    const window = await split()

    pressed('p')
    await settles()

    expect(overNote(window)).toBe('Root')
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

  it('says the vault is still being read where that is why it did nothing', async () => {
    said.ready = false
    said.opening = null
    const window = await drawn()

    await chose(window, 'title')

    expect(window.find('[role="alert"]').text()).toBe('The vault is still being read')
  })

  it('says nothing where it was taken up', async () => {
    const window = await drawn()

    await chose(window, 'title')

    expect(window.find('[role="alert"]').exists()).toBe(false)
    expect(window.findComponent(Palette).props('crumb')).toBe('Change title')
  })
})

/**
 * The palette drawn as a person meets it, so that what a keystroke reaches is
 * what the row the keyboard opened on offers.
 */
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

describe('the window with no note to show', () => {
  it('says nothing was read when the vault could not be read', async () => {
    said.opening = null
    said.failed = 'the vault folder is not there'

    const window = await drawn()

    expect(window.find('.waiting').text()).toBe('nothing was read')
    expect(window.find('.warning').text()).toContain('the vault folder is not there')
  })

  it('says nothing when the vault was read and holds none', async () => {
    said.opening = null
    said.failed = ''

    const window = await drawn()

    expect(window.find('.waiting').exists()).toBe(false)
    expect(window.find('.warning').exists()).toBe(false)
  })
})
