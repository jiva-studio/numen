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
import { Agent, Editor, Palette, Plex, Reader } from '@numen/ui'
import AgentTab from './agent/AgentTab.vue'
import DocumentTab from './document/DocumentTab.vue'
import NoteTab from './note/NoteTab.vue'
import PlexTab from './plex/PlexTab.vue'

const { said, held } = vi.hoisted(() => ({
  /** What the mocked vault answers about itself, set before the window draws. */
  said: { ready: true, failed: '', opening: 'Root.md' as string | null },
  /** A stream that stays open, so nothing the window follows ever ends. */
  async *held(): AsyncGenerator<never> {
    await new Promise<never>(() => {})
  },
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
    changes: held,
    editing: held,
    tasks: held,
    focus: held,
    quitting: held,
    flushed: async () => {},
    names: async () => [],
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
