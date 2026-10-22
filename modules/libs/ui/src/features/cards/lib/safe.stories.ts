/**
 * What a card's HTML evaluates to, measured over thousands of arrangements of it in
 * Chromium and in WebKit, which are the engines a card is drawn in. The
 * measuring is a `DOMParser` reading and writing markup twice, and each engine
 * makes its own of `<svg/onload=` or of a title attribute holding `</noscript>`.
 *
 * The corpus is generated from a seed the run does not change and a grammar of
 * the pieces an attempt is assembled out of, seeded with the arrangements that
 * are already known.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'
import { safe, scheme } from './safe'

/** How many arrangements one run measures. */
const ARRANGEMENTS = 2_000

/** The seed the corpus is drawn from. A run measures what the run before did. */
const SEED = 0x9e3779b9

/**
 * The tags that carry a card nowhere: each either runs something, fetches
 * something, collects something, or is read by one parser as markup and by the
 * next as text. They are written out here, and never read out of the measuring.
 */
const RUNS = [
  'script',
  'style',
  'iframe',
  'object',
  'embed',
  'noscript',
  'template',
  'svg',
  'math',
  'link',
  'meta',
  'base',
  'form',
  'input',
  'button',
  'textarea',
  'select',
  'option',
  'frame',
  'frameset',
  'applet',
  'canvas',
  'audio',
  'video',
  'source',
  'track',
  'portal',
]

/** The schemes a link may lead to. */
const LEADS = ['http:', 'https:', 'mailto:', 'tel:']

/** The declarations a card may be styled with. */
const STYLED = [
  'color',
  'background-color',
  'font-family',
  'font-size',
  'font-style',
  'font-weight',
  'text-align',
  'text-decoration',
  'vertical-align',
]

/** An image standing in the text itself, which is the one src that may name a scheme. */
const INLINE_IMAGE = /^data:image\/(png|jpeg|jpg|gif|webp|avif);base64,/i

/** The arrangements already known, which the generated ones are built around. */
const KNOWN = [
  '<script>window.ran = 1</script>',
  '<SCRIPT SRC=//example.org/x.js></SCRIPT>',
  '<img src=x onerror=window.ran=1>',
  '<svg/onload=window.ran=1>',
  '<svg><animate onbegin="window.ran = 1"></animate></svg>',
  '<noscript><p title="</noscript><img src=x onerror=window.ran=1>">',
  '<math><mtext><table><mglyph><style><img src=x onerror=window.ran=1>',
  '<a href="java\tscript:window.ran=1">press</a>',
  '<a href=" &#x6a;avascript:window.ran=1">press</a>',
  '<form action="https://example.org"><input name="p"></form>',
  '<div style="background-color: url(https://example.org/pixel.png)">a</div>',
  '<span style="width: expression(window.ran = 1)">a</span>',
  '<img src="//tracker.example/pixel.png">',
  '<img src="\\\\tracker.example/pixel.png">',
  '<a href="llama.md">a note beside this one</a>',
  '<p>a <strong>bold</strong> and <em>slanted</em> word</p>',
  '<table><tbody><tr><th colspan="2">a</th><td>b</td></tr></tbody></table>',
  '<ruby>漢<rt>kan</rt></ruby>',
  'बगीचे की खाद और हरी खाद',
  '',
]

/** The pieces an arrangement is assembled out of. */
const PIECES = [
  '<p>',
  '</p>',
  '<div>',
  '</div>',
  '<span>',
  '</span>',
  '<b>',
  '</b>',
  '<a href="{url}">',
  '</a>',
  '<img src="{url}">',
  '<img src="{url}" {on}>',
  '<{runs}>',
  '</{runs}>',
  '<{runs}/{on}>',
  '<{runs} {on}>',
  '<p {on}>',
  '<p title="{quote}">',
  '<p style="{style}">',
  '<!--',
  '-->',
  '<!--[if IE]>',
  '<![endif]-->',
  '<![CDATA[',
  ']]>',
  '<',
  '>',
  '"',
  "'",
  '/',
  '=',
  '\t',
  '\n',
  ' ',
  '',
  'a word',
  '&lt;',
  '&#x6a;',
  '&nope;',
]

/** The addresses a link or a picture is pointed at. */
const URLS = [
  'https://example.org/a',
  'llama.md',
  '#one',
  'mailto:someone@example.org',
  'javascript:window.ran=1',
  'java\tscript:window.ran=1',
  ' javascript:alert(1)',
  'data:text/html;base64,PHNjcmlwdD53aW5kb3cucmFuPTE8L3NjcmlwdD4=',
  'data:image/png;base64,iVBORw0KGgo=',
  '//tracker.example/pixel.png',
  '\\\\tracker.example/pixel.png',
  'vbscript:window.ran=1',
  '',
]

/** The handlers a tag is given, written the several ways they are written. */
const HANDLERS = [
  'onerror=window.ran=1',
  'onload="window.ran = 1"',
  'ONMOUSEOVER=window.ran=1',
  'onbegin=window.ran=1',
  'onfocus=window.ran=1 autofocus',
]

/** The styles a tag is given. */
const STYLES = [
  'color: red',
  'position: fixed; color: red',
  'background-color: url(https://example.org/x)',
  'font-size: 2em; behavior: url(#x)',
  'color: expression(window.ran = 1)',
  'text-align: right',
  'color: \\72 ed',
]

/** What a title carries, which one parser reads as text and the next as markup. */
const QUOTES = [
  '</noscript><img src=x onerror=window.ran=1>',
  '"><script>window.ran=1</script>',
  '</title><svg onload=window.ran=1>',
  'an ordinary title',
]

/**
 * A number generator a run can be repeated from: the same seed gives the same
 * corpus, so a failure names an arrangement that can be looked at again.
 */
const numbers = (from: number): (() => number) => {
  let target = from >>> 0
  return () => {
    target = (target + 0x6d2b79f5) >>> 0
    let x = target
    x = Math.imul(x ^ (x >>> 15), x | 1)
    x ^= x + Math.imul(x ^ (x >>> 7), x | 61)
    return ((x ^ (x >>> 14)) >>> 0) / 4294967296
  }
}

/** One arrangement: the known ones first, then pieces put together at random. */
const createArrangement = (next: () => number, pos: number): string => {
  if (pos < KNOWN.length) return KNOWN[pos]!

  const amongItems = <T>(pool: readonly T[]): T => pool[Math.floor(next() * pool.length)]!
  const pieces = 1 + Math.floor(next() * 12)
  let out = ''
  for (let i = 0; i < pieces; i++) {
    out += amongItems(PIECES)
      .replace('{url}', () => amongItems(URLS))
      .replace('{on}', () => amongItems(HANDLERS))
      .replace('{runs}', () => amongItems(RUNS))
      .replace('{style}', () => amongItems(STYLES))
      .replace('{quote}', () => amongItems(QUOTES))
  }
  // Half of them carry a known arrangement inside, so the generator reaches
  // past what it could have stumbled on.
  if (next() < 0.5) {
    const into = Math.floor(next() * (out.length + 1))
    out = out.slice(0, into) + amongItems(KNOWN) + out.slice(into)
  }
  return out
}

/** A URL with the spaces and control characters a scheme may be hidden behind dropped. */
const stripControls = (url: string): string => url.replace(/[\x00-\x20]/g, '')

/** How many comments the markup holds, wherever they stand. */
const comments = (root: Node): number => {
  let target = 0
  for (const child of root.childNodes) {
    if (child.nodeType === Node.COMMENT_NODE) target += 1
    else target += comments(child)
  }
  return target
}

/** What is wrong with what one element came to, and nothing where it is drawable. */
const wrong = (element: Element): string | null => {
  const tag = element.tagName.toLowerCase()
  if (RUNS.includes(tag)) return `<${tag}> survived`

  for (const attribute of element.attributes) {
    const name = attribute.name.toLowerCase()
    if (name.startsWith('on')) return `${tag} carries ${name}`
    if (name === 'srcdoc' || name === 'formaction' || name === 'xlink:href') {
      return `${tag} carries ${name}`
    }
    if (name === 'style') {
      for (const response of attribute.value.split(';')) {
        if (response.trim() === '') continue
        const property = response.slice(0, response.indexOf(':')).trim().toLowerCase()
        if (!STYLED.includes(property)) return `${tag} is styled with ${property}`
        if (/url\(|expression|\\/i.test(response)) return `${tag} is styled with ${response.trim()}`
      }
    }
  }

  const href = element.getAttribute('href')
  if (href !== null) {
    const response = scheme(href)
    if (response !== null && !LEADS.includes(response)) return `a link leads to ${response}`
  }

  const src = element.getAttribute('src')
  if (src !== null) {
    const response = stripControls(src)
    if (scheme(response) === 'data:') {
      if (!INLINE_IMAGE.test(response)) return `a picture stands in the text as ${response.slice(0, 30)}`
    } else if (scheme(response) !== null || /^[\\/]{2}/.test(response)) {
      return `a picture would be fetched from ${response.slice(0, 40)}`
    }
  }

  return null
}

/** What a run measured, as the story draws it. */
interface Tally {
  arrangements: number
  /** How many came to markup a browser reads as elements. */
  drawn: number
  elements: number
}

const measureArrangements = (): Tally => {
  const next = numbers(SEED)
  const tally: Tally = { arrangements: 0, drawn: 0, elements: 0 }

  for (let at = 0; at < ARRANGEMENTS; at++) {
    const content = createArrangement(next, at)
    const once = safe(content)
    tally.arrangements += 1

    // Read twice is the whole of the measuring: what one reading writes out is
    // what the next parser is handed, and the second reading has to find
    // nothing left to take out.
    if (safe(once) !== once) {
      throw new Error(`read twice, ${JSON.stringify(content)} came to ${JSON.stringify(once)}`)
    }

    const readItem = new DOMParser().parseFromString(once, 'text/html')
    const elements = [...readItem.body.querySelectorAll('*')]
    if (elements.length > 0) tally.drawn += 1
    tally.elements += elements.length

    for (const element of elements) {
      const response = wrong(element)
      if (response !== null) {
        throw new Error(`${response}: ${JSON.stringify(content)} came to ${JSON.stringify(once)}`)
      }
    }
    if (comments(readItem.body) > 0) {
      throw new Error(`a comment survived ${JSON.stringify(content)}`)
    }
  }
  return tally
}

const meta = {
  title: 'Cards/Safe',
  render: () => ({
    setup: () => ({ tally: measureArrangements() }),
    template: `
      <div class="numen" style="padding:32px;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans);font-size:var(--numen-font-size)">
        <p data-tally>{{ tally.arrangements }} arrangements, {{ tally.drawn }} of them drawn as {{ tally.elements }} elements</p>
      </div>
    `,
  }),
} satisfies Meta

export default meta
type Story = StoryObj

/**
 * Nothing that runs, fetches, collects or changes under a second reading is
 * left in what a card is drawn with — over two thousand arrangements, in this
 * engine.
 *
 * A run that measured almost nothing would be a run that proved nothing, so
 * what it drew is counted and drawn: an arrangement that evaluates to plain text
 * has not been measured against anything.
 */
export const Arrangements: Story = {
  play: ({ canvasElement }) => {
    const saidText = canvasElement.querySelector('[data-tally]')?.textContent ?? ''
    expect(saidText).toContain(`${ARRANGEMENTS} arrangements`)

    const drawnCount = Number(/, (\d+) of them drawn/.exec(saidText)?.[1] ?? 0)
    const elements = Number(/drawn as (\d+) elements/.exec(saidText)?.[1] ?? 0)
    expect(drawnCount).toBeGreaterThan(ARRANGEMENTS / 4)
    expect(elements).toBeGreaterThan(ARRANGEMENTS)
  },
}

/**
 * Nothing runs when what a card evaluates to is put on a live page. An image whose
 * address is no address fires as soon as it is attached, so the frames it would
 * fire in are waited for.
 */
export const NothingRuns: Story = {
  play: async ({ canvasElement }) => {
    const view = canvasElement.ownerDocument.defaultView as (Window & { ran?: number }) | null
    if (view === null) throw new Error('the story is drawn in no window')
    view.ran = 0

    const element = canvasElement.ownerDocument.createElement('div')
    canvasElement.append(element)

    const next = numbers(SEED)
    for (let at = 0; at < ARRANGEMENTS; at++) {
      element.innerHTML = safe(createArrangement(next, at))
    }
    await new Promise((settle) => view.requestAnimationFrame(() => view.setTimeout(settle, 0)))

    expect(view.ran).toBe(0)
    element.remove()
  },
}
