/** How a slot drag ends when the release never arrives. Every plugin here is invented. */
import { effectScope, type EffectScope } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Clock } from '@numen/ui'
import { SLOT_ATTRIBUTE, SLOT_ENTRY_ATTRIBUTE } from '@numen/plugin-host'
import { createSlotDrag, DRAGGING_ATTRIBUTE } from './slotDrag'

const KEY = 'titleBar.status|org.example.tide/status'
const immediateClock: Clock = {
  now: () => 0,
  schedule: (run) => {
    run(0)
    return 0
  },
  cancel: () => undefined,
}

const fire = (target: EventTarget, type: string, init: MouseEventInit = {}): void => {
  target.dispatchEvent(new MouseEvent(type, { bubbles: true, cancelable: true, ...init }))
}

describe('a slot drag whose release never arrives', () => {
  let scope: EffectScope
  let entry: HTMLElement
  let writeLayout: ReturnType<typeof vi.fn<() => Promise<void>>>
  const isDragging = (): boolean => document.documentElement.hasAttribute(DRAGGING_ATTRIBUTE)
  const press = (): void => fire(entry, 'pointerdown', { ctrlKey: true, clientX: 10, clientY: 10 })
  const move = (): void => fire(window, 'pointermove', { clientX: 30, clientY: 10 })
  const release = (): void => fire(window, 'pointerup', { clientX: 30, clientY: 10 })

  beforeEach(() => {
    document.body.innerHTML = `<div ${SLOT_ATTRIBUTE}="titleBar.status"><button ${SLOT_ENTRY_ATTRIBUTE}="item">Tasks</button></div>`
    entry = document.querySelector('button') as HTMLElement
    document.elementFromPoint = () => entry.parentElement
    writeLayout = vi.fn<() => Promise<void>>().mockResolvedValue(undefined)
    scope = effectScope()
    scope.run(() =>
      createSlotDrag({
        isMac: () => false,
        findLayoutKey: () => KEY,
        listLayoutKeys: () => [KEY],
        getLayout: () => ({}),
        writeLayout,
        getClock: () => immediateClock,
      }),
    )
  })

  afterEach(() => {
    scope.stop()
    document.body.innerHTML = ''
  })

  it('ends when a new press begins, and a release without movement leaves the page as it was', () => {
    press()
    move()
    expect(isDragging()).toBe(true)

    press()
    expect(isDragging()).toBe(false)
    release()

    expect(isDragging()).toBe(false)
    expect(writeLayout).not.toHaveBeenCalled()
  })

  it('is cancelled when the window loses focus, and the release after it writes nothing', () => {
    press()
    move()
    expect(isDragging()).toBe(true)

    fire(window, 'blur')
    expect(isDragging()).toBe(false)
    release()

    expect(isDragging()).toBe(false)
    expect(writeLayout).not.toHaveBeenCalled()
  })
})
