/**
 * Measuring a title. jsdom has no canvas of its own, so the one this stands in
 * for it counts characters — what is under test is what the measurer does with
 * a measurement, not what a font does with a letter.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { titleWidths } from './measure'
import type { PlexNode } from './model'

const node = (title: string): PlexNode => ({ id: title, title, seat: 'child' })

const PER_CHARACTER = 7

/** A canvas that measures by the character, and remembers what it was asked. */
function stubCanvas() {
  const asked: string[] = []
  const context = {
    font: '',
    measureText(text: string) {
      asked.push(text)
      return { width: text.length * PER_CHARACTER } as TextMetrics
    },
  }

  vi.stubGlobal('CanvasRenderingContext2D', function () {})
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(
    context as unknown as CanvasRenderingContext2D,
  )

  return { asked, context }
}

/** The tokens on the root, which is where a theme writes them. */
function themed(size = '13px', padding = '10px'): void {
  const root = document.documentElement
  root.style.setProperty('--numen-font-size', size)
  root.style.setProperty('--numen-font-sans', 'Test Sans, sans-serif')
  root.style.setProperty('--numen-node-padding', padding)
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  document.documentElement.removeAttribute('style')
})

describe('where there is nothing to measure with', () => {
  it('gives nothing where the platform has no canvas', () => {
    // jsdom, and a page rendered on a server. The arrangement is still defined:
    // handed no measurer, it draws every box at its widest.
    themed()
    expect(titleWidths()).toBeUndefined()
  })
})

describe('what a title needs', () => {
  it('adds the padding on both sides of the text', () => {
    stubCanvas()
    themed()
    const measure = titleWidths()!
    expect(measure(node('abcd'))).toBe(4 * PER_CHARACTER + 20)
  })

  it('takes the type and the padding from the root', () => {
    const { context } = stubCanvas()
    themed('17px', '4px')
    const measure = titleWidths()!
    expect(context.font).toBe('17px Test Sans, sans-serif')
    expect(measure(node('ab'))).toBe(2 * PER_CHARACTER + 8)
  })

  it('gives an empty title the padding alone', () => {
    stubCanvas()
    themed()
    const measure = titleWidths()!
    expect(measure(node(''))).toBe(20)
  })

  it('answers in whole pixels', () => {
    const { context } = stubCanvas()
    themed()
    context.measureText = (text: string) => ({ width: text.length * 6.4 }) as TextMetrics
    const measure = titleWidths()!
    expect(measure(node('abc'))).toBe(Math.ceil(3 * 6.4) + 20)
  })

  it('measures a string once, however many nodes carry it', () => {
    const { asked } = stubCanvas()
    themed()
    const measure = titleWidths()!

    measure({ id: 'a', title: 'Adapter', seat: 'child' })
    measure({ id: 'b', title: 'Adapter', seat: 'parent' })
    measure(node('Port'))

    expect(asked).toStrictEqual(['Adapter', 'Port'])
  })
})
