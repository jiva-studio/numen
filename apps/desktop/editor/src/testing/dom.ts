/**
 * Queries over the document for the page of connections and the connection rows of a plugin, by
 * the names the user sees: the text of a control, its accessible name or its role.
 */
import { flushPromises, type VueWrapper } from '@vue/test-utils'

/** The root a query runs in: the document body, so a menu or a dialog drawn outside the component is found. */
export const root = (): HTMLElement => document.body

/** The accessible name of an element: its label, else its title, else its text. */
export const nameOf = (element: Element): string =>
  element.getAttribute('aria-label') ??
  element.getAttribute('title') ??
  element.textContent.replace(/\s+/g, ' ').trim()

const buttons = (scope: ParentNode = root()): HTMLElement[] =>
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

const findTexts = (text: string | RegExp, scope: ParentNode = root()): HTMLElement[] => {
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

/** The password inputs in a scope. */
export const maskedInputs = (scope: ParentNode = root()): HTMLInputElement[] =>
  Array.from(scope.querySelectorAll<HTMLInputElement>('input[type="password"]'))

/** The single-line inputs in a scope; a masked input is not one. */
export const textInputs = (scope: ParentNode = root()): HTMLInputElement[] =>
  Array.from(scope.querySelectorAll<HTMLInputElement>('input[type="text"]'))

/** Types into an input and commits with Enter, a change and a blur. */
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

/** The menu item whose text matches, in the menu drawn into the document. */
export const findMenuItem = (pattern: RegExp): HTMLElement => {
  const items = Array.from(root().querySelectorAll<HTMLElement>('[role="menuitem"]'))
  const found = items.filter((one) => pattern.test(nameOf(one)))
  if (found.length !== 1) {
    throw new Error(
      `${found.length} menu items match ${pattern}; the items are: ${items.map(nameOf).join(' | ')}`,
    )
  }
  return found[0] as HTMLElement
}
