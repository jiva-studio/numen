/**
 * What the keyboard reaches in a story, and what it is told when it gets
 * there. Run after every story, in the browser that drew it. What is made of
 * the answer is `reach.ts`.
 */
import { userEvent } from 'vitest/browser'
import type { Stop, Walk } from './reach'

/**
 * How many stops the walk visits before it stops.
 *
 * Every Tab is a real key struck by the browser, and the four crowded stories
 * that reach this number draw the same row over and over: whatever is wrong
 * with the fortieth of them is wrong with the first.
 */
const FURTHEST = 40

/**
 * How many parts of its own a native date or time field spends Tab on before
 * it hands it on: a year, a month, a day, an hour, a minute, a second, a
 * fraction of one, a morning or afternoon, and the picker the browser draws in
 * it. It is the length of a list the HTML date and time states fix.
 */
const PARTS = 9

/** The fields the browser walks Tab through part by part. */
const PARTED =
  'input[type="time"], input[type="date"], input[type="datetime-local"], input[type="month"], input[type="week"]'

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

/** What the browser may put the keyboard on. */
const REACHABLE =
  'a[href], area[href], button, input, select, textarea, [tabindex], [contenteditable]'

/**
 * Whether the page holds somewhere else for the keyboard to go. A story with
 * one control is one the browser walks in a circle, and a control it comes back
 * to is where the walk ends and not something keeping it.
 */
const elsewhere = (from: Element): boolean =>
  [...document.querySelectorAll(REACHABLE)].some(
    (one) =>
      !from.contains(one) &&
      !one.contains(from) &&
      (one as HTMLElement).tabIndex >= 0 &&
      !one.matches(':disabled') &&
      shown(one),
  )

/** A dialog holds the keyboard for as long as it stands, which is what modal means. */
const modal = (element: Element): boolean => element.closest('[aria-modal="true"]') !== null

/**
 * Nothing moves while the census is taken, so a colour read off an element is
 * the colour it settles on and not a frame of the way there.
 */
const STILL = '*, *::before, *::after { transition: none !important; animation: none !important }'

/**
 * How long the page may go on moving before it is called restless. It bounds
 * the moving and not the waiting: still frames are taken however slowly the
 * machine draws them.
 */
const SETTLING = 500

/**
 * What Vue writes on an element for as long as it is entering or leaving. It
 * is on there a frame before the transition it describes exists, which is the
 * whole reason to read it: `getAnimations` lists what has started, and a
 * transition Vue has only just queued has not.
 */
const MOVING = '[class*="-enter-active"], [class*="-leave-active"]'

/** One drawn frame, which is when a transition the page has queued begins. */
const frame = (): Promise<void> =>
  new Promise((wake) => {
    requestAnimationFrame(() => wake())
  })

/**
 * Whether anything on the page is on its way somewhere. An animation that has
 * finished stays on the list when it fills forwards, and where it stopped is
 * where it stays.
 */
const moving = (): boolean =>
  document.getAnimations().some((one) => one.playState === 'running') ||
  document.querySelector(MOVING) !== null

/** How many still frames running say the page is done and not between halves. */
const STILLNESS = 3

/**
 * The page as it comes to rest, and whether it got there. A swap runs its two
 * halves one after the other and holds nothing moving between them, so several
 * still frames running are asked for and one proves nothing.
 */
const settles = async (): Promise<boolean> => {
  const until = performance.now() + SETTLING
  let still = 0
  while (still < STILLNESS) {
    await frame()
    if (!moving()) {
      still += 1
      continue
    }
    still = 0
    if (performance.now() >= until) return false
  }
  return true
}

/**
 * The keyboard walked through a story from the start, Tab by Tab, until it
 * comes back where it began or leaves the page, and every stop it made along
 * the way.
 */
export async function walk(): Promise<Walk> {
  const settled = await settles()

  const still = document.createElement('style')
  still.textContent = STILL
  document.head.append(still)

  document.body.setAttribute('tabindex', '-1')
  document.body.focus()

  const stops: Stop[] = []
  const seen = new Set<Element>()
  let trapped: string | null = null

  /**
   * Tab struck until the keyboard moves: the element it moved to, null once it
   * has left the page, and the element it started on where that one kept it.
   * A field with parts of its own is struck once for each of them.
   */
  const onward = async (from: Element): Promise<Element | null> => {
    const strikes = from.matches?.(PARTED) ? PARTS + 1 : 1
    for (let held = 0; held < strikes; held += 1) {
      await userEvent.tab()
      const here = document.activeElement
      if (!here || here === document.body || here === document.documentElement) return null
      if (here !== from) return here
    }
    return from
  }

  let at: Element = document.body
  for (let step = 0; step < FURTHEST; step += 1) {
    const here = await onward(at)
    if (!here) break
    if (here === at) {
      if (!modal(here) && elsewhere(here)) trapped = whereOf(here)
      break
    }
    at = here

    if (seen.has(here)) break
    seen.add(here)

    stops.push({
      where: whereOf(here),
      name: nameOf(here),
      shown: shown(here),
      // The stop itself, or whatever it is drawn inside: a row on its way in
      // carries every control standing on it.
      moving: here.closest(MOVING) !== null,
    })
  }

  document.body.focus()
  document.body.removeAttribute('tabindex')
  still.remove()
  return { stops, trapped, settled }
}
