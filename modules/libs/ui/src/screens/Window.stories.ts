/**
 * The whole window at rest: the plex on the left, the note it is focused on
 * beside it, and the agent along the trailing edge.
 *
 * Nothing here moves and nothing here waits. It is the screen a picture of the
 * product is taken from, so every piece is drawn in the state it settles in.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { Book, FileText, Folder, FolderOpen } from '@lucide/vue'
import { ref } from 'vue'
import Workspace from '@/workspace/Workspace.vue'
import Plex from '@/plex/Plex.vue'
import Editor from '@/editor/Editor.vue'
import Palette from '@/palette/Palette.vue'
import Reader from '@/reader/Reader.vue'
import Tree from '@/tree/Tree.vue'
import Agent from './Agent.vue'
import { branch, pane, type Tab, type Workspace as State } from '@/workspace/model'
import { keyChord } from '@/palette/model'
import type { PaletteBand, PaletteItem, PaletteKeys, PaletteSpan } from '@/palette/model'
import type { PlexPart } from '@/plex/inside'
import { RELATED_SEATS } from '@/plex/model'
import type { PlexEdge, PlexNeighbourhood, PlexNode, PlexRelatedSeat } from '@/plex/model'
import type { Row } from '@/tree/model'
import type { Turn } from '@/thread/model'

const PLEX = 'plex'
const NOTE = 'note'
const BOOK = 'book'
const AGENT = 'agent'
const FILES = 'files'

const TABS: readonly Tab[] = [
  { id: PLEX, title: 'Plex' },
  { id: NOTE, title: 'Entropy' },
  { id: BOOK, title: 'Boltzmann 1877' },
  { id: AGENT, title: 'Agent' },
  { id: FILES, title: 'Files' },
]

/** The folders of the vault the rest of this file is written about. */
const ROWS: readonly Row[] = [
  {
    id: 'physics',
    name: 'Physics',
    holds: true,
    rows: [
      { id: 'entropy', name: 'Entropy.md', holds: false },
      { id: 'mixing', name: 'Entropy of mixing.md', holds: false },
      { id: 'second-law', name: 'The second law.md', holds: false },
      { id: 'free-energy', name: 'Free energy.md', holds: false },
    ],
  },
  {
    id: 'computation',
    name: 'Computation',
    holds: true,
    rows: [
      { id: 'shannon-entropy', name: 'Shannon entropy.md', holds: false },
      { id: 'landauer', name: "Landauer's principle.md", holds: false },
    ],
  },
  {
    id: 'reading',
    name: 'Reading',
    holds: true,
    rows: [
      { id: 'boltzmann-pdf', name: 'Boltzmann 1877.pdf', holds: false },
      { id: 'shannon-pdf', name: 'Shannon 1948.pdf', holds: false },
    ],
  },
  { id: 'inbox', name: 'Inbox.md', holds: false },
  { id: 'reading-list', name: 'Reading list.md', holds: false },
]

const OPEN = ['physics', 'computation', 'reading']

/** What is drawn beside a row: a folder says whether what it holds is drawn. */
const iconFor = (id: string, open: boolean) => {
  const row = ROWS.find((one) => one.id === id) ?? ROWS.flatMap((one) => one.rows ?? []).find((one) => one.id === id)
  if (row?.holds) return open ? FolderOpen : Folder
  if (row?.name.endsWith('.pdf')) return Book
  return FileText
}

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

/** What a pane sharing its column with a note has the width for. */
const CLOSE = around(RELATED.slice(0, 6))

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
 * The headings of the notes on the map, by the node each stands in. More of
 * them than stand at once, so the mark saying there is more to wind to is
 * drawn as well.
 */
const PARTS: Readonly<Record<string, readonly PlexPart[]>> = {
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

/**
 * The document a search sends a person into, as the lines standing on each of
 * its pages. It is set here rather than fetched, so the picture of a book is
 * a book and not a placeholder — and broken into lines by hand, because what
 * draws a page below cannot break one.
 */
const BOOK_PAGES: readonly (readonly string[])[] = [
  [
    '§ 6.  THE COUNT OF ARRANGEMENTS',
    '',
    'Let a quantity of gas be enclosed, and let the total energy',
    'it holds be fixed. The molecules may then be arranged in a',
    'great many ways without that total being disturbed, and no',
    'measurement made upon the gas as a whole distinguishes one',
    'such arrangement from another.',
    '',
    'The number of these arrangements is not a property of any',
    'one of them. It belongs to the description: it says how much',
    'that description has left open. Where the description is',
    'exact, the number is one, and its logarithm vanishes.',
    '',
    'It is this logarithm, and not the heat, that the second law',
    'is a statement about. A quantity of gas passes from an',
    'arrangement that could have been reached in few ways to one',
    'that could have been reached in many, because there are more',
    'of the latter to be reached.',
  ],
  [
    'The same counting serves wherever a description leaves some-',
    'thing open. A channel that may carry any of a great number of',
    'messages is described by the logarithm of that number, and',
    'the description is improved exactly as the number falls.',
    '',
    'Nothing in the argument is peculiar to molecules. What is',
    'counted are the possibilities a statement admits, and the',
    'measure of them is the same whether the statement is made',
    'about a gas, about a message, or about the contents of a',
    'memory that is about to be cleared.',
    '',
    'The objection has been raised that a sufficiently attentive',
    'observer, able to see each molecule and to open a door for',
    'the swift ones alone, would lower the count without expend-',
    'ing work, and so defeat the law.',
  ],
  [
    'The objection does not hold. To open the door for the swift',
    'ones the observer must know which are swift, and the knowing',
    'occupies a memory. That memory is finite; to go on sorting,',
    'the observer must clear it.',
    '',
    'The clearing is the payment. It is exacted not at the door',
    'but at the moment the record of a molecule is destroyed, and',
    'it is exacted at the same rate at which the sorting reduced',
    'the count. The books balance, and the law stands.',
    '',
    'What the law forbids, then, is not a clever machine but a',
    'description that improves itself for nothing.',
  ],
]

/** How a page of that document is drawn: letter paper, one line at a time. */
const PAPER = { wide: 612, high: 792 }
const MARGIN = 84
const FIRST = 132
const LEADING = 27

const escaped = (line: string) =>
  line.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

const drawnPage = (page: number): string => {
  const lines = BOOK_PAGES[page] ?? []
  const set = lines
    .map((line, at) => {
      const y = FIRST + at * LEADING
      const weight = at === 0 && page === 0 ? ' font-weight="600" letter-spacing="1.2"' : ''
      return `<text x="${MARGIN}" y="${y}" font-family="Georgia, serif" font-size="16"${weight} fill="#1b1b1b">${escaped(line)}</text>`
    })
    .join('')
  const number = `<text x="${PAPER.wide / 2}" y="${PAPER.high - 54}" text-anchor="middle" font-family="Georgia, serif" font-size="13" fill="#5a5a5a">${373 + page}</text>`
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${PAPER.wide}" height="${PAPER.high}"><rect width="100%" height="100%" fill="#fbfaf7"/>${set}${number}</svg>`
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
}

/** Where a run of lines stands on the page, in fractions of it. */
const litLines = (from: number, to: number, ends: number) => ({
  minX: (MARGIN - 4) / PAPER.wide,
  maxX: (MARGIN + ends) / PAPER.wide,
  minY: (FIRST + from * LEADING - 18) / PAPER.high,
  maxY: (FIRST + to * LEADING + 6) / PAPER.high,
})

/** The passage a search found in it: the sentence the count is defined by. */
const LIT = [litLines(8, 9, 420), litLines(10, 11, 360)]

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

/** A keyboard to draw the chords for: this file settles on one. */
const KEYBOARD = 'Macintosh'

const RUN = [{ id: 'run', text: 'Run it' }]

const command = (id: string, title: string, keys?: PaletteKeys): PaletteItem => ({
  id,
  title,
  actions: RUN,
  ...(keys ? { keys } : {}),
})

/** What the commands offer, in the three bands the window draws them in. */
const COMMANDS: readonly PaletteBand[] = [
  {
    id: 'note',
    title: 'This note',
    items: [
      command('read', 'Open the note'),
      command('beside', 'Open beside'),
      command('travel', 'Show in plex', keyChord('p', KEYBOARD, true)),
      command('child', 'New child note', keyChord('c', KEYBOARD, true)),
      command('parent', 'New parent note'),
      command('ask', 'Ask the agent about this note'),
    ],
  },
  {
    id: 'window',
    title: 'This window',
    items: [
      command('note', 'New note', keyChord('n', KEYBOARD)),
      command('plex', 'New plex'),
      command('agent', 'New agent', keyChord('a', KEYBOARD, true)),
      command('appearance', 'Change the theme'),
    ],
  },
  {
    id: 'vault',
    title: 'This vault',
    items: [
      command('goto', 'Go to a note', keyChord('g', KEYBOARD)),
      command('openVault', 'Open vault'),
    ],
  },
]

/** A note carrying a table, for the picture of one being typed in. */
const TABLED = `The three counts, held against each other. Same logarithm, three
places it is taken.

| Where | What is counted | Written |
| --- | --- | --- |
| A gas | arrangements of one energy | Boltzmann, 1877 |
| A channel | messages it could carry | Shannon, 1948 |
| A memory | what clearing one bit costs | Landauer, 1961 |

The middle column is the one to watch: it is the same sentence three times, and
only the noun changes.
`

interface Screen {
  /** How the window is divided, and which tab each pane shows. */
  readonly workspace: () => State
  /** What stands over it: the search, the commands, or nothing. */
  readonly panel?: 'search' | 'commands' | null
  /** What the map is looking at, which follows the room it is drawn in. */
  readonly neighbourhood?: PlexNeighbourhood
  /** What the note in a tab holds. */
  readonly markdown?: string
  /** The headings a node hangs under its box, and none where there are none. */
  readonly parts?: (id: string) => readonly PlexPart[]
  /**
   * The seats a new note may be made in. None leaves the map at rest, with no
   * handle on any node and nothing coming out from under one.
   */
  readonly creatable?: readonly PlexRelatedSeat[]
}

/** One arrangement of the window, drawn from the pieces above. */
const screen = ({
  workspace,
  panel = null,
  neighbourhood = NEIGHBOURHOOD,
  markdown = MARKDOWN,
  parts = () => [],
  creatable = [],
}: Screen) => ({
  components: { Workspace, Plex, Editor, Agent, Palette, Reader, Tree },
  setup() {
    const held = ref<State>(workspace())
    const text = ref(markdown)
    const asking = ref('')
    const typed = ref(panel === 'search' ? 'entrop' : '')
    const at = ref(0)

    return {
      held,
      text,
      asking,
      typed,
      at,
      panel,
      neighbourhood,
      parts,
      creatable,
      bands: panel === 'commands' ? COMMANDS : BANDS,
      placeholder: panel === 'commands' ? 'Type a command' : 'Search',
      pages: BOOK_PAGES.length,
      sheets: BOOK_PAGES.map(() => PAPER),
      picture: (page: number) => drawnPage(page),
      litOn: (page: number) => (page === 0 ? LIT : []),
      go: (page: number) => {
        at.value = Math.min(Math.max(page, 0), BOOK_PAGES.length - 1)
      },
      TURNS,
      TABS,
      ROWS,
      OPEN,
      iconFor,
      PLEX,
      NOTE,
      BOOK,
      AGENT,
      FILES,
    }
  },
  template: `
    <div style="height: 100vh">
      <Workspace v-model="held" :tabs="TABS">
        <template #tab="{ id }">
          <Plex
            v-if="id === PLEX"
            :neighbourhood="neighbourhood"
            :creatable="creatable"
            :duration="0"
            :parts="parts"
          />
          <Editor v-else-if="id === NOTE" v-model="text" />
          <Agent
            v-else-if="id === AGENT"
            v-model="asking"
            :turns="TURNS"
            placeholder="Ask about the vault"
          />
          <Reader
            v-else-if="id === BOOK"
            class="h-full"
            :pages="pages"
            :sheets="sheets"
            :at="at"
            :picture="picture"
            :lit="litOn"
            @go="go"
          />
          <Tree
            v-else-if="id === FILES"
            :rows="ROWS"
            :open="OPEN"
            name="The folders and files of the vault"
          >
            <template #icon="{ id, open }">
              <component :is="iconFor(id, open)" class="size-4 shrink-0 text-hushed" />
            </template>
          </Tree>
        </template>
      </Workspace>
      <Palette
        v-if="panel"
        v-model="typed"
        :bands="bands"
        open
        :placeholder="placeholder"
      />
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

/** The map with the whole window to itself. */
export const Mapping: Story = {
  render: () =>
    screen({
      workspace: () => ({
        root: pane('main', [PLEX, NOTE, BOOK], PLEX),
        axis: 'horizontal',
        focus: 'main',
      }),
      neighbourhood: WIDE,
    }),
}

/**
 * The map with a hand left on a node, which hangs the headings of its note
 * under the box.
 */
export const Hanging: Story = {
  render: () =>
    screen({
      workspace: () => ({
        root: pane('main', [PLEX, NOTE, BOOK], PLEX),
        axis: 'horizontal',
        focus: 'main',
      }),
      neighbourhood: WIDE,
      parts: (id: string) => PARTS[id] ?? [],
      creatable: RELATED_SEATS,
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
      panel: 'search',
    }),
}

/** The commands, in the three bands they are drawn in. */
export const Commanding: Story = {
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
      panel: 'commands',
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

/** A document open where a search found something, and the note beside it. */
export const Reading: Story = {
  render: () =>
    screen({
      workspace: () => ({
        root: branch(
          'root',
          [pane('main', [BOOK, NOTE], BOOK), pane('aside', [AGENT])],
          [0.66, 0.34],
        ),
        axis: 'horizontal',
        focus: 'main',
      }),
    }),
}

/** The folders of the vault, with rows chosen across two of them. */
export const Filing: Story = {
  render: () =>
    screen({
      workspace: () => ({
        root: branch(
          'root',
          [
            pane('main', [FILES]),
            // A branch inside a horizontal one is drawn the other way, so the
            // map stands over the note it is about.
            branch('middle', [pane('map', [PLEX]), pane('note', [NOTE, BOOK], NOTE)], [0.52, 0.48]),
            pane('aside', [AGENT]),
          ],
          [0.22, 0.5, 0.28],
        ),
        axis: 'horizontal',
        focus: 'map',
      }),
      neighbourhood: CLOSE,
    }),
}

/** A note holding a table, which is typed in as a table. */
export const Tabling: Story = {
  render: () =>
    screen({
      workspace: () => ({
        root: branch(
          'root',
          [pane('main', [PLEX]), pane('middle', [NOTE, BOOK], NOTE)],
          [0.3, 0.7],
        ),
        axis: 'horizontal',
        focus: 'middle',
      }),
      markdown: TABLED,
    }),
}
