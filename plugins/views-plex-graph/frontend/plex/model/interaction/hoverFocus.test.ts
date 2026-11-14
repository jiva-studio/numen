/** When a node counts as hovered or keyboard-focused, which the plex draws the same way. */
import { effectScope, nextTick } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { EXPANSION_DELAY, useExpansion } from '../sizing/expansion'
import { stubClock } from '../../../fixtures/clock'
import { useHoverFocus } from './hoverFocus'

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
  it('is on while the pointer is over it, and off once the pointer exits', () => {
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

  it('stays while focus moves to something inside, and ends when it exits', () => {
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

  it('stays while focus moves to another of its parts, and ends when it exits all of them', () => {
    const group = document.createElement('div')
    const lifted = document.createElement('div')
    const handle = document.createElement('button')
    lifted.append(handle)
    const hoverFocus = useHoverFocus(() => [group, lifted])
    const keyboardOn = vi.spyOn(group, 'matches').mockReturnValue(true)
    group.addEventListener('focusin', hoverFocus.onFocusIn)
    group.addEventListener('focusout', hoverFocus.onFocusOut)
    group.dispatchEvent(new FocusEvent('focusin'))
    expect(keyboardOn).toHaveBeenCalledWith(':focus-visible')

    group.dispatchEvent(new FocusEvent('focusout', { relatedTarget: handle }))
    expect(hoverFocus.isOn.value).toBe(true)

    // The lifted part is heard from the document, which holds everything and is not the node.
    document.addEventListener('focusout', hoverFocus.onFocusOut)
    document.dispatchEvent(new FocusEvent('focusout', { relatedTarget: document.body }))
    document.removeEventListener('focusout', hoverFocus.onFocusOut)
    expect(hoverFocus.isOn.value).toBe(false)
  })
})

describe('release', () => {
  it('takes hover off without a pointer exit', () => {
    const hoverFocus = useHoverFocus()
    hoverFocus.onPointerEnter()

    hoverFocus.release()

    expect(hoverFocus.isOn.value).toBe(false)
  })
})

describe('release during the wait before a box opens', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('leaves the box closed when the wait ends', async () => {
    vi.useFakeTimers()
    const scope = effectScope()
    const { hoverFocus, open, frames } = scope.run(() => {
      const hoverFocus = useHoverFocus()
      const frames = stubClock()
      const open = useExpansion(
        () => (hoverFocus.isOn.value ? 'node' : null),
        () => EXPANSION_DELAY,
        frames.clock,
      )
      return { hoverFocus, open, frames }
    })!

    hoverFocus.onPointerEnter()
    await nextTick()
    vi.advanceTimersByTime(EXPANSION_DELAY / 5)
    hoverFocus.release()
    await nextTick()
    vi.advanceTimersByTime(EXPANSION_DELAY * 2)
    frames.run()

    expect(hoverFocus.isOn.value).toBe(false)
    expect(open.value).toBe(0)
    scope.stop()
  })

  it('opens the box when the pointer stays through the whole wait', async () => {
    vi.useFakeTimers()
    const scope = effectScope()
    const { hoverFocus, open, frames } = scope.run(() => {
      const hoverFocus = useHoverFocus()
      const frames = stubClock()
      const open = useExpansion(
        () => (hoverFocus.isOn.value ? 'node' : null),
        () => EXPANSION_DELAY,
        frames.clock,
      )
      return { hoverFocus, open, frames }
    })!

    hoverFocus.onPointerEnter()
    await nextTick()
    vi.advanceTimersByTime(EXPANSION_DELAY)
    frames.run()

    expect(open.value).toBe(1)
    scope.stop()
  })
})
