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

/** Types into an input and commits with Enter, a change and a blur. */
export async function enterText(input: HTMLInputElement, text: string): Promise<void> {
  input.value = text
  input.dispatchEvent(new Event('input', { bubbles: true }))
  input.dispatchEvent(new Event('change', { bubbles: true }))
  input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
  await flushPromises()
}

/** Types a name into a name input and commits with Enter. */
export async function typeName(input: HTMLInputElement, name: string): Promise<void> {
  input.value = name
  input.dispatchEvent(new Event('input', { bubbles: true }))
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

/** The text inputs in a scope. */
export const textInputs = (scope: ParentNode = root()): HTMLInputElement[] =>
  Array.from(scope.querySelectorAll<HTMLInputElement>('input[type="text"]'))

/** The input a secret's name is typed into, by the name it holds, or null where there is none. */
export const findNameInput = (name: string, scope: ParentNode = root()): HTMLInputElement | null =>
  textInputs(scope).find((one) => one.value === name) ?? null

/** The row of the secret with that name: the smallest element around its name input that holds a masked input. */
export const rowOf = (name: string): HTMLElement => {
  const input = findNameInput(name)
  if (input === null) throw new Error(`no name input holding ${name} on the page`)
  return ancestorWhere(input, (one) => maskedInputs(one).length > 0)
}

/** The section of the group with that name: the smallest element around the name that holds the group actions button. */
export const sectionOf = (group: string): HTMLElement => {
  const text = findText(group)
  if (text === null) throw new Error(`no text ${group} on the page`)
  return ancestorWhere(text, (one) => findButtons(/group actions/i, one).length > 0)
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

/** Opens the group actions menu in a scope and chooses the item whose text matches. */
export async function chooseGroupAction(
  pattern: RegExp,
  scope: ParentNode = root(),
): Promise<void> {
  await click(findButton(/group actions/i, scope))
  await click(findMenuItem(pattern))
}

/** Opens the menu of a secret row and chooses the item whose text matches. */
export async function chooseRowAction(row: HTMLElement, pattern: RegExp): Promise<void> {
  await click(findButton(/secret actions/i, row))
  await click(findMenuItem(pattern))
}

/** Opens the menu of a secret row and returns its items, closing nothing. */
export async function openRowMenu(row: HTMLElement): Promise<HTMLElement[]> {
  await click(findButton(/secret actions/i, row))
  return Array.from(root().querySelectorAll<HTMLElement>('[role="menuitem"]'))
}
