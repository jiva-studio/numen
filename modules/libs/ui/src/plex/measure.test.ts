/**
 * Measuring a title. jsdom has neither a canvas nor a style engine, so the two
 * below stand in for them: one counts characters, the other resolves a token
 * into the pixels a browser would give. What is under test is what the measurer
 * does with a measurement, not what a font does with a letter.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { effectScope } from 'vue'
import { useTitleWidths } from './measure'
import type { PlexNode } from './node'

const node = (title: string): PlexNode => ({ id: title, title, seat: 'child' })

/** The measurers read once, where nothing on the platform watches a box. */
const widths = (icon = 0) => useTitleWidths(() => icon).value

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

/** The tokens as a theme writes them, which is in whatever unit it likes. */
let theme: Record<string, string> = {}

function themed(written: Record<string, string> = {}): void {
  theme = {
    '--numen-font-size': '13px',
    '--numen-font-sans': 'Test Sans, sans-serif',
    '--numen-node-padding': '10px',
    '--numen-node-gap': '6px',
    '--numen-edge-label-size': '10px',
    ...written,
  }
}

/** A root of 16px, which is what a browser resolves a rem against. */
const inPixels = (written: string): string =>
  written.endsWith('rem') ? `${Number.parseFloat(written) * 16}px` : written

const resolved = (declaration: string): string => {
  const token = /^var\((--[a-z-]+)\)$/.exec(declaration)?.[1]
  if (!token) return declaration
  const written = theme[token]
  return written ? inPixels(written) : ''
}

/**
 * The document, resolving the declarations an element carries. Anything that
 * carries none of them is answered by jsdom as before.
 */
function stubStyles() {
  const engine = window.getComputedStyle.bind(window)

  vi.spyOn(window, 'getComputedStyle').mockImplementation(((
    element: Element,
    pseudo?: string | null,
  ) => {
    const style = (element as HTMLElement).style
    const declared = (property: string) => resolved(style?.getPropertyValue(property) ?? '')
    if (!declared('font-size')) return engine(element, pseudo)

    return {
      fontSize: declared('font-size'),
      fontFamily: declared('font-family'),
      paddingInlineStart: declared('padding-inline'),
      columnGap: declared('column-gap'),
    } as CSSStyleDeclaration
  }) as typeof window.getComputedStyle)
}

/** A window that says when a box of the plex's type has changed size. */
function stubObserver(): (() => void)[] {
  const observers: (() => void)[] = []

  vi.stubGlobal(
    'ResizeObserver',
    class {
      constructor(ring: () => void) {
        observers.push(ring)
      }
      observe() {}
      disconnect() {}
    },
  )

  return observers
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  document.body.replaceChildren()
})

describe('where there is nothing to measure with', () => {
  it('gives nothing where the platform has no canvas', () => {
    // jsdom, and a page rendered on a server. The arrangement is still defined:
    // handed no measurer, it draws every box at its widest.
    themed()
    stubStyles()
    expect(widths()).toBeUndefined()
  })

  it('leaves nothing of itself in the page', () => {
    themed()
    stubStyles()
    widths()
    expect(document.body.children).toHaveLength(0)
  })
})

describe('what a title needs', () => {
  it('adds the padding on both sides of the text', () => {
    stubCanvas()
    stubStyles()
    themed()
    const measures = widths()!
    expect(measures.node(node('abcd'))).toBe(4 * PER_CHARACTER + 20)
  })

  it('takes the type and the padding in pixels, however the theme wrote them', () => {
    // The tokens are read off a box the document has resolved, so a theme in
    // rem is measured as one in px is.
    const { context } = stubCanvas()
    stubStyles()
    themed({ '--numen-font-size': '1.0625rem', '--numen-node-padding': '0.25rem' })
    const measures = widths()!
    expect(measures.node(node('ab'))).toBe(2 * PER_CHARACTER + 8)
    expect(context.font).toBe('17px Test Sans, sans-serif')
  })

  it('gives an empty title the padding alone', () => {
    stubCanvas()
    stubStyles()
    themed()
    const measures = widths()!
    expect(measures.node(node(''))).toBe(20)
  })

  it('answers in whole pixels', () => {
    const { context } = stubCanvas()
    stubStyles()
    themed()
    context.measureText = (text: string) => ({ width: text.length * 6.4 }) as TextMetrics
    const measures = widths()!
    expect(measures.node(node('abc'))).toBe(Math.ceil(3 * 6.4) + 20)
  })

  it('measures a string once, however many nodes carry it', () => {
    const { asked } = stubCanvas()
    stubStyles()
    themed()
    const measures = widths()!

    measures.node({ id: 'a', title: 'Adapter', seat: 'child' })
    measures.node({ id: 'b', title: 'Adapter', seat: 'parent' })
    measures.node(node('Port'))

    // The sample word comes first: how deep a line stands is taken off the
    // type when the measurer is made.
    expect(asked).toStrictEqual(['Hxg', 'Adapter', 'Port'])
  })
})

describe('where a caller draws an icon', () => {
  it('keeps the room it asked for, and a gap between icon and title', () => {
    stubCanvas()
    stubStyles()
    themed()
    expect(widths(18)!.node(node('abcd'))).toBe(4 * PER_CHARACTER + 20 + 18 + 6)
  })

  it('keeps neither where no icon is drawn', () => {
    stubCanvas()
    stubStyles()
    themed()
    expect(widths(0)!.node(node('abcd'))).toBe(4 * PER_CHARACTER + 20)
  })
})

describe('what a label needs', () => {
  it('is the words alone: a label stands on a line and carries no padding', () => {
    stubCanvas()
    stubStyles()
    themed()
    expect(widths()!.label('names')).toBe(5 * PER_CHARACTER)
  })

  it('is measured in the size a label is set at, not a title', () => {
    const { context } = stubCanvas()
    stubStyles()
    themed()
    const measures = widths()!

    measures.node(node('Domain'))
    expect(context.font).toBe('13px Test Sans, sans-serif')

    measures.label('names')
    expect(context.font).toBe('10px Test Sans, sans-serif')
  })

  it('stands as deep as the font sets its letters about the baseline', () => {
    const { context } = stubCanvas()
    stubStyles()
    themed()
    context.measureText = (text: string) =>
      ({
        width: text.length * PER_CHARACTER,
        fontBoundingBoxAscent: 9.5,
        fontBoundingBoxDescent: 2.5,
      }) as TextMetrics

    expect(widths()!.labelDepth).toBe(12)
  })

  it('is as deep as a label is set, however tall a title is set', () => {
    // The two sizes are separate tokens, and a theme is free to raise one of
    // them alone.
    stubCanvas()
    stubStyles()
    themed({ '--numen-font-size': '20px', '--numen-edge-label-size': '11px' })

    expect(widths()!.labelDepth).toBe(11)
  })
})

describe('a theme changed under a plex already standing', () => {
  const scoped = <T,>(run: () => T): { value: T; stop: () => void } => {
    const scope = effectScope()
    return { value: scope.run(run)!, stop: () => scope.stop() }
  }

  it('measures again when the type has changed', () => {
    stubCanvas()
    stubStyles()
    themed()
    const observers = stubObserver()
    const { value: measures, stop } = scoped(() => useTitleWidths(() => 0))

    expect(measures.value!.node(node('abcd'))).toBe(4 * PER_CHARACTER + 20)

    themed({ '--numen-node-padding': '1rem' })
    for (const ring of observers) ring()

    expect(measures.value!.node(node('abcd'))).toBe(4 * PER_CHARACTER + 32)
    stop()
  })

  it('is the same measurer while the type stands, so nothing is re-arranged', () => {
    stubCanvas()
    stubStyles()
    themed()
    const observers = stubObserver()
    const { value: measures, stop } = scoped(() => useTitleWidths(() => 0))

    const first = measures.value
    for (const ring of observers) ring()

    expect(measures.value).toBe(first)
    stop()
  })

  it('takes its box back out of the page when the plex goes', () => {
    stubCanvas()
    stubStyles()
    themed()
    stubObserver()
    const { stop } = scoped(() => useTitleWidths(() => 0))

    expect(document.body.children).toHaveLength(1)
    stop()
    expect(document.body.children).toHaveLength(0)
  })

  it('follows nothing where the platform cannot say a box has changed', () => {
    stubCanvas()
    stubStyles()
    themed()
    const { value: measures, stop } = scoped(() => useTitleWidths(() => 0))

    expect(measures.value!.node(node('abcd'))).toBe(4 * PER_CHARACTER + 20)
    expect(document.body.children).toHaveLength(0)
    stop()
  })
})
