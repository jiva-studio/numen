/**
 * What the keyboard reaches in a story, and what it is told when it gets
 * there. Run after every story, in the browser that drew it. What is made of
 * the answer is `reach.ts`.
 */
import { userEvent } from 'vitest/browser'
import type { Stop, Walk } from './reach'

/** How far the walk goes before it calls the tab order a ring it cannot leave. */
const FURTHEST = 120

/** What is compared before the keyboard arrives and once it is there. */
const PAINTED = [
  'outline-style',
  'outline-width',
  'outline-color',
  'outline-offset',
  'box-shadow',
  'background-color',
  'background-image',
  'border-top-color',
  'border-top-width',
  'border-top-style',
  'color',
  'opacity',
  'text-decoration-line',
  'filter',
  // What a shape drawn in SVG rings with, where there is no box to outline.
  'stroke',
  'stroke-width',
  'stroke-dasharray',
  'stroke-opacity',
  'fill',
  'fill-opacity',
] as const

/** The stop, what draws it and what it is drawn inside: any of them may ring. */
const around = (element: Element): Element[] => {
  const out: Element[] = [element, ...element.querySelectorAll('*')]
  let one = element.parentElement
  for (let up = 0; one && up < 4; up += 1, one = one.parentElement) out.push(one)
  return out
}

/** A ring is as often drawn on a box that is not there as on the element. */
const FACES = [null, '::before', '::after'] as const

const painting = (element: Element): string => {
  let out = ''
  for (const face of FACES) {
    const style = getComputedStyle(element, face)
    for (const name of PAINTED) out += `${style.getPropertyValue(name)}|`
    out += style.content + '|'
  }
  return out
}

/** The roles whose name never comes from what stands inside them. */
const HELD = new Set(['textbox', 'combobox', 'searchbox', 'spinbutton', 'slider'])

/** What a screen reader would call this, as far as the DOM alone can say. */
export const nameOf = (element: Element): string => {
  const label = element.getAttribute('aria-label')
  if (label?.trim()) return label.trim()
  const by = element.getAttribute('aria-labelledby')
  if (by) {
    const said = by
      .split(/\s+/)
      .map((id) => document.getElementById(id)?.textContent?.trim() ?? '')
      .join(' ')
      .trim()
    if (said) return said
  }
  if (element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement) {
    const id = element.getAttribute('id')
    const tied = id ? document.querySelector(`label[for="${CSS.escape(id)}"]`) : null
    if (tied?.textContent?.trim()) return tied.textContent.trim()
    const wrapping = element.closest('label')?.textContent?.trim()
    if (wrapping) return wrapping
    const holder = element.getAttribute('placeholder')
    if (holder?.trim()) return `(placeholder) ${holder.trim()}`
    const title = element.getAttribute('title')
    if (title?.trim()) return `(title) ${title.trim()}`
    return ''
  }
  // A box typed into is never named by what is in it: what a person has
  // written is the value, and the name has to come from outside.
  const role = element.getAttribute('role') ?? ''
  const text = (element.textContent ?? '').replace(/\s+/g, ' ').trim()
  if (text && !HELD.has(role)) return text
  const title = element.getAttribute('title')
  if (title?.trim()) return `(title) ${title.trim()}`
  const alt = element.querySelector('img[alt], [aria-label]')?.getAttribute('aria-label')
  if (alt?.trim()) return alt.trim()
  return ''
}

/** Where this stop stands, said so a person can find it in the source. */
export const whereOf = (element: Element): string => {
  const tag = element.tagName.toLowerCase()
  const role = element.getAttribute('role')
  const cls = (element.getAttribute('class') ?? '')
    .split(/\s+/)
    .filter((one) => one.includes('__'))
    .slice(0, 2)
    .join('.')
  return `${tag}${role ? `[role=${role}]` : ''}${cls ? `.${cls}` : ''}`
}

/** Whether the browser draws this where a person could see it. */
const shown = (element: Element): boolean => {
  if (element.getClientRects().length === 0) return false
  const style = getComputedStyle(element)
  return style.visibility !== 'hidden' && Number(style.opacity) > 0.01
}

/** What is typed into, and shows where the keyboard is with its own caret. */
const TYPED = 'input, textarea, [contenteditable], [role="textbox"], [role="combobox"]'

/**
 * Nothing moves while the census is taken, so a colour read off an element is
 * the colour it settles on and not a frame of the way there.
 */
const STILL = '*, *::before, *::after { transition: none !important; animation: none !important }'

/**
 * The keyboard walked through a story from the start, Tab by Tab, until it
 * comes back where it began or leaves the page; then every stop is looked at
 * twice, with the keyboard away from it and with the keyboard on it. The walk
 * itself is what puts the browser in its keyboard temper, so a ring drawn only
 * for `:focus-visible` is a difference these two readings can see.
 */
export async function walk(): Promise<Walk> {
  const still = document.createElement('style')
  still.textContent = STILL
  document.head.append(still)

  document.body.setAttribute('tabindex', '-1')
  document.body.focus()

  // What everything on the page is painted as with the keyboard nowhere near
  // it. A ring is a difference from this, read the moment the keyboard lands.
  const away = new Map<Element, string>()
  for (const one of document.body.querySelectorAll('*')) away.set(one, painting(one))

  /** Each stop as it stood with the keyboard on it, and what was painted then. */
  const reached: { stop: Omit<Stop, 'rings'>; drawn: [Element, string][] }[] = []
  const seen = new Set<Element>()
  let trapped: string | null = null
  let last: Element | null = null
  let stuck = 0

  for (let step = 0; step < FURTHEST; step += 1) {
    await userEvent.tab()
    const here = document.activeElement
    if (!here || here === document.body || here === document.documentElement) break

    if (here === last) {
      stuck += 1
      if (stuck >= 2) {
        trapped = whereOf(here)
        break
      }
      continue
    }
    stuck = 0
    last = here

    if (seen.has(here)) break
    seen.add(here)

    reached.push({
      stop: {
        where: whereOf(here),
        name: nameOf(here),
        shown: shown(here),
        typed: here.matches(TYPED),
      },
      drawn: around(here).map((one) => [one, painting(one)] as [Element, string]),
    })
  }

  document.body.focus()

  // A ring is anything painted differently while the keyboard stood there.
  // What the page held before the walk is the reading it is compared against;
  // what only appeared once the keyboard arrived is compared against the page
  // as it stands now, and what went with the keyboard is a ring in itself.
  const stops = reached.map(({ stop, drawn }) => ({
    ...stop,
    rings: drawn.some(([one, held]) => {
      const before = away.get(one)
      if (before !== undefined) return before !== held
      return one.isConnected ? painting(one) !== held : true
    }),
  }))

  document.body.removeAttribute('tabindex')
  still.remove()
  return { stops, trapped }
}
