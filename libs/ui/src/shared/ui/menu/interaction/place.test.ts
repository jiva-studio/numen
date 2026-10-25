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
    const { placed, setSize } = useMenuPlacement({
      at: () => ({ x: 450, y: 350 }),
      viewport: () => null,
      margin: () => 0,
    })
    setSize({ width: 100, height: 100 })
    expect(placed.value).toStrictEqual({ x: 350, y: 250 })
  })
})
