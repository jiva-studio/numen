/**
 * A deck tab drawn: where a mark lands, and what a gesture in the grid reaches.
 *
 * What is wrong with a card is drawn inside that card's tile, so this asks the
 * tile and not the tab.
 */
// @vitest-environment jsdom
import { enableAutoUnmount, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { StopReason } from '@numen/protocol'
import { asFailure, asValue } from '@numen/wire'
import type { ErrorCode } from '@/shared/errors'
import type { Cards, VaultCard, DeckProblem } from '@/entities/deck'
import { DEFAULTS, NOWHERE, NO_BOUNDS, type PresetChoice, type Presets } from '@/entities/deck'
import { fileOpeners } from '@/entities/tab'
import { useWindowTabs } from '@/entities/tab'
import { DECK } from '@/entities/tab'
import DeckTab from './DeckTab.vue'
import { useDeckTabs, type DeckTabState } from '../model/useDeckTabs'
import { WORDS as words } from '@/entities/deck'

/** A preset that schedules, which is what every preset here is. */
const SCHEDULING = { stops: StopReason.NOTHING, stopsOn: StopReason.NOTHING }

/** The one place a file is opened from. Nothing here opens one. */
const tabOpeners = () => fileOpeners({ fileKinds: async () => new Map() })

/** A moment for whatever the tab asked the vault for to come back. */
const settle = () => new Promise((done) => setTimeout(done, 0))

// Each tab is drawn into the page, so the one before it goes before the next
// stands: a mark is teleported to the tile a selector finds in the whole page.
enableAutoUnmount(afterEach)

afterEach(() => {
  // A menu is drawn at the end of the document, so one left open would stand
  // there while the next test looks for its own.
  document.body.innerHTML = ''
})

const CARDS: readonly VaultCard[] = [
  {
    mark: 'k7m2xq9fzp',
    sectionIndex: 0,
    heading: 'Llama',
    stencilLink: 'Animal',
    stencilPath: 'Animal.md',
    preamble: '',
    values: [
      { field: 'Name', text: 'Llama' },
      { field: 'Height', text: '45"' },
    ],
  },
  {
    mark: '3n8vr4tqch',
    sectionIndex: 0,
    heading: '',
    stencilLink: 'Animal',
    stencilPath: 'Animal.md',
    preamble: '',
    values: [],
  },
]

/** The one section the deck stands in, which every card of it is under. */
const SECTIONS = [{ name: 'Roots', preamble: '' }]

/** A window with one deck open, drawn. */
const mountDeck = async (
  problems: readonly DeckProblem[] = [],
  schedule: {
    /** The presets the vault holds. */
    presets?: readonly PresetChoice[]
    /** The preset the deck names, and nothing for a deck naming none. */
    by?: string
    /** What is said against what the deck names. */
    saying?: string
    /** What putting the deck on a preset is refused for. */
    notScheduled?: ErrorCode
  } = {},
) => {
  const core: Cards = {
    stencils: async () => ({
      stencils: [{ path: 'Animal.md', title: 'Animal', fields: ['Name', 'Height'] }],
      held: 1,
    }),
    createDeck: async (title) => ({ ok: true, value: { path: `${title}.md` } }),
    createStencil: async (title) => ({ ok: true, value: { path: `${title}.md` } }),
    renameField: async () => ({
      decks: [],
      cards: 0,
      notWritten: [],
      error: null,
      changed: false,
      at: '',
    }),
    readDeck: async (path) =>
      asValue({
        deck: {
          path,
          title: 'Animals',
          preamble: '',
          cards: CARDS,
          sections: SECTIONS,
          tail: '',
          problems,
        },
        at: 'read',
      }),
    writeDeck: async () => asValue({ at: 'written' }),
    readStencil: async () => asFailure('missing' as const),
    writeStencil: async () => asValue({ at: '' }),
  }

  /** Which preset the deck names, as the vault answers it. */
  let by = schedule.by ?? ''
  /** Every deck put on a preset, as the tab asked for it. */
  const put: string[] = []

  const presets: Presets = {
    read: async (path) =>
      asValue({
        preset: { path, title: '', settings: DEFAULTS, problems: [], ...SCHEDULING },
        at: '',
        bounds: NO_BOUNDS,
      }),
    list: async () =>
      schedule.presets ?? [
        { path: 'Sanskrit.md', title: 'Sanskrit' },
        { path: 'presets/Slow.md', title: '' },
      ],
    createPreset: async () => asValue({ path: '' }),
    getDeckPreset: async () =>
      asValue({
        preset: {
          path: by,
          title: by === 'Sanskrit.md' ? 'Sanskrit' : '',
          settings: DEFAULTS,
          problems: schedule.saying ? [schedule.saying] : [],
          ...SCHEDULING,
        },
        at: '',
        bounds: NO_BOUNDS,
      }),
    scheduleDeck: async (_deck, preset) => {
      put.push(preset)
      if (schedule.notScheduled) return asFailure(schedule.notScheduled)
      by = preset
      return asValue({ at: 'scheduled' })
    },
    write: async () => asValue({ at: '' }),
    curve: async () => ({
      goal: 'minutes',
      grid: [],
      days: [],
      at: [],
      now: NOWHERE,
      suggested: NOWHERE,
      decks: 0,
      cards: 0,
      overdue: 0,
      unbegun: 0,
      honest: true,
    }),
  }

  const held = useWindowTabs()
  const decks = useDeckTabs(core, presets, held.handle, tabOpeners())
  held.registerKinds([decks.kind])
  const id = await held.openTabOfKind(DECK, 'Animals.md')
  await settle()
  const tab = held.handle.getTabState<DeckTabState>(DECK, id) as DeckTabState
  // A mark is teleported into the tile it is about, so the grid has to stand in
  // the document for the tile to be found.
  const window = mount(DeckTab, { props: { state: tab }, attachTo: document.body })
  await settle()
  return { window, tab, decks, put }
}

/** The tile one card is drawn as, by the identity the window gave that card. */
const tileOf = (window: Awaited<ReturnType<typeof mountDeck>>['window'], card: string) =>
  window.find(`[data-card="${card}"]`)

const stencilless: DeckProblem = {
  fault: 'cardWithoutAStencil',
  card: 1,
  face: null,
  field: '',
  text: 'a card under no stencil',
}

describe('a deck drawn', () => {
  it('draws a tile for every card the vault read', async () => {
    const { window } = await mountDeck()

    expect(window.findAll('[data-card]')).toHaveLength(2)
  })

  it('offers every stencil the vault holds once the plus is pressed', async () => {
    const { window } = await mountDeck()

    expect(window.find('[data-cut="Animal"]').exists()).toBe(false)
    await window.find('[data-plus]').find('button').trigger('click')

    expect(window.find('[data-cut="Animal"]').exists()).toBe(true)
  })

  it('writes a card cut by the stencil that was chosen', async () => {
    const { window, tab } = await mountDeck()

    // The plus of the last section, so the card it makes stands last of all.
    await window.findAll('[data-plus]').at(-1)!.find('button').trigger('click')
    await window.find('[data-cut="Animal"]').trigger('click')

    expect(tab.deck.value.cards.length).toBe(3)
    expect(tab.deck.value.cards.at(-1)?.stencilLink).toBe('Animal')
    expect(tab.deck.value.cards.at(-1)?.values).toStrictEqual([
      { field: 'Name', text: '' },
      { field: 'Height', text: '' },
    ])
  })

  it('draws a heading for every section the vault read, with the cards under it', async () => {
    const { window, tab } = await mountDeck()
    const roots = tab.sections.value[0]?.id ?? ''

    expect(window.get(`[data-section-head="${roots}"]`).get('input').element.value).toBe('Roots')
    expect(window.findAll(`[data-section="${roots}"]`)).toHaveLength(2)
  })

  it('makes a section at the end of the deck', async () => {
    const { window, tab } = await mountDeck()

    await window.get('[data-add-section]').trigger('click')

    expect(tab.sections.value.map((section) => section.name)).toStrictEqual(['Roots', 'Section 1'])
  })
})

describe('a mark on a tile', () => {
  it('is drawn inside the tile of the card it was read against', async () => {
    const { window, tab } = await mountDeck([stencilless])
    const second = tab.deck.value.cards[1]?.id ?? ''

    expect(tileOf(window, second).find('[data-wrong]').text()).toBe('a card under no stencil')
  })

  it('is drawn on no other tile', async () => {
    const { window, tab } = await mountDeck([stencilless])
    const first = tab.deck.value.cards[0]?.id ?? ''

    expect(tileOf(window, first).find('[data-wrong]').exists()).toBe(false)
  })

  it('is nowhere at all where the vault reported nothing wrong', async () => {
    const { window } = await mountDeck()

    expect(window.findAll('[data-wrong]')).toHaveLength(0)
  })

  it('goes when the card it stood on is taken out of the deck', async () => {
    const { window, tab } = await mountDeck([stencilless])
    const second = tab.deck.value.cards[1]?.id ?? ''

    tab.removeCard(second)
    await settle()

    expect(window.findAll('[data-wrong]')).toHaveLength(0)
  })

  it('stands with the card and not with its place, so removing another leaves it', async () => {
    const { window, tab } = await mountDeck([stencilless])
    const first = tab.deck.value.cards[0]?.id ?? ''
    const second = tab.deck.value.cards[1]?.id ?? ''

    tab.removeCard(first)
    await settle()

    expect(tileOf(window, second).find('[data-wrong]').text()).toBe('a card under no stencil')
  })
})

describe('what is wrong with the file itself', () => {
  it('is drawn above the grid, standing on no tile', async () => {
    const { window } = await mountDeck([
      { fault: 'unknown', card: null, face: null, field: '', text: 'the whole file' },
    ])

    expect(window.find(`[aria-label="${words.problems}"]`).text()).toBe('the whole file')
    expect(window.findAll('[data-wrong]')).toHaveLength(0)
  })

  it('is drawn nowhere where every problem stands on a card', async () => {
    const { window } = await mountDeck([stencilless])

    expect(window.find(`[aria-label="${words.problems}"]`).exists()).toBe(false)
  })
})

describe('a gesture in the grid', () => {
  it('writes into the card the box belongs to', async () => {
    const { window, tab } = await mountDeck()
    const first = tab.deck.value.cards[0]?.id ?? ''
    const box = tileOf(window, first).find('[data-value="Name"]')

    await box.setValue('Vicuña')

    expect(tab.deck.value.cards[0]?.values[0]).toStrictEqual({ field: 'Name', text: 'Vicuña' })
  })

  it('leaves every other card as it was', async () => {
    const { window, tab } = await mountDeck()
    const first = tab.deck.value.cards[0]?.id ?? ''
    const box = tileOf(window, first).find('[data-value="Name"]')

    await box.setValue('Vicuña')

    expect(tab.deck.value.cards[1]?.values).toStrictEqual([])
  })

  it('renames the section the box belongs to', async () => {
    const { window, tab } = await mountDeck()
    const roots = tab.sections.value[0]?.id ?? ''
    const box = window.get(`[data-section-head="${roots}"]`).get('input')

    await box.setValue('Roots and shoots')
    await box.trigger('change')

    expect(tab.sections.value[0]?.name).toBe('Roots and shoots')
  })

  it('takes a section away, and leaves the cards that stood under it', async () => {
    const { window, tab } = await mountDeck()
    const roots = tab.sections.value[0]?.id ?? ''

    await window.get(`[data-section-head="${roots}"]`).get('.remove-button').trigger('click')

    expect(tab.sections.value).toStrictEqual([])
    expect(window.findAll('[data-card]')).toHaveLength(2)
  })
})

describe('the question a file that changed on disk puts', () => {
  it('is not drawn while the deck is as the file has it', async () => {
    const { window } = await mountDeck()

    expect(window.text()).not.toContain(words.stale)
  })
})

describe('the preset a deck is scheduled by', () => {
  it('stands on a line at the top of the deck, saying which one is in force', async () => {
    const { window } = await mountDeck([], { by: 'Sanskrit.md' })

    const line = window.get('.deck-tab__scheduled')
    expect(line.text()).toContain(words.scheduledBy)
    expect(line.get('.deck-tab__choice').text()).toContain('Sanskrit')
  })

  it('says the defaults for a deck naming no preset', async () => {
    const { window } = await mountDeck()

    expect(window.get('.deck-tab__choice').text()).toContain(words.defaults)
  })

  it('offers the defaults and every preset the vault holds', async () => {
    const { window } = await mountDeck([], { by: 'Sanskrit.md' })
    const line = window.get('.deck-tab__choice')
    expect(line.attributes('aria-haspopup')).toBe('menu')
    expect(document.body.querySelectorAll('.menu__item')).toHaveLength(0)

    await line.trigger('click')

    const offered = [...document.body.querySelectorAll('.menu__item')].map(
      (one) => one.textContent?.trim() ?? '',
    )
    expect(offered).toStrictEqual([words.defaults, 'Sanskrit', 'Slow.md'])
  })

  it('marks the preset in force among the ones offered', async () => {
    const { window } = await mountDeck([], { by: 'Sanskrit.md' })
    await window.get('.deck-tab__choice').trigger('click')

    const checked = [...document.body.querySelectorAll('.menu__item')].filter(
      (one) => one.getAttribute('aria-checked') === 'true',
    )
    expect(checked.map((one) => one.textContent?.trim())).toStrictEqual(['Sanskrit'])
  })

  it('puts the deck on the preset that was chosen, and says it afterwards', async () => {
    const { window, put } = await mountDeck()
    await window.get('.deck-tab__choice').trigger('click')
    const chosen = [...document.body.querySelectorAll<HTMLElement>('.menu__item')].find(
      (one) => one.textContent?.trim() === 'Sanskrit',
    )

    chosen?.click()
    await settle()
    await window.vm.$nextTick()

    expect(put).toStrictEqual(['Sanskrit.md'])
    expect(window.get('.deck-tab__choice').text()).toContain('Sanskrit')
  })

  it('takes the deck back to the defaults', async () => {
    const { window, put } = await mountDeck([], { by: 'Sanskrit.md' })
    await window.get('.deck-tab__choice').trigger('click')
    const chosen = [...document.body.querySelectorAll<HTMLElement>('.menu__item')].find(
      (one) => one.textContent?.trim() === words.defaults,
    )

    chosen?.click()
    await settle()
    await window.vm.$nextTick()

    expect(put).toStrictEqual([''])
    expect(window.get('.deck-tab__choice').text()).toContain(words.defaults)
  })

  it('says on the line what the deck names and the vault does not hold', async () => {
    const { window } = await mountDeck([], {
      saying: 'Sanskrit reaches no note, and the defaults stand',
    })

    expect(window.get('.deck-tab__scheduled').text()).toContain('reaches no note')
    expect(window.get('.deck-tab__choice').text()).toContain(words.defaults)
  })

  it('says on the line that a choice was not written', async () => {
    const { window } = await mountDeck([], { notScheduled: 'notAPreset' })
    await window.get('.deck-tab__choice').trigger('click')
    const chosen = [...document.body.querySelectorAll<HTMLElement>('.menu__item')].find(
      (one) => one.textContent?.trim() === 'Sanskrit',
    )

    chosen?.click()
    await settle()
    await window.vm.$nextTick()

    expect(window.get('.deck-tab__scheduled').text()).toContain(words.notScheduled)
  })
})
