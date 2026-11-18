import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { describe, expect, it } from 'vitest'
import type { Viewport } from '@/shared/api/viewport'
import Tooltip from './Tooltip.vue'

const mountTooltip = (slot: string, viewport?: Viewport) => {
  return mount(Tooltip, {
    props: { at: { x: 100, y: 200, width: 20, height: 20 }, ...(viewport ? { viewport } : {}) },
    slots: { default: slot },
  })
}

describe('the tooltip of the library', () => {
  it('is announced as one, and says what it was given to say', () => {
    const tooltip = mountTooltip('What this is')

    expect(tooltip.findAll('[role="tooltip"]')).toHaveLength(1)
    expect(tooltip.text()).toBe('What this is')
  })

  it('says the tags it was given, in the order it was given them', () => {
    const tooltip = mountTooltip('Port, Sea, Quay')

    expect(tooltip.text()).toBe('Port, Sea, Quay')
  })

  it('is no wider than the free width of the viewport less a margin on each side', async () => {
    const viewport: Viewport = {
      watch: () => () => undefined,
      watchWhole: () => () => undefined,
      watchFreeSpace: (took) => {
        took({ width: 300, height: 200 })
        return () => undefined
      },
    }

    const tooltip = mountTooltip('Port, Sea, Quay', viewport)
    await nextTick()

    expect(tooltip.get<HTMLElement>('[role="tooltip"]').element.style.maxInlineSize).toBe('284px')
  })
})
