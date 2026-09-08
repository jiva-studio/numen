/**
 * The pill saying how many cards are due, and what it says while the figure is
 * still on its way.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import DueCount from './DueCount.vue'

const mountDueCount = (props: Record<string, unknown> = {}) =>
  mount(DueCount, { props: { due: 12, ...props } })

describe('a figure still on its way', () => {
  it('is read out as being counted, and carries a role to be read out by', () => {
    const held = mountDueCount({ due: null })
    const pill = held.get('.due-count')

    expect(pill.attributes('role')).toBe('status')
    expect(pill.attributes('aria-label')).toBe('still being counted')
  })

  it('gives way to the figure, which is then what is read', () => {
    const pill = mountDueCount().get('.due-count')

    expect(pill.attributes('aria-label')).toBeUndefined()
    expect(pill.text()).toBe('12 to review')
  })

  it('says the figure alone where it is asked to be bare', () => {
    expect(mountDueCount({ bare: true }).get('.due-count').text()).toBe('12')
  })
})
