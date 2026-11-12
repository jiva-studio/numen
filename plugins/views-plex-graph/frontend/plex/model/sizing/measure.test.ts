/**
 * Measuring a title. jsdom has neither a canvas nor a style engine, so the two
 * below stand in for them: one counts characters, the other resolves a token
 * into the pixels a browser would give. What is under test is what the measurer
 * does with a measurement, not what a font does with a letter.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { effectScope } from 'vue'
import { useTitleWidths, type PlexType, type TitleRoom } from './measure'
import type { PlexNode } from '../../../types'

const node = (title: string): PlexNode => ({ id: title, title, relation: 'child' })

const PER_CHARACTER = 7

function stubCanvas() {
  const request: string[] = []
  const context = {
    font: '',
    measureText(text: string) {
      request.push(text)
      return { width: text.length * PER_CHARACTER } as TextMetrics
    },
  }

  vi.stubGlobal('CanvasRenderingContext2D', class {})
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(
    context as unknown as CanvasRenderingContext2D,
  )

  return { request: request, context }
}

/** The tokens as a theme writes them, which is in whatever unit it likes. */
let theme: Record<string, string> = {}

function setTheme(tokens: Record<string, string> = {}): void {
  theme = {
    '--numen-font-size': '13px',
    '--numen-font-sans': 'Test Sans, sans-serif',
    '--numen-node-gap': '6px',
    '--numen-edge-label-size': '10px',
    ...tokens,
  }
}

const inPixels = (value: string): string =>
  value.endsWith('rem') ? `${Number.parseFloat(value) * 16}px` : value

const getToken = (token: string): string => inPixels(theme[token] ?? '')

const readThemeType = (): PlexType => {
  const family = getToken('--numen-font-sans') || 'sans-serif'
  return {
    font: `${getToken('--numen-font-size')} ${family}`,
    labelFont: `${getToken('--numen-edge-label-size')} ${family}`,
    gap: Number.parseFloat(getToken('--numen-node-gap')) || 0,
  }
}

/** The padding the plex hands its measurer, and nothing drawn beside a title. */
const PADDING = 10
const NOTHING_BESIDE: TitleRoom = { padding: PADDING, beside: [] }

const widths = (room: TitleRoom = NOTHING_BESIDE) => useTitleWidths(() => room, readThemeType).value

const withIcon = (icon: number): TitleRoom => ({ padding: PADDING, beside: icon > 0 ? [icon] : [] })

function stubStyles() {
  // Type values are read by readThemeType above.
}

function stubObserver(): (() => void)[] {
  const observers: (() => void)[] = []

  vi.stubGlobal(
    'ResizeObserver',
    class {
      constructor(ring: () => void) {
        observers.push(ring)
      }
      observe = vi.fn()
      disconnect = vi.fn()
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
    setTheme()
    stubStyles()
    expect(widths()).toBeUndefined()
  })

  it('leaves nothing of itself in the page', () => {
    setTheme()
    stubStyles()
    widths()
    expect(document.body.children).toHaveLength(0)
  })
})

describe('what a title needs', () => {
  it('adds the padding on both sides of the text', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    const measures = widths()!
    expect(measures.node(node('abcd'))).toBe(4 * PER_CHARACTER + 20)
  })

  it('takes the type in pixels, however the theme wrote it', () => {
    // The tokens are read off a box the document has resolved, so a theme in
    // rem is measured as one in px is.
    const { context } = stubCanvas()
    stubStyles()
    setTheme({ '--numen-font-size': '1.0625rem' })
    const measures = widths()!
    expect(measures.node(node('ab'))).toBe(2 * PER_CHARACTER + 20)
    expect(context.font).toBe('17px Test Sans, sans-serif')
  })

  it('gives an empty title the padding alone', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    const measures = widths()!
    expect(measures.node(node(''))).toBe(20)
  })

  it('answers in whole pixels', () => {
    const { context } = stubCanvas()
    stubStyles()
    setTheme()
    context.measureText = (text: string) => ({ width: text.length * 6.4 }) as TextMetrics
    const measures = widths()!
    expect(measures.node(node('abc'))).toBe(Math.ceil(3 * 6.4) + 20)
  })

  it('measures a string once, however many nodes carry it', () => {
    const { request } = stubCanvas()
    stubStyles()
    setTheme()
    const measures = widths()!

    measures.node({ id: 'a', title: 'Adapter', relation: 'child' })
    measures.node({ id: 'b', title: 'Adapter', relation: 'parent' })
    measures.node(node('Port'))

    // The three letters of the ink are measured first: they are read from the type when
    // the measurer is created.
    expect(request).toStrictEqual(['H', 'g', 'x', 'Adapter', 'Port'])
  })
})

describe('where a caller draws an icon', () => {
  it('keeps the room it requested for, and a gap between icon and title', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    expect(widths(withIcon(18))!.node(node('abcd'))).toBe(4 * PER_CHARACTER + 20 + 18 + 6)
  })

  it('keeps neither where no icon is drawn', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    expect(widths(withIcon(0))!.node(node('abcd'))).toBe(4 * PER_CHARACTER + 20)
  })
})

describe('the padding of a title and the widths drawn before it', () => {
  it('uses the padding from the plex options, not a theme token', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    expect(widths({ padding: 4, beside: [] })!.node(node('abcd'))).toBe(4 * PER_CHARACTER + 8)
    expect(widths({ padding: 4, beside: [] })!.heading('abc')).toBe(3 * PER_CHARACTER + 8)
  })

  it('keeps a gap after each thing drawn before the title', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    const room = { padding: PADDING, beside: [6, 18] }
    expect(widths(room)!.node(node('abcd'))).toBe(4 * PER_CHARACTER + 20 + 6 + 6 + 18 + 6)
  })
})

describe('what a label needs', () => {
  it('is the words alone: a label relies on a line and carries no padding', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    expect(widths()!.label('names')).toBe(5 * PER_CHARACTER)
  })

  it('is measured in the size a label is set at, not a title', () => {
    const { context } = stubCanvas()
    stubStyles()
    setTheme()
    const measures = widths()!

    measures.node(node('Domain'))
    expect(context.font).toBe('13px Test Sans, sans-serif')

    measures.label('names')
    expect(context.font).toBe('10px Test Sans, sans-serif')
  })

  it('takes the ink of a label from the font: a capital, a descender and a lowercase x', () => {
    const { context } = stubCanvas()
    stubStyles()
    setTheme()
    const ascent: Record<string, number> = { H: 9.5, x: 5.25 }
    const descent: Record<string, number> = { g: 2.5 }
    context.measureText = (text: string) =>
      ({
        width: text.length * PER_CHARACTER,
        actualBoundingBoxAscent: ascent[text] ?? 0,
        actualBoundingBoxDescent: descent[text] ?? 0,
      }) as TextMetrics

    expect(widths()!.labelType).toStrictEqual({ ascent: 9.5, descent: 2.5, xHeight: 5.25 })
  })

  it('falls back to the size a label is set at, however tall a title is set, where the platform reports no ink', () => {
    // The two sizes are separate tokens, and a theme is free to raise one of
    // them alone.
    stubCanvas()
    stubStyles()
    setTheme({ '--numen-font-size': '20px', '--numen-edge-label-size': '11px' })

    const { ascent, descent, xHeight } = widths()!.labelType
    expect(ascent + descent).toBeCloseTo(0.94 * 11)
    expect(xHeight).toBeCloseTo(0.5 * 11)
  })
})

describe('a theme changed under a plex already standing', () => {
  const scoped = <T>(run: () => T): { value: T; stop: () => void } => {
    const scope = effectScope()
    return { value: scope.run(run)!, stop: () => scope.stop() }
  }

  it('measures again when the type has changed', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    const observers = stubObserver()
    const { value: measures, stop } = scoped(() =>
      useTitleWidths(() => withIcon(18), readThemeType),
    )

    expect(measures.value!.node(node('abcd'))).toBe(4 * PER_CHARACTER + 20 + 18 + 6)

    setTheme({ '--numen-node-gap': '0.5rem' })
    for (const ring of observers) ring()

    expect(measures.value!.node(node('abcd'))).toBe(4 * PER_CHARACTER + 20 + 18 + 8)
    stop()
  })

  it('is the same measurer while the type remains, so nothing is re-arranged', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    const observers = stubObserver()
    const { value: measures, stop } = scoped(() =>
      useTitleWidths(() => NOTHING_BESIDE, readThemeType),
    )

    const first = measures.value
    for (const ring of observers) ring()

    expect(measures.value).toBe(first)
    stop()
  })

  it('takes its box back out of the page when the plex is disposed', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    stubObserver()
    const { stop } = scoped(() => useTitleWidths(() => NOTHING_BESIDE, readThemeType))

    expect(document.body.children).toHaveLength(1)
    stop()
    expect(document.body.children).toHaveLength(0)
  })

  it('follows nothing where the platform cannot say a box has changed', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    const { value: measures, stop } = scoped(() =>
      useTitleWidths(() => NOTHING_BESIDE, readThemeType),
    )

    expect(measures.value!.node(node('abcd'))).toBe(4 * PER_CHARACTER + 20)
    expect(document.body.children).toHaveLength(0)
    stop()
  })
})

describe('a page the plex cannot stand a box in', () => {
  it('gives nothing where the document has no body', () => {
    setTheme()
    stubStyles()
    Object.defineProperty(document, 'body', { configurable: true, get: () => null })

    try {
      expect(widths()).toBeUndefined()
    } finally {
      Reflect.deleteProperty(document, 'body')
    }
  })

  it('gives nothing where there is no document at all', () => {
    vi.stubGlobal('document', undefined)

    expect(widths()).toBeUndefined()
  })

  it('gives nothing where the platform has no canvas class though a canvas answers', () => {
    const context = { font: '', measureText: () => ({ width: 1 }) as TextMetrics }
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(
      context as unknown as CanvasRenderingContext2D,
    )
    setTheme()
    stubStyles()

    expect(widths()).toBeUndefined()
  })
})

describe('the box a plex measures its type against', () => {
  /** An observer that records the targets it observes and whether they were released. */
  function stubWatching() {
    const observe = vi.fn<(target: Element, options?: ResizeObserverOptions) => void>()
    const disconnect = vi.fn<() => void>()
    vi.stubGlobal(
      'ResizeObserver',
      class {
        observe = observe
        disconnect = disconnect
      },
    )
    return { observe, disconnect }
  }

  it('is hidden from assistive technology and holds a title word and a label word', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    stubWatching()
    widths()

    const box = document.body.children[0] as HTMLElement
    expect(box.getAttribute('aria-hidden')).toBe('true')
    expect(box.children).toHaveLength(2)

    const [title, label] = Array.from(box.children) as HTMLElement[]
    expect(title!.textContent).toBe('Hxg')
    expect(label!.textContent).toBe('Hxg')
  })

  it('is watched for the size of its border box', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    const { observe } = stubWatching()
    widths()

    expect(observe).toHaveBeenCalledExactlyOnceWith(document.body.children[0], {
      box: 'border-box',
    })
  })

  it('is released when the plex is disposed', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    const { disconnect } = stubWatching()
    const scope = effectScope()
    scope.run(() => useTitleWidths(() => NOTHING_BESIDE, readThemeType))
    expect(disconnect).not.toHaveBeenCalled()

    scope.stop()
    expect(disconnect).toHaveBeenCalledOnce()
  })

  it('is read without a warning where no scope is there to dispose of it', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    stubWatching()
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined)

    widths()

    expect(warn).not.toHaveBeenCalled()
  })
})

describe('what a heading needs', () => {
  it('is its words with the padding on both sides', () => {
    stubCanvas()
    stubStyles()
    setTheme()

    expect(widths()!.heading('abc')).toBe(3 * PER_CHARACTER + 20)
  })

  it('is measured in the size a label is set at, in whole pixels', () => {
    const { context } = stubCanvas()
    stubStyles()
    setTheme()
    context.measureText = (text: string) => ({ width: text.length * 6.4 }) as TextMetrics
    const measures = widths()!

    expect(measures.heading('abc')).toBe(Math.ceil(3 * 6.4 + 20))
    expect(context.font).toBe('10px Test Sans, sans-serif')
  })
})

describe('the canvas a plex measures with', () => {
  it('is a two-dimensional one', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    widths()

    expect(HTMLCanvasElement.prototype.getContext).toHaveBeenCalledWith('2d')
  })
})

describe('a theme changed in one token under a plex already standing', () => {
  const scoped = <T>(run: () => T): { value: T; stop: () => void } => {
    const scope = effectScope()
    return { value: scope.run(run)!, stop: () => scope.stop() }
  }

  it.each([
    ['the title size', { '--numen-font-size': '14px' }],
    ['the label size', { '--numen-edge-label-size': '11px' }],
    ['the gap', { '--numen-node-gap': '8px' }],
  ])('measures again when only %s has changed', (_name, tokens) => {
    stubCanvas()
    stubStyles()
    setTheme()
    const observers = stubObserver()
    const { value: measures, stop } = scoped(() =>
      useTitleWidths(() => NOTHING_BESIDE, readThemeType),
    )
    const first = measures.value

    setTheme(tokens)
    for (const ring of observers) ring()

    expect(measures.value).not.toBe(first)
    stop()
  })
})

describe('type the document leaves unsaid or a window refuses', () => {
  it('falls back to a generic family where the theme names none', () => {
    const { context } = stubCanvas()
    stubStyles()
    setTheme({ '--numen-font-sans': '' })
    widths()!.node(node('ab'))

    expect(context.font).toBe('13px sans-serif')
  })

  it('gives nothing where the window throws on making a canvas context', () => {
    vi.stubGlobal('CanvasRenderingContext2D', class {})
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(() => {
      throw new Error('no canvas')
    })
    setTheme()
    stubStyles()

    expect(widths()).toBeUndefined()
  })
})

describe('a canvas the page cannot make', () => {
  it('gives nothing where the canvas has no 2d context to give', () => {
    vi.stubGlobal('CanvasRenderingContext2D', class {})
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(() => {
      throw new Error('no 2d context')
    })
    setTheme()
    stubStyles()

    expect(widths()).toBeUndefined()
  })
})
