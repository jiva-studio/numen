/**
 * The whole window at rest: the plex on the left, the note it is focused on
 * beside it, and the agent along the trailing edge.
 *
 * Nothing here moves and nothing here waits. It is the screen a picture of the
 * product is taken from, so every piece is drawn in the state it settles in.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor, within } from 'storybook/test'
import {
  ArrowRightLeft,
  Book,
  BookOpen,
  Bot,
  Copy,
  CornerDownRight,
  CornerLeftUp,
  FileText,
  Folder,
  FolderOpen,
  FolderTree,
  Trash2,
  Type,
  Waypoints,
} from '@lucide/vue'
import { ref } from 'vue'
import { WorkspaceLayout } from '@/features/workspace'
import { Plex } from '@/features/plex'
import { Editor } from '@/features/editor'
import { Palette } from '@/features/palette'
import { Reader } from '@/features/reader'
import { Tree } from '@/features/tree'
import { Menu } from '@/shared/ui/menu'
import type { MenuItem } from '@/shared/ui/menu'
import Agent from './Agent.vue'
import { branch, pane, type Tab, type Workspace as State } from '@/features/workspace'
import { keyChord } from '@/features/palette'
import type { PaletteAction, PaletteGroup, PaletteItem } from '@/features/palette'
import type { Span } from '@/shared/lib/span'
import type { PaletteKeys } from '@/shared/ui/key-cap'
import type { PlexPart } from '@/features/plex'
import { RELATED_SEATS, type PlexRelatedSeat } from '@/features/plex'
import type { PlexEdge } from '@/features/plex'
import type { PlexNeighbourhood } from '@/features/plex'
import type { PlexNode } from '@/features/plex'
import type { Row } from '@/features/tree'
import type { Turn } from '@/features/thread'
import { hoverOver } from '@/shared/fixtures/colour'

const PLEX = 'plex'
const NOTE = 'note'
const BOOK = 'book'
const AGENT = 'agent'
const FILES = 'files'
const OTHER_NOTE = 'demon'

const TABS: readonly Tab[] = [
  { id: PLEX, title: 'Plex' },
  { id: NOTE, title: 'Entropy' },
  { id: BOOK, title: 'Boltzmann 1877' },
  { id: OTHER_NOTE, title: "Maxwell's demon" },
  { id: AGENT, title: 'Agent' },
  { id: FILES, title: 'Files' },
]

/** What each tab is drawn as before its name, as the application draws it. */
const TAB_ICONS: Readonly<Record<string, typeof Waypoints>> = {
  [PLEX]: Waypoints,
  [NOTE]: FileText,
  [OTHER_NOTE]: FileText,
  [BOOK]: BookOpen,
  [AGENT]: Bot,
  [FILES]: FolderTree,
}

const iconOfTab = (id: string) => TAB_ICONS[id]

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
const edgeFor = (neighbour: PlexNode): PlexEdge => {
  const label = LABELS[neighbour.id]
  if (neighbour.seat === 'sibling') return { from: 'thermodynamics', to: neighbour.id }
  if (neighbour.seat === 'parent' || neighbour.seat === 'jump') {
    return { from: neighbour.id, to: 'focus', ...(label ? { label } : {}) }
  }
  return { from: 'focus', to: neighbour.id, ...(label ? { label } : {}) }
}

const around = (neighbours: readonly PlexNode[]): PlexNeighbourhood => ({
  nodes: [node('focus', 'Entropy', 'focus'), ...neighbours],
  edges: neighbours.map(edgeFor),
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

/** A second note, for a pane opened beside the first one. */
const OTHER = `A thought experiment of 1867: a being small enough to see single molecules stands at a door between two halves of a box, and lets the fast ones through one way and the slow ones the other.

## Why it looks free

Nothing is pushed and nothing is heated. The gas ends sorted, hot on one side and cold on the other, and no work appears to have been done to sort it.

## Where the bill arrives

The being has to know which molecule is which, and it has to hold that knowledge somewhere. The memory fills, and clearing it costs what [[Landauer's principle]] says a bit costs.

> The demon does not break the second law. It pays for the sorting in a currency nobody was counting.
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

/**
 * The volume those pages were bound in, and where in it they stand. The reader
 * counts the pages of the document it is given, so the number it says and the
 * number printed on the page are the same number.
 */
const BOOK_LEAVES = 412
const BOOK_FIRST = 372

/** How a page of that document is drawn: letter paper, one line at a time. */
const PAPER = { wide: 612, high: 792 }
const MARGIN = 84
const FIRST = 132
const LEADING = 27

const escapeHtml = (line: string) =>
  line.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

const renderPage = (page: number): string => {
  const leaf = page - BOOK_FIRST
  const lines = BOOK_PAGES[leaf] ?? []
  const set = lines
    .map((line, at) => {
      const y = FIRST + at * LEADING
      const weight = at === 0 && leaf === 0 ? ' font-weight="600" letter-spacing="1.2"' : ''
      return `<text x="${MARGIN}" y="${y}" font-family="Georgia, serif" font-size="16"${weight} fill="#1b1b1b">${escapeHtml(line)}</text>`
    })
    .join('')
  const number = `<text x="${PAPER.wide / 2}" y="${PAPER.high - 54}" text-anchor="middle" font-family="Georgia, serif" font-size="13" fill="#5a5a5a">${page + 1}</text>`
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${PAPER.wide}" height="${PAPER.high}"><rect width="100%" height="100%" fill="#fbfaf7"/>${set}${number}</svg>`
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
}

/** Where a run of lines stands on the page, in fractions of it. */
const overLines = (from: number, to: number, ends: number) => ({
  minX: (MARGIN - 4) / PAPER.wide,
  maxX: (MARGIN + ends) / PAPER.wide,
  minY: (FIRST + from * LEADING - 18) / PAPER.high,
  maxY: (FIRST + to * LEADING + 6) / PAPER.high,
})

/** The passage a search found in it: the sentence the count is defined by. */
const HIGHLIGHTS = [overLines(8, 9, 420), overLines(10, 11, 360)]

const createAsked = (id: string, text: string): Turn => ({ id, voice: 'asked', text })

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
  createAsked('1', 'What does this note leave out?'),
  did('2', 'note neighbourhood', 'Entropy', '4 links'),
  back(
    '3',
    'Two things. The note says the demon has to pay, but not where the payment ' +
      'lands — that is Landauer, and you have no note for him. And the passage on ' +
      'p. 373 counts arrangements of a **fixed** energy, which is the assumption ' +
      'the free-energy note quietly drops. Everywhere else you treat the two as ' +
      'the same measure, so one of them is wrong and it is not the one you wrote ' +
      'first.',
  ),
  createAsked('4', 'Then make the Landauer note and put it under Entropy.'),
  did('5', 'note create', "Landauer's principle", 'under Entropy'),
  back(
    '6',
    'Made and filed under Entropy, written from p. 373 and the 1961 paper: what ' +
      'clearing one bit costs, and why the sorting is what pays it. I linked it ' +
      "across to Maxwell's demon, since that is where the debt is run up, and " +
      'left the free-energy note alone — its assumption is still worth a note of ' +
      'its own.',
  ),
]

/** Where a run of a title stands, every time it stands there. */
const marks = (title: string, word: string): Span[] => {
  const spans: Span[] = []
  const lower = title.toLowerCase()
  for (let at = lower.indexOf(word); at >= 0; at = lower.indexOf(word, at + 1)) {
    spans.push({ from: at, to: at + word.length })
  }
  return spans
}

const TRAVEL = { id: 'travel', text: 'Show in plex' }
const READ = { id: 'read', text: 'Open the note' }
const READ_AT = { id: 'readAt', text: 'Open at this heading' }
const READ_DOCUMENT = { id: 'readDocument', text: 'Open the document here' }

/** A note whose name carries the words: the name, and nothing under it. */
const createItem = (id: string, title: string): PaletteItem => ({
  id,
  title,
  at: marks(title, 'entrop'),
  actions: [TRAVEL, READ],
})

/** A heading that carries them: the heading, and the note it stands in. */
const heading = (id: string, title: string, note: string): PaletteItem => ({
  id,
  title,
  at: marks(title, 'entrop'),
  detail: note,
  actions: [READ_AT, TRAVEL],
})

/** A passage: what it came out of, and the words around the hit under it. */
const passage = (
  id: string,
  source: string,
  text: string,
  actions: readonly PaletteAction[],
): PaletteItem => ({
  id,
  title: source,
  detail: text,
  detailAt: marks(text, 'entrop'),
  actions,
})

/**
 * What one search turns up, in the three groups the window draws: the names it
 * matched, the text it was found in, and what means the same without saying it.
 */
const GROUPS: readonly PaletteGroup[] = [
  {
    id: 'names',
    title: 'Names',
    items: [
      createItem('n1', 'Entropy'),
      createItem('n2', 'Entropy of mixing'),
      heading('n3', 'Entropy as missing information', 'Shannon entropy'),
    ],
  },
  {
    id: 'text',
    title: 'Text',
    items: [
      passage(
        't1',
        'Boltzmann 1877',
        'the entropy of a state is the logarithm of the number of arrangements it '
          + 'could have been made of',
        [READ_DOCUMENT],
      ),
      passage(
        't2',
        'The second law',
        'entropy never falls in a closed system, which is the whole of it stated '
          + 'in one line',
        [READ, TRAVEL],
      ),
    ],
  },
  {
    id: 'meaning',
    title: 'Meaning',
    items: [
      passage(
        'm1',
        "Landauer's principle",
        'clearing one bit of memory costs at least kT ln 2 of heat, which is what '
          + 'the sorting has to pay for',
        [READ, TRAVEL],
      ),
      passage(
        'm2',
        "Maxwell's demon",
        'a demon that sorts fast molecules from slow ones appears to lower the '
          + 'disorder of a gas for nothing',
        [READ, TRAVEL],
      ),
    ],
  },
]

/**
 * What the menu on a node offers, in the groups the window draws it in: what
 * opens the note, what is done to its file, what is made off it in the plex,
 * what is asked of the agent, and what takes it out of the vault.
 */
const MENU: readonly MenuItem[] = [
  { id: 'read', text: 'Open the note', group: 'open' },
  { id: 'travel', text: 'Show in plex', group: 'open' },
  { id: 'copy', text: 'Copy path', group: 'file' },
  { id: 'reveal', text: 'Show this note in the files', group: 'file' },
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

/** A keyboard to draw the chords for: this file settles on one. */
const KEYBOARD = 'Macintosh'

const RUN = [{ id: 'run', text: 'Run it' }]

const command = (id: string, title: string, keys?: PaletteKeys): PaletteItem => ({
  id,
  title,
  actions: RUN,
  ...(keys ? { keys } : {}),
})

/** What the commands offer, in the three groups the window draws them in. */
const COMMANDS: readonly PaletteGroup[] = [
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
  /** Where the menu on a node stands, and none where it stands nowhere. */
  readonly menu?: { x: number; y: number } | null
  /** The rows of the vault the selection stands on. */
  readonly selected?: readonly string[]
}

/** One arrangement of the window, drawn from the pieces above. */
const screen = ({
  workspace,
  panel = null,
  neighbourhood = NEIGHBOURHOOD,
  markdown = MARKDOWN,
  parts = () => [],
  creatable = [],
  menu = null,
  selected = [],
}: Screen) => ({
  components: { WorkspaceLayout, Plex, Editor, Agent, Palette, Reader, Tree, Menu },
  setup() {
    const held = ref<State>(workspace())
    const text = ref(markdown)
    const other = ref(OTHER)
    const asking = ref('')
    const typed = ref(panel === 'search' ? 'entrop' : '')
    const at = ref(BOOK_FIRST)

    return {
      held,
      text,
      other,
      asking,
      typed,
      at,
      panel,
      neighbourhood,
      parts,
      creatable,
      menu,
      selected,
      MENU,
      menuIcon,
      groups: panel === 'commands' ? COMMANDS : GROUPS,
      placeholder: panel === 'commands' ? 'Type a command' : 'Search',
      pages: Array.from({ length: BOOK_LEAVES }, () => PAPER),
      picture: (page: number) => renderPage(page),
      highlightsOn: (page: number) => (page === BOOK_FIRST ? HIGHLIGHTS : []),
      go: (page: number) => {
        at.value = Math.min(Math.max(page, 0), BOOK_LEAVES - 1)
      },
      TURNS,
      TABS,
      ROWS,
      OPEN,
      iconFor,
      iconOfTab,
      PLEX,
      NOTE,
      BOOK,
      OTHER_NOTE,
      AGENT,
      FILES,
    }
  },
  template: `
    <div style="height: 100vh">
      <WorkspaceLayout v-model="held" :tabs="TABS">
        <!-- Lucide draws on a 24 grid, and the stroke is given in those units. -->
        <template #icon="{ id }">
          <component
            :is="iconOfTab(id)"
            v-if="iconOfTab(id)"
            style="inline-size: 100%; block-size: 100%; stroke-width: 1.75; opacity: 0.75"
          />
        </template>

        <template #tab="{ id }">
          <Plex
            v-if="id === PLEX"
            :neighbourhood="neighbourhood"
            :creatable="creatable"
            :duration="0"
            :parts="parts"
          />
          <Editor v-else-if="id === NOTE" v-model="text" />
          <Editor v-else-if="id === OTHER_NOTE" v-model="other" />
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
            :at="at"
            :picture="picture"
            :highlights="highlightsOn"
            @go="go"
          />
          <Tree
            v-else-if="id === FILES"
            :rows="ROWS"
            :open="OPEN"
            :selected="selected"
            name="The folders and files of the vault"
          >
            <template #icon="{ id, open }">
              <component :is="iconFor(id, open)" class="size-4 shrink-0 text-hushed" />
            </template>
          </Tree>
        </template>
      </WorkspaceLayout>
      <Palette
        v-if="panel"
        v-model="typed"
        :groups="groups"
        open
        :placeholder="placeholder"
      />

      <Menu v-if="menu" :items="MENU" :at="menu" open name="What can be done to this note">
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

const meta: Meta = {
  title: 'Application/Window',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

/** A window holding a note: Tab indents in the editor until Escape hands it back. */
const WRITING = { reach: { keeps: 'Escape hands Tab back to the page' } }

/** One pane of the window, by the name it was divided under. */
const paneOf = (canvas: HTMLElement, id: string): HTMLElement => {
  const held = canvas.querySelector<HTMLElement>(`[data-workspace-pane="${id}"]`)
  if (!held) throw new Error(`no pane ${id}`)
  return held
}

/** Whether one pane stands entirely past another's trailing edge. */
const past = (later: HTMLElement, earlier: HTMLElement): boolean =>
  later.getBoundingClientRect().left >= earlier.getBoundingClientRect().right - 1

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
  play: async ({ canvasElement }) => {
    const map = paneOf(canvasElement, 'main')
    const aside = paneOf(canvasElement, 'aside')

    // The map is drawn around the note the window is focused on, and it has the
    // room: most of the width, and the whole neighbourhood in it.
    await waitFor(() => expect(within(map).getAllByLabelText(/, focus$/)).toHaveLength(1))
    expect(within(map).getAllByLabelText(/, (parent|child|sibling|jump)$/).length).toBeGreaterThan(3)
    expect(map.getBoundingClientRect().width).toBeGreaterThan(
      aside.getBoundingClientRect().width,
    )

    // The agent is along the trailing edge, and is something to ask with.
    within(aside).getByPlaceholderText('Ask about the vault')
    expect(past(aside, map)).toBe(true)
  },
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
      parts: (id: string) => PARTS[id] ?? [],
      creatable: RELATED_SEATS,
      // Asked for on a node off to the trailing side, where it stands clear of
      // the map and of what a node is hanging.
      menu: { x: 902, y: 396 },
    }),
  play: async ({ canvasElement }) => {
    // The map has the window: one pane, and no second one beside it.
    expect(canvasElement.querySelectorAll('[data-workspace-pane]')).toHaveLength(1)
    await waitFor(() =>
      expect(within(canvasElement).getAllByLabelText(/, focus$/)).toHaveLength(1),
    )

    // The menu was asked for on a node, and every item of it is in the window
    // rather than off the edge it was asked near.
    const menu = within(document.body).getByRole('menu', {
      name: 'What can be done to this note',
    })
    const box = menu.getBoundingClientRect()
    expect(box.width).toBeGreaterThan(0)
    expect(box.left).toBeGreaterThanOrEqual(0)
    expect(box.right).toBeLessThanOrEqual(window.innerWidth + 1)
    expect(box.top).toBeGreaterThanOrEqual(0)
    expect(box.bottom).toBeLessThanOrEqual(window.innerHeight + 1)
    expect(within(menu).getAllByRole('menuitem').length).toBeGreaterThan(0)
  },
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
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const focus = await waitFor(() => canvas.getByLabelText(/, focus$/))

    // A hand left on the node hangs the headings of its note under the box.
    // They come in one after another, so the picture is let settle first.
    const HEADINGS = (PARTS['focus'] ?? []).map((part) => part.text)
    const hung = () => HEADINGS.filter((words) => canvas.queryByText(words))
    await hoverOver(focus)
    await waitFor(() => expect(hung().length).toBeGreaterThan(4), { timeout: 5000 })

    // As many as the panel has room for, taken from the top of the note: what
    // does not fit is wound to, not dropped from the middle.
    const shown = hung()
    expect(shown).toEqual(HEADINGS.slice(0, shown.length))

    // They hang under the box, in the order the note sets them out. The box is
    // the node's own rectangle: the group around it has grown to hold them.
    const box = focus.querySelector('rect')!.getBoundingClientRect()
    const tops = shown.map((words) => canvas.getByText(words).getBoundingClientRect().top)
    expect(Math.min(...tops)).toBeGreaterThan(box.bottom - 1)
    expect([...tops].sort((one, other) => one - other)).toEqual(tops)
  },
}

/** A note being written, with the map it stands in beside it. */
export const Writing: Story = {
  parameters: WRITING,
  render: () =>
    screen({
      workspace: () => ({
        root: branch(
          'root',
          [
            pane('main', [PLEX]),
            // A branch inside a horizontal one is drawn the other way, so the
            // two notes stand one above the other.
            branch(
              'beside',
              [pane('upper', [NOTE, BOOK], NOTE), pane('lower', [OTHER_NOTE], OTHER_NOTE)],
              [0.5, 0.5],
            ),
          ],
          [0.52, 0.48],
        ),
        axis: 'horizontal',
        focus: 'upper',
      }),
      neighbourhood: WIDE,
    }),
  play: async ({ canvasElement }) => {
    // Writing means somewhere to type: the note in the pane is drawn as text a
    // person may put a caret in.
    await waitFor(() => {
      const written = canvasElement.querySelector('.cm-content')
      expect(written).not.toBeNull()
      expect(written?.getAttribute('contenteditable')).toBe('true')
    })
  },
}

/** The palette, over everything the window holds. */
export const Searching: Story = {
  parameters: WRITING,
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
  play: async () => {
    // Searching means something to search with: the field carries what was
    // typed, and what it found is on offer under it.
    await waitFor(() => {
      const field = document.body.querySelector<HTMLInputElement>('[data-palette="field"]')
      expect(field?.value).toBe('entrop')
      expect(
        document.body.querySelectorAll('[data-palette="list"] [role="option"]').length,
      ).toBeGreaterThan(0)
    })
  },
}

/** The commands, in the three groups they are drawn in. */
export const Commanding: Story = {
  parameters: WRITING,
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
  play: async () => {
    // Commanding means the commands are there to be run, in the three groups
    // the window sorts them into.
    await waitFor(() => {
      const groups = [...document.body.querySelectorAll('[data-palette="title"]')]
      expect(groups.map((group) => group.textContent?.trim())).toEqual([
        'This note',
        'This window',
        'This vault',
      ])
    })
  },
}

/** The agent beside the note it is being asked about. */
export const Asking: Story = {
  parameters: WRITING,
  render: () =>
    screen({
      workspace: () => ({
        root: branch(
          'root',
          [
            pane('source', [BOOK], BOOK),
            pane('middle', [NOTE], NOTE),
            pane('aside', [AGENT]),
          ],
          [0.4, 0.3, 0.3],
        ),
        axis: 'horizontal',
        focus: 'aside',
      }),
    }),
  play: async ({ canvasElement }) => {
    const source = paneOf(canvasElement, 'source')
    const note = paneOf(canvasElement, 'middle')
    const aside = paneOf(canvasElement, 'aside')

    // The three stand in the order they were divided in, the agent last.
    expect(past(note, source)).toBe(true)
    expect(past(aside, note)).toBe(true)

    // The agent is something to ask with, and it is already carrying the
    // asking it is beside the note about.
    within(aside).getByPlaceholderText('Ask about the vault')
    expect(within(aside).getAllByText(/entropy/i).length).toBeGreaterThan(0)
  },
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
  play: async ({ canvasElement }) => {
    // Reading means the document is open at the page the search landed on,
    // with the passage it found highlighted on it.
    await waitFor(() => {
      const page = canvasElement.querySelector(`.reader__page[data-page="${BOOK_FIRST}"]`)
      expect(page).not.toBeNull()
      expect(page?.querySelectorAll('.reader__highlight').length).toBeGreaterThan(0)
    })
  },
}

/** The folders of the vault, with rows chosen across two of them. */
export const Filing: Story = {
  parameters: WRITING,
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
      selected: ['entropy', 'landauer'],
    }),
  play: async ({ canvasElement }) => {
    const tree = within(canvasElement).getByRole('tree', {
      name: 'The folders and files of the vault',
    })
    const rows = within(tree).getAllByRole('treeitem')
    const chosen = rows.filter((row) => row.getAttribute('aria-selected') === 'true')

    // The selection stands on two rows, and they are held in different
    // folders, which is the case a single run of rows would not cover.
    expect(chosen.map((row) => row.textContent?.trim())).toEqual([
      'Entropy.md',
      "Landauer's principle.md",
    ])
    const folders = rows.filter((row) => within(row).queryByText(/^(Physics|Computation)$/))
    expect(folders).toHaveLength(2)
    for (const [at, folder] of folders.entries()) {
      expect(rows.indexOf(folder)).toBeLessThan(rows.indexOf(chosen[at]!))
    }
  },
}

/** A note holding a table, which is typed in as a table. */
export const Tabling: Story = {
  parameters: WRITING,
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
  play: async ({ canvasElement }) => {
    // The table in the note is drawn as a table, not as the pipes it is
    // written with, and every cell of it is a cell to type in.
    const table = await waitFor(() => within(canvasElement).getByRole('table'))
    const heads = within(table).getAllByRole('columnheader')
    expect(heads.map((head) => head.textContent?.trim())).toEqual([
      'Where',
      'What is counted',
      'Written',
    ])

    const cells = within(table).getAllByRole('cell')
    expect(cells).toHaveLength(9)
    for (const cell of cells) expect(cell).toHaveAttribute('contenteditable', 'plaintext-only')
    expect(within(canvasElement).queryByText(/\| --- \|/)).toBeNull()
  },
}
