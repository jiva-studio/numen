/**
 * What the panel does with what it is given. Where the things on it are laid
 * and what happens to one taller than the panel are the browser's answer, and
 * are asked in the stories.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Panel from './Panel.vue'

describe('the panel', () => {
  it('holds what is put on it', () => {
    expect(mount(Panel, { slots: { default: '<p>said</p>' } }).text()).toContain('said')
  })

  // What a caller put on the panel is what the panel stacks, so nothing may
  // come between them.
  it('puts what it is given straight on it, in the order it was given', () => {
    const panel = mount(Panel, { slots: { default: '<p>first</p><span>second</span>' } })
    expect([...panel.element.children].map((child) => child.outerHTML)).toEqual([
      '<p>first</p>',
      '<span>second</span>',
    ])
  })

  it('is one box, and is still one with nothing on it', () => {
    const empty = mount(Panel)
    expect(empty.element.tagName).toBe('DIV')
    expect(empty.element.children).toHaveLength(0)
  })
})
