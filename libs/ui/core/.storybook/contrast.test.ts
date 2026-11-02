import { describe, expect, it } from 'vitest'
import {
  blend,
  getContrastRatio,
  getLuminance,
  getShortfalls,
  hasOwnText,
  isAccent,
  layOver,
  measureFill,
  measureText,
  resolveColours,
  type ParseColour,
  type Rgb,
} from './contrast'

/** Reads `#rrggbb`, `#rrggbbaa`, `rgb()` and `rgba()`, the forms jsdom reports; the browser parses the rest. */
const parse: ParseColour = (css) => {
  if (css === 'transparent') return { r: 0, g: 0, b: 0, a: 0 }
  const hex = /^#([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})?$/i.exec(css)
  if (hex) {
    const [, r = '0', g = '0', b = '0', a] = hex
    return {
      r: Number.parseInt(r, 16),
      g: Number.parseInt(g, 16),
      b: Number.parseInt(b, 16),
      a: a === undefined ? 1 : Number.parseInt(a, 16) / 255,
    }
  }
  const fn = /^rgba?\(([^)]+)\)$/.exec(css)
  if (fn) {
    const [r = 0, g = 0, b = 0, a = 1] = (fn[1] ?? '').split(/[\s,/]+/).map(Number)
    return { r, g, b, a }
  }
  throw new Error(`the test parser cannot read ${css}`)
}

const WHITE: Rgb = { r: 255, g: 255, b: 255 }
const BLACK: Rgb = { r: 0, g: 0, b: 0 }

/** A page of nested boxes, each given its style, with the text in the innermost. */
function nest(...styles: string[]): HTMLElement {
  document.body.innerHTML = ''
  let parent: HTMLElement = document.body
  for (const style of styles) {
    const box = document.createElement('div')
    box.setAttribute('style', style)
    parent.append(box)
    parent = box
  }
  return parent
}

describe('the ratio', () => {
  it('is 21 for black on white and 1 for a colour on itself', () => {
    expect(getContrastRatio(BLACK, WHITE)).toBeCloseTo(21, 5)
    expect(getContrastRatio(WHITE, WHITE)).toBe(1)
  })

  it('does not depend on which colour is the text', () => {
    const grey = { r: 119, g: 119, b: 119 }
    expect(getContrastRatio(grey, WHITE)).toBe(getContrastRatio(WHITE, grey))
  })

  it('is the published 4.48 for #777 on white', () => {
    expect(getContrastRatio({ r: 119, g: 119, b: 119 }, WHITE)).toBeCloseTo(4.48, 2)
  })

  it('takes the luminance of the primaries as the standard weights them', () => {
    expect(getLuminance({ r: 255, g: 0, b: 0 })).toBeCloseTo(0.2126, 4)
    expect(getLuminance({ r: 0, g: 255, b: 0 })).toBeCloseTo(0.7152, 4)
    expect(getLuminance({ r: 0, g: 0, b: 255 })).toBeCloseTo(0.0722, 4)
  })
})

describe('compositing', () => {
  it('lays half of a colour over a ground as the midpoint', () => {
    expect(layOver(BLACK, { ...WHITE, a: 0.5 })).toEqual({ r: 127.5, g: 127.5, b: 127.5 })
  })

  it('leaves the ground under a colour that covers nothing', () => {
    expect(layOver(WHITE, { r: 0, g: 0, b: 0, a: 0 })).toEqual(WHITE)
  })

  it('blends a subtree at full opacity as the subtree and at none as the backdrop', () => {
    expect(blend(WHITE, BLACK, 1)).toEqual(BLACK)
    expect(blend(WHITE, BLACK, 0)).toEqual(WHITE)
  })
})

describe('the colours of an element', () => {
  it('takes the ground from the nearest opaque ancestor', () => {
    const text = nest('background:#faf4e8', '', 'color:#3a322a')
    const { ground, text: ink } = resolveColours(text, parse, WHITE)
    expect(ground).toEqual({ r: 0xfa, g: 0xf4, b: 0xe8 })
    expect(ink).toEqual({ r: 0x3a, g: 0x32, b: 0x2a })
  })

  it('lays a translucent ground over the ground behind it', () => {
    const text = nest('background:#000000', 'background:#ffffff80', 'color:#ffffff')
    const { ground } = resolveColours(text, parse, WHITE)
    expect(ground.r).toBeCloseTo(127.5, 0)
  })

  it('falls to the canvas where no ancestor draws a ground', () => {
    const text = nest('', 'color:#000000')
    expect(resolveColours(text, parse, WHITE).ground).toEqual(WHITE)
  })

  it('lays a translucent text colour over its own ground', () => {
    const text = nest('background:#ffffff', 'color:#00000080')
    expect(resolveColours(text, parse, WHITE).text.r).toBeCloseTo(127.5, 1)
  })

  it('dims text and ground together when an element has an opacity', () => {
    const text = nest('background:#ffffff', 'background:#000000;color:#ffffff;opacity:0.5')
    const { text: ink, ground } = resolveColours(text, parse, WHITE)
    expect(ground.r).toBeCloseTo(127.5, 1)
    expect(ink.r).toBeCloseTo(255, 1)
  })

  it('multiplies the opacity of nested elements', () => {
    const text = nest('background:#ffffff', 'opacity:0.5', 'background:#000000;opacity:0.5')
    const { ground } = resolveColours(text, parse, WHITE)
    expect(ground.r).toBeCloseTo(191.25, 1)
  })

  it('refuses a ground that is an image', () => {
    const text = nest('background-image:linear-gradient(red, blue)', 'color:#000000')
    expect(() =>
      resolveColours(
        text,
        parse,
        WHITE,
        (el) =>
          ({
            ...getComputedStyle(el),
            backgroundImage: 'linear-gradient(red, blue)',
            backgroundColor: '',
            opacity: '1',
            color: '#000000',
          }) as CSSStyleDeclaration,
      ),
    ).toThrow(/image/)
  })
})

describe('measuring', () => {
  it('measures dark text on a pale ground as readable and cream on cream as not', () => {
    const readable = nest('background:#faf4e8', 'color:#3a322a')
    expect(measureText(readable, parse, WHITE)).toBeGreaterThan(10)
    const faint = nest('background:#faf4e8', 'color:#f5eddc')
    expect(measureText(faint, parse, WHITE)).toBeLessThan(1.2)
  })

  it('measures a fill against the element it sits on', () => {
    const thumb = nest('background:#8a7f70', 'background:#fffbf2')
    expect(measureFill(thumb, parse, WHITE)).toBeGreaterThan(3)
  })

  it('refuses to measure a fill with nothing under it', () => {
    expect(() => measureFill(document.documentElement, parse, WHITE)).toThrow(/sits on/)
  })
})

describe('the sweep', () => {
  it('lists the readings below a minimum and none above it', () => {
    const readings = [
      { name: 'a', ratio: 2 },
      { name: 'b', ratio: 4.5 },
    ]
    expect(getShortfalls(readings, 4.5)).toEqual([{ name: 'a', ratio: 2 }])
  })

  it('takes a colour a hover has moved a little for the accent, and a distant one for not', () => {
    const accent = { r: 168, g: 88, b: 22 }
    expect(isAccent({ r: 175, g: 95, b: 30 }, accent)).toBe(true)
    expect(isAccent({ r: 250, g: 244, b: 232 }, accent)).toBe(false)
  })

  it('tells an element holding text from one holding only elements', () => {
    const outer = nest('', '')
    outer.parentElement?.append('words')
    expect(hasOwnText(outer)).toBe(false)
    expect(hasOwnText(outer.parentElement as Element)).toBe(true)
  })
})
