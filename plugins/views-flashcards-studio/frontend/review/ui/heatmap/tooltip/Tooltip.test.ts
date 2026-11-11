/**
 * The tooltip: announced as one, and showing what it was given to say. Where it
 * stands is computed by getAdjacentPlace, which has its own tests.
 */
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { describe, expect, it } from 'vitest'
import Tooltip from './Tooltip.vue'

describe('the tooltip itself', () => {
  it('is announced as one, and says what it was given to say', () => {
    const tooltip = mount(Tooltip, {
      props: { at: { x: 100, y: 200, width: 20, height: 20 } },
      slots: { default: 'What this is' },
    })

    expect(tooltip.findAll('[role="tooltip"]')).toHaveLength(1)
    expect(tooltip.text()).toBe('What this is')
  })
})

describe('the thing it is about moving', () => {
  it('is placed again alongside where that thing now is', async () => {
    const tooltip = mount(Tooltip, {
      props: { at: { x: 100, y: 200, width: 20, height: 20 } },
      slots: { default: 'What this is' },
    })
    const stand = () => (tooltip.get('[role="tooltip"]').element as HTMLElement).style.insetInlineStart
    await nextTick()
    const before = parseFloat(stand())

    await tooltip.setProps({ at: { x: 400, y: 200, width: 20, height: 20 } })
    await nextTick()

    expect(parseFloat(stand()) - before).toBe(300)
  })
})
