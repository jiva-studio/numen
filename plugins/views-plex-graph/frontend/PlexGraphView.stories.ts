/**
 * A plex tab as the plugin's own kind opens it: the graph around one note, the
 * window it is drawn in and the vault behind it made up for the story.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor, within } from 'storybook/test'
import { defineComponent, h } from 'vue'
import PlexGraphView from './PlexGraphView.vue'
import { createPlexKind } from './kind'
import { createHost } from './fixtures/host'
import { viewOn, type NeighbourSpec } from './fixtures'

/** A plex tab opened by the kind on a note and the notes joined to it. */
const createTab = (focus: string, neighbours: readonly NeighbourSpec[]) =>
  defineComponent({
    name: 'PlexKindTab',
    setup() {
      const { kind } = createPlexKind(
        createHost({ opening: focus }).host,
        () => viewOn(focus, neighbours).view,
      )
      const state = kind.open!(focus, 'plex:story')
      return () =>
        h('div', { style: { width: '100vw', height: '100vh' } }, [h(PlexGraphView, { state })])
    },
  })

const meta = {
  title: 'Plex/Tab',
  parameters: { layout: 'fullscreen' },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

/** A note with a parent, two children and a jump, each drawn as a node of its own. */
export const AroundANote: Story = {
  render: () => ({
    components: {
      Tab: createTab('Ontology.md', [
        { path: 'Philosophy.md', relation: 'parent' },
        { path: 'Being.md', relation: 'child' },
        { path: 'Becoming.md', relation: 'child' },
        { path: 'Logic.md', relation: 'jump' },
      ]),
    },
    template: '<Tab />',
  }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    for (const title of ['Ontology', 'Philosophy', 'Being', 'Becoming', 'Logic']) {
      await waitFor(() => expect(canvas.getAllByText(title).length).toBeGreaterThan(0))
    }
  },
}

/** Arrows are drawn where the records carry a marker: here a head at one end and at both ends. */
export const WithArrows: Story = {
  render: () => ({
    components: {
      Tab: createTab('Ontology.md', [
        { path: 'Being.md', relation: 'child', marker: 'arrow' },
        { path: 'Logic.md', relation: 'jump', marker: 'arrow', reverseMarker: 'arrow' },
        { path: 'Ethics.md', relation: 'jump' },
      ]),
    },
    template: '<Tab />',
  }),
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.querySelectorAll('.plex__edge-arrow')).toHaveLength(3))
  },
}

/** A note joined to nothing: the plex draws it alone. */
export const Alone: Story = {
  render: () => ({ components: { Tab: createTab('Solitude.md', []) }, template: '<Tab />' }),
  play: async ({ canvasElement }) => {
    await waitFor(() =>
      expect(within(canvasElement).getAllByText('Solitude').length).toBeGreaterThan(0),
    )
  },
}
