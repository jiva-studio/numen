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

  describe('with a separator between two groups', () => {
    const grouped = [
      {
        slot: 'titleBar.commands',
        title: 'Actions',
        contributors: [
          { key: 'a/command/one', label: 'one', pluginName: 'Alpha', group: 'navigate' },
          { key: 'b/command/two', label: 'two', pluginName: 'Beta', group: 'settings' },
          { key: 'c/command/three', label: 'three', pluginName: 'Gamma', group: 'settings' },
        ],
      },
    ]

    it('draw a divider row with no handle and no switch', () => {
      const page = mount(PluginSlots, { props: { groups: grouped, layout: undefined } })

      expect(page.findAll('li')).toHaveLength(4)
      expect(page.findAll('li hr')).toHaveLength(1)
      expect(page.findAll('[aria-label^="Reorder"]')).toHaveLength(3)
      expect(page.findAll('[aria-label^="Show"]')).toHaveLength(3)
    })

    it('store the whole arrangement when a grip moves a row across the separator', async () => {
      const page = mount(PluginSlots, { props: { groups: grouped, layout: undefined } })

      await page.get('[aria-label="Reorder two"]').trigger('keydown', { key: 'ArrowUp' })

      expect(page.emitted('change')).toStrictEqual([
        [
          {
            'titleBar.commands': [
              { id: 'a/command/one' },
              { id: 'b/command/two' },
              { id: '---' },
              { id: 'c/command/three' },
            ],
          },
        ],
      ])
    })

    it('drop the separator of a group a row leaves empty', async () => {
      const page = mount(PluginSlots, {
        props: {
          groups: grouped,
          layout: {
            'titleBar.commands': [
              { id: 'a/command/one' },
              { id: '---' },
              { id: 'b/command/two' },
              { id: 'c/command/three' },
            ],
          },
        },
      })

      await page.get('[aria-label="Reorder one"]').trigger('keydown', { key: 'ArrowDown' })

      expect(page.emitted('change')).toStrictEqual([
        [
          {
            'titleBar.commands': [
              { id: 'a/command/one' },
              { id: 'b/command/two' },
              { id: 'c/command/three' },
            ],
          },
        ],
      ])
    })
  })
})
