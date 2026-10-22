/** When a node counts as hovered or keyboard-focused, which the plex draws the same way. */
import { describe, expect, it, vi } from 'vitest'
import { useHoverFocus } from './hoverFocus'

/** A real element wired to the handlers, answering `:focus-visible` as told. */
function createFocusTarget(matches: (selector: string) => boolean) {
  const hoverFocus = useHoverFocus()
  const element = document.createElement('button')
  const request: string[] = []
  vi.spyOn(element, 'matches').mockImplementation((selector: string) => {
    request.push(selector)
    return matches(selector)
  })
  element.addEventListener('focusin', hoverFocus.onFocusIn)
  const focus = () => element.dispatchEvent(new FocusEvent('focusin'))
  return { hoverFocus, element, focus, request: request }
}

describe('hover', () => {
  it('is on while the pointer is over it, and off once it leaves', () => {
    const hoverFocus = useHoverFocus()
    expect(hoverFocus.isOn.value).toBe(false)

    hoverFocus.onPointerEnter()
    expect(hoverFocus.isOn.value).toBe(true)

    hoverFocus.onPointerLeave()
    expect(hoverFocus.isOn.value).toBe(false)
  })
})

describe('keyboard focus', () => {
  it('is on where the focused element matches :focus-visible', () => {
    const { hoverFocus, focus, request } = createFocusTarget(
      (selector) => selector === ':focus-visible',
    )
    focus()

    expect(request).toStrictEqual([':focus-visible'])
    expect(hoverFocus.isOn.value).toBe(true)
  })

  it('is off where focus arrived by the pointer', () => {
    const { hoverFocus, focus } = createFocusTarget(() => false)
    focus()

    expect(hoverFocus.isOn.value).toBe(false)
  })

  it('is off where the platform does not know :focus-visible', () => {
    const { hoverFocus, focus } = createFocusTarget(() => {
      throw new SyntaxError('unsupported selector')
    })
    focus()

    expect(hoverFocus.isOn.value).toBe(false)
  })

  it('stays while focus moves to something inside, and goes when it leaves', () => {
    const { hoverFocus, element, focus } = createFocusTarget(() => true)
    const inside = document.createElement('span')
    const outside = document.createElement('span')
    element.append(inside)
    element.addEventListener('focusout', hoverFocus.onFocusOut)
    focus()

    element.dispatchEvent(new FocusEvent('focusout', { relatedTarget: inside }))
    expect(hoverFocus.isOn.value).toBe(true)

    element.dispatchEvent(new FocusEvent('focusout', { relatedTarget: outside }))
    expect(hoverFocus.isOn.value).toBe(false)
  })
})
