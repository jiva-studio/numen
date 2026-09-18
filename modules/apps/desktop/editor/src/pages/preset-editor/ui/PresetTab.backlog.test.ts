/**
 * The plot of what stands overdue, drawn under the picture over the days ahead.
 *
 * It is read off the place the knob stands at and never dragged, so it is no
 * stop on the way round the screen. Nothing overdue is the floor it is read up
 * from, and the room it is drawn in is the same box in every state it has.
 */
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import type { Curve } from '../types'
import { mountPresetTab, heights, point } from '../fixtures'
import { WORDS as words } from '../words'

// Past the day a person names, the preset schedules nothing: no card is
// answered and the pile stacks up. Every series stops at that day.
describe('what a goal of a date draws', () => {
  const climbing = [0, 0, 0, 4, 9, 16, 27, 42, 57, 69, 80, 91, 100]

  const createDatedTab = () =>
    mountPresetTab(
      {
        goal: 'date',
        grid: [4, 8, 13],
        days: ['2026-09-04', '2026-09-08', '2026-09-13'],
        cards: 79,
        now: { at: 0, value: 4, day: '2026-09-04' },
        at: [
          point({ backlog: climbing }),
          point({ backlog: climbing }),
          point({ backlog: climbing }),
        ],
      },
      { goal: 'date', byDate: '2026-09-04' },
    )

  /** How many days the backlog is drawn over, which is how many places it has. */
  const days = (tab: ReturnType<typeof mount>) =>
    (tab.get('[data-backlog="line"]').attributes('d') ?? '').split(/[ML]/).length - 1

  it('stops the backlog at the day the place stands for', async () => {
    const { tab } = createDatedTab()
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'Home' })
    expect(days(tab)).toBe(4)

    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'End' })
    expect(days(tab)).toBe(13)
  })

  // A shorter run must not make the picture jump, so every place is drawn
  // against the one extent: the most any of them ever stands at.
  it('keeps one height for the backlog, whatever place the knob stands on', async () => {
    const { tab } = createDatedTab()
    const control = tab.get('[data-control="picture"][role="slider"]')
    await control.trigger('keydown', { key: 'End' })
    const tallest = heights(tab.get('[data-backlog="line"]').attributes('d') ?? '')
    await control.trigger('keydown', { key: 'Home' })
    const shortest = heights(tab.get('[data-backlog="line"]').attributes('d') ?? '')

    // The first days are the same days, and they are drawn at the same height.
    expect(shortest.slice(0, 3)).toStrictEqual(tallest.slice(0, 3))
  })

  it('draws every day of the run under the goals that name no day', () => {
    const { tab } = mountPresetTab({
      at: [point(), point(), point({ backlog: climbing }), point()],
    })
    expect(days(tab)).toBe(climbing.length)
  })
})

describe('the plot of what stands overdue', () => {
  /** A curve whose place the knob stands at carries a backlog that climbs. */
  const createClimbingTab = (backlog: readonly number[] = [16, 21, 55, 66, 65, 78]) =>
    mountPresetTab({
      at: [point(), point(), point({ reviews: 80, backlog }), point()],
    })

  it('is a plot of its own under the picture, drawn over the days ahead', () => {
    const { tab } = createClimbingTab()
    expect(tab.findAll('[data-backlog="line"]')).toHaveLength(1)
    expect(tab.get('[data-backlog="line"]').attributes('d')?.startsWith('M')).toBe(true)
  })

  // A day too short to carry what falls due adds to the pile, and the climb is
  // what the picture is for. The extent is scaled to this one place's own run.
  it('draws a backlog that climbs as climbing, and not flat', () => {
    const heights = (d: string) =>
      d
        .split(/[ML]/)
        .slice(1)
        .map((one) => Number(one.split(' ')[1]))
    const drawnAt = heights(
      createClimbingTab().tab.get('[data-backlog="line"]').attributes('d') ?? '',
    )
    expect(drawnAt[0]).toBeGreaterThan(drawnAt[5] ?? 0)
    expect(drawnAt[2]).toBeLessThan(drawnAt[1] ?? 0)
    expect(new Set(drawnAt).size).toBeGreaterThan(4)
  })

  // Nothing overdue is the foot, so the height is read against a floor that
  // means something.
  it('reads from nothing overdue to the most this pace ever stands at', () => {
    const said = createClimbingTab()
      .tab.findAll('[data-control="number"]')
      .map((one) => one.text())
    expect(said).toContain(words.backlogHeightAt(78))
    // A number the line stands on gives way to it, so the foot is read off a
    // run that leaves the left edge of the backlog clear.
    const falling = createClimbingTab([78, 60, 40, 20, 5, 0])
      .tab.findAll('[data-control="number"]')
      .map((one) => one.text())
    expect(falling).toContain(words.backlogHeightAt(0))
  })

  it('names both its axes, in the same voice as the picture over it', () => {
    const { tab } = createClimbingTab()
    const names = tab.findAll('[data-control="name"][data-axis="y"]').map((one) => one.text())
    expect(names).toStrictEqual([words.axisY('minutes'), words.backlogY])
    expect(tab.findAll('[data-control="name"][data-axis="x"]').map((one) => one.text())).toContain(
      words.backlogX,
    )
  })

  it('carries the days at either end, which the goal’s grid says nothing about', () => {
    const { tab } = createClimbingTab()
    const ends = tab.findAll('[data-control="ends"]').map((one) => one.text())
    expect(ends[1]).toContain(words.backlogWidthAt(6))
  })

  // The backlog is read and never dragged, so it is no stop on the way round the
  // screen and offers nothing to the keyboard.
  it('is read and not dragged, so the one control stays the one control', () => {
    const { tab } = createClimbingTab()
    expect(tab.findAll('[data-control="picture"][role="slider"]')).toHaveLength(1)
    expect(tab.findAll('[data-control="picture"], [data-backlog="picture"]')).toHaveLength(2)
  })

  // The page keeps its height whether or not there is a backlog to draw.
  it('keeps its room where the place the knob stands carries no backlog', () => {
    const { tab } = mountPresetTab()
    expect(tab.findAll('[data-backlog="line"]')).toHaveLength(0)
    expect(tab.findAll('[data-control="room"]')).toHaveLength(2)
    expect(tab.findAll('[data-control="name"][data-axis="y"]').map((one) => one.text())).toContain(
      words.backlogY,
    )
  })

  // The room a plot is drawn in is the same box in every state it has, so
  // nothing under the picture moves when the answer lands.
  it('holds one room for the plot, waiting, drawn and empty areNeighbourhoodsEqual', () => {
    const room = (over: Partial<Curve> = {}) =>
      mountPresetTab(over)
        .tab.findAll('[data-control="room"]')
        .map((one) => one.attributes('style'))
    expect(room({ isHonest: false })).toStrictEqual(room())
    expect(room({ at: [point(), point(), point(), point()] })).toStrictEqual(room())
  })

  // Nothing overdue is the floor the backlog is read up from, and a run holding
  // nothing at all lies along it.
  it('lays a run of nothing overdue along the floor, not through the middle', () => {
    const heights = (d: string) =>
      d
        .split(/[ML]/)
        .slice(1)
        .map((one) => Number(one.split(' ')[1]))
    const flat = heights(
      createClimbingTab([0, 0, 0, 0, 0, 0]).tab.get('[data-backlog="line"]').attributes('d') ?? '',
    )
    const floor = heights(
      createClimbingTab([12, 8, 4, 0, 0, 0]).tab.get('[data-backlog="line"]').attributes('d') ?? '',
    )
    // Every place of the empty run stands where the run that clears comes to
    // rest, which is the foot the axis is drawn along.
    expect(new Set(flat).size).toBe(1)
    expect(flat[0]).toBe(floor[5])
    expect(flat[0]).toBe(floor[4])
  })

  it('says nothing overdue once, on the floor, where the run holds nothing', () => {
    const said = createClimbingTab([0, 0, 0, 0, 0, 0])
      .tab.findAll('[data-control="number"]')
      .map((one) => one.text())
    expect(said).toContain(words.backlogHeightAt(0))
  })

  // The backlog is read off the place the knob stands at, so walking the grid
  // walks the backlog with it.
  it('follows the knob, since each place of the grid keeps its own', async () => {
    const { tab } = mountPresetTab({
      at: [
        point({ backlog: [1, 2, 3] }),
        point({ backlog: [9, 9, 9] }),
        point({ backlog: [4, 3, 2] }),
        point({ backlog: [8, 4, 0] }),
      ],
    })
    const line = () => tab.get('[data-backlog="line"]').attributes('d')
    const was = line()
    await tab.get('[data-control="picture"][role="slider"]').trigger('keydown', { key: 'End' })
    expect(line()).not.toBe(was)
  })
})
