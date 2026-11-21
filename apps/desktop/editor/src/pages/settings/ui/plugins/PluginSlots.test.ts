import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import PluginSlots from './PluginSlots.vue'

const groups = [
  {
    slot: 'plexGraph.nodeBelow',
    title: 'Below a node',
    contributors: [
      { key: 'a/view/one', label: 'one', pluginName: 'Alpha', group: '' },
      { key: 'b/view/two', label: 'two', pluginName: 'Beta', group: '' },
    ],
  },
  { slot: 'plexGraph.node', title: 'Over a node', contributors: [] },
]

describe('the slots of a plugin page', () => {
  it('list the entries of each slot under its title', () => {
    const page = mount(PluginSlots, { props: { groups, layout: undefined } })

    expect(page.text()).toContain('Below a node')
    expect(page.text()).toContain('Alpha')
    expect(page.text()).toContain('No entries in this slot.')
  })

  it('store the order after a move', async () => {
    const page = mount(PluginSlots, { props: { groups, layout: undefined } })

    await page.get('[aria-label="Move one down"]').trigger('click')

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
