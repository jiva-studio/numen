/**
 * Where a menu is drawn as its size becomes known, and which area it stays in.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useMenuPlacement } from './place'
import type { Size } from '@/shared/lib/math/geometry'

const ROOM: Size = { width: 1000, height: 800 }

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('where a menu is placed', () => {
  it('remains at the requested point while its size is unknown', () => {
    const { placed } = useMenuPlacement({
      at: () => ({ x: 100, y: 120 }),
      viewport: () => ROOM,
      margin: () => 8,
    })
    expect(placed.value).toStrictEqual({ x: 100, y: 120 })
  })

  it('folds back over the point once its size is set', () => {
    const { placed, setSize } = useMenuPlacement({
      at: () => ({ x: 950, y: 100 }),
      viewport: () => ROOM,
      margin: () => 8,
    })
    expect(placed.value.x).toBe(950)

    setSize({ width: 200, height: 100 })
    expect(placed.value).toStrictEqual({ x: 750, y: 100 })
  })

  it('stays inside the browser window where the caller measures no area', () => {
    vi.stubGlobal('innerWidth', 500)
    vi.stubGlobal('innerHeight', 400)
    const { placed, setSize, resetSize, menuStyle } = useMenuPlacement({
      at: () => ({ x: 450, y: 350 }),
      viewport: () => null,
      margin: () => 0,
    })
    expect(menuStyle.value['--asking']).toBe('0px')
    expect(menuStyle.value['--opened-width']).toBe('0px')
    setSize({ width: 100, height: 100 })
    expect(placed.value).toStrictEqual({ x: 350, y: 250 })
    expect(menuStyle.value['--opened-width']).toBe('100px')
    setSize({ width: 60, height: 100 })
    expect(menuStyle.value['--opened-width']).toBe('100px')
    resetSize()
    expect(menuStyle.value['--opened-width']).toBe('0px')

    const withAsking = useMenuPlacement({
      at: () => ({ x: 10, y: 20 }),
      viewport: () => ROOM,
      margin: () => 8,
      asking: () => 180,
    })
    expect(withAsking.menuStyle.value['--asking']).toBe('180px')
  })

  it('is brought in off an edge it was requested for right against', () => {
    const { placed, menuStyle } = useMenuPlacement({
      at: () => ({ x: 0, y: 0 }),
      viewport: () => ROOM,
      margin: () => 12,
    })
    expect(placed.value).toStrictEqual({ x: 12, y: 12 })
    expect(menuStyle.value['left']).toBe('12px')
    expect(menuStyle.value['top']).toBe('12px')
  })
})

describe('how tall a menu grows', () => {
  const getRoom = (viewport: Size | null, margin: number) =>
    useMenuPlacement({ at: () => ({ x: 0, y: 0 }), viewport: () => viewport, margin: () => margin })
      .menuStyle.value['--room']

  it('leaves the area its margin on both sides', () => {
    expect(getRoom({ width: 900, height: 300 }, 10)).toBe('280px')
  })

  it('reads the room off the browser window where no area is given', () => {
    expect(getRoom(null, 8)).toBe('calc(100dvh - 16px)')
  })

  it('never leaves a negative room', () => {
    expect(getRoom({ width: 900, height: 10 }, 8)).toBe('0px')
  })
})
