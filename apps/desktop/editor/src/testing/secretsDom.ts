/**
 * Queries over the document for the page of secrets and the secret fields of a plugin, by the
 * names the user sees: the text of a control, its accessible name or its role.
 */
import { flushPromises, type VueWrapper } from '@vue/test-utils'

/** The root a query runs in: the document body, so a menu or a dialog drawn outside the component is found. */
export const root = (): HTMLElement => document.body

/** The accessible name of an element: its label, else its title, else its text. */
export const nameOf = (element: Element): string =>
  element.getAttribute('aria-label') ??
  element.getAttribute('title') ??
  (element.textContent ?? '').replace(/\s+/g, ' ').trim()

/** Every button, or element with the role button, in a scope. */
export const buttons = (scope: ParentNode = root()): HTMLElement[] =>
  Array.from(scope.querySelectorAll<HTMLElement>('button, [role="button"]'))

/** The buttons in a scope whose accessible name matches. */
export const findButtons = (pattern: RegExp, scope: ParentNode = root()): HTMLElement[] =>
  buttons(scope).filter((one) => pattern.test(nameOf(one)))

/** The one button in a scope whose accessible name matches. */
export const findButton = (pattern: RegExp, scope: ParentNode = root()): HTMLElement => {
  const found = findButtons(pattern, scope)
  if (found.length !== 1) {
    throw new Error(
      `${found.length} buttons match ${pattern}; the buttons are: ${buttons(scope).map(nameOf).join(' | ')}`,
    )
  }
  return found[0] as HTMLElement
}

/** Every smallest element in a scope that holds the text, in document order. */
export const findTexts = (text: string | RegExp, scope: ParentNode = root()): HTMLElement[] => {
  const matches = (element: Element): boolean => {
    const own = Array.from(element.childNodes)
      .filter((node) => node.nodeType === Node.TEXT_NODE)
      .map((node) => node.textContent ?? '')
      .join('')
      .replace(/\s+/g, ' ')
      .trim()
    return typeof text === 'string' ? own === text : text.test(own)
  }
  return Array.from(scope.querySelectorAll<HTMLElement>('*')).filter(matches)
}

/** The first smallest element in a scope that holds the text. */
export const findText = (text: string | RegExp, scope: ParentNode = root()): HTMLElement | null =>
  findTexts(text, scope)[0] ?? null

/** The position of an element in the document, for the order two elements appear in. */
export const positionOf = (element: Element): number =>
  Array.from(root().querySelectorAll('*')).indexOf(element)

/** The password inputs in a scope. */
export const maskedInputs = (scope: ParentNode = root()): HTMLInputElement[] =>
  Array.from(scope.querySelectorAll<HTMLInputElement>('input[type="password"]'))

/** Types into an input and commits with Enter. */
export async function enterText(input: HTMLInputElement, text: string): Promise<void> {
  input.value = text
  input.dispatchEvent(new Event('input', { bubbles: true }))
  input.dispatchEvent(new Event('change', { bubbles: true }))
  input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
  await flushPromises()
}

/** Clicks an element and lets the component render. */
export async function click(element: Element): Promise<void> {
  element.dispatchEvent(new MouseEvent('click', { bubbles: true }))
  await flushPromises()
}

/** Unmounts a wrapper and clears what it drew outside itself. */
export function dispose(wrapper: VueWrapper): void {
  wrapper.unmount()
  document.body.innerHTML = ''
}

/** The smallest ancestor of the element that satisfies the test. */
const ancestorWhere = (
  element: Element,
  test: (candidate: HTMLElement) => boolean,
): HTMLElement => {
  for (let one = element.parentElement; one !== null; one = one.parentElement) {
    if (test(one)) return one
  }
  throw new Error(`no ancestor of <${element.tagName}> ${nameOf(element)} satisfies the test`)
}

/**
 * The row of the secret with that name: the smallest element around the name that holds a delete
 * button or a masked input. The last match is taken, as a heading of the same text comes first.
 */
export const rowOf = (name: string): HTMLElement => {
  const text = findTexts(name).at(-1)
  if (text === undefined) throw new Error(`no text ${name} on the page`)
  return ancestorWhere(
    text,
    (one) => findButtons(/delete/i, one).length > 0 || maskedInputs(one).length > 0,
  )
}

/** The section of the group with that name: the smallest element around the name that holds a button to add a secret. */
export const sectionOf = (group: string): HTMLElement => {
  const text = findText(group)
  if (text === null) throw new Error(`no text ${group} on the page`)
  return ancestorWhere(text, (one) => findButtons(/add/i, one).length > 0)
}
