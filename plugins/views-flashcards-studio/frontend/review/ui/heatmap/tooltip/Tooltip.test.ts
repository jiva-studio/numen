/**
 * The tooltip: announced as one, and showing what it was given to say. Where it
 * stands is computed by getAdjacentPlace, which has its own tests.
 */
import { mount } from '@vue/test-utils'
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
