/** The plex at another size: what a size carries, and what it leaves alone. */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope } from 'vue'
import { DESIGNED_TYPE, optionsForType, scaleOptions, useTypeSize } from './plexScale'
import { arrangePlex, DEFAULT_OPTIONS, type PlexOptions } from '../../lib/arrange'
import { build } from '../../fixtures/build'

const HALF_AGAIN = 1.5

describe('the size the plex was designed at', () => {
  it('is the size a label is set at with nothing multiplying it', () => {
    expect(optionsForType(DESIGNED_TYPE)).toStrictEqual(DEFAULT_OPTIONS)
  })

  it('is what a plex nobody measured is drawn at', () => {
    expect(optionsForType(0)).toStrictEqual(DEFAULT_OPTIONS)
  })

  it('places every node where it stood', () => {
    const crowd = build('A node', { parent: 3, child: 12, jump: 4, sibling: 4 })
    const viewport = { width: 1200, height: 800 }
    const was = arrangePlex(crowd, { options: { viewport } })
    const now = arrangePlex(crowd, {
      options: { ...optionsForType(DESIGNED_TYPE), viewport },
    })
    expect(now).toStrictEqual(was)
  })
})

describe('a label half again as large', () => {
  let larger: PlexOptions

  beforeEach(() => {
    larger = optionsForType(DESIGNED_TYPE * HALF_AGAIN)
  })

  it('is retained by a box half again as large', () => {
    expect(larger.nodeSize).toStrictEqual({ width: 216, height: 54 })
    expect(larger.focusSize).toStrictEqual({ width: 264, height: 66 })
  })

  it('is cleared by every length between the boxes', () => {
    expect(larger.gap).toBe(24)
    expect(larger.lineGap).toBe(30)
    expect(larger.focusGap).toBe(84)
    expect(larger.margin).toBe(24)
    expect(larger.routing.minReach).toBe(33)
  })

  it('is retained by every length inside a box, and alongside the line it hangs on', () => {
    expect(larger.minWidth).toBe(108)
    expect(larger.iconWidth).toBe(24)
    expect(larger.routing.arrowClearance).toBe(30)
  })

  // Every length is multiplied, so one added and left out of the multiplying
  // would draw at the size the plex was designed at while the rest grew.
  it('leaves no length behind', () => {
    const lengths = (options: PlexOptions): number[] =>
      Object.entries(options)
        .flatMap(([name, value]) =>
          name === 'routing' ? Object.entries(value as object) : [[name, value] as const],
        )
        .filter(([name, value]) => typeof value === 'number' && name !== 'curvature')
        .filter(
          ([name]) =>
            !['maxPerLine', 'maxLines', 'maxHeadings', 'spread', 'squeeze'].includes(name),
        )
        .map(([, value]) => value as number)
        .concat([options.nodeSize, options.focusSize].flatMap((size) => [size.width, size.height]))

    const was = lengths(DEFAULT_OPTIONS)
    const now = lengths(larger)
    expect(now).toHaveLength(was.length)
    for (let i = 0; i < was.length; i++) expect(now[i]).toBeCloseTo(was[i]! * HALF_AGAIN)
  })

  it('leaves the counts, the fractions and the directions where they were', () => {
    expect(larger.maxPerLine).toBe(DEFAULT_OPTIONS.maxPerLine)
    expect(larger.maxLines).toBe(DEFAULT_OPTIONS.maxLines)
    expect(larger.maxHeadings).toBe(DEFAULT_OPTIONS.maxHeadings)
    expect(larger.routing.curvature).toBe(DEFAULT_OPTIONS.routing.curvature)
    expect(larger.spread).toBe(DEFAULT_OPTIONS.spread)
    expect(larger.squeeze).toBe(DEFAULT_OPTIONS.squeeze)
    expect(larger.motion).toStrictEqual(DEFAULT_OPTIONS.motion)
    expect(larger.gesture).toStrictEqual(DEFAULT_OPTIONS.gesture)
    expect(larger.direction).toStrictEqual(DEFAULT_OPTIONS.direction)
  })

  it('carries boxes the caller chose, and not only the ones here', () => {
    const request = { ...DEFAULT_OPTIONS, nodeSize: { width: 200, height: 40 } }
    expect(optionsForType(DESIGNED_TYPE * HALF_AGAIN, request).nodeSize).toStrictEqual({
      width: 300,
      height: 60,
    })
  })

  it('is drawn in the window the screen really has', () => {
    const viewport = { width: 1200, height: 800 }
    expect(scaleOptions({ ...DEFAULT_OPTIONS, viewport }, HALF_AGAIN).viewport).toStrictEqual(
      viewport,
    )
  })

  it('wraps a row sooner, so nothing reaches past the edge', () => {
    const viewport = { width: 900, height: 700 }
    const crowd = build('A node', { parent: 3, child: 12, jump: 4, sibling: 4 })
    const frame = arrangePlex(crowd, { options: { ...larger, viewport } })
    const escapes = frame.nodes.filter(
      (node) =>
        Math.abs(node.x) + node.width / 2 > viewport.width / 2 - larger.margin ||
        Math.abs(node.y) + node.height / 2 > viewport.height / 2 - larger.margin,
    )
    expect(escapes.map((node) => node.title)).toStrictEqual([])
  })
})

/** A window that reports a size for the boxes it observes. */
function stubObserver() {
  const watched: Element[] = []
  const state = { ring: (_entries: unknown[]): void => undefined, disconnected: 0 }

  vi.stubGlobal(
    'ResizeObserver',
    class {
      constructor(ring: (entries: unknown[]) => void) {
        state.ring = ring
      }
      observe(element: Element) {
        watched.push(element)
      }
      disconnect() {
        state.disconnected++
      }
    },
  )

  return { watched, state }
}

const tell = (state: { ring: (entries: unknown[]) => void }, width: number) =>
  state.ring([{ contentRect: { width } }])

describe('the size a label is being set at', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    document.body.replaceChildren()
  })

  it('is the size the plex was designed at where nothing watches a box', () => {
    const scope = effectScope()
    const type = scope.run(() => useTypeSize())
    expect(type?.value).toBe(DESIGNED_TYPE)
    expect(document.body.children).toHaveLength(0)
    scope.stop()
  })

  it('is the size the plex was designed at where there is no document', () => {
    const { watched } = stubObserver()
    vi.stubGlobal('document', undefined)
    const scope = effectScope()
    const type = scope.run(() => useTypeSize())
    expect(type?.value).toBe(DESIGNED_TYPE)
    expect(watched).toHaveLength(0)
    scope.stop()
  })

  it('is measured off a hidden box of the window’s own type', () => {
    const { watched } = stubObserver()
    const scope = effectScope()
    scope.run(() => useTypeSize())
    const probe = document.body.firstElementChild as HTMLElement
    expect(probe.tagName).toBe('DIV')
    expect(probe.getAttribute('aria-hidden')).toBe('true')
    expect(probe.style.fontSize).toBe('var(--numen-font-size)')
    expect(watched).toStrictEqual([probe])
    scope.stop()
  })

  it('follows the width the box takes', () => {
    const { state } = stubObserver()
    const scope = effectScope()
    const type = scope.run(() => useTypeSize())!
    tell(state, 19.5)
    expect(type.value).toBe(19.5)
    scope.stop()
  })

  it('keeps its size when the box has no width yet', () => {
    const { state } = stubObserver()
    const scope = effectScope()
    const type = scope.run(() => useTypeSize())!
    tell(state, 0)
    expect(type.value).toBe(DESIGNED_TYPE)
    tell(state, -3)
    expect(type.value).toBe(DESIGNED_TYPE)
    scope.stop()
  })

  it('keeps its size when the observer reports no box at all', () => {
    const { state } = stubObserver()
    const scope = effectScope()
    const type = scope.run(() => useTypeSize())!
    state.ring([])
    expect(type.value).toBe(DESIGNED_TYPE)
    scope.stop()
  })

  it('stops watching and leaves nothing in the page when its scope ends', () => {
    const { state } = stubObserver()
    const scope = effectScope()
    scope.run(() => useTypeSize())
    expect(document.body.children).toHaveLength(1)
    scope.stop()
    expect(state.disconnected).toBe(1)
    expect(document.body.children).toHaveLength(0)
  })

  it('is read without a warning where no scope holds it', () => {
    stubObserver()
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined)
    useTypeSize()
    expect(warn).not.toHaveBeenCalled()
    warn.mockRestore()
  })
})
