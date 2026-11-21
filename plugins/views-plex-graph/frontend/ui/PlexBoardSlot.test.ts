import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import PlexBoardSlot from './PlexBoardSlot.vue'

const control = (name: string) =>
  defineComponent({ render: () => h('button', { 'data-control': name }, name) })

const board = { board: 'plex:1', path: 'Root.md', go: () => undefined }

describe('PlexBoardSlot', () => {
  it('draws its views in order in the place given', () => {
    const wrapper = mount(PlexBoardSlot, {
      props: {
        place: 'bottom',
        ...board,
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
    const wrapper = mount(PlexBoardSlot, { props: { place: 'top', entries: [], ...board } })

    expect(wrapper.find('.plex-board').exists()).toBe(false)
  })

  it('takes no room while none of its views draws anything', () => {
    const silent = defineComponent({ render: () => null })
    const wrapper = mount(PlexBoardSlot, {
      props: { place: 'top', entries: [{ key: 'a/view/one', component: silent }], ...board },
    })

    expect(wrapper.get('.plex-board').attributes('style')).toContain('display: none')
  })

  it('takes room once a view draws', async () => {
    const drawing = ref(false)
    const sometimes = defineComponent({ render: () => (drawing.value ? h('button', 'go') : null) })
    const wrapper = mount(PlexBoardSlot, {
      props: { place: 'top', entries: [{ key: 'a/view/one', component: sometimes }], ...board },
    })

    drawing.value = true

    await vi.waitFor(() =>
      expect(wrapper.get('.plex-board').attributes('style') ?? '').not.toContain('display: none'),
    )
  })

  it('gives each view the board key, the path of the note in front and go', () => {
    const go = () => undefined
    const probe = defineComponent({
      props: { board: String, path: String, go: Function },
      render: () => h('i', { 'data-probe': '' }),
    })
    const wrapper = mount(PlexBoardSlot, {
      props: { place: 'top', entries: [{ key: 'a/view/one', component: probe }], ...board, go },
    })

    expect(wrapper.findComponent(probe).props()).toStrictEqual({
      board: 'plex:1',
      path: 'Root.md',
      go,
    })
  })
})
