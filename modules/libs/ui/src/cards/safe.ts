/**
 * A card's HTML, measured against what a card may be drawn with.
 *
 * A deck may come from another person and the window is a webview, so a tag,
 * an attribute or a scheme that is not named here does not survive.
 */

/** The tags a card is drawn with. */
const DRAWN = new Set([
  'p', 'br', 'hr', 'span', 'div',
  'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
  'strong', 'b', 'em', 'i', 'u', 's', 'del', 'ins', 'mark', 'small',
  'sub', 'sup', 'code', 'pre', 'kbd', 'samp', 'var', 'abbr', 'q', 'cite',
  'dfn', 'time', 'bdi', 'bdo', 'ruby', 'rt', 'rp',
  'blockquote',
  'ul', 'ol', 'li', 'dl', 'dt', 'dd',
  'table', 'thead', 'tbody', 'tfoot', 'tr', 'th', 'td', 'caption',
  'colgroup', 'col',
  'a', 'img', 'figure', 'figcaption',
])

/** The tags that go, and everything they hold with them. */
const STRUCK = new Set([
  'script', 'style', 'iframe', 'object', 'embed', 'noscript', 'template',
  'svg', 'math', 'link', 'meta', 'base', 'form', 'input', 'button',
  'textarea', 'select', 'option', 'head', 'title', 'frame', 'frameset',
  'applet', 'canvas', 'audio', 'video', 'source', 'track', 'portal',
])

/** The attributes any tag may carry. */
const ANY = new Set(['class', 'dir', 'lang', 'title'])

/** What each tag may carry beyond those. */
const OWN: Readonly<Record<string, readonly string[]>> = {
  a: ['href'],
  img: ['src', 'alt', 'width', 'height'],
  td: ['colspan', 'rowspan'],
  th: ['colspan', 'rowspan', 'scope'],
  ol: ['start', 'reversed'],
  time: ['datetime'],
  col: ['span'],
  colgroup: ['span'],
}

/** The schemes a link may point at. An address naming none is the caller's to resolve. */
const SCHEMES = new Set(['http:', 'https:', 'mailto:', 'tel:'])

/** The declarations a tag may be styled with. */
const STYLED = new Set([
  'color',
  'background-color',
  'font-family',
  'font-size',
  'font-style',
  'font-weight',
  'text-align',
  'text-decoration',
  'vertical-align',
])

/** An image standing in the text itself. */
const INLINE_IMAGE = /^data:image\/(png|jpeg|jpg|gif|webp|avif);base64,/i

/** A URL with the spaces and control characters a scheme may be hidden behind dropped. */
const bare = (url: string): string => url.replace(/[\u0000-\u0020]/g, '')

/** The scheme a URL names, and nothing where it names none. */
export const scheme = (url: string): string | null =>
  /^[a-zA-Z][a-zA-Z0-9+.-]*:/.exec(bare(url))?.[0]?.toLowerCase() ?? null

const points = (url: string): boolean => {
  const said = scheme(url)
  return said === null || SCHEMES.has(said)
}

const shows = (url: string): boolean =>
  scheme(url) === 'data:' ? INLINE_IMAGE.test(bare(url)) : points(url)

/** A style with every declaration that is not drawn with dropped. */
const styled = (value: string): string =>
  value
    .split(';')
    .map((each) => each.trim())
    .filter((each) => {
      const at = each.indexOf(':')
      if (at === -1) return false
      const property = each.slice(0, at).trim().toLowerCase()
      const said = each.slice(at + 1).toLowerCase()
      if (!STYLED.has(property)) return false
      return !said.includes('url(') && !said.includes('expression') && !said.includes('\\')
    })
    .join('; ')

/** Comments go: what a parser makes of one is not what the next parser makes. */
const uncommented = (root: ParentNode & Node): void => {
  const walk = (node: Node): void => {
    for (const child of [...node.childNodes]) {
      if (child.nodeType === 8) child.parentNode?.removeChild(child)
      else walk(child)
    }
  }
  walk(root)
}

const strip = (element: Element): void => {
  const tag = element.tagName.toLowerCase()
  for (const attribute of [...element.attributes]) {
    const name = attribute.name.toLowerCase()

    if (name === 'style') {
      const said = styled(attribute.value)
      if (said === '') element.removeAttribute(attribute.name)
      else element.setAttribute('style', said)
      continue
    }

    if (!ANY.has(name) && !(OWN[tag] ?? []).includes(name)) {
      element.removeAttribute(attribute.name)
      continue
    }

    if (name === 'href' && !points(attribute.value)) element.removeAttribute(attribute.name)
    if (name === 'src' && !shows(attribute.value)) element.removeAttribute(attribute.name)
  }
}

/** One reading of the text, with everything not drawn with taken out of it. */
const pass = (html: string): string => {
  const read = new DOMParser().parseFromString(html, 'text/html')
  uncommented(read.body)

  for (const element of [...read.body.querySelectorAll('*')]) {
    const tag = element.tagName.toLowerCase()
    if (STRUCK.has(tag)) {
      element.remove()
      continue
    }
    if (!DRAWN.has(tag)) {
      element.replaceWith(...element.childNodes)
      continue
    }
    strip(element)
  }

  return read.body.innerHTML
}

/**
 * Text a card may be drawn with. It is read twice, because what one reading
 * writes out is what the next parser is handed.
 */
export const safe = (html: string): string => pass(pass(html))
