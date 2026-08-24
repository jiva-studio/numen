/**
 * The whole window at rest: the plex on the left, the note it is focused on
 * beside it, and the agent along the trailing edge.
 *
 * Nothing here moves and nothing here waits. It is the screen a picture of the
 * product is taken from, so every piece is drawn in the state it settles in.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { ref } from 'vue'
import Workspace from '@/workspace/Workspace.vue'
import Plex from '@/plex/Plex.vue'
import Editor from '@/editor/Editor.vue'
import Palette from '@/palette/Palette.vue'
import Agent from './Agent.vue'
import { branch, pane, type Tab, type Workspace as State } from '@/workspace/model'
import type { PaletteBand, PaletteItem, PaletteSpan } from '@/palette/model'
import type { PlexEdge, PlexNeighbourhood, PlexNode } from '@/plex/model'
import type { Turn } from '@/thread/model'

const PLEX = 'plex'
const NOTE = 'note'
const BOOK = 'book'
const AGENT = 'agent'

const TABS: readonly Tab[] = [
  { id: PLEX, title: 'Plex' },
  { id: NOTE, title: 'Entropy' },
  { id: BOOK, title: 'Boltzmann 1877' },
  { id: AGENT, title: 'Agent' },
]

const node = (id: string, title: string, seat: PlexNode['seat']): PlexNode => ({
  id,
  title,
  seat,
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
const edgeFor = (related: PlexNode): PlexEdge => {
  const label = LABELS[related.id]
  if (related.seat === 'sibling') return { from: 'thermodynamics', to: related.id }
  if (related.seat === 'parent' || related.seat === 'jump') {
    return { from: related.id, to: 'focus', ...(label ? { label } : {}) }
  }
  return { from: 'focus', to: related.id, ...(label ? { label } : {}) }
}

const around = (related: readonly PlexNode[]): PlexNeighbourhood => ({
  nodes: [node('focus', 'Entropy', 'focus'), ...related],
  edges: related.map(edgeFor),
})

/** What a pane of a divided window has the width for. */
const NEIGHBOURHOOD = around(RELATED)

/**
 * What a pane with the window to itself has the room for: wider at the sides,
 * and no deeper, so nothing is left off the picture.
 */
const WIDE = around([
  ...RELATED,
  node('landauer', "Landauer's principle", 'jump'),
  node('carnot', 'Carnot cycle', 'jump'),
  node('volume', 'Volume', 'sibling'),
  node('work', 'Work', 'sibling'),
])

/**
 * A note is the prose below its frontmatter, which is what a tab holds. The
 * title is the frontmatter's, so the body does not open by saying it again.
 */
const MARKDOWN = `How many arrangements a state could have been made of, counted the same way in a gas and in a message. Boltzmann counted the microstates a temperature leaves open; Shannon counted the messages a channel could have sent, and the two counts came out as one expression.

## Where the two meet

A demon that sorts fast molecules from slow ones appears to lower the entropy of the gas for nothing. It does not: sorting means knowing which molecule is which, and the knowing is paid for when the memory holding it is cleared.

- Boltzmann, 1877 — *Über die Beziehung*, p. 373
- Shannon, 1948 — the same logarithm, in bits
- Landauer, 1961 — what clearing one bit costs

> The measure is not about heat. It is about how much a description leaves open.

## What is still open

Whether [[Free energy]] belongs under this note or beside it. The counting there assumes a temperature the surroundings hold fixed, and that assumption is nowhere above.
`

const said = (id: string, text: string): Turn => ({ id, voice: 'asked', text })

const did = (id: string, text: string, about: string, aside: string): Turn => ({
  id,
  voice: 'doing',
  text,
  about,
  aside,
  opens: true,
})

const back = (id: string, text: string): Turn => ({ id, voice: 'answered', text })

/** A tool is named the way the panel says it: as a program is named, spoken. */
const TURNS: readonly Turn[] = [
  said('1', 'What does this note leave out?'),
  did('2', 'note neighbourhood', 'Entropy', '4 links'),
  did('3', 'source read', 'Boltzmann 1877 · p. 373', '2,140 characters'),
  back(
    '4',
    'Two things. The note says the demon has to pay, but not where the payment ' +
      'lands — that is Landauer, and you have no note for him. And the passage on ' +
      'p. 373 counts arrangements of a **fixed** energy, which is the assumption ' +
      'the free-energy note quietly drops.',
  ),
  said('5', "Link it to Maxwell's demon and say why."),
  did('6', 'link add', "Entropy → Maxwell's demon", 'jump'),
  back('7', 'Written, as a jump, noted "what the sorting has to pay for".'),
]

/** Where a run of a title stands, every time it stands there. */
const marks = (title: string, word: string): PaletteSpan[] => {
  const spans: PaletteSpan[] = []
  const lower = title.toLowerCase()
  for (let at = lower.indexOf(word); at >= 0; at = lower.indexOf(word, at + 1)) {
    spans.push({ from: at, to: at + word.length })
  }
  return spans
}

const found = (id: string, title: string, detail: string): PaletteItem => ({
  id,
  title,
  at: marks(title, 'entrop'),
  detail,
  actions: [
    { id: 'travel', text: 'Show in plex' },
    { id: 'read', text: 'Open the note' },
  ],
})

const BANDS: readonly PaletteBand[] = [
  {
    id: 'notes',
    title: 'Notes',
    items: [
      found('n1', 'Entropy', 'physics/Entropy.md'),
      found('n2', 'Entropy of mixing', 'physics/Entropy of mixing.md'),
      found('n3', 'Shannon entropy', 'computation/Shannon entropy.md'),
    ],
  },
  {
    id: 'passages',
    title: 'In the books',
    items: [
      {
        id: 'p1',
        title: 'the entropy of a state is the logarithm of the number of…',
        detail: 'Boltzmann 1877 · p. 373',
        actions: [{ id: 'read', text: 'Open the page' }],
      },
      {
        id: 'p2',
        title: '…the same measure appears where a channel is described',
        detail: 'Shannon 1948 · § 6',
        actions: [{ id: 'read', text: 'Open the page' }],
      },
    ],
  },
]

interface Screen {
  /** How the window is divided, and which tab each pane shows. */
  readonly workspace: () => State
  /** Whether the palette stands over it. */
  readonly palette?: boolean
  /** What the map is looking at, which follows the room it is drawn in. */
  readonly neighbourhood?: PlexNeighbourhood
}

/** One arrangement of the window, drawn from the pieces above. */
const screen = ({
  workspace,
  palette = false,
  neighbourhood = NEIGHBOURHOOD,
}: Screen) => ({
  components: { Workspace, Plex, Editor, Agent, Palette },
  setup() {
    const held = ref<State>(workspace())
    const text = ref(MARKDOWN)
    const asking = ref('')
    const typed = ref(palette ? 'entrop' : '')
    return {
      held,
      text,
      asking,
      typed,
      palette,
      neighbourhood,
      TURNS,
      BANDS,
      TABS,
      PLEX,
      NOTE,
      AGENT,
    }
  },
  template: `
    <div style="height: 100vh">
      <Workspace v-model="held" :tabs="TABS">
        <template #tab="{ id }">
          <Plex
            v-if="id === PLEX"
            :neighbourhood="neighbourhood"
            :creatable="[]"
            :duration="0"
          />
          <Editor v-else-if="id === NOTE" v-model="text" />
          <Agent
            v-else-if="id === AGENT"
            v-model="asking"
            :turns="TURNS"
            placeholder="Ask about this note"
          />
        </template>
      </Workspace>
      <Palette v-if="palette" v-model="typed" :bands="BANDS" open placeholder="Search" />
    </div>
  `,
})

const meta: Meta = {
  title: 'Application/Window',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

/** The map with the room, and the agent along the trailing edge. */
export const Map: Story = {
  render: () =>
    screen({
      workspace: () => ({
        root: branch('root', [pane('main', [PLEX]), pane('aside', [AGENT])], [0.71, 0.29]),
        axis: 'horizontal',
        focus: 'main',
      }),
      neighbourhood: WIDE,
    }),
}

/** A note being written, with the map it stands in beside it. */
export const Writing: Story = {
  render: () =>
    screen({
      workspace: () => ({
        root: branch(
          'root',
          [pane('main', [PLEX]), pane('middle', [NOTE, BOOK], NOTE)],
          [0.34, 0.66],
        ),
        axis: 'horizontal',
        focus: 'middle',
      }),
    }),
}

/** The palette, over everything the window holds. */
export const Searching: Story = {
  render: () =>
    screen({
      workspace: () => ({
        root: branch(
          'root',
          [pane('main', [PLEX]), pane('middle', [NOTE, BOOK], NOTE)],
          [0.46, 0.54],
        ),
        axis: 'horizontal',
        focus: 'middle',
      }),
      palette: true,
    }),
}

/** The agent beside the note it is being asked about. */
export const Asking: Story = {
  render: () =>
    screen({
      workspace: () => ({
        root: branch(
          'root',
          [pane('middle', [NOTE, BOOK], NOTE), pane('aside', [AGENT])],
          [0.52, 0.48],
        ),
        axis: 'horizontal',
        focus: 'aside',
      }),
    }),
}
