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

    it('draw a divider row with a handle and a remove button but no switch', () => {
      const page = mount(PluginSlots, { props: { groups: grouped, layout: undefined } })

      expect(page.findAll('li')).toHaveLength(4)
      expect(page.findAll('li hr')).toHaveLength(1)
      expect(page.findAll('[aria-label^="Reorder"]')).toHaveLength(4)
      expect(page.findAll('[aria-label="Reorder separator"]')).toHaveLength(1)
      expect(page.findAll('[aria-label="Remove separator"]')).toHaveLength(1)
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

    it('keep the separator of a group a row leaves empty, as the first row', async () => {
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
              { id: '---' },
              { id: 'a/command/one' },
              { id: 'b/command/two' },
              { id: 'c/command/three' },
            ],
          },
        ],
      ])
    })

    it('store a move of the separator by its grip', async () => {
      const page = mount(PluginSlots, { props: { groups: grouped, layout: undefined } })

      await page.get('[aria-label="Reorder separator"]').trigger('keydown', { key: 'ArrowDown' })

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

    it('store the list without the separator its button removes', async () => {
      const page = mount(PluginSlots, { props: { groups: grouped, layout: undefined } })

      await page.get('[aria-label="Remove separator"]').trigger('click')

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

  it('add a separator after the first row from the heading of the island', async () => {
    const page = mount(PluginSlots, { props: { groups, layout: undefined } })

    await page
      .get('[data-testid="slots-plexGraph.nodeBelow"] [aria-label="Add separator"]')
      .trigger('click')

    expect(page.emitted('change')).toStrictEqual([
      [{ 'plexGraph.nodeBelow': [{ id: 'a/view/one' }, { id: '---' }, { id: 'b/view/two' }] }],
    ])
  })

  it('offer Reset only on a slot with a stored list, and drop the key of that slot', async () => {
    const layout = { 'plexGraph.nodeBelow': [{ id: 'b/view/two' }], other: [{ id: 'x' }] }
    const page = mount(PluginSlots, { props: { groups, layout } })

    expect(page.findAll('[aria-label^="Reset"]')).toHaveLength(1)
    await page.get('[aria-label="Reset Below a node to its default"]').trigger('click')

    expect(page.emitted('change')).toStrictEqual([[{ other: [{ id: 'x' }] }]])
  })

  it('offer no Reset while nothing is stored', () => {
    const page = mount(PluginSlots, { props: { groups, layout: undefined } })

    expect(page.find('[aria-label^="Reset"]').exists()).toBe(false)
  })
})

describe('the slots of a plugin page while a row is held', () => {
  const press = (page: ReturnType<typeof mount>) => {
    const grip = page.get('[aria-label="Reorder one"]').element
    grip.dispatchEvent(
      new PointerEvent('pointerdown', { bubbles: true, button: 0, clientY: 10, pointerId: 1 }),
    )
    for (const y of [20, 30])
      window.dispatchEvent(
        new PointerEvent('pointermove', { bubbles: true, clientY: y, pointerId: 1 }),
      )
  }

  it('keep the row held and the others shifting when the groups are read again equal', async () => {
    const page = mount(PluginSlots, {
      props: { groups, layout: undefined },
      attachTo: document.body,
    })
    press(page)
    await page.vm.$nextTick()
    const held = (): boolean =>
      page
        .get('[data-testid="slots-plexGraph.nodeBelow"] li')
        .classes()
        .includes('plugin-slots__row--lifted')
    expect(held()).toBe(true)

    await page.setProps({ groups: structuredClone(groups) })

    expect(held()).toBe(true)
    expect(page.find('.plugin-slots__list--pressed').exists()).toBe(true)
    page.unmount()
  })

  it('create no row again when the groups are read again equal', async () => {
    let mounted = 0
    const page = mount(PluginSlots, {
      props: { groups, layout: undefined },
      global: {
        mixins: [
          {
            mounted() {
              mounted += 1
            },
          },
        ],
      },
    })
    const before = mounted

    await page.setProps({ groups: structuredClone(groups), layout: { x: [] } })

    expect(mounted).toBe(before)
  })
})
