import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Panel from './Panel.vue'

describe('the panel', () => {
  it('holds what is put on it', () => {
    expect(mount(Panel, { slots: { default: '<p>said</p>' } }).text()).toContain('said')
  })

  it('is still a panel with nothing on it', () => {
    expect(mount(Panel).find('.rounded-panel').exists()).toBe(true)
  })

  it('stacks what it is given in a column', () => {
    const classes = mount(Panel).classes()
    expect(classes).toContain('flex')
    expect(classes).toContain('flex-col')
  })

  it('does not scroll, which is for whatever is put on it', () => {
    expect(mount(Panel).classes()).not.toContain('overflow-y-auto')
  })

  it('carries the tokens with it, so it paints outside a themed page', () => {
    expect(mount(Panel).classes()).toContain('numen')
  })
})
