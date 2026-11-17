/** The views plugins place on one side of a node are drawn at the node in the order of the manifest, each given the path of the note. */
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h, ref, type Ref } from 'vue'
import { afterEach, describe, expect, it } from 'vitest'
import PlexNodeWidgets from './PlexNodeWidgets.vue'
import type { PlacedNode } from '../../lib/node'

const node: PlacedNode = {
  id: 'one',
  title: 'Entropy',
  relation: 'child',
  x: 40,
  y: 100,
  width: 144,
  height: 36,
  order: 0,
  opacity: 1,
}

const createWidget = (name: string) =>
  defineComponent({
    props: { path: { type: String, required: true } },
    render() {
      return h('span', { 'data-widget': name }, this.path)
    },
  })

const entries = (...names: string[]) =>
  names.map((name) => ({ key: `org.example.${name}/view/${name}`, component: createWidget(name) }))

const mountWidgets = (props: Partial<InstanceType<typeof PlexNodeWidgets>['$props']> = {}) =>
  mount(PlexNodeWidgets, {
    attachTo: document.body,
    props: {
      node,
      side: 'above',
      entries: entries('tags', 'parents'),
      path: 'notes/Entropy.md',
      ...props,
    },
  })

afterEach(() => {
  document.body.replaceChildren()
})

describe('the widgets of a side of a node', () => {
  it('are drawn once each, in the order of the entries', () => {
    const widgets = mountWidgets({ entries: entries('parents', 'tags', 'related') })

    expect(
      widgets.findAll('[data-widget]').map((one) => one.attributes('data-widget')),
    ).toStrictEqual(['parents', 'tags', 'related'])
  })

  it('are each given the path of the note of the node', () => {
    const widgets = mountWidgets()

    expect(widgets.findAll('[data-widget]').map((one) => one.text())).toStrictEqual([
      'notes/Entropy.md',
      'notes/Entropy.md',
    ])
  })

  it('are given the new path when the node shows another note', async () => {
    const widgets = mountWidgets()

    await widgets.setProps({ path: 'notes/Heat.md' })

    expect(widgets.findAll('[data-widget]').map((one) => one.text())).toStrictEqual([
      'notes/Heat.md',
      'notes/Heat.md',
    ])
  })

  it('are drawn in a box as wide as the node, centered on it', () => {
    const widgets = mountWidgets({ side: 'below' })
    const box = widgets.get('foreignObject')

    expect(Number(box.attributes('width'))).toBe(node.width)
    expect(Number(box.attributes('x'))).toBe(node.x - node.width / 2)
  })

  it('are not drawn when no plugin places a view on the side', () => {
    const widgets = mountWidgets({ entries: [] })

    expect(widgets.find('foreignObject').exists()).toBe(false)
    expect(widgets.find('[data-widget]').exists()).toBe(false)
  })
  it('share one element for measuring', async () => {
    const widgets = mountWidgets({ entries: entries('parents', 'tags') })
    await flushPromises()

    const [group] = widgets.emitted('widgets')?.at(-1) ?? []

    expect(group).toBeInstanceOf(Element)
    expect((group as Element).querySelectorAll('[data-widget]')).toHaveLength(2)
  })

  it('give no element to be measured once the side has no entries', async () => {
    const widgets = mountWidgets()

    await widgets.setProps({ entries: [] })

    expect(widgets.emitted('widgets')?.at(-1)).toStrictEqual([null])
  })
})

describe('the group of the widgets of a side of a node', () => {
  const createOptional = (drawing: Ref<boolean>) =>
    defineComponent({
      props: { path: { type: String, required: true } },
      render() {
        return drawing.value ? h('span', { 'data-widget': 'tags' }, this.path) : null
      },
    })

  it('is not given to be measured while none of the widgets draws anything', () => {
    const widgets = mountWidgets({
      entries: [{ key: 'org.example.tags/view/tags', component: createOptional(ref(false)) }],
    })

    expect(widgets.emitted('widgets')?.at(-1) ?? [null]).toStrictEqual([null])
    expect(widgets.find('.plex__widgets-group').exists()).toBe(false)
  })

  it('is given to be measured once a widget starts drawing, and taken away when the last one stops', async () => {
    const drawing = ref(false)
    const widgets = mountWidgets({
      entries: [{ key: 'org.example.tags/view/tags', component: createOptional(drawing) }],
    })

    drawing.value = true
    await flushPromises()
    expect(widgets.emitted('widgets')?.at(-1)?.[0]).toBeInstanceOf(Element)

    drawing.value = false
    await flushPromises()
    expect(widgets.emitted('widgets')?.at(-1)).toStrictEqual([null])
  })
})
