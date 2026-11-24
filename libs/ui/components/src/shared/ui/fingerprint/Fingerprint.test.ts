import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Fingerprint from './Fingerprint.vue'

const VALUE = 'F9D2 40ED 59AF 4B0D 55BB 09D1 D750 C8AD'

describe('a fingerprint', () => {
  it('renders the whole value', () => {
    expect(mount(Fingerprint, { props: { value: VALUE } }).text()).toBe(VALUE)
  })

  it('keeps the last two groups in a part of their own', () => {
    const tail = mount(Fingerprint, { props: { value: VALUE } }).find('.fingerprint__tail')
    expect(tail.text()).toBe('D750 C8AD')
  })

  it('holds the whole value in the title and the label', () => {
    const wrapper = mount(Fingerprint, { props: { value: VALUE } })
    expect(wrapper.attributes('title')).toBe(VALUE)
    expect(wrapper.attributes('aria-label')).toBe(VALUE)
  })
})
