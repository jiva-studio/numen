/** How long the corner has been in a person's hand, and what that does to the moment cards are read against. */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { shallowRef } from 'vue'
import { useNoticeStack, type NoticeStackState } from './stack'

const START = 1000

let container: HTMLElement
let card: HTMLButtonElement
let sibling: HTMLButtonElement
let outside: HTMLButtonElement
let time: number
let isHidden: boolean
let state: NoticeStackState

const pointerEvent = (type: string, related: EventTarget | null = null): PointerEvent =>
  new MouseEvent(type, { bubbles: true, relatedTarget: related }) as PointerEvent

const focusEvent = (type: string, related: EventTarget | null = null): FocusEvent =>
  new FocusEvent(type, { relatedTarget: related })

/** A pointer arriving on the card, as the card's own listener hears it. */
const pointOver = (): void => {
  const event = pointerEvent('pointerover')
  card.addEventListener('pointerover', state.onPointerOver, { once: true })
  card.dispatchEvent(event)
}

const advanceTo = (to: number): void => {
  time = to
  state.beat()
}

beforeEach(() => {
  container = document.createElement('div')
  card = document.createElement('button')
  sibling = document.createElement('button')
  outside = document.createElement('button')
  container.append(card, sibling)
  document.body.append(container, outside)
  time = START
  isHidden = false
  state = useNoticeStack(
    shallowRef<HTMLElement | null>(container),
    () => time,
    () => isHidden,
  )
})

afterEach(() => {
  container.remove()
  outside.remove()
})

describe('the moment cards are read against', () => {
  it('follows the clock while nothing holds the corner', () => {
    advanceTo(1500)
    expect(state.now.value).toBe(1500)
    expect(state.read.value).toBe(1500)
  })

  it('remains still while a pointer is on the stack, and moves on once it leaves', () => {
    pointOver()
    advanceTo(2500)
    expect(state.now.value).toBe(2500)
    expect(state.read.value).toBe(START)

    state.onPointerOut(pointerEvent('pointerout', outside))
    advanceTo(3000)
    expect(state.read.value).toBe(1500)
  })

  it('remains still while the keyboard is in the stack', () => {
    card.focus()
    state.onFocusIn()
    advanceTo(2500)
    expect(state.read.value).toBe(START)
  })

  it('remains still while the window is hidden', () => {
    isHidden = true
    advanceTo(2000)
    expect(state.read.value).toBe(START)

    isHidden = false
    advanceTo(3000)
    expect(state.read.value).toBe(2000)
  })

  it('counts the time before and after a hold as read', () => {
    advanceTo(1500)
    pointOver()
    advanceTo(2500)
    state.onPointerOut(pointerEvent('pointerout'))
    advanceTo(3000)
    expect(state.read.value).toBe(2000)
  })
})

describe('a pointer on the stack', () => {
  it('keeps holding while it moves to another part of the stack', () => {
    pointOver()
    state.onPointerOut(pointerEvent('pointerout', sibling))
    advanceTo(2000)
    expect(state.read.value).toBe(START)
  })

  it('is let go when the card it is on has been taken out of the page', () => {
    pointOver()
    card.remove()
    advanceTo(2000)
    expect(state.read.value).toBe(2000)
  })

  it('holds when the card it went over cannot be named', () => {
    const event = pointerEvent('pointerover')
    card.dispatchEvent(event)
    state.onPointerOver(event)
    advanceTo(2000)
    expect(state.read.value).toBe(START)
  })
})

describe('the keyboard in the stack', () => {
  it('is not a hold until it has been reported', () => {
    card.focus()
    advanceTo(2000)
    expect(state.read.value).toBe(2000)
  })

  it('is let go when focus has moved out of the stack without being reported', () => {
    card.focus()
    state.onFocusIn()
    outside.focus()
    advanceTo(2000)
    expect(state.read.value).toBe(2000)
  })

  it('keeps holding while focus moves to another part of the stack', () => {
    card.focus()
    state.onFocusIn()
    sibling.focus()
    state.onFocusOut(focusEvent('focusout', sibling))
    advanceTo(2000)
    expect(state.read.value).toBe(START)
  })

  it('is let go when focus leaves the stack', () => {
    card.focus()
    state.onFocusIn()
    state.onFocusOut(focusEvent('focusout', outside))
    advanceTo(2000)
    expect(state.read.value).toBe(2000)
  })

  it('is let go when focus leaves for nowhere', () => {
    card.focus()
    state.onFocusIn()
    state.onFocusOut(focusEvent('focusout'))
    advanceTo(2000)
    expect(state.read.value).toBe(2000)
  })

  it('keeps holding when focus is reported moving to another part of the stack', () => {
    card.focus()
    state.onFocusIn()
    state.onFocusOut(focusEvent('focusout', sibling))
    advanceTo(2000)
    expect(state.read.value).toBe(START)
  })
})

describe('a stack that is not on the page', () => {
  beforeEach(() => {
    state = useNoticeStack(
      shallowRef<HTMLElement | null>(null),
      () => time,
      () => isHidden,
    )
  })

  it('lets go of the keyboard whenever it is requested', () => {
    state.onFocusIn()
    advanceTo(2000)
    expect(state.read.value).toBe(2000)
  })

  it('lets go of the keyboard and the pointer when they leave for an element', () => {
    state.onFocusIn()
    state.onFocusOut(focusEvent('focusout', outside))
    pointOver()
    state.onPointerOut(pointerEvent('pointerout', outside))
    advanceTo(2000)
    expect(state.read.value).toBe(2000)
  })
})
