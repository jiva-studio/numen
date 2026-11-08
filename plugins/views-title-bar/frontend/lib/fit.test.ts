import { describe, expect, it } from 'vitest'
import { fitBar, type FitEntry } from './fit'

const BUTTON = 32
const GAP = 8
const MORE = 32

const command = (key: string): FitEntry => ({ key, kind: 'command', width: BUTTON })
const submenu = (key: string): FitEntry => ({ key, kind: 'submenu', width: 40 })
const view = (key: string, width = 80): FitEntry => ({ key, kind: 'view', width })
const separator = (key: string): FitEntry => ({ key, kind: 'separator', width: GAP })

const COMMANDS = [command('a'), command('b'), separator('s1'), command('c'), command('d')]
const STATUS = [view('tasks'), view('stats', 60), command('e')]
const FULL = 4 * BUTTON + GAP + 80 + 60 + BUTTON

describe('fitBar', () => {
  it('draws every entry while the room is not measured', () => {
    expect(fitBar([COMMANDS, STATUS], null, MORE)).toStrictEqual({
      hidden: new Set(),
      overflow: [],
    })
  })

  it('draws every entry when they all fit', () => {
    expect(fitBar([COMMANDS, STATUS], FULL, MORE)).toStrictEqual({
      hidden: new Set(),
      overflow: [],
    })
  })

  it('moves the commands from the edge into the menu and keeps room for the menu button', () => {
    const fit = fitBar([COMMANDS, STATUS], FULL - 1, MORE)

    expect(fit.overflow).toStrictEqual(['d', 'e'])
    expect([...fit.hidden].sort()).toStrictEqual(['d', 'e'])
  })

  it('lists the menu in the order of the bar, the first slot before the second', () => {
    const fit = fitBar([COMMANDS, STATUS], 2 * BUTTON + 80 + 60 + MORE, MORE)

    expect(fit.overflow).toStrictEqual(['c', 'd', 'e'])
  })

  it('hides a separator that has no drawn entry on one side', () => {
    const fit = fitBar([COMMANDS, STATUS], 2 * BUTTON + 80 + 60 + MORE, MORE)

    expect(fit.hidden.has('s1')).toBe(true)
  })

  it('never moves a view into the menu, and hides views from the edge once every command is in it', () => {
    const fit = fitBar([COMMANDS, STATUS], 80 + MORE, MORE)

    expect(fit.overflow).toStrictEqual(['a', 'b', 'c', 'd', 'e'])
    expect(fit.hidden.has('stats')).toBe(true)
    expect(fit.hidden.has('tasks')).toBe(false)
  })

  it('hides every entry and moves every command into the menu when there is no room', () => {
    const fit = fitBar([COMMANDS, STATUS], 0, MORE)

    expect(fit.overflow).toStrictEqual(['a', 'b', 'c', 'd', 'e'])
    expect(fit.hidden).toStrictEqual(new Set(['a', 'b', 's1', 'c', 'd', 'tasks', 'stats', 'e']))
  })

  it('moves a submenu into the menu as it moves a command', () => {
    const fit = fitBar([[command('a'), submenu('new')]], 70, MORE)

    expect(fit.overflow).toStrictEqual(['new'])
  })

  it('hides views without a menu button when the slots hold no command', () => {
    const fit = fitBar([[], [view('tasks'), view('stats', 60)]], 100, MORE)

    expect(fit.overflow).toStrictEqual([])
    expect([...fit.hidden]).toStrictEqual(['stats'])
  })

  it('keeps a view drawn whose width is not measured yet', () => {
    const fit = fitBar([[view('tasks', 0), command('a')]], BUTTON, MORE)

    expect(fit).toStrictEqual({ hidden: new Set(), overflow: [] })
  })
})
