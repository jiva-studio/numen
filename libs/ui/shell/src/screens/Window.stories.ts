/**
 * The whole window at rest: the files, the notes, a scanned book and the agent
 * in the panes they were divided into.
 *
 * Nothing here moves and nothing here waits. It is the screen a picture of the
 * product is taken from, so every piece is drawn in the state it settles in.
 * The book is a picture of its page: the reader that draws a document is the
 * PDF plugin's, and its stories are in the plugins' Storybook. The agent's
 * pane is empty for the same reason: views-agent-chat draws the agent chat. The
 * notes are plain text: the notes plugin draws the note editor.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor, within } from 'storybook/test'
import { Book, BookOpen, Bot, FileText, Folder, FolderOpen, FolderTree } from '@lucide/vue'
import { ref } from 'vue'
import { Tree, keyChord } from '@numen/ui'
import type { PaletteKeys, Row, TextRange } from '@numen/ui'
import {
  WorkspaceLayout,
  branch,
  pane,
  type Tab,
  type Workspace as State,
} from '@/features/workspace'
import { Palette } from '@/features/palette'
import type { PaletteAction, PaletteGroup, PaletteItem } from '@/features/palette'

const NOTE = 'note'
const BOOK = 'book'
const AGENT = 'agent'
const FILES = 'files'
const OTHER_NOTE = 'demon'

const TABS: readonly Tab[] = [
  { id: NOTE, title: 'Entropy' },
  { id: BOOK, title: 'Boltzmann 1877' },
  { id: OTHER_NOTE, title: "Maxwell's demon" },
  { id: AGENT, title: 'Agent' },
  { id: FILES, title: 'Files' },
]

/** What each tab is drawn as before its name, as the application draws it. */
const TAB_ICONS: Readonly<Record<string, typeof FileText>> = {
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
    hasChildren: true,
    rows: [
      { id: 'entropy', name: 'Entropy.md', hasChildren: false },
      { id: 'mixing', name: 'Entropy of mixing.md', hasChildren: false },
      { id: 'second-law', name: 'The second law.md', hasChildren: false },
      { id: 'free-energy', name: 'Free energy.md', hasChildren: false },
    ],
  },
  {
    id: 'computation',
    name: 'Computation',
    hasChildren: true,
    rows: [
      { id: 'shannon-entropy', name: 'Shannon entropy.md', hasChildren: false },
      { id: 'landauer', name: "Landauer's principle.md", hasChildren: false },
    ],
  },
  {
    id: 'reading',
    name: 'Reading',
    hasChildren: true,
    rows: [
      { id: 'boltzmann-pdf', name: 'Boltzmann 1877.pdf', hasChildren: false },
      { id: 'shannon-pdf', name: 'Shannon 1948.pdf', hasChildren: false },
    ],
  },
  { id: 'inbox', name: 'Inbox.md', hasChildren: false },
  { id: 'reading-list', name: 'Reading list.md', hasChildren: false },
]

const OPEN = ['physics', 'computation', 'reading']

/** What is drawn beside a row: a folder says whether what it holds is drawn. */
const getIcon = (id: string, open: boolean) => {
  const row =
    ROWS.find((one) => one.id === id) ??
    ROWS.flatMap((one) => one.rows ?? []).find((one) => one.id === id)
  if (row?.hasChildren) return open ? FolderOpen : Folder
  if (row?.name.endsWith('.pdf')) return Book
  return FileText
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
 * its pages. It is set here and not fetched, so the picture of a book is
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

/** Where in the volume those pages stand, which is the number printed on them. */
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
  const activeSet = lines
    .map((line, index) => {
      const y = FIRST + index * LEADING
      const weight = index === 0 && leaf === 0 ? ' font-weight="600" letter-spacing="1.2"' : ''
      return `<text x="${MARGIN}" y="${y}" font-family="Georgia, serif" font-size="16"${weight} fill="#1b1b1b">${escapeHtml(line)}</text>`
    })
    .join('')
  const number = `<text x="${PAPER.wide / 2}" y="${PAPER.high - 54}" text-anchor="middle" font-family="Georgia, serif" font-size="13" fill="#5a5a5a">${page + 1}</text>`
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${PAPER.wide}" height="${PAPER.high}"><rect width="100%" height="100%" fill="#fbfaf7"/>${activeSet}${number}</svg>`
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
}

/** Where a run of a title stands, every time it stands there. */
const marks = (title: string, word: string): TextRange[] => {
  const spans: TextRange[] = []
  const lower = title.toLowerCase()
  for (let pos = lower.indexOf(word); pos >= 0; pos = lower.indexOf(word, pos + 1)) {
    spans.push({ start: pos, end: pos + word.length })
  }
  return spans
}

const TRAVEL = { id: 'travel', text: 'Show in plex' }
const READ = { id: 'read', text: 'Open note' }
const READ_AT = { id: 'readAt', text: 'Open at this heading' }
const READ_DOCUMENT = { id: 'readDocument', text: 'Open the document here' }

/** A note whose name carries the words: the name, and nothing under it. */
const createItem = (id: string, title: string): PaletteItem => ({
  id,
  title,
  at: marks(title, 'entrop'),
  actions: [TRAVEL, READ],
})

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
        'the entropy of a state is the logarithm of the number of arrangements it ' +
          'could have been made of',
        [READ_DOCUMENT],
      ),
      passage(
        't2',
        'The second law',
        'entropy never falls in a closed system, which is the whole of it stated ' + 'in one line',
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
        'clearing one bit of memory costs at least kT ln 2 of heat, which is what ' +
          'the sorting has to pay for',
        [READ, TRAVEL],
      ),
      passage(
        'm2',
        "Maxwell's demon",
        'a demon that sorts fast molecules from slow ones appears to lower the ' +
          'disorder of a gas for nothing',
        [READ, TRAVEL],
      ),
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

/** The commands over the window itself, offered whatever is in front. */
const OVER_WINDOW: PaletteGroup = {
  id: 'window',
  title: 'General',
  items: [
    command('note', 'New note', keyChord('n', KEYBOARD)),
    command('plex', 'New plex'),
    command('agent', 'New agent', keyChord('a', KEYBOARD, true)),
    command('appearance', 'Change theme'),
  ],
}

/** The commands over the vault, offered whatever is in front. */
const OVER_VAULT: PaletteGroup = {
  id: 'vault',
  title: 'Vault',
  items: [
    command('goto', 'Go to a note', keyChord('g', KEYBOARD)),
    command('openVault', 'Open vault'),
  ],
}

/** What the commands offer over a note, in the three groups the window draws them in. */
const COMMANDS: readonly PaletteGroup[] = [
  {
    id: 'note',
    title: 'Note',
    items: [
      command('read', 'Open note'),
      command('beside', 'Open beside'),
      command('travel', 'Show in plex', keyChord('p', KEYBOARD, true)),
      command('child', 'New child note', keyChord('c', KEYBOARD, true)),
      command('parent', 'New parent note'),
      command('ask', 'Ask the agent'),
    ],
  },
  OVER_WINDOW,
  OVER_VAULT,
]

/**
 * What the commands offer over a scanned book: the texts the plugins make of
 * the file, each by the plugin that makes it, above the window and the vault.
 */
const FILE_COMMANDS: readonly PaletteGroup[] = [
  {
    id: 'file',
    title: 'File',
    items: [
      command('recognise', 'Recognise the text with Document OCR'),
      command('proofread', 'Proofread the text with LLM Proofreading'),
    ],
  },
  OVER_WINDOW,
  OVER_VAULT,
]

interface Screen {
  /** How the window is divided, and which tab each pane shows. */
  readonly workspace: () => State
  readonly panel?: 'search' | 'commands' | null
  /** What the commands panel offers, where it is the panel drawn. */
  readonly commands?: readonly PaletteGroup[]
  /** The rows of the vault the selection relies on. */
  readonly selected?: readonly string[]
}

/** One arrangement of the window, drawn from the pieces above. */
const screen = ({ workspace, panel = null, commands = COMMANDS, selected = [] }: Screen) => ({
  components: { WorkspaceLayout, Palette, Tree },
  setup() {
    const target = ref<State>(workspace())
    const text = ref(MARKDOWN)
    const other = ref(OTHER)
    const searchQuery = ref(panel === 'search' ? 'entrop' : '')
    return {
      held: target,
      text,
      other,
      typed: searchQuery,
      panel,
      selected,
      groups: panel === 'commands' ? commands : GROUPS,
      placeholder: panel === 'commands' ? 'Type a command' : 'Search',
      book: renderPage(BOOK_FIRST),
      TABS,
      ROWS,
      OPEN,
      getIcon,
      iconOfTab,
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
          <textarea v-if="id === NOTE" v-model="text" aria-label="Note" class="h-full w-full resize-none bg-transparent p-6 font-sans text-base text-ink" />
          <textarea v-else-if="id === OTHER_NOTE" v-model="other" aria-label="Other note" class="h-full w-full resize-none bg-transparent p-6 font-sans text-base text-ink" />
          <section v-else-if="id === AGENT" class="h-full" aria-label="Agent" />
          <div
            v-else-if="id === BOOK"
            style="block-size: 100%; overflow: auto; display: flex; justify-content: center; padding: 24px"
          >
            <img :src="book" alt="Page 373 of Boltzmann 1877" style="inline-size: 100%; max-inline-size: 612px; block-size: auto; align-self: flex-start" />
          </div>
          <Tree
            v-else-if="id === FILES"
            :rows="ROWS"
            :open="OPEN"
            :selected="selected"
            name="Vault files"
          >
            <template #icon="{ id, open }">
              <component :is="getIcon(id, open)" class="size-4 shrink-0 text-hushed" />
            </template>
          </Tree>
        </template>
      </WorkspaceLayout>
      <Palette
        v-if="panel"
        v-model="typed"
        :groups="groups"
        is-open
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

/** One pane of the window, by the name it was divided under. */
const getPane = (canvas: HTMLElement, id: string): HTMLElement => {
  const element = canvas.querySelector<HTMLElement>(`[data-workspace-pane="${id}"]`)
  if (!element) throw new Error(`no pane ${id}`)
  return element
}

/** Whether one pane stands entirely past another's trailing edge. */
const isPast = (later: HTMLElement, earlier: HTMLElement): boolean =>
  later.getBoundingClientRect().left >= earlier.getBoundingClientRect().right - 1

export const Writing: Story = {
  render: () =>
    screen({
      workspace: () => ({
        root: branch(
          'root',
          [
            pane('main', [FILES]),
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
    }),
  play: async ({ canvasElement }) => {
    // Writing means somewhere to type: the note in the pane is text a person may
    // put a caret in.
    const note = await waitFor(() => within(canvasElement).getByRole('textbox', { name: 'Note' }))
    expect(note).not.toHaveAttribute('readonly')
  },
}

/** The palette, over everything the window holds. */
export const Searching: Story = {
  render: () =>
    screen({
      workspace: () => ({
        root: branch(
          'root',
          [pane('main', [FILES]), pane('middle', [NOTE, BOOK], NOTE)],
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
  render: () =>
    screen({
      workspace: () => ({
        root: branch(
          'root',
          [pane('main', [FILES]), pane('middle', [NOTE, BOOK], NOTE)],
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
      expect(groups.map((group) => group.textContent?.trim())).toEqual(['Note', 'General', 'Vault'])
    })
  },
}

/** The agent beside the note it is being asked about. */
export const Asking: Story = {
  render: () =>
    screen({
      workspace: () => ({
        root: branch(
          'root',
          [pane('source', [BOOK], BOOK), pane('middle', [NOTE], NOTE), pane('aside', [AGENT])],
          [0.4, 0.3, 0.3],
        ),
        axis: 'horizontal',
        focus: 'aside',
      }),
    }),
  play: async ({ canvasElement }) => {
    const source = getPane(canvasElement, 'source')
    const note = getPane(canvasElement, 'middle')
    const aside = getPane(canvasElement, 'aside')

    // The three stand in the order they were divided in, the agent last.
    expect(isPast(note, source)).toBe(true)
    expect(isPast(aside, note)).toBe(true)

    // The aside holds the agent's pane, which the views-agent-chat plugin draws.
    within(aside).getByRole('region', { name: 'Agent' })
  },
}

/**
 * The commands over a scanned book beside the files, offering to read its pages
 * with the plugin that recognises text.
 */
export const Recognising: Story = {
  render: () =>
    screen({
      workspace: () => ({
        root: branch(
          'root',
          [pane('aside', [FILES]), pane('main', [BOOK, NOTE], BOOK)],
          [0.26, 0.74],
        ),
        axis: 'horizontal',
        focus: 'main',
      }),
      panel: 'commands',
      commands: FILE_COMMANDS,
    }),
  play: async () => {
    // The file in front is a scan, so the commands open on what can be made of
    // it, before the window's own and the vault's.
    await waitFor(() => {
      const groups = [...document.body.querySelectorAll('[data-palette="title"]')]
      expect(groups.map((group) => group.textContent?.trim())).toEqual(['File', 'General', 'Vault'])
      expect(within(document.body).getByText('Recognise the text with Document OCR')).toBeVisible()
    })
  },
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
            branch(
              'middle',
              [pane('map', [OTHER_NOTE]), pane('note', [NOTE, BOOK], NOTE)],
              [0.52, 0.48],
            ),
            pane('aside', [AGENT]),
          ],
          [0.22, 0.5, 0.28],
        ),
        axis: 'horizontal',
        focus: 'map',
      }),
      selected: ['entropy', 'landauer'],
    }),
  play: async ({ canvasElement }) => {
    const tree = within(canvasElement).getByRole('tree', {
      name: 'Vault files',
    })
    const rows = within(tree).getAllByRole('treeitem')
    const selected = rows.filter((row) => row.getAttribute('aria-selected') === 'true')

    // The selection relies on two rows, and they are held in different
    // folders, which is the case a single run of rows would not cover.
    expect(selected.map((row) => row.textContent?.trim())).toEqual([
      'Entropy.md',
      "Landauer's principle.md",
    ])
    const folders = rows.filter((row) => within(row).queryByText(/^(Physics|Computation)$/))
    expect(folders).toHaveLength(2)
    for (const [at, folder] of folders.entries()) {
      expect(rows.indexOf(folder)).toBeLessThan(rows.indexOf(selected[at]!))
    }
  },
}
