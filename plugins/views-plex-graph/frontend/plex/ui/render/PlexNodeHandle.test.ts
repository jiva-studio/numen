/** The handle reaches out from a press of the primary button alone. */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import PlexNodeHandle from './PlexNodeHandle.vue'

const press = (options: PointerEventInit) => {
  const handle = mount(PlexNodeHandle, { props: { at: { x: 0, y: 0 } } })
  const event = new PointerEvent('pointerdown', { bubbles: true, cancelable: true, ...options })
  handle.get('.plex__handle').element.dispatchEvent(event)
  return { handle, event }
}

describe('a press on the handle', () => {
  it('reaches out from the primary button', () => {
    const { handle } = press({ button: 0 })

    expect(handle.emitted('reach')).toHaveLength(1)
  })

  it('does not reach out from the secondary button', () => {
    expect(press({ button: 2 }).handle.emitted('reach')).toBeUndefined()
  })

  it('does not reach out from a control click, which opens a menu on macOS', () => {
    const { handle, event } = press({ button: 0, ctrlKey: true })

    expect(handle.emitted('reach')).toBeUndefined()
    expect(event.defaultPrevented).toBe(false)
  })
})
