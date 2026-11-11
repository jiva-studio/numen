/** Where the focus is a field that takes typed letters. */
import { afterEach, describe, expect, it } from 'vitest'
import { isFieldFocused } from './isFieldFocused'

const focusOn = (html: string, selector: string): void => {
  document.body.innerHTML = html
  document.querySelector<HTMLElement>(selector)?.focus()
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('the focus in a field', () => {
  it.each([
    ['an input', '<input id="x">'],
    ['a textarea', '<textarea id="x"></textarea>'],
    ['a select', '<select id="x"><option>a</option></select>'],
    ['an editable element', '<div id="x" contenteditable="true" tabindex="0"></div>'],
    [
      'an element editable by its bare attribute',
      '<div id="x" contenteditable tabindex="0"></div>',
    ],
    [
      'the content of the note editor',
      '<div class="cm-editor"><div id="x" class="cm-content" contenteditable="true"></div></div>',
    ],
  ])('is held by %s', (_name, html) => {
    focusOn(html, '#x')

    expect(isFieldFocused()).toBe(true)
  })

  it.each([
    ['a button', '<button id="x">Go</button>'],
    ['an element that is not editable', '<div id="x" contenteditable="false" tabindex="0"></div>'],
  ])('is not held by %s', (_name, html) => {
    focusOn(html, '#x')

    expect(isFieldFocused()).toBe(false)
  })

  it('is false without focus', () => {
    document.body.innerHTML = '<input id="x">'

    expect(isFieldFocused()).toBe(false)
  })
})
