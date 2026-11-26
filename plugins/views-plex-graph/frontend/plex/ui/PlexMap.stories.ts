/**
 * The plex of a note on entropy: the map and the menu on a node. The manual and
 * the landing page take their plex images from these stories.
 */
import { getVisibleFraction } from '@numen/ui/testing'
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor, within } from 'storybook/test'
import {
  ArrowRightLeft,
  Bot,
  Copy,
  CornerDownRight,
  CornerLeftUp,
  FileText,
  FolderOpen,
  Trash2,
  Type,
  Waypoints,
} from '@lucide/vue'
import { ref } from 'vue'
import { Menu, type MenuItem } from '@numen/ui'
import Plex from './Plex.vue'
import { RELATION_NAMES } from '../lib/relation'
import type { PlexRelation, PlexEdge, PlexGraph, PlexNode } from '../../types'

const node = (id: string, title: string, relation: PlexNode['relation']): PlexNode => ({
  id,
  title,
  relation,
})

const RELATED: readonly PlexNode[] = [
  node('thermodynamics', 'Thermodynamics', 'parent'),
  node('information', 'Information theory', 'parent'),
  node('second-law', 'The second law', 'child'),
  node('boltzmann', "Boltzmann's constant", 'child'),
  node('heat-death', 'Heat death', 'child'),
  node('free-energy', 'Free energy', 'child'),
  node('demon', "Maxwell's demon", 'jump'),
  node('shannon', 'Shannon 1948', 'jump'),
  node('temperature', 'Temperature', 'sibling'),
  node('pressure', 'Pressure', 'sibling'),
]

const LABELS: Readonly<Record<string, string>> = {
  thermodynamics: 'measures',
  'second-law': 'states',
  demon: 'costs',
}

const createLine = (from: string, to: string, text: string, end: 'from' | 'to'): PlexEdge => ({
  from,
  to,
  labels: [{ text }],
  arrows: [end],
})

const createPair = (from: string, to: string, texts: readonly [string, string]): PlexEdge => ({
  from,
  to,
  labels: texts.map((text) => ({ text })),
  arrows: ['to', 'from'],
})

const createEdge = (neighbor: PlexNode): PlexEdge => {
  const label = LABELS[neighbor.id]
  if (neighbor.relation === 'sibling') return { from: 'thermodynamics', to: neighbor.id }
  if (neighbor.relation === 'parent' || neighbor.relation === 'jump') {
    return label
      ? createLine(neighbor.id, 'focus', label, 'from')
      : { from: neighbor.id, to: 'focus' }
  }
  return label ? createLine('focus', neighbor.id, label, 'to') : { from: 'focus', to: neighbor.id }
}

const createGraph = (neighbors: readonly PlexNode[]): PlexGraph => ({
  nodes: [node('focus', 'Entropy', 'focus'), ...neighbors],
  edges: neighbors.map(createEdge),
})

/** A wide graph: more nodes at the sides than in depth. */
const WIDE = createGraph([
  ...RELATED,
  node('landauer', "Landauer's principle", 'jump'),
  node('carnot', 'Carnot cycle', 'jump'),
  node('volume', 'Volume', 'sibling'),
  node('work', 'Work', 'sibling'),
])

/**
 * The menu items of a node, in display groups: open the note, file actions,
 * create from it in the plex, agent actions, and removal from the vault.
 */
const MENU: readonly MenuItem[] = [
  { id: 'app:file.open', text: 'Open note', group: 'open' },
  { id: 'travel', text: 'Show note in plex', group: 'open' },
  { id: 'app:file.copyPath', text: 'Copy note path', group: 'file' },
  { id: 'reveal', text: 'Show note in file tree', group: 'file' },
  { id: 'app:note.createChild', text: 'Create child note', group: 'plex' },
  { id: 'app:note.createParent', text: 'Create parent note', group: 'plex' },
  { id: 'app:note.createJump', text: 'Create jump note', group: 'plex' },
  { id: 'app:note.rename', text: 'Rename note', group: 'plex' },
  { id: 'ask', text: 'Ask agent about note', group: 'agent' },
  { id: 'app:file.remove', text: 'Move to trash', group: 'remove' },
]

const MENU_ICONS: Readonly<Record<string, typeof Waypoints>> = {
  read: FileText,
  travel: Waypoints,
  copy: Copy,
  reveal: FolderOpen,
  child: CornerDownRight,
  parent: CornerLeftUp,
  jump: ArrowRightLeft,
  title: Type,
  ask: Bot,
  remove: Trash2,
}

const menuIcon = (id: string) => MENU_ICONS[id]

interface MapScreen {
  readonly creatable?: readonly PlexRelation[]
  readonly menu?: { x: number; y: number } | null
}

const map = ({ creatable = [], menu = null }: MapScreen) => ({
  components: { Plex, Menu },
  setup() {
    const graph = ref<PlexGraph>(WIDE)
    return { graph, creatable, menu, MENU, menuIcon }
  },
  template: `
    <div style="height: 100vh">
      <Plex
        :graph="graph"
        :creatable="creatable"
        :duration="420"
      />
      <Menu v-if="menu" :items="MENU" :at="menu" is-open name="What can be done to this note">
        <template #icon="{ id }">
          <component
            :is="menuIcon(id)"
            v-if="menuIcon(id)"
            style="inline-size: 100%; block-size: 100%; stroke-width: 1.75; opacity: 0.75"
          />
        </template>
      </Menu>
    </div>
  `,
})

const meta = {
  title: 'Plex/Map',
  parameters: { layout: 'fullscreen' },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

/** The map with the menu open on a node. */
export const Mapping: Story = {
  tags: ['screenshot'],
  parameters: {
    screenshot: [
      { name: 'plex', app: 'docs' },
      { name: 'window-graph', app: 'landing', over: '[aria-label="The second law, child"]' },
      { name: 'map', over: '[aria-label="The second law, child"]' },
    ],
  },
  render: () =>
    map({
      creatable: RELATION_NAMES,
      // Position of a node at the trailing side, clear of the map.
      menu: { x: 902, y: 396 },
    }),
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(within(canvasElement).getAllByLabelText(/, focus$/)).toHaveLength(1))

    // The menu opens on a node and is entirely inside the viewport.
    const menu = within(document.body).getByRole('menu', {
      name: 'What can be done to this note',
    })
    await expect(await getVisibleFraction(menu)).toBe(1)
    await expect(within(menu).getAllByRole('menuitem').length).toBeGreaterThan(0)
  },
}

/**
 * Every kind of line the plex draws, each with a record pointing at one end:
 * the focus with a parent, a child and a jump; two siblings; two children. A
 * parent and a sibling have a record written by each note, one at either end of
 * their line. A line with a record and no arrowhead is the last.
 */
const EVERY_LINE: PlexGraph = {
  nodes: [
    node('focus', 'Entropy', 'focus'),
    node('thermodynamics', 'Thermodynamics', 'parent'),
    node('second-law', 'The second law', 'child'),
    node('heat-death', 'Heat death', 'child'),
    node('demon', "Maxwell's demon", 'jump'),
    node('temperature', 'Temperature', 'sibling'),
    node('pressure', 'Pressure', 'sibling'),
  ],
  edges: [
    createLine('thermodynamics', 'focus', 'measures', 'from'),
    createLine('focus', 'second-law', 'states', 'to'),
    createLine('demon', 'focus', 'costs', 'to'),
    createPair('thermodynamics', 'pressure', ['includes', 'part of']),
    createLine('pressure', 'temperature', 'drives', 'to'),
    createLine('second-law', 'heat-death', 'leads to', 'to'),
    { from: 'focus', to: 'heat-death', labels: [{ text: 'is about' }] },
  ],
}

/**
 * The same arrowhead and the same label are drawn on a line whichever notes it joins. The room around a note is
 * at its most, 40, so every line is long enough for its words.
 */
export const EveryLine: Story = {
  render: () => ({
    components: { Plex },
    setup: () => ({ graph: ref(EVERY_LINE) }),
    template:
      '<div style="height: 100vh"><Plex :graph="graph" :creatable="[]" :duration="0" :options="{ room: 40 }" /></div>',
  }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    for (const text of [
      'measures',
      'states',
      'costs',
      'includes',
      'part of',
      'drives',
      'leads to',
      'is about',
    ]) {
      await waitFor(() => expect(canvas.getAllByText(text).length).toBeGreaterThan(0))
    }
    await expect(canvasElement.querySelectorAll('.plex__edge-arrow')).toHaveLength(7)

    // The two records of one line are each read in the direction of their record.
    const getTextPath = (text: string) => {
      const href = canvas.getAllByText(text)[0]!.closest('textPath')!.getAttribute('href')!
      return canvasElement.ownerDocument.getElementById(href.slice(1)) as unknown as SVGPathElement
    }
    const one = { path: getTextPath('includes') }
    const two = { path: getTextPath('part of') }

    // The first record is read along the line from its note, the second back along it.
    const start = one.path.getPointAtLength(0)
    const end = two.path.getPointAtLength(two.path.getTotalLength())
    await expect(Math.hypot(start.x - end.x, start.y - end.y)).toBeLessThan(0.5)
  },
}
