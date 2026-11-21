import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import PlexBoardSlot from './PlexBoardSlot.vue'

const control = (name: string) =>
  defineComponent({ render: () => h('button', { 'data-control': name }, name) })

describe('PlexBoardSlot', () => {
  it('draws its views in order in the place given', () => {
    const wrapper = mount(PlexBoardSlot, {
      props: {
        place: 'bottom',
        entries: [
          { key: 'a/view/one', component: control('one') },
          { key: 'b/view/two', component: control('two') },
        ],
      },
    })

    expect(wrapper.classes()).toContain('plex-board--bottom')
    expect(wrapper.findAll('[data-control]').map((one) => one.text())).toStrictEqual(['one', 'two'])
  })

  it('draws nothing without views', () => {
    const wrapper = mount(PlexBoardSlot, { props: { place: 'top', entries: [] } })

    expect(wrapper.find('.plex-board').exists()).toBe(false)
  })
})
