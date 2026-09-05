/**
 * What a card's HTML comes to, measured over thousands of arrangements of it
 * and in the engine that draws the card.
 *
 * A story is run in a real browser by `@storybook/addon-vitest`, in Chromium
 * and in WebKit, which is what this has to be: the measuring is a `DOMParser`
 * reading and writing markup twice, and what one parser makes of `<svg/onload=`
 * or of a title attribute holding `</noscript>` is not what the next one makes
 * of it. A parser written in JavaScript agreeing with itself proves nothing
 * about the two engines a card is ever drawn in.
 *
 * The corpus is generated: a seed the run does not change, a grammar of the
 * pieces an attempt is assembled out of, and the arrangements a person could
 * not have thought to write down. The seeds of it are the arrangements that are
 * already known — a handler however the tag is closed, a scheme behind control
 * characters, markup that comes back as different markup when it is read twice.
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
 * next as text.
 *
 * They are written out here rather than read out of the measuring, because a
 * test that takes its answer from the thing it measures measures nothing.
 */
const RUNS = [
  'script', 'style', 'iframe', 'object', 'embed', 'noscript', 'template',
  'svg', 'math', 'link', 'meta', 'base', 'form', 'input', 'button',
  'textarea', 'select', 'option', 'frame', 'frameset', 'applet', 'canvas',
  'audio', 'video', 'source', 'track', 'portal',
]

/** The schemes a link may lead to. */
const LEADS = ['http:', 'https:', 'mailto:', 'tel:']

/** The declarations a card may be styled with. */
const STYLED = [
  'color', 'background-color', 'font-family', 'font-size', 'font-style',
  'font-weight', 'text-align', 'text-decoration', 'vertical-align',
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
  '<p>', '</p>', '<div>', '</div>', '<span>', '</span>', '<b>', '</b>',
  '<a href="{url}">', '</a>', '<img src="{url}">', '<img src="{url}" {on}>',
  '<{runs}>', '</{runs}>', '<{runs}/{on}>', '<{runs} {on}>',
  '<p {on}>', '<p title="{quote}">', '<p style="{style}">',
  '<!--', '-->', '<!--[if IE]>', '<![endif]-->', '<![CDATA[', ']]>',
  '<', '>', '"', "'", '/', '=', '\t', '\n', ' ', '',
  'a word', '&lt;', '&#x6a;', '&nope;',
]

/** The addresses a link or a picture is pointed at. */
const URLS = [
  'https://example.org/a', 'llama.md', '#one', 'mailto:someone@example.org',
  'javascript:window.ran=1', 'java\tscript:window.ran=1', ' javascript:alert(1)',
  'data:text/html;base64,PHNjcmlwdD53aW5kb3cucmFuPTE8L3NjcmlwdD4=',
  'data:image/png;base64,iVBORw0KGgo=', '//tracker.example/pixel.png',
  '\\\\tracker.example/pixel.png', 'vbscript:window.ran=1', '',
]

/** The handlers a tag is given, written the several ways they are written. */
const HANDLERS = [
  'onerror=window.ran=1', 'onload="window.ran = 1"', 'ONMOUSEOVER=window.ran=1',
  'onbegin=window.ran=1', 'onfocus=window.ran=1 autofocus',
]

/** The styles a tag is given. */
const STYLES = [
  'color: red', 'position: fixed; color: red',
  'background-color: url(https://example.org/x)',
  'font-size: 2em; behavior: url(#x)', 'color: expression(window.ran = 1)',
  'text-align: right', 'color: \\72 ed',
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
const numbers = (seed: number): (() => number) => {
  let held = seed >>> 0
  return () => {
    held = (held + 0x6d2b79f5) >>> 0
    let x = held
    x = Math.imul(x ^ (x >>> 15), x | 1)
    x ^= x + Math.imul(x ^ (x >>> 7), x | 61)
    return ((x ^ (x >>> 14)) >>> 0) / 4294967296
  }
}

/** One arrangement: the known ones first, then pieces put together at random. */
const arranged = (next: () => number, at: number): string => {
  if (at < KNOWN.length) return KNOWN[at]!

  const among = <T,>(pool: readonly T[]): T => pool[Math.floor(next() * pool.length)]!
  const pieces = 1 + Math.floor(next() * 12)
  let out = ''
  for (let i = 0; i < pieces; i++) {
    out += among(PIECES)
      .replace('{url}', () => among(URLS))
      .replace('{on}', () => among(HANDLERS))
      .replace('{runs}', () => among(RUNS))
      .replace('{style}', () => among(STYLES))
      .replace('{quote}', () => among(QUOTES))
  }
  // Half of them carry a known arrangement inside, so the generator reaches
  // past what it could have stumbled on.
  if (next() < 0.5) {
    const into = Math.floor(next() * (out.length + 1))
    out = out.slice(0, into) + among(KNOWN) + out.slice(into)
  }
  return out
}

/** A URL with the spaces and control characters a scheme may be hidden behind dropped. */
const bare = (url: string): string => url.replace(/[\x00-\x20]/g, '')

/** How many comments the markup holds, wherever they stand. */
const comments = (root: Node): number => {
  let held = 0
  for (const child of root.childNodes) {
    if (child.nodeType === Node.COMMENT_NODE) held += 1
    else held += comments(child)
  }
  return held
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
      for (const said of attribute.value.split(';')) {
        if (said.trim() === '') continue
        const property = said.slice(0, said.indexOf(':')).trim().toLowerCase()
        if (!STYLED.includes(property)) return `${tag} is styled with ${property}`
        if (/url\(|expression|\\/i.test(said)) return `${tag} is styled with ${said.trim()}`
      }
    }
  }

  const href = element.getAttribute('href')
  if (href !== null) {
    const said = scheme(href)
    if (said !== null && !LEADS.includes(said)) return `a link leads to ${said}`
  }

  const src = element.getAttribute('src')
  if (src !== null) {
    const said = bare(src)
    if (scheme(said) === 'data:') {
      if (!INLINE_IMAGE.test(said)) return `a picture stands in the text as ${said.slice(0, 30)}`
    } else if (scheme(said) !== null || /^[\\/]{2}/.test(said)) {
      return `a picture would be fetched from ${said.slice(0, 40)}`
    }
  }

  return null
}

/** What a run measured, as the story draws it. */
interface Tally {
  arrangements: number
  /** How many came to markup a browser reads as elements, rather than to text. */
  drawn: number
  elements: number
}

const measured = (): Tally => {
  const next = numbers(SEED)
  const tally: Tally = { arrangements: 0, drawn: 0, elements: 0 }

  for (let at = 0; at < ARRANGEMENTS; at++) {
    const written = arranged(next, at)
    const once = safe(written)
    tally.arrangements += 1

    // Read twice is the whole of the measuring: what one reading writes out is
    // what the next parser is handed, and the second reading has to find
    // nothing left to take out.
    if (safe(once) !== once) {
      throw new Error(`read twice, ${JSON.stringify(written)} came to ${JSON.stringify(once)}`)
    }

    const read = new DOMParser().parseFromString(once, 'text/html')
    const elements = [...read.body.querySelectorAll('*')]
    if (elements.length > 0) tally.drawn += 1
    tally.elements += elements.length

    for (const element of elements) {
      const said = wrong(element)
      if (said !== null) {
        throw new Error(`${said}: ${JSON.stringify(written)} came to ${JSON.stringify(once)}`)
      }
    }
    if (comments(read.body) > 0) {
      throw new Error(`a comment survived ${JSON.stringify(written)}`)
    }
  }
  return tally
}

const meta = {
  title: 'Cards/Safe',
  render: () => ({
    setup: () => ({ tally: measured() }),
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
 * what it drew is counted and drawn: an arrangement that comes to plain text
 * has not been measured against anything.
 */
export const Arrangements: Story = {
  play: ({ canvasElement }) => {
    const said = canvasElement.querySelector('[data-tally]')?.textContent ?? ''
    expect(said).toContain(`${ARRANGEMENTS} arrangements`)

    const drawn = Number(/, (\d+) of them drawn/.exec(said)?.[1] ?? 0)
    const elements = Number(/drawn as (\d+) elements/.exec(said)?.[1] ?? 0)
    expect(drawn).toBeGreaterThan(ARRANGEMENTS / 4)
    expect(elements).toBeGreaterThan(ARRANGEMENTS)
  },
}

/**
 * Nothing runs when what a card comes to is put on a live page.
 *
 * A handler that survived is measured by the reading above; a handler this
 * window would fire is a second question, and the page is the only place it is
 * answered. An image whose address is no address fires as soon as it is
 * attached, so the frames are waited for rather than assumed.
 */
export const NothingRuns: Story = {
  play: async ({ canvasElement }) => {
    const view = canvasElement.ownerDocument.defaultView as (Window & { ran?: number }) | null
    if (view === null) throw new Error('the story is drawn in no window')
    view.ran = 0

    const held = canvasElement.ownerDocument.createElement('div')
    canvasElement.append(held)

    const next = numbers(SEED)
    for (let at = 0; at < ARRANGEMENTS; at++) {
      held.innerHTML = safe(arranged(next, at))
    }
    await new Promise((settle) => view.requestAnimationFrame(() => view.setTimeout(settle, 0)))

    expect(view.ran).toBe(0)
    held.remove()
  },
}
