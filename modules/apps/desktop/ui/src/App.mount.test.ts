/**
 * The window drawn, in a document, with both ports mocked away.
 *
 * A vault that could not be read and a vault that holds nothing both leave the
 * window with no note to show, and the window says which of the two it is. The
 * failure of this is silent: one wrong line and an unreadable vault reads as an
 * empty one.
 */
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

const { said, held } = vi.hoisted(() => ({
  /** What the mocked vault answers about itself, set before the window draws. */
  said: { ready: true, failed: '' },
  /** A stream that stays open, so nothing the window follows ever ends. */
  async *held(): AsyncGenerator<never> {
    await new Promise<never>(() => {})
  },
}))

vi.mock('./vault', () => ({
  vault: {},
  documents: {},
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
    // No note to open: the window has nothing to show either way.
    opening: async () => null,
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

/**
 * The window mounted and left to settle. Every child is stubbed, so what is
 * asked here is what the window itself draws.
 */
async function drawn() {
  const window = mount(App, { shallow: true })
  await new Promise((settle) => setTimeout(settle, 0))
  return window
}

describe('the window with no note to show', () => {
  it('says nothing was read when the vault could not be read', async () => {
    said.ready = true
    said.failed = 'the vault folder is not there'

    const window = await drawn()

    expect(window.find('.waiting').text()).toBe('nothing was read')
    expect(window.text()).not.toContain('this vault holds no notes')
    expect(window.find('.warning').text()).toContain('the vault folder is not there')
  })

  it('says the vault holds no notes when it was read and holds none', async () => {
    said.ready = true
    said.failed = ''

    const window = await drawn()

    expect(window.find('.waiting').text()).toBe('this vault holds no notes')
    expect(window.text()).not.toContain('nothing was read')
    expect(window.find('.warning').exists()).toBe(false)
  })
})
