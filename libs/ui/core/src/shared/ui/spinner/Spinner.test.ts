/** A spinner is decorative: the region that renders it says it is busy. */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Spinner from './Spinner.vue'

describe('a spinner', () => {
  it('is hidden from a screen reader', () => {
    const spinner = mount(Spinner)
    expect(spinner.attributes('aria-hidden')).toBe('true')
    expect(spinner.text()).toBe('')
  })

  it('keeps the class of the caller next to its own', () => {
    const spinner = mount(Spinner, { attrs: { class: 'own' } })
    expect(spinner.classes()).toEqual(expect.arrayContaining(['spinner', 'own']))
  })
})
