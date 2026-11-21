import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import PluginSlots from './PluginSlots.vue'

const groups = [
  {
    slot: 'plexGraph.nodeBelow',
    title: 'Below a node',
    contributors: [
      { key: 'a/view/one', label: 'one', pluginName: 'Alpha' },
      { key: 'b/view/two', label: 'two', pluginName: 'Beta' },
    ],
  },
  {
    slot: 'plexGraph.nodeAbove',
    title: 'Above a node',
    contributors: [{ key: 'c/view/three', label: 'three', pluginName: 'Gamma' }],
  },
]

describe('the slots of a plugin page', () => {
  it('draw an island for each slot, with its entries and the plugin of each', () => {
    const page = mount(PluginSlots, { props: { groups, layout: undefined } })

    expect(page.findAll('section')).toHaveLength(2)
    expect(page.get('[data-testid="slots-plexGraph.nodeBelow"]').text()).toContain('Below a node')
    expect(page.text()).toContain('Alpha')
    expect(page.findAll('header')).toHaveLength(2)
  })

  it('keep a handle on every row, dimmed in a slot of one entry, and draw no arrows', () => {
    const page = mount(PluginSlots, { props: { groups, layout: undefined } })

    expect(page.findAll('[aria-label^="Reorder"]')).toHaveLength(3)
    expect(page.get('[aria-label="Reorder one"]').attributes('disabled')).toBeUndefined()
    expect(page.get('[aria-label="Reorder three"]').attributes('disabled')).toBeDefined()
    expect(page.find('[aria-label^="Move"]').exists()).toBe(false)
    expect(page.find('[aria-label="Show three"]').exists()).toBe(true)
  })

  it('store the order after a move of a grip by the keyboard', async () => {
    const page = mount(PluginSlots, { props: { groups, layout: undefined } })

    await page.get('[aria-label="Reorder one"]').trigger('keydown', { key: 'ArrowDown' })

    expect(page.emitted('change')).toStrictEqual([
      [{ 'plexGraph.nodeBelow': [{ id: 'b/view/two' }, { id: 'a/view/one' }] }],
    ])
  })

  it('store a hidden entry', async () => {
    const page = mount(PluginSlots, { props: { groups, layout: undefined } })

    await page.get('[aria-label="Show two"]').trigger('click')

    expect(page.emitted('change')).toStrictEqual([
      [{ 'plexGraph.nodeBelow': [{ id: 'a/view/one' }, { id: 'b/view/two', hidden: true }] }],
    ])
  })
})
