/**
 * A deck tab drawn: where a mark lands, and what a gesture in the grid reaches.
 *
 * What is wrong with a card is drawn inside that card's tile, so this asks the
 * tile and not the tab.
 */
// @vitest-environment jsdom
import { enableAutoUnmount, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import type { Cards, Carded, Problem } from '../core'
import { putting } from '../putting'
import { windowing } from '../windowing'
import { DECK } from '../workspace'
import DeckTab from './DeckTab.vue'
import { decking, type Held } from './deck'
import { WORDS as words } from './words'

/** The one place a file is opened from. Nothing here opens one. */
const puts = () => putting({ standing: async () => new Map() })

/** A moment for whatever the tab asked the vault for to come back. */
const settles = () => new Promise((done) => setTimeout(done, 0))

// Each tab is drawn into the page, so the one before it goes before the next
// stands: a mark is teleported to the tile a selector finds in the whole page.
enableAutoUnmount(afterEach)

const CARDS: readonly Carded[] = [
  {
    name: 'Llama',
    stencil: 'Animal',
    stencilAt: 'Animal.md',
    lead: '',
    values: [{ field: 'Height', text: '45"' }],
  },
  { name: '', stencil: 'Animal', stencilAt: 'Animal.md', lead: '', values: [] },
]

/** A window with one deck open, drawn. */
const drawn = async (problems: readonly Problem[] = []) => {
  const core: Cards = {
    stencils: async () => ({
      stencils: [{ path: 'Animal.md', title: 'Animal', fields: ['Name', 'Height'] }],
      held: 1,
    }),
    makeDeck: async (title) => ({ path: `${title}.md`, refusal: null }),
    makeStencil: async (title) => ({ path: `${title}.md`, refusal: null }),
    renameField: async () => ({
      decks: [],
      cards: 0,
      notWritten: [],
      refusal: null,
      changed: false,
      at: '',
    }),
    readDeck: async (path) => ({
      deck: { path, title: 'Animals', preamble: '', cards: CARDS, tail: '', problems },
      refusal: null,
      at: 'read',
      bound: 0,
    }),
    writeDeck: async () => ({ refusal: null, changed: false, at: 'written', bound: 0 }),
    readStencil: async () => ({ stencil: null, refusal: 'missing', at: '' }),
    writeStencil: async () => ({ refusal: null, changed: false, at: '' }),
  }

  const held = windowing()
  const decks = decking(core, held.host, puts())
  held.declares([decks.kind])
  const id = await held.opens(DECK, 'Animals.md')
  await settles()
  const tab = held.host.holds<Held>(DECK, id) as Held
  // A mark is teleported into the tile it is about, so the grid has to stand in
  // the document for the tile to be found.
  const window = mount(DeckTab, { props: { held: tab }, attachTo: document.body })
  await settles()
  return { window, tab, decks }
}

/** The tile one card is drawn as, by the identity the window gave that card. */
const tileOf = (window: Awaited<ReturnType<typeof drawn>>['window'], card: string) =>
  window.find(`[data-card="${card}"]`)

const nameless: Problem = {
  fault: 'cardWithoutAName',
  card: 1,
  face: null,
  field: '',
  text: 'a card with no name',
}

describe('a deck drawn', () => {
  it('draws a tile for every card the vault read', async () => {
    const { window } = await drawn()

    expect(window.findAll('[data-card]')).toHaveLength(2)
  })

  it('offers every stencil the vault holds once the plus is pressed', async () => {
    const { window } = await drawn()

    expect(window.find('[data-cut="Animal"]').exists()).toBe(false)
    await window.find('[data-plus]').find('button').trigger('click')

    expect(window.find('[data-cut="Animal"]').exists()).toBe(true)
  })

  it('writes a card cut by the stencil that was chosen', async () => {
    const { window, tab } = await drawn()

    await window.find('[data-plus]').find('button').trigger('click')
    await window.find('[data-cut="Animal"]').trigger('click')

    expect(tab.deck().cards.at(-1)?.stencil).toBe('Animal')
    expect(tab.deck().cards.at(-1)?.values).toStrictEqual([{ field: 'Height', text: '' }])
  })
})

describe('a mark on a tile', () => {
  it('is drawn inside the tile of the card it was read against', async () => {
    const { window, tab } = await drawn([nameless])
    const second = tab.deck().cards[1]?.id ?? ''

    expect(tileOf(window, second).find('[data-wrong]').text()).toBe('a card with no name')
  })

  it('is drawn on no other tile', async () => {
    const { window, tab } = await drawn([nameless])
    const first = tab.deck().cards[0]?.id ?? ''

    expect(tileOf(window, first).find('[data-wrong]').exists()).toBe(false)
  })

  it('is nowhere at all where the vault reported nothing wrong', async () => {
    const { window } = await drawn()

    expect(window.findAll('[data-wrong]')).toHaveLength(0)
  })

  it('goes when the card it stood on is taken out of the deck', async () => {
    const { window, tab } = await drawn([nameless])
    const second = tab.deck().cards[1]?.id ?? ''

    tab.removes(second)
    await settles()

    expect(window.findAll('[data-wrong]')).toHaveLength(0)
  })

  it('stands with the card and not with its place, so removing another leaves it', async () => {
    const { window, tab } = await drawn([nameless])
    const first = tab.deck().cards[0]?.id ?? ''
    const second = tab.deck().cards[1]?.id ?? ''

    tab.removes(first)
    await settles()

    expect(tileOf(window, second).find('[data-wrong]').text()).toBe('a card with no name')
  })
})

describe('what is wrong with the file itself', () => {
  it('is drawn above the grid, standing on no tile', async () => {
    const { window } = await drawn([
      { fault: 'unknown', card: null, face: null, field: '', text: 'the whole file' },
    ])

    expect(window.find(`[aria-label="${words.problems}"]`).text()).toBe('the whole file')
    expect(window.findAll('[data-wrong]')).toHaveLength(0)
  })

  it('is drawn nowhere where every problem stands on a card', async () => {
    const { window } = await drawn([nameless])

    expect(window.find(`[aria-label="${words.problems}"]`).exists()).toBe(false)
  })
})

describe('a gesture in the grid', () => {
  it('renames the card the box belongs to', async () => {
    const { window, tab } = await drawn()
    const first = tab.deck().cards[0]?.id ?? ''
    const box = tileOf(window, first).find('[data-value="Name"]')

    await box.setValue('Vicuña')
    await box.trigger('change')

    expect(tab.deck().cards[0]?.name).toBe('Vicuña')
  })

  it('leaves every other card as it was', async () => {
    const { window, tab } = await drawn()
    const first = tab.deck().cards[0]?.id ?? ''
    const box = tileOf(window, first).find('[data-value="Name"]')

    await box.setValue('Vicuña')
    await box.trigger('change')

    expect(tab.deck().cards[1]?.name).toBe('')
  })
})

describe('the question a file that changed on disk puts', () => {
  it('is not drawn while the deck is as the file has it', async () => {
    const { window } = await drawn()

    expect(window.text()).not.toContain(words.overtaken)
  })
})
