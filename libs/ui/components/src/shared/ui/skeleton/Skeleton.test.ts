/**
 * What a placeholder is drawn as, and that it says nothing to a screen reader.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Skeleton from './Skeleton.vue'

describe('what it is drawn as', () => {
  it('is rounded to its own ends only where it is a pill', async () => {
    const skeleton = mount(Skeleton)
    expect(skeleton.classes()).not.toContain('skeleton--pill')

    await skeleton.setProps({ isPill: true })
    expect(skeleton.classes()).toContain('skeleton--pill')
  })

  it('is hidden from a screen reader, because it says a value is pending and nothing more', () => {
    expect(mount(Skeleton).attributes('aria-hidden')).toBe('true')
  })
})
