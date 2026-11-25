import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ProgressRing from './ProgressRing.vue'

describe('a progress ring', () => {
  it('turns an arc and writes no percent where the progress is not known', () => {
    const ring = mount(ProgressRing, { props: { size: 'large', label: 'Downloading' } })
    expect(ring.attributes('data-known')).toBe('false')
    expect(ring.text()).toBe('')
    expect(ring.attributes('aria-valuenow')).toBeUndefined()
    expect(ring.find('.progress-ring__arc').attributes('stroke-dasharray')).toBe('25 75')
  })

  it('is filled and writes the percent inside the large ring where it is known', () => {
    const ring = mount(ProgressRing, {
      props: { value: 0.29, size: 'large', label: 'Downloading' },
    })
    expect(ring.attributes('data-known')).toBe('true')
    expect(ring.text()).toBe('29%')
    expect(ring.attributes('aria-valuenow')).toBe('29')
    expect(ring.find('.progress-ring__arc').attributes('stroke-dasharray')).toBe('29 100')
  })

  it('writes no text in the small ring and is decorative without a label', () => {
    const ring = mount(ProgressRing, { props: { value: 0.5 } })
    expect(ring.text()).toBe('')
    expect(ring.attributes('aria-hidden')).toBe('true')
    expect(ring.attributes('role')).toBeUndefined()
  })
})
