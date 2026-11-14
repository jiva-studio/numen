/**
 * A plex tab as the plugin's own kind opens it: the graph around one note, the
 * window it is drawn in and the vault behind it made up for the story.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor, within } from 'storybook/test'
import type { Connection } from '@numen/plugin-sdk'
import { defineComponent, h, ref } from 'vue'
import PlexGraphView from './PlexGraphView.vue'
import { createPlexKind } from '../model/kind'
import { createHost } from '../fixtures/host'
import { createLine, viewOn, type RelatedNoteInput } from '../fixtures'

const createTab = (
  focus: string,
  relatedNotes: readonly RelatedNoteInput[],
  { lines = [], noteRoom }: { lines?: readonly Connection[]; noteRoom?: number } = {},
) =>
  defineComponent({
    name: 'PlexKindTab',
    setup() {
      const { host } = createHost({ opening: focus })
      const look = { ...host.look, noteRoom: ref(noteRoom ?? host.look.noteRoom.value) }
      const { kind } = createPlexKind(
        { ...host, look },
        () => viewOn(focus, relatedNotes, {}, lines).view,
        PlexGraphView,
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

/**
 * A record is drawn the same way on every line: a head at the end it points at
 * and its words near it, on a line at the focus, from a parent to a sibling,
 * and between two children. The room around a note is at its most, 40, so every
 * line is long enough for its words.
 */
export const OnEveryLine: Story = {
  render: () => ({
    components: {
      Tab: createTab(
        'Ontology.md',
        [
          { path: 'Philosophy.md', relation: 'parent', label: 'part of', marker: 'arrow' },
          { path: 'Being.md', relation: 'child', marker: 'arrow' },
          { path: 'Becoming.md', relation: 'child' },
          {
            path: 'Metaphysics.md',
            relation: 'sibling',
            through: 'Philosophy.md',
            label: 'broader',
            marker: 'arrow',
          },
        ],
        {
          lines: [
            createLine('Being.md', {
              path: 'Becoming.md',
              relation: 'child',
              label: 'precedes',
              marker: 'arrow',
            }),
          ],
          noteRoom: 40,
        },
      ),
    },
    template: '<Tab />',
  }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    for (const words of ['part of', 'broader', 'precedes']) {
      await waitFor(() => expect(canvas.getAllByText(words).length).toBeGreaterThan(0))
    }
    await expect(canvasElement.querySelectorAll('.plex__edge-arrow')).toHaveLength(4)
  },
}
