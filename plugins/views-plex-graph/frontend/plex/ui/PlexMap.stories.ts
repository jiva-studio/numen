/**
 * The plex of a note on entropy: the map, the menu on a node, and the
 * headings of a note shown under its box. The manual and the landing page take
 * their plex images from these stories.
 */
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
import { hoverOver } from '../../fixtures/colour'
import Plex from './Plex.vue'
import { RELATION_NAMES, type PlexRelation } from '../lib/relation'
import type { PlexEdge } from '../lib/edge'
import type { PlexHeading } from '../lib/inside'
import type { PlexNeighbourhood } from '../lib/neighbourhood'
import type { PlexNode } from '../lib/node'

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

/** A sibling hangs off the parent it shares; everything else meets the focus. */
const createEdge = (neighbour: PlexNode): PlexEdge => {
  const label = LABELS[neighbour.id]
  if (neighbour.relation === 'sibling') return { from: 'thermodynamics', to: neighbour.id }
  if (neighbour.relation === 'parent' || neighbour.relation === 'jump') {
    return { from: neighbour.id, to: 'focus', ...(label ? { label } : {}) }
  }
  return { from: 'focus', to: neighbour.id, ...(label ? { label } : {}) }
}

const createNeighbourhood = (neighbours: readonly PlexNode[]): PlexNeighbourhood => ({
  nodes: [node('focus', 'Entropy', 'focus'), ...neighbours],
  edges: neighbours.map(createEdge),
})

/** A wide neighbourhood: more nodes at the sides than in depth. */
const WIDE = createNeighbourhood([
  ...RELATED,
  node('landauer', "Landauer's principle", 'jump'),
  node('carnot', 'Carnot cycle', 'jump'),
  node('volume', 'Volume', 'sibling'),
  node('work', 'Work', 'sibling'),
])

/**
 * The headings of two notes. The focus note has more headings than fit at once,
 * so the overflow mark is drawn as well.
 */
const HEADINGS: Readonly<Record<string, readonly PlexHeading[]>> = {
  focus: [
    { id: '4', text: 'What it measures', level: 2 },
    { id: '19', text: "Boltzmann's formula", level: 3 },
    { id: '38', text: 'Gibbs and the ensemble', level: 3 },
    { id: '57', text: 'The arrow of time', level: 2 },
    { id: '76', text: 'Missing information', level: 2 },
    { id: '94', text: "Shannon's H", level: 3 },
    { id: '118', text: 'The demon, and its bookkeeping', level: 2 },
  ],
  'second-law': [
    { id: '3', text: 'As Clausius put it', level: 2 },
    { id: '21', text: 'As Kelvin put it', level: 2 },
    { id: '44', text: 'Why the two are one', level: 2 },
  ],
}

/**
 * The menu items of a node, in display groups: open the note, file actions,
 * create from it in the plex, agent actions, and removal from the vault.
 */
const MENU: readonly MenuItem[] = [
  { id: 'read', text: 'Open note', group: 'open' },
  { id: 'travel', text: 'Show in plex', group: 'open' },
  { id: 'copy', text: 'Copy path', group: 'file' },
  { id: 'reveal', text: 'Show in files', group: 'file' },
  { id: 'child', text: 'New child note', group: 'plex' },
  { id: 'parent', text: 'New parent note', group: 'plex' },
  { id: 'jump', text: 'New jump note', group: 'plex' },
  { id: 'title', text: 'Change title', group: 'plex' },
  { id: 'ask', text: 'Ask the agent about this note', group: 'agent' },
  { id: 'remove', text: 'Remove note', group: 'remove' },
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
  /** The headings a node hangs under its box, and none where there are none. */
  readonly headings?: (id: string) => readonly PlexHeading[]
  /** The relations a new note may be made in. */
  readonly creatable?: readonly PlexRelation[]
  /** The position of the node menu, or null when no menu is open. */
  readonly menu?: { x: number; y: number } | null
}

/** The map around the note on entropy, with the whole window to itself. */
const map = ({ headings = () => [], creatable = [], menu = null }: MapScreen) => ({
  components: { Plex, Menu },
  setup() {
    const neighbourhood = ref<PlexNeighbourhood>(WIDE)
    return { neighbourhood, headings, creatable, menu, MENU, menuIcon }
  },
  template: `
    <div style="height: 100vh">
      <Plex
        :neighbourhood="neighbourhood"
        :creatable="creatable"
        :duration="420"
        :headings="headings"
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
  render: () =>
    map({
      headings: (id: string) => HEADINGS[id] ?? [],
      creatable: RELATION_NAMES,
      // Position of a node at the trailing side, clear of the map and of the
      // headings shown under a node.
      menu: { x: 902, y: 396 },
    }),
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(within(canvasElement).getAllByLabelText(/, focus$/)).toHaveLength(1))

    // The menu opens on a node and is entirely inside the viewport.
    const menu = within(document.body).getByRole('menu', {
      name: 'What can be done to this note',
    })
    const box = menu.getBoundingClientRect()
    await expect(box.width).toBeGreaterThan(0)
    await expect(box.left).toBeGreaterThanOrEqual(0)
    await expect(box.right).toBeLessThanOrEqual(window.innerWidth + 1)
    await expect(box.top).toBeGreaterThanOrEqual(0)
    await expect(box.bottom).toBeLessThanOrEqual(window.innerHeight + 1)
    await expect(within(menu).getAllByRole('menuitem').length).toBeGreaterThan(0)
  },
}

/** The pointer on the focus node, with the headings of its note shown under the box. */
export const Nesting: Story = {
  render: () =>
    map({
      headings: (id: string) => HEADINGS[id] ?? [],
      creatable: RELATION_NAMES,
    }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const focus = await waitFor(() => canvas.getByLabelText(/, focus$/))

    // The headings appear one after another, so wait until the display settles.
    const titles = (HEADINGS['focus'] ?? []).map((heading) => heading.text)
    const suspended = () => titles.filter((words) => canvas.queryByText(words))
    await hoverOver(focus)
    await waitFor(() => expect(suspended().length).toBeGreaterThan(4), { timeout: 5000 })

    // The headings shown are the leading ones of the note, as many as fit; the
    // remainder is truncated at the end, not dropped from the middle.
    const suspendedView = suspended()
    await expect(suspendedView).toEqual(titles.slice(0, suspendedView.length))

    // The headings are under the box, in note order. The box is the node's own
    // rectangle; the surrounding group has grown to contain them.
    const box = focus.querySelector('rect')!.getBoundingClientRect()
    const tops = suspendedView.map((words) => canvas.getByText(words).getBoundingClientRect().top)
    await expect(Math.min(...tops)).toBeGreaterThan(box.bottom - 1)
    await expect([...tops].sort((one, other) => one - other)).toEqual(tops)
  },
}
