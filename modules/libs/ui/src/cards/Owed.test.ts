/**
 * The pill saying how much is waiting, and what it says while the figure is
 * still on its way.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Owed from './Owed.vue'

const mountOwed = (props: Record<string, unknown> = {}) =>
  mount(Owed, { props: { waiting: 12, ...props } })

describe('a figure still on its way', () => {
  it('is read out as being counted, and carries a role to be read out by', () => {
    const held = mountOwed({ waiting: null })
    const pill = held.get('.owed')

    expect(pill.attributes('role')).toBe('status')
    expect(pill.attributes('aria-label')).toBe('still being counted')
  })

  it('gives way to the figure, which is then what is read', () => {
    const pill = mountOwed().get('.owed')

    expect(pill.attributes('aria-label')).toBeUndefined()
    expect(pill.text()).toBe('12 to review')
  })

  it('says the figure alone where it is asked to be bare', () => {
    expect(mountOwed({ bare: true }).get('.owed').text()).toBe('12')
  })
})
