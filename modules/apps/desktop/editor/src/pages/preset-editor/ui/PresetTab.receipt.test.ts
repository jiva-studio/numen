/**
 * The receipt in figures under the picture: a row of what the preset is set
 * to, and when the material is learned.
 *
 * The figures are read off the very run the line is drawn from. Nothing on a
 * row says who put its value there: a preset is a file, and a file carries
 * values and not a history of them.
 */
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { mountPresetTab, point } from '../fixtures'

describe('a row of the receipt', () => {
  it('carries no mark of its own and offers nothing back to the goal', () => {
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      const { tab } = mountPresetTab({ goal }, { goal })
      const rows = tab.findAll('[data-preset-row]')
      expect(rows.length).toBeGreaterThan(0)
      // A mark on the row that shows who put the value there would be one row
      // drawn unlike the others; every row is drawn alike.
      expect(new Set(rows.map((one) => one.attributes('class'))).size).toBe(1)
      // A way back under the goal stood in what the row says. What it says is
      // its name and what it means, and nothing beside them.
      for (const row of rows) {
        expect(row.get('[data-preset="name"]').element.parentElement?.children).toHaveLength(2)
      }
    }
  })
})

describe('when the material is learned', () => {
  const getLearnedTiles = (tab: ReturnType<typeof mount>) =>
    tab
      .findAll('[data-control="learned"] [data-control="tile"]')
      .map((one) => [one.get('[data-control="figure"]').text(), one.get('[data-control="word"]').text()])

  it('says the days it takes and how much of it stands learned today', () => {
    const { tab } = mountPresetTab({
      cards: 79,
      at: [point(), point(), point({ learns: 41, learned: 0 }), point()],
    })
    expect(getLearnedTiles(tab)).toStrictEqual([
      ['41', 'days to learn it'],
      ['0 of 79', 'learned today'],
    ])
  })

  it('follows the knob, since each place of the grid learns at its own pace', async () => {
    const { tab } = mountPresetTab({
      cards: 79,
      at: [
        point({ learns: 70, learned: 0 }),
        point({ learns: 41, learned: 0 }),
        point({ learns: 3, learned: 77 }),
        point({ learns: 0, learned: 79 }),
      ],
    })
    expect(getLearnedTiles(tab)[0]).toStrictEqual(['3', 'days to learn it'])
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'Home' })
    expect(getLearnedTiles(tab)[0]).toStrictEqual(['70', 'days to learn it'])
  })

  // A pace that does not get there has no day to name, so it says so.
  it('says a pace that never gets there in words, and not as a figure', () => {
    const { tab } = mountPresetTab({
      cards: 79,
      at: [point(), point(), point({ learns: -1, learned: 4 }), point()],
    })
    expect(getLearnedTiles(tab)).toStrictEqual([
      ['not yet', 'in the days ahead'],
      ['4 of 79', 'learned today'],
    ])
  })

  it('says a material already learned is learned today', () => {
    const { tab } = mountPresetTab({
      cards: 79,
      at: [point(), point(), point({ learns: 0, learned: 79 }), point()],
    })
    expect(getLearnedTiles(tab)[0]).toStrictEqual(['today', 'all of it learned'])
  })

  it('says nothing at all until the answer lands', () => {
    const { tab } = mountPresetTab({ honest: false })
    expect(tab.findAll('[data-control="learned"] [data-control="tile"]')).toHaveLength(0)
  })

  // There is no day the whole of it stands learned on under every rule, and
  // where the application carries none the window says nothing in its place.
  it('draws no tile for a day the answer does not carry', () => {
    const { tab } = mountPresetTab({
      cards: 79,
      at: [point(), point(), point({ learned: 77 }), point()],
    })
    expect(getLearnedTiles(tab)).toStrictEqual([['77 of 79', 'learned today']])
    expect(tab.text()).not.toContain('not yet')
    expect(tab.text()).not.toContain('days to learn it')
  })

  // Under a goal of a date the day is the answer and what qualifies it is said
  // in the bubble, so the tiles say what stands learned now and nothing else.
  it('says what stands learned today under a goal of a date, and no day', () => {
    const { tab } = mountPresetTab(
      {
        goal: 'date',
        grid: [14, 30, 90, 180],
        days: ['2026-09-14', '2026-09-30', '2026-11-29', '2027-02-27'],
        cards: 79,
        at: [point(), point(), point({ learned: 41, short: 11 }), point()],
      },
      { goal: 'date', byDate: '2026-11-29' },
    )
    expect(getLearnedTiles(tab)).toStrictEqual([['41 of 79', 'learned today']])
  })
})
