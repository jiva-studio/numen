/**
 * Every kind of tab the window declares, drawn in a document.
 *
 * The window is told about its kinds in one list and draws whichever the tab
 * holds. Nothing else asks whether that list and what is drawn agree: a kind
 * declared and never drawn looks exactly like a window with fewer kinds.
 */
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { Agent, Editor, Plex, Reader } from '@numen/ui'
import AgentTab from './agent/AgentTab.vue'
import DocumentTab from './document/DocumentTab.vue'
import NoteTab from './note/NoteTab.vue'
import PlexTab from './plex/PlexTab.vue'

const { held } = vi.hoisted(() => ({
  /** A stream that stays open, so nothing the window follows ever ends. */
  async *held(): AsyncGenerator<never> {
    await new Promise<never>(() => {})
  },
}))

/** A vault holding one note, and one document to read. */
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
      ready: true,
      failed: '',
      unwatched: '',
      unreachable: '',
      chunks: 0n,
      embedded: 0n,
      embedding: false,
    }),
    opening: async () => ({ path: 'Root.md' }),
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

vi.mock('./agent', () => ({ core: { ask: held, finish: async () => {} } }))

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
 * The window drawn, with what each kind draws inside it stubbed. The tabs
 * themselves are the window's own, and they are what is asked about here.
 */
async function drawn() {
  const window = mount(App, {
    global: { stubs: { Plex: true, Editor: editor, Agent: true, Reader: reader, Palette: true } },
  })
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
