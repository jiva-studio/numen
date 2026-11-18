/**
 * The widgets plugins place over and under the nodes of a plex: which side of a node each slot draws on, their order, and the plex with none.
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

const getStack = (canvasElement: HTMLElement, title: string): Element => {
  const stack = canvasElement.querySelector(`[aria-label^="${title}"]`)?.closest('.plex__stack')
  if (!stack) throw new Error(`no node titled ${title}`)
  return stack
}

const getSide = (stack: Element, side: 'above' | 'below'): Element => {
  const found = stack.querySelector(`.plex__widgets--${side}`)
  if (!found) throw new Error(`no widgets ${side} the node`)
  return found
}

const listWidgets = (side: Element): string[] =>
  [...side.querySelectorAll('[data-widget]')].map((one) => one.getAttribute('data-widget') ?? '')

/** The widgets of the slot above a node are drawn over it and those of the slot below under it, each of them visible. */
export const WidgetsAboveAndBelow: Story = {
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.querySelectorAll('[data-widget]')).toHaveLength(9))
    const stack = getStack(canvasElement, 'Raised beds')

    await expect(listWidgets(getSide(stack, 'above'))).toStrictEqual(['parents', 'tags'])
    await expect(listWidgets(getSide(stack, 'below'))).toStrictEqual(['children'])
    for (const widget of stack.querySelectorAll('[data-widget]')) await expect(widget).toBeVisible()
  },
}

/** The widgets of one slot appear in the order of the manifest. */
export const WidgetsOfASlotFollowTheManifest: Story = {
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.querySelectorAll('[data-widget]')).toHaveLength(9))

    for (const title of ['Kitchen garden', 'Raised beds', 'Garden']) {
      await expect(listWidgets(getSide(getStack(canvasElement, title), 'above'))).toStrictEqual([
        'parents',
        'tags',
      ])
    }
  },
}

/** Every node is given the path of its own note by each widget. */
export const EachWidgetIsGivenThePathOfItsNote: Story = {
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.querySelectorAll('[data-widget]')).toHaveLength(9))

    for (const [title, path] of [
      ['Kitchen garden', 'focus.md'],
      ['Raised beds', 'beds.md'],
      ['Garden', 'garden.md'],
    ] as const) {
      for (const widget of getStack(canvasElement, title).querySelectorAll('[data-widget]'))
        await expect(widget.getAttribute('data-path')).toBe(path)
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
