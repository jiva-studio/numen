/**
 * What survives a card's HTML and what does not.
 *
 * A deck may come from another person and the window is a webview, so every
 * attempt below is one that has to be gone by the time it reaches a screen.
 */
import MarkdownIt from 'markdown-it'
import { describe, expect, it } from 'vitest'
import { safe } from './safe'
import { drawn } from './render'

/** The marks alone, told to read tags as tags and told nothing else. */
const unmeasured = new MarkdownIt({ html: true, linkify: true })

/** What the text comes to, with the case of the tags settled. */
const cleaned = (html: string): string => safe(html).toLowerCase()

/** Every element the text comes to, read back the way a window reads it. */
const elements = (html: string): readonly Element[] => [
  ...new DOMParser().parseFromString(safe(html), 'text/html').body.querySelectorAll('*'),
]

/** The tags the text comes to. */
const tags = (html: string): readonly string[] =>
  elements(html).map((each) => each.tagName.toLowerCase())

/** Every attribute the text comes to carries, named. */
const attributes = (html: string): readonly string[] =>
  elements(html).flatMap((each) =>
    [...each.attributes].map((attribute) => attribute.name.toLowerCase()),
  )

describe('safe, what a card may be drawn with', () => {
  it('keeps the tags a card is written with', () => {
    expect(safe('<p>a <strong>bold</strong> and <em>slanted</em> word</p>')).toBe(
      '<p>a <strong>bold</strong> and <em>slanted</em> word</p>',
    )
  })

  it('keeps a table whole', () => {
    const table = '<table><tbody><tr><th colspan="2">a</th><td>b</td></tr></tbody></table>'
    expect(safe(table)).toBe(table)
  })

  it('keeps ruby, which is what a card of another script is written with', () => {
    expect(safe('<ruby>漢<rt>kan</rt></ruby>')).toBe('<ruby>漢<rt>kan</rt></ruby>')
  })

  it('keeps a link that points somewhere a link may point', () => {
    expect(safe('<a href="https://example.org">there</a>')).toBe(
      '<a href="https://example.org">there</a>',
    )
    expect(safe('<a href="llama.md">there</a>')).toBe('<a href="llama.md">there</a>')
  })

  it('keeps the words of a tag it does not draw, and drops the tag', () => {
    expect(cleaned('<marquee>still here</marquee>')).toBe('still here')
  })
})

describe('safe, what does not survive', () => {
  it('drops a script and everything in it', () => {
    expect(cleaned('<script>alert(1)</script>')).toBe('')
    expect(cleaned('a<script>alert(1)</script>b')).toBe('ab')
  })

  it('drops a script whose tag is written in capitals', () => {
    expect(cleaned('<SCRIPT>alert(1)</SCRIPT>')).toBe('')
  })

  it('drops a handler on a tag it does keep', () => {
    expect(cleaned('<img src="x" onerror="alert(1)">')).toBe('<img src="x">')
  })

  it('drops a handler however it is written', () => {
    expect(cleaned('<p ONLOAD="alert(1)" onclick="alert(2)">a</p>')).toBe('<p>a</p>')
  })

  it('drops a link that would run something', () => {
    expect(cleaned('<a href="javascript:alert(1)">press</a>')).toBe('<a>press</a>')
  })

  it('drops a link hiding its scheme behind spaces and control characters', () => {
    expect(cleaned('<a href="java\tscript:alert(1)">press</a>')).toBe('<a>press</a>')
    expect(cleaned('<a href=" javascript:alert(1)">press</a>')).toBe('<a>press</a>')
  })

  it('drops a frame, an object and an embed', () => {
    expect(cleaned('<iframe src="https://example.org"></iframe>')).toBe('')
    expect(cleaned('<object data="x.swf"></object>')).toBe('')
    expect(cleaned('<embed src="x.swf">')).toBe('')
  })

  it('drops a stylesheet and a style block', () => {
    expect(cleaned('<style>body{display:none}</style>')).toBe('')
    expect(cleaned('<link rel="stylesheet" href="x.css">')).toBe('')
  })

  it('drops a form and everything it would collect', () => {
    expect(cleaned('<form action="https://example.org"><input name="p"></form>')).toBe('')
  })

  it('drops an image whose source would run something', () => {
    expect(cleaned('<img src="javascript:alert(1)">')).toBe('<img>')
  })

  it('drops an image standing in the text that is not an image', () => {
    expect(cleaned('<img src="data:text/html;base64,aaa">')).toBe('<img>')
  })

  it('keeps an image standing in the text that is one', () => {
    const said = 'data:image/png;base64,iVBORw0KGgo='
    expect(safe(`<img src="${said}">`)).toBe(`<img src="${said}">`)
  })

  it('drops a comment, which is read differently by every parser', () => {
    expect(cleaned('a<!--[if IE]><script>alert(1)</script><![endif]-->b')).toBe('ab')
  })

  it('drops an svg, and the handler it would carry', () => {
    expect(cleaned('<svg><animate onbegin="alert(1)"></animate></svg>')).toBe('')
  })

  it('leaves no handler in what a second reading makes of the text', () => {
    const said = '<noscript><p title="</noscript><img src=x onerror=alert(1)>">'
    expect(attributes(said).filter((name) => name.startsWith('on'))).toEqual([])
    expect(tags(said)).not.toContain('img')
  })

  it('leaves no handler on anything, however the tag was closed', () => {
    const said = '<div><p onmouseover=alert(1)>a<svg/onload=alert(2)></p></div>'
    expect(attributes(said).filter((name) => name.startsWith('on'))).toEqual([])
    expect(tags(said)).not.toContain('svg')
  })

  it('drops an attribute a tag it keeps may not carry', () => {
    expect(cleaned('<a href="x" target="_blank" id="taken">a</a>')).toBe('<a href="x">a</a>')
  })
})

describe('safe, the styles a card may carry', () => {
  it('keeps a declaration a card is styled with', () => {
    expect(cleaned('<span style="color: red">a</span>')).toBe('<span style="color: red">a</span>')
  })

  it('keeps the alignment a table column is written with', () => {
    expect(cleaned('<table><tr><td style="text-align:right">a</td></tr></table>')).toContain(
      '<td style="text-align:right">a</td>',
    )
  })

  it('drops a declaration that would fetch something', () => {
    expect(cleaned('<span style="background-color: url(https://example.org/x)">a</span>')).toBe(
      '<span>a</span>',
    )
  })

  it('drops a declaration a card is not styled with', () => {
    expect(cleaned('<span style="position: fixed; color: red">a</span>')).toBe(
      '<span style="color: red">a</span>',
    )
  })
})

describe('drawn', () => {
  it('reads the marks as marks', () => {
    expect(drawn('**bold**')).toBe('<p><strong>bold</strong></p>\n')
  })

  it('reads the tags among the marks as tags', () => {
    expect(drawn('a <u>marked</u> word')).toBe('<p>a <u>marked</u> word</p>\n')
  })

  it('draws no script a person wrote among the marks', () => {
    expect(drawn('before\n\n<script>alert(1)</script>\n\nafter')).not.toContain('alert')
  })

  it('draws no handler a person wrote among the marks', () => {
    expect(drawn('<img src="x" onerror="alert(1)">')).not.toContain('onerror')
  })

  it('draws text that is not Latin as it was written', () => {
    expect(drawn('बगीचे की खाद और हरी खाद')).toContain('बगीचे की खाद और हरी खाद')
  })

  it('draws nothing for text with nothing in it', () => {
    expect(drawn('')).toBe('')
  })

  /* The marks alone would draw each of these, so what takes them out is the
     measuring and nothing else. */
  it.each([
    '<script>alert(1)</script>',
    '<img src="x" onerror="alert(1)">',
    '<iframe src="https://example.org"></iframe>',
  ])('is what takes %s out, which the marks alone would draw', (said) => {
    expect(unmeasured.render(said)).toContain(said)
    expect(drawn(said)).not.toContain(said)
  })
})
