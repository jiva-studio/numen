import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import PlexNavigation from './PlexNavigation.vue'

const ICON_TIMEOUT_MS = 30_000

const getButtons = (props: { canBack: boolean; canForward: boolean }) =>
  mount(PlexNavigation, { props })

describe('PlexNavigation', () => {
  it('disables both buttons without history', () => {
    const wrapper = getButtons({ canBack: false, canForward: false })

    expect(wrapper.findAll('button').map((one) => one.attributes('disabled'))).toStrictEqual([
      '',
      '',
    ])
  })

  it('enables only the button that has a node to go to', () => {
    const wrapper = getButtons({ canBack: true, canForward: false })

    expect(wrapper.findAll('button').map((one) => one.element.disabled)).toStrictEqual([
      false,
      true,
    ])
  })

  it(
    'emits back and forward when pressed',
    async () => {
      const wrapper = getButtons({ canBack: true, canForward: true })
      await wrapper.get('[aria-label="Back"]').trigger('click')
      await wrapper.get('[aria-label="Forward"]').trigger('click')

      expect(Object.keys(wrapper.emitted())).toEqual(expect.arrayContaining(['back', 'forward']))
    },
    ICON_TIMEOUT_MS,
  )
})
