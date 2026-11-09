/** A window blur ends the whole press of a slot drag, so the next click runs. Every plugin here is invented. */
import { effectScope, type EffectScope } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Clock } from '@numen/ui'
import { SLOT_ATTRIBUTE, SLOT_ENTRY_ATTRIBUTE } from '@numen/plugin-host'
import { createSlotDrag } from './slotDrag'

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

describe('a slot drag ended by a window blur', () => {
  let scope: EffectScope
  let entry: HTMLElement
  let other: HTMLElement
  let writeLayout: ReturnType<typeof vi.fn<() => Promise<void>>>

  beforeEach(() => {
    document.body.innerHTML = `<div ${SLOT_ATTRIBUTE}="titleBar.status"><button ${SLOT_ENTRY_ATTRIBUTE}="item">Tasks</button></div><button id="other">Other</button>`
    entry = document.querySelector(`[${SLOT_ENTRY_ATTRIBUTE}]`) as HTMLElement
    other = document.getElementById('other') as HTMLElement
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

  it('lets the next click on an unrelated button run and writes nothing', () => {
    const clicks = vi.fn()
    other.addEventListener('click', clicks)

    fire(entry, 'pointerdown', { ctrlKey: true, clientX: 10, clientY: 10 })
    fire(window, 'pointermove', { clientX: 30, clientY: 10 })
    fire(window, 'blur')
    fire(window, 'pointermove', { clientX: 50, clientY: 10 })
    fire(other, 'pointerdown', { clientX: 5, clientY: 5 })
    fire(window, 'pointerup', { clientX: 5, clientY: 5 })
    fire(other, 'click')

    expect(clicks).toHaveBeenCalledTimes(1)
    expect(writeLayout).not.toHaveBeenCalled()
  })
})
