/**
 * Measuring a title. jsdom has neither a canvas nor a style engine, so the two
 * below stand in for them: one counts characters, the other resolves a token
 * into the pixels a browser would give. What is under test is what the measurer
 * does with a measurement, not what a font does with a letter.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { effectScope } from 'vue'
import { useTitleWidths } from './measure'
import type { PlexNode } from '../lib/node'

const node = (title: string): PlexNode => ({ id: title, title, relation: 'child' })

/** The measurers read once, where nothing on the platform watches a box. */
const widths = (icon = 0) => useTitleWidths(() => icon).value

const PER_CHARACTER = 7

/** A canvas that measures by the character, and remembers what it was asked. */
function stubCanvas() {
  const request: string[] = []
  const context = {
    font: '',
    measureText(text: string) {
      request.push(text)
      return { width: text.length * PER_CHARACTER } as TextMetrics
    },
  }

  vi.stubGlobal('CanvasRenderingContext2D', function () {})
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(
    context as unknown as CanvasRenderingContext2D,
  )

  return { asked: request, context }
}

/** The tokens as a theme writes them, which is in whatever unit it likes. */
let theme: Record<string, string> = {}

function setTheme(tokens: Record<string, string> = {}): void {
  theme = {
    '--numen-font-size': '13px',
    '--numen-font-sans': 'Test Sans, sans-serif',
    '--numen-node-padding': '10px',
    '--numen-node-gap': '6px',
    '--numen-edge-label-size': '10px',
    ...tokens,
  }
}

/** A root of 16px, which is what a browser resolves a rem against. */
const inPixels = (value: string): string =>
  value.endsWith('rem') ? `${Number.parseFloat(value) * 16}px` : value

const resolveValue = (declaration: string): string => {
  const token = /^var\((--[a-z-]+)\)$/.exec(declaration)?.[1]
  if (!token) return declaration
  const content = theme[token]
  return content ? inPixels(content) : ''
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
    const getDeclared = (property: string) => resolveValue(style?.getPropertyValue(property) ?? '')
    if (!getDeclared('font-size')) return engine(element, pseudo)

    return {
      fontSize: getDeclared('font-size'),
      fontFamily: getDeclared('font-family'),
      paddingInlineStart: getDeclared('padding-inline'),
      columnGap: getDeclared('column-gap'),
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

  it('takes the type and the padding in pixels, however the theme wrote them', () => {
    // The tokens are read off a box the document has resolved, so a theme in
    // rem is measured as one in px is.
    const { context } = stubCanvas()
    stubStyles()
    setTheme({ '--numen-font-size': '1.0625rem', '--numen-node-padding': '0.25rem' })
    const measures = widths()!
    expect(measures.node(node('ab'))).toBe(2 * PER_CHARACTER + 8)
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
    const { asked } = stubCanvas()
    stubStyles()
    setTheme()
    const measures = widths()!

    measures.node({ id: 'a', title: 'Adapter', relation: 'child' })
    measures.node({ id: 'b', title: 'Adapter', relation: 'parent' })
    measures.node(node('Port'))

    // The sample word comes first: how deep a line stands is taken off the
    // type when the measurer is made.
    expect(asked).toStrictEqual(['Hxg', 'Adapter', 'Port'])
  })
})

describe('where a caller draws an icon', () => {
  it('keeps the room it requested for, and a gap between icon and title', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    expect(widths(18)!.node(node('abcd'))).toBe(4 * PER_CHARACTER + 20 + 18 + 6)
  })

  it('keeps neither where no icon is drawn', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    expect(widths(0)!.node(node('abcd'))).toBe(4 * PER_CHARACTER + 20)
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

  it('remains as deep as the font sets its letters about the baseline', () => {
    const { context } = stubCanvas()
    stubStyles()
    setTheme()
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
    setTheme({ '--numen-font-size': '20px', '--numen-edge-label-size': '11px' })

    expect(widths()!.labelDepth).toBe(11)
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
    const { value: measures, stop } = scoped(() => useTitleWidths(() => 0))

    expect(measures.value!.node(node('abcd'))).toBe(4 * PER_CHARACTER + 20)

    setTheme({ '--numen-node-padding': '1rem' })
    for (const ring of observers) ring()

    expect(measures.value!.node(node('abcd'))).toBe(4 * PER_CHARACTER + 32)
    stop()
  })

  it('is the same measurer while the type remains, so nothing is re-arranged', () => {
    stubCanvas()
    stubStyles()
    setTheme()
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
    setTheme()
    stubObserver()
    const { stop } = scoped(() => useTitleWidths(() => 0))

    expect(document.body.children).toHaveLength(1)
    stop()
    expect(document.body.children).toHaveLength(0)
  })

  it('follows nothing where the platform cannot say a box has changed', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    const { value: measures, stop } = scoped(() => useTitleWidths(() => 0))

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
  /** An observer that remembers what it was asked to watch, and whether it was let go. */
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
    expect(title!.style.getPropertyValue('font-size')).toBe('')
    expect(label!.textContent).toBe('Hxg')
    expect(label!.style.getPropertyValue('font-size')).toBe('var(--numen-edge-label-size)')
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

  it('is release when the plex goes', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    const { disconnect } = stubWatching()
    const scope = effectScope()
    scope.run(() => useTitleWidths(() => 0))
    expect(disconnect).not.toHaveBeenCalled()

    scope.stop()
    expect(disconnect).toHaveBeenCalledOnce()
  })

  it('is read without a warning where no scope is there to dispose of it', () => {
    stubCanvas()
    stubStyles()
    setTheme()
    stubWatching()
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})

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
    ['the padding', { '--numen-node-padding': '12px' }],
    ['the gap', { '--numen-node-gap': '8px' }],
  ])('measures again when only %s has changed', (_name, tokens) => {
    stubCanvas()
    stubStyles()
    setTheme()
    const observers = stubObserver()
    const { value: measures, stop } = scoped(() => useTitleWidths(() => 0))
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
    vi.stubGlobal('CanvasRenderingContext2D', function () {})
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
    vi.stubGlobal('CanvasRenderingContext2D', function () {})
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(() => {
      throw new Error('no 2d context')
    })
    setTheme()
    stubStyles()

    expect(widths()).toBeUndefined()
  })
})
