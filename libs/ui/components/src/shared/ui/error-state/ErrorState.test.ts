/**
 * What a pane that cannot show its content says: the message, under a heading when there is one.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ErrorState from './ErrorState.vue'

describe('the error state of a pane', () => {
  it('says its message as an alert', () => {
    const view = mount(ErrorState, { props: { message: 'There is no note at Reef.md' } })

    expect(view.attributes('role')).toBe('alert')
    expect(view.text()).toBe('There is no note at Reef.md')
    expect(view.find('svg').attributes('aria-hidden')).toBe('true')
  })

  it('draws the heading only where one is given', () => {
    const plain = mount(ErrorState, { props: { message: 'm' } })
    const titled = mount(ErrorState, { props: { message: 'm', heading: 'Note missing' } })

    expect(plain.find('h3').exists()).toBe(false)
    expect(titled.get('h3').text()).toBe('Note missing')
  })

  it('draws what it is given below the message', () => {
    const view = mount(ErrorState, {
      props: { message: 'm' },
      slots: { default: '<button>Retry</button>' },
    })

    expect(view.get('button').text()).toBe('Retry')
  })
})
