/**
 * The widgets plugins place over and under the nodes of a plex: their place against the node, how a pair shares a line, and the plex with none.
 * Every graph and every widget here is made up.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor } from 'storybook/test'
import { defineComponent, h } from 'vue'
import Plex from './Plex.vue'
import type { PlexGraph } from '../../types'

const graph: PlexGraph = {
  nodes: [
    { id: 'focus', title: 'Kitchen garden', relation: 'focus' },
    { id: 'beds', title: 'Raised beds', relation: 'child' },
    { id: 'garden', title: 'Garden', relation: 'parent' },
  ],
  edges: [
    { from: 'garden', to: 'focus' },
    { from: 'focus', to: 'beds' },
  ],
}

const createWidget = (name: string, text: string) =>
  defineComponent({
    props: { path: { type: String, required: true } },
    render() {
      return h(
        'span',
        {
          'data-widget': name,
          'data-path': this.path,
          style:
            'display:inline-block;padding:0.125rem 0.25rem;margin:0.0625rem;border:1px solid currentColor',
        },
        text,
      )
    },
  })

const entry = (name: string, text: string) => ({
  key: `org.example.${name}/view/${name}`,
  component: createWidget(name, text),
})

interface Knobs {
  hasWidgets: boolean
}

const getPathOf = (id: string) => `${id}.md`

const meta = {
  title: 'Plex/Widgets',
  component: Plex,
  parameters: { layout: 'fullscreen' },
  args: { hasWidgets: true },
  render: (args: Knobs) => ({
    components: { Plex },
    setup: () => ({
      args,
      graph,
      getPathOf,
      above: [entry('parents', 'Up'), entry('tags', 'Tag')],
      below: [entry('children', 'Down')],
    }),
    template: `
      <div style="height:100vh">
        <Plex
          :graph="graph"
          :duration="0"
          :get-path="getPathOf"
          :widgets-above="args.hasWidgets ? above : []"
          :widgets-below="args.hasWidgets ? below : []"
        />
      </div>`,
  }),
} satisfies Meta<Knobs>

export default meta
type Story = StoryObj<Knobs>

const getBox = (canvasElement: HTMLElement, title: string): DOMRect => {
  const node = canvasElement.querySelector(`[aria-label^="${title}"]`)
  const box = node?.querySelector('.plex__box') ?? node
  if (!box) throw new Error(`no node titled ${title}`)
  return box.getBoundingClientRect()
}

const getWidgets = (canvasElement: HTMLElement, name: string, path: string): DOMRect => {
  const found = [...canvasElement.querySelectorAll(`[data-widget="${name}"]`)].find(
    (one) => one.getAttribute('data-path') === path,
  )
  if (!found) throw new Error(`no widget ${name} for ${path}`)
  return found.getBoundingClientRect()
}

/** Widgets of the manifest sit over a note and under it, each centered with its group on the node. */
export const WidgetsAboveAndBelow: Story = {
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.querySelectorAll('[data-widget]')).toHaveLength(9))
    const node = getBox(canvasElement, 'Raised beds')
    const parents = getWidgets(canvasElement, 'parents', 'beds.md')
    const tags = getWidgets(canvasElement, 'tags', 'beds.md')
    const children = getWidgets(canvasElement, 'children', 'beds.md')

    await expect(parents.bottom).toBeLessThanOrEqual(node.top + 1)
    await expect(tags.bottom).toBeLessThanOrEqual(node.top + 1)
    await expect(children.top).toBeGreaterThanOrEqual(node.bottom - 1)

    const groupCenter =
      (Math.min(parents.left, tags.left) + Math.max(parents.right, tags.right)) / 2
    await expect(Math.abs(groupCenter - (node.left + node.right) / 2)).toBeLessThan(2)
    await expect(
      Math.abs((children.left + children.right) / 2 - (node.left + node.right) / 2),
    ).toBeLessThan(2)
  },
}

/** Two widgets that fit on one line sit side by side, in the order of the manifest. */
export const AnArrangedPairSharesALine: Story = {
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.querySelectorAll('[data-widget]')).toHaveLength(9))
    const parents = getWidgets(canvasElement, 'parents', 'beds.md')
    const tags = getWidgets(canvasElement, 'tags', 'beds.md')

    await expect(Math.abs(parents.top - tags.top)).toBeLessThan(2)
    await expect(parents.right).toBeLessThanOrEqual(tags.left + 1)
  },
}

/** The room of the widgets keeps the other nodes out: no widget of a node covers another node. */
export const WidgetsKeepClearOfOtherNodes: Story = {
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.querySelectorAll('[data-widget]')).toHaveLength(9))
    const widgets = getWidgets(canvasElement, 'children', 'garden.md')

    for (const title of ['Kitchen garden', 'Raised beds']) {
      const other = getBox(canvasElement, title)
      const isOverlapping =
        widgets.left < other.right &&
        other.left < widgets.right &&
        widgets.top < other.bottom &&
        other.top < widgets.bottom
      await expect(isOverlapping).toBe(false)
    }
  },
}

/** A plex that no plugin places widgets in draws the nodes alone. */
export const WithNoWidgets: Story = {
  args: { hasWidgets: false },
  play: async ({ canvasElement }) => {
    await waitFor(() =>
      expect(canvasElement.querySelector('[aria-label^="Raised beds"]')).not.toBeNull(),
    )

    await expect(canvasElement.querySelectorAll('[data-widget]')).toHaveLength(0)
  },
}
