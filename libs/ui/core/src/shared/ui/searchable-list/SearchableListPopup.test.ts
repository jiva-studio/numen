/**
 * What the popup adds to the list: where it is drawn, how it is put away, and
 * where the keyboard goes afterwards. It is drawn at the end of the document, so
 * everything here is read off the document.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import SearchableListPopup from './SearchableListPopup.vue'

type PopupProps = InstanceType<typeof SearchableListPopup>['$props']

let mounted: { unmount: () => void } | null = null

const mountPopup = (props: Partial<PopupProps> = {}) => {
  const popup = mount(SearchableListPopup, {
    props: {
      items: [
        { id: 'small', text: 'Small' },
        { id: 'large', text: 'Large' },
      ],
      label: 'Sizes',
      at: { x: 40, y: 60 },
      ...props,
    },
    attachTo: document.body,
  })
  mounted = popup
  return popup
}

const getFrame = () => document.body.querySelector<HTMLElement>('[data-numen-searchable-list]')

afterEach(() => {
  mounted?.unmount()
  mounted = null
})

describe('the popup', () => {
  it('is drawn at the end of the document, at its own point', async () => {
    mountPopup({ asking: 300 })
    await nextTick()
    const frame = getFrame()
    expect(frame?.parentElement).toBe(document.body)
    expect(frame?.style.left).toBe('40px')
    expect(frame?.style.top).toBe('60px')
    expect(frame?.style.getPropertyValue('--asking')).toBe('300px')
  })

  it('asks to close on Escape and on a press outside', async () => {
    const popup = mountPopup()
    await nextTick()
    getFrame()!
      .querySelector('input')!
      .dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    document.body.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    expect(popup.emitted('close')?.length).toBeGreaterThanOrEqual(2)
  })

  it('passes what is chosen on, and the item in force back', async () => {
    const popup = mountPopup()
    await nextTick()
    document.body.querySelectorAll<HTMLElement>('[role="option"]')[1]!.click()
    expect(popup.emitted('select')).toStrictEqual([['large']])
    expect(popup.emitted('update:modelValue')).toStrictEqual([['large']])
  })

  it('returns the keyboard to its from element on closing', () => {
    const asker = document.createElement('button')
    document.body.append(asker)
    const popup = mountPopup({ from: asker })
    popup.unmount()
    mounted = null
    expect(document.activeElement).toBe(asker)
    asker.remove()
  })
})
