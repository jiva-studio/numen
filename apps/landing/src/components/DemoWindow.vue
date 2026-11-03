<script setup lang="ts">
/**
 * Interactive landing-page simulation of the Numen desktop window, composed
 * from the real application components (`FilesTab`, `PlexGraphView`, `Editor`,
 * `Reader`, `RecordingTab`, `DeckTab`, `Agent`, `TabLinkIndicator`,
 * `TabLinkMenu`, `WorkspaceLayout`, `Palette`).
 */
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
  watch,
  type Component,
} from 'vue'
import {
  ArrowRightLeft,
  AudioLines,
  Book,
  BookOpen,
  Bot,
  Copy,
  CornerDownRight,
  CornerLeftUp,
  FilePlus,
  FileText,
  Folder,
  FolderOpen,
  FolderPlus,
  FolderTree,
  Layers,
  Link,
  PenLine,
  Plus,
  Trash2,
  Type,
  Waypoints,
} from '@lucide/vue'
import { keyChord, type PaletteKeys, type TextRange } from '@numen/ui'
import {
  Palette,
  WorkspaceLayout,
  branch,
  closeTab,
  commandKeyChord,
  getPanes,
  openTab,
  openTabAdjacent,
  pane,
  paneById,
} from '@numen/ui-shell'
import type {
  ActionWords,
  PaletteAction,
  PaletteGroup,
  PaletteItem,
  Tab,
  Workspace as State,
} from '@numen/ui-shell'
import {
  NOTE_TYPE,
  asFailure,
  asValue,
  isNotePath,
  type Connection,
  type MenuEntry,
  type PathMenuRow,
  type VaultEntry,
} from '@numen/plugin-sdk'
import {
  FilesTab,
  ROOT,
  createFileTree,
  useFilesTab,
  type FilesTabState,
} from '@plugins/views-file-tree/frontend'
import {
  PlexGraphView,
  createPlexTab,
  usePlexView,
  type Neighbour,
  type Neighbourhood,
  type NoteHeading,
} from '@plugins/views-plex-graph/frontend'
import type { PlexRelation } from '@plugins/views-plex-graph/frontend/plex'
import { Editor } from '@plugins/notes/frontend/editor'
import { Reader } from '@plugins/readers-pdf/frontend/reader'
import RecordingTab from '@plugins/views-media-player/frontend/RecordingTab.vue'
import UrlTab from './url-tab/UrlTab.vue'
import type { Cue } from '@plugins/views-media-player/frontend/lib/cues'
import { createAudioPlayer } from '@plugins/views-media-player/frontend/model/player'
import type {
  MenuEntry as MediaMenuEntry,
  Recordings,
} from '@plugins/views-media-player/frontend/types'
import DeckTab from '@plugins/views-flashcards-studio/frontend/deck/ui/DeckTab.vue'
import { createMarks } from '@plugins/views-flashcards-studio/frontend/cards'
import type { DeckTabState } from '@plugins/views-flashcards-studio/frontend/deck/types'
import type {
  CardModel,
  DeckCard,
  DeckSection,
} from '@plugins/views-flashcards-studio/frontend/ui/cards'
import { Agent } from '@plugins/views-agent-chat/frontend/agent'
import type { AgentModelOption } from '@plugins/views-agent-chat/frontend/message-composer'
import type { Turn } from '@plugins/views-agent-chat/frontend/thread'
import {
  TabLinkIndicator,
  TabLinkMenu,
  useTabLinkMenu,
  useTabLinks,
} from '@editor-features/tab-linking'

import '@numen/ui/styles.css'
import '@numen/ui-shell/styles.css'

/* --------------------------------- State ---------------------------------- */
const FILES_TAB = 'files'
const PLEX_TAB = 'plex:main'
const AGENT_TAB = 'agent:main'
const RECORDING_PATH = 'lectures/Lecture 4.mp3'
const YOUTUBE_LINK_PATH = 'lectures/Feynman 1964.md'
const DECK_PATH = 'decks/Thermodynamics.md'

interface VaultNoteRecord {
  path: string
  title: string
  folder: string
  body: string
  parents: string[]
  jumps: string[]
  isHiddenFromFolderList?: boolean
  headings?: readonly NoteHeading[]
}

const INITIAL_NOTES: readonly VaultNoteRecord[] = [
  {
    path: 'physics/Entropy.md',
    title: 'Entropy',
    folder: 'physics',
    parents: ['physics/Thermodynamics.md', 'computation/Information theory.md'],
    jumps: ["physics/Maxwell's demon.md", 'computation/Shannon entropy.md'],
    headings: [
      { line: 1, text: 'What it measures', level: 2 },
      { line: 3, text: 'Where the two meet', level: 2 },
      { line: 13, text: 'What is still open', level: 2 },
    ],
    body: `How many arrangements a state could have been made of, counted the same way in a gas and in a message. Boltzmann counted the microstates a temperature leaves open; Shannon counted the messages a channel could have sent, and the two counts came out as one expression.

## Where the two meet

A demon that sorts fast molecules from slow ones appears to lower the entropy of the gas for nothing. It does not: sorting means knowing which molecule is which, and the knowing is paid for when the memory holding it is cleared.

- Boltzmann, 1877 — *Über die Beziehung*, p. 373
- Shannon, 1948 — the same logarithm, in bits
- Landauer, 1961 — what clearing one bit costs

> The measure is not about heat. It is about how much a description leaves open.

## What is still open

Whether [[Free energy]] belongs under this note or beside it. The counting there assumes a temperature the surroundings hold fixed, and that assumption is nowhere above.
`,
  },
  {
    path: 'physics/Entropy of mixing.md',
    title: 'Entropy of mixing',
    folder: 'physics',
    parents: ['physics/Entropy.md'],
    jumps: ['physics/Free energy.md'],
    body: `When two ideal gases at the same temperature and pressure are allowed to mix by removing a partition, the total [[Entropy]] rises because each molecule now has the whole volume open to it.

## Gibbs paradox

If the two gases are identical, removing the partition changes nothing macroscopic. Counting arrangements requires dividing by $N!$ for indistinguishable particles.
`,
  },
  {
    path: 'physics/The second law.md',
    title: 'The second law',
    folder: 'physics',
    parents: ['physics/Entropy.md'],
    jumps: ['physics/Heat death.md'],
    headings: [
      { line: 3, text: 'As Clausius put it', level: 2 },
      { line: 7, text: 'As Kelvin put it', level: 2 },
      { line: 11, text: 'Why the two are one', level: 2 },
    ],
    body: `Entropy never falls in an isolated system, which is the whole law stated in one line.

## As Clausius put it

Heat can never pass from a colder to a warmer body without some other change, connected therewith, occurring at the same time.

## As Kelvin put it

No cyclic process can convert heat from a single reservoir entirely into mechanical work.

## Why the two are one

A system moves from macrostates with fewer microstates to macrostates with overwhelmingly more, because there are more of the latter to be reached.
`,
  },
  {
    path: 'physics/Free energy.md',
    title: 'Free energy',
    folder: 'physics',
    parents: ['physics/Entropy.md'],
    jumps: ["computation/Landauer's principle.md"],
    body: `When a system is held at fixed temperature $T$ by a heat bath, the quantity that falls at equilibrium is not energy alone and not negative [[Entropy]] alone, but the Helmholtz free energy:

$$F = U - T S$$

## Work extractable

The decrease $-\\Delta F$ bounds the maximum useful work that can be drawn from a system in thermal contact with its surroundings.
`,
  },
  {
    path: 'physics/Thermodynamics.md',
    title: 'Thermodynamics',
    folder: 'physics',
    parents: [],
    jumps: [],
    isHiddenFromFolderList: true,
    body: `The bookkeeping of heat, work, and the state variables of macroscopic matter.

## Laws and potentials

1. Conservation of total energy ($dU = \\delta Q - \\delta W$)
2. [[Entropy]] never decreases in an isolated system — [[The second law]]
3. [[Free energy]] measures the work extractable at constant temperature
`,
  },
  {
    path: 'computation/Information theory.md',
    title: 'Information theory',
    folder: 'computation',
    parents: [],
    jumps: ['reading/Shannon 1948.md'],
    isHiddenFromFolderList: true,
    body: `What a channel can carry, measured by the uncertainty removed when a message arrives.

## Core ideas

- [[Shannon entropy]] counts missing information in bits: $H(X) = -\\sum p(x) \\log_2 p(x)$
- [[Entropy]] in statistical mechanics and in coding share the same logarithmic form
- [[Landauer's principle]] ties logical erasure back to physical heat
`,
  },
  {
    path: "physics/Maxwell's demon.md",
    title: "Maxwell's demon",
    folder: 'physics',
    parents: ['physics/Thermodynamics.md'],
    jumps: ['physics/Entropy.md', "computation/Landauer's principle.md"],
    isHiddenFromFolderList: true,
    headings: [
      { line: 3, text: 'Why it looks free', level: 2 },
      { line: 7, text: 'Where the bill arrives', level: 2 },
    ],
    body: `A thought experiment of 1867: a being small enough to see single molecules stands at a door between two halves of a box, and lets the fast ones through one way and the slow ones the other.

## Why it looks free

Nothing is pushed and nothing is heated. The gas ends sorted, hot on one side and cold on the other, and no work appears to have been done to sort it.

## Where the bill arrives

The being has to know which molecule is which, and it has to hold that knowledge somewhere. The memory fills, and clearing it costs what [[Landauer's principle]] says a bit costs.

> The demon does not break the second law. It pays for the sorting in a currency nobody was counting.
`,
  },
  {
    path: 'computation/Shannon entropy.md',
    title: 'Shannon entropy',
    folder: 'computation',
    parents: ['computation/Information theory.md'],
    jumps: ['physics/Entropy.md', 'reading/Shannon 1948.md'],
    body: `The expected surprise of a discrete random variable, derived in 1948 as the unique measure satisfying continuity, monotonicity, and branching additivity.

## Entropy as missing information

For a source with distribution $P = (p_1, \\dots, p_n)$:

$$H(P) = -\\sum_{i=1}^n p_i \\log_2 p_i$$

When all outcomes are equally likely, $H$ reduces to $\\log_2 n$, matching Boltzmann's [[Entropy]] formula up to units.
`,
  },
  {
    path: "computation/Landauer's principle.md",
    title: "Landauer's principle",
    folder: 'computation',
    parents: ['physics/Entropy.md', 'computation/Information theory.md'],
    jumps: ["physics/Maxwell's demon.md", 'physics/Free energy.md'],
    body: `Clearing one bit of memory costs at least $k_B T \\ln 2$ of heat dissipated into the surroundings, which is what the sorting in [[Maxwell's demon]] has to pay for.

## Logical irreversibility

A 2-to-1 map such as \`RESET -> 0\` compresses two logical states into one. By Liouville's theorem, phase-space volume cannot shrink, so the compressed factor of 2 must be exported to the thermal bath as [[Entropy]].
`,
  },
  {
    path: "physics/Boltzmann's constant.md",
    title: "Boltzmann's constant",
    folder: 'physics',
    parents: ['physics/The second law.md'],
    jumps: [],
    isHiddenFromFolderList: true,
    body: `The bridge factor $k_B = 1.380649 \\times 10^{-23}\\,\\text{J/K}$ converting a pure count of arrangements ($\\ln W$) into thermodynamic units of energy per kelvin.
`,
  },
  {
    path: 'physics/Heat death.md',
    title: 'Heat death',
    folder: 'physics',
    parents: ['physics/The second law.md'],
    jumps: [],
    isHiddenFromFolderList: true,
    body: `The asymptotic state in which all gradients of temperature, pressure, and chemical potential have been spent, leaving maximum [[Entropy]] and no [[Free energy]] to drive further work.
`,
  },
  {
    path: 'Inbox.md',
    title: 'Inbox',
    folder: '',
    parents: [],
    jumps: [],
    body: `Quick notes waiting to be filed into the graph:

- Compare Jaynes' maximum-entropy principle with Gibbs ensembles
- Check page 373 of *Boltzmann 1877* on discrete energy cells
- Link [[Shannon entropy]] to channel capacity notes
`,
  },
  {
    path: 'Reading list.md',
    title: 'Reading list',
    folder: '',
    parents: ['physics/Thermodynamics.md'],
    jumps: ['reading/Shannon 1948.md'],
    body: `Primary sources in this vault:

1. **Boltzmann (1877)** — *Über die Beziehung zwischen dem zweiten Hauptsatze der mechanischen Wärmetheorie und der Wahrscheinlichkeitsrechnung*
2. **Shannon (1948)** — *A Mathematical Theory of Communication*
3. **Landauer (1961)** — *Irreversibility and Heat Generation in the Computing Process*
`,
  },
  {
    path: 'reading/Shannon 1948.md',
    title: 'Shannon 1948',
    folder: 'reading',
    parents: ['physics/Thermodynamics.md', 'computation/Information theory.md'],
    jumps: ['computation/Shannon entropy.md'],
    isHiddenFromFolderList: true,
    body: `Reading notes on Claude Shannon's *A Mathematical Theory of Communication* (Bell System Technical Journal, 1948).

See [[Shannon entropy]] and [[Information theory]].
`,
  },
]

const notesByPath = reactive<Record<string, VaultNoteRecord>>(
  Object.fromEntries(
    INITIAL_NOTES.map((n) => [
      n.path,
      {
        ...n,
        parents: [...n.parents],
        jumps: [...n.jumps],
      },
    ]),
  ),
)

const customFolders = ref<string[]>(['physics', 'computation', 'reading', 'lectures', 'decks'])

const EDGE_LABELS = reactive<Record<string, string>>({
  'physics/Thermodynamics.md->physics/Entropy.md': 'measures',
  'physics/Entropy.md->physics/The second law.md': 'states',
  'physics/Entropy.md->physics/Free energy.md': 'costs',
  'computation/Information theory.md->computation/Shannon entropy.md': 'defines',
  "physics/Maxwell's demon.md->physics/Entropy.md": 'tests',
  'computation/Shannon entropy.md->physics/Entropy.md': 'see also',
})

const PDF_DOCS: Readonly<
  Record<
    string,
    {
      title: string
      leaves: number
      firstPage: number
      pages: readonly (readonly string[])[]
    }
  >
> = {
  'reading/Boltzmann 1877.pdf': {
    title: 'Boltzmann 1877',
    leaves: 412,
    firstPage: 372,
    pages: [
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
      ],
    ],
  },
  'reading/Shannon 1948.pdf': {
    title: 'Shannon 1948',
    leaves: 55,
    firstPage: 10,
    pages: [
      [
        '§ 6.  CHOICE, UNCERTAINTY AND ENTROPY',
        '',
        'Suppose we have a set of possible events whose probabilities',
        'of occurrence are p_1, p_2, ..., p_n. These probabilities are',
        'known but that is all we know concerning which event will',
        'occur. Can we find a measure of how much "choice" is involved',
        'in the selection of the event or of how uncertain we are of',
        'the outcome?',
        '',
        'If there is such a measure, say H(p_1, p_2, ..., p_n), it is',
        'reasonable to require of it that H be continuous in the p_i,',
        'and if all the p_i are equal, p_i = 1/n, then H should be a',
        'monotonic increasing function of n.',
        '',
        'Theorem 2: The only H satisfying the three assumptions is of',
        'the form H = -K sum(p_i log p_i), where K is a positive',
        'constant that merely amounts to a choice of a unit of measure.',
      ],
      [
        'Quantities of the form H = -sum(p_i log p_i) play a central',
        'role in information theory as measures of information, choice',
        'and uncertainty. The form of H will be recognized as that of',
        'entropy as defined in certain formulations of statistical',
        'mechanics where p_i is the probability of a system being in',
        'cell i of its phase space.',
        '',
        "H is then, for example, the H in Boltzmann's famous H-theorem.",
        'We shall call H = -sum(p_i log p_i) the entropy of the set of',
        'probabilities p_1, ..., p_n.',
      ],
    ],
  },
}

const pdfPagesState = reactive<Record<string, number>>({
  'reading/Boltzmann 1877.pdf': PDF_DOCS['reading/Boltzmann 1877.pdf']!.firstPage,
  'reading/Shannon 1948.pdf': PDF_DOCS['reading/Shannon 1948.pdf']!.firstPage,
})

/* ------------------- Real `RecordingTab` (Media Player) ------------------- */
const LECTURE_MEDIA_URL = '/lecture-4.mp3'

const lectureCues = ref<Cue[]>([
  {
    from: 0,
    to: 4_780,
    text: 'To Sherlock Holmes she is always the woman.',
  },
  {
    from: 4_780,
    to: 9_580,
    text: 'I have seldom heard him mention her under any other name.',
  },
  {
    from: 9_580,
    to: 15_400,
    text: 'In his eyes she eclipses and predominates the whole of her sex.',
  },
  {
    from: 15_400,
    to: 23_400,
    text: 'It was not that he felt any emotion akin to love for Irene Adler.',
  },
  {
    from: 23_400,
    to: 32_520,
    text: 'All emotions, and that one particularly, were abhorrent to his cold, precise, but admirably balanced mind.',
  },
  {
    from: 32_520,
    to: 40_880,
    text: 'He was, I take it, the most perfect reasoning and observing machine that the world has seen,',
  },
  {
    from: 40_880,
    to: 45_880,
    text: 'but as a lover he would have placed himself in a false position.',
  },
  {
    from: 45_880,
    to: 51_720,
    text: 'He never spoke of the softer passions, save with a gibe and a sneer.',
  },
  {
    from: 51_720,
    to: 60_800,
    text: "They were admirable things for the observer — excellent for drawing the veil from men's motives and actions.",
  },
  {
    from: 60_800,
    to: 77_400,
    text: 'But for the trained reasoner to admit such intrusions into his own delicate and finely adjusted temperament was to introduce a distracting factor which might throw a doubt upon all his mental results.',
  },
  {
    from: 77_400,
    to: 92_320,
    text: 'Grit in a sensitive instrument, or a crack in one of his own high-power lenses, would not be more disturbing than a strong emotion in a nature such as his.',
  },
  {
    from: 92_320,
    to: 101_800,
    text: 'And yet there was but one woman to him, and that woman was the late Irene Adler, of dubious and questionable memory.',
  },
])

const TRANSCRIBED_MENU: readonly MediaMenuEntry[] = [
  {
    pluginId: 'md.numen.engines.transcription',
    pluginName: 'Transcription',
    command: 'delete-transcript',
    title: 'Delete transcript',
    icon: '',
    versionId: 'md.numen.engines.transcription/lecture-4',
    isLive: true,
  },
]

const YOUTUBE_EMBED_URL = 'https://www.youtube-nocookie.com/embed/kYXlvBiv9V4?enablejsapi=1&hl=en'
const youtubeCues = ref<Cue[]>([
  {
    from: 0,
    to: 6_480,
    text: 'A historian, although he talks about the past, can do it by talking about the future.',
  },
  {
    from: 6_480,
    to: 16_720,
    text: 'When he says Napoleon exists or Napoleon was, or that the French Revolution was in 1783,',
  },
  {
    from: 16_720,
    to: 24_520,
    text: "he means that if you look in another book about this French Revolution, you'll find the same date.",
  },
  {
    from: 24_560,
    to: 27_920,
    text: '1789 maybe.',
  },
  {
    from: 27_920,
    to: 35_239,
    text: "That's pretty accurate for a physicist to have the third decimal figure — the three figures.",
  },
  {
    from: 104_159,
    to: 110_880,
    text: "Now, it's obvious to everybody that the phenomena of the world are evidently irreversible.",
  },
  {
    from: 110_880,
    to: 113_840,
    text: "By that, I mean that things happen that don't happen the other way.",
  },
  {
    from: 113_840,
    to: 123_640,
    text: 'You drop a cup and it breaks, and you can sit there a long time waiting for the pieces to come together, come back into your hand.',
  },
  {
    from: 124_240,
    to: 136_400,
    text: 'If you watch the waves breaking at the sea, you stand there and wait for the great moment when the foam collects together, rises up out of the sea, and falls back further out from the shore.',
  },
  {
    from: 136_400,
    to: 139_360,
    text: 'Would be very pretty.',
  },
  {
    from: 139_360,
    to: 148_879,
    text: 'As a matter of fact, the demonstration of this in such lectures is usually made by having a section of moving picture in which you take a number of phenomena and running the thing backwards, and then see all the laughter.',
  },
  {
    from: 148_879,
    to: 154_160,
    text: "The laughter just means this ain't going to happen in the real world.",
  },
  {
    from: 154_160,
    to: 170_239,
    text: "But actually that's a kind of a weak way to put something which is so obvious and deep as the difference between the past and the future, because even without an experiment our very experiences inside are completely different for past and future.",
  },
  {
    from: 170_239,
    to: 172_800,
    text: "We remember the past; we don't remember the future.",
  },
  {
    from: 172_800,
    to: 180_000,
    text: 'We have a different kind of awareness about what might happen than we have of what most likely has happened.',
  },
  {
    from: 180_000,
    to: 184_560,
    text: 'The past and future look completely different psychologically, and so forth.',
  },
  {
    from: 184_560,
    to: 195_440,
    text: 'And the questions of memory and of apparent freedom of will, in the sense that we feel that we can do something to affect the future —',
  },
  {
    from: 195_440,
    to: 202_480,
    text: "but none of us, or very few of us, believe that there's anything you can do to affect the past.",
  },
  {
    from: 202_480,
    to: 212_560,
    text: 'And remorse and regret and hope and so forth are all words which distinguish perfectly obviously the past and the future.',
  },
  {
    from: 212_560,
    to: 232_640,
    text: 'Now if the world of nature is made of atoms, and we too are made of atoms that obey physical laws, this obvious distinction between what happened in the past and the future, and this obvious irreversibility of all phenomena, you would think would most likely have its interpretation in that some of the motion laws of the atoms are going one way —',
  },
  {
    from: 232_640,
    to: 246_720,
    text: "that the atom laws are not such that they can go either way, that there is somewhere in the works some kind of a principle that X's only make Y's and never vice versa.",
  },
  {
    from: 246_720,
    to: 260_639,
    text: 'And so the world is turning from X-like character to Y-like character all the time, and that this one-way business of the interactions of things is the thing that makes the whole phenomena of the world seem to go one way.',
  },
  {
    from: 260_639,
    to: 271_680,
    text: "And yet we haven't found it yet. That is, in all the laws of physics that we have found so far, there doesn't seem to be any distinction of the past and the future.",
  },
  {
    from: 271_680,
    to: 280_479,
    text: 'The moving picture should work the same way going both ways, and the physicist who looks at it should not laugh. Details now to be explained.',
  },
  {
    from: 280_479,
    to: 296_560,
    text: "Now let's take the law of gravitation as our standard example. If I have a sun and a planet that I started off in some direction going around to here, and then take a moving picture of this — say it gets to here.",
  },
  {
    from: 296_560,
    to: 307_280,
    text: 'Now take a movie picture of it, excuse me, and run the movie backward and look at it. What happened? The planet goes around the sun in an ellipse.',
  },
])

const demoAudioPlayer = createAudioPlayer()

function getCuesRefForPath(path: string) {
  return path === YOUTUBE_LINK_PATH ? youtubeCues : lectureCues
}

const demoRecordings: Recordings = {
  getSource: async (path) =>
    path === YOUTUBE_LINK_PATH
      ? { mediaUrl: YOUTUBE_EMBED_URL, mediaType: 'text/html' }
      : { mediaUrl: LECTURE_MEDIA_URL, mediaType: 'audio/mpeg' },
  readTranscript: async (path) => getCuesRefForPath(path).value,
  writeTranscript: async (path, text) => {
    const targetCues = getCuesRefForPath(path)
    const paragraphs = text.split(/\n\n+/).filter(Boolean)
    targetCues.value = paragraphs.map((p, index) => {
      const existing = targetCues.value[index]
      const fallbackFrom = index === 0 ? 0 : (targetCues.value[index - 1]?.to ?? index * 8_000)
      return {
        text: p,
        from: existing?.from ?? fallbackFrom,
        to: existing?.to ?? fallbackFrom + 8_000,
      }
    })
  },
  findCueTime: async (path, offset) => {
    const list = getCuesRefForPath(path).value
    let charPos = 0
    for (const cue of list) {
      const endPos = charPos + cue.text.length
      if (offset <= endPos) return cue.from
      charPos = endPos + 2
    }
    return list[0]?.from ?? 0
  },
  getMenu: async () => TRANSCRIBED_MENU,
  runMenuEntry: async (path) => {
    getCuesRefForPath(path).value = []
  },
  watchWork: (_path, onChange) => {
    onChange(false)
    return () => {}
  },
  watchChanges: () => () => {},
}

const recordingHow = {
  audioPlayer: demoAudioPlayer,
  plays: () => true,
}

/* ------------------- Real `DeckTab` (Flashcards Studio) ------------------- */
const CONCEPT_MODEL: CardModel = {
  name: 'Concept',
  fields: ['Concept', 'Formula', 'Meaning', 'Source'],
}
const LAW_MODEL: CardModel = {
  name: 'Law',
  fields: ['Law', 'Statement', 'Consequence'],
}

const deckCardsList = ref<DeckCard[]>([
  {
    id: 'boltzmann-entropy',
    section: 'entropy',
    cardModel: 'Concept',
    filled: [
      { field: 'Concept', text: 'Boltzmann entropy' },
      { field: 'Formula', text: 'S = k_B ln W' },
      { field: 'Meaning', text: 'Logarithm of the microstates consistent with a macrostate' },
      { field: 'Source', text: 'Boltzmann, 1877 — p. 373' },
    ],
  },
  {
    id: 'shannon-entropy',
    section: 'entropy',
    cardModel: 'Concept',
    filled: [
      { field: 'Concept', text: 'Shannon entropy' },
      { field: 'Formula', text: 'H(X) = -∑ p(x) log₂ p(x)' },
      { field: 'Meaning', text: 'Expected bits of uncertainty removed when a message arrives' },
      { field: 'Source', text: 'Shannon, 1948 — § 6' },
    ],
  },
  {
    id: 'helmholtz-free-energy',
    section: 'entropy',
    cardModel: 'Concept',
    filled: [
      { field: 'Concept', text: 'Helmholtz free energy' },
      { field: 'Formula', text: 'F = U - T S' },
      { field: 'Meaning', text: 'Maximum work extractable at constant temperature T' },
      { field: 'Source', text: 'Thermodynamics' },
    ],
  },
  {
    id: 'landauer-bound',
    section: 'erasure',
    cardModel: 'Law',
    filled: [
      { field: 'Law', text: "Landauer's principle" },
      { field: 'Statement', text: 'Erasing 1 bit of information dissipates at least k_B T ln 2' },
      { field: 'Consequence', text: "Pays the entropy bill run up by Maxwell's demon" },
    ],
  },
  {
    id: 'second-law-card',
    section: 'erasure',
    cardModel: 'Law',
    filled: [
      { field: 'Law', text: 'Second law of thermodynamics' },
      { field: 'Statement', text: 'ΔS ≥ 0 in any isolated system' },
      { field: 'Consequence', text: 'No description improves its own precision for nothing' },
    ],
  },
])

const deckSectionsList = ref<DeckSection[]>([
  { id: 'entropy', name: 'Counting arrangements' },
  { id: 'erasure', name: 'Information & erasure' },
])

const deckScheduledPreset = ref({
  path: 'Physics.md',
  name: 'Physics',
  errorMessage: '',
})

const DECK_CHOICES = [
  { path: '', name: 'Defaults' },
  { path: 'Physics.md', name: 'Physics' },
  { path: 'Computation.md', name: 'Computation' },
]

let nextCardNum = 1
const deckState: DeckTabState = {
  id: DECK_PATH,
  note: computed(() => ({ path: DECK_PATH, body: '', state: 'clean', error: null })),
  deck: computed(() => ({ preamble: '', cards: [], sections: [], tail: '' })),
  deckCards: computed(() => deckCardsList.value),
  sections: computed(() => deckSectionsList.value),
  cardModels: computed(() => [CONCEPT_MODEL, LAW_MODEL]),
  marks: computed(() => createMarks([], [], [])),
  errorMessage: computed(() => ''),
  scheduled: computed(() => deckScheduledPreset.value),
  choices: computed(() => DECK_CHOICES),
  setSchedule: (presetPath: string) => {
    const found = DECK_CHOICES.find((c) => c.path === presetPath)
    if (found) {
      deckScheduledPreset.value = { path: found.path, name: found.name, errorMessage: '' }
    }
  },
  addCard: (cardModel, values, section) => {
    nextCardNum += 1
    deckCardsList.value = [
      ...deckCardsList.value,
      {
        id: `card-${nextCardNum}`,
        section: section ?? deckSectionsList.value[0]?.id ?? 'entropy',
        cardModel,
        filled: [...values],
      },
    ]
  },
  removeCard: (cardId) => {
    deckCardsList.value = deckCardsList.value.filter((c) => c.id !== cardId)
  },
  moveCard: (cardId, to) => {
    const moving = deckCardsList.value.find((c) => c.id === cardId)
    if (!moving) return
    const rest = deckCardsList.value.filter((c) => c.id !== cardId)
    const updated: DeckCard = { ...moving, section: to.section }
    if (to.before !== null) {
      const idx = rest.findIndex((c) => c.id === to.before)
      if (idx >= 0) {
        deckCardsList.value = [...rest.slice(0, idx), updated, ...rest.slice(idx)]
        return
      }
    }
    deckCardsList.value = [...rest, updated]
  },
  writeCardField: (cardId, field, _index, text) => {
    deckCardsList.value = deckCardsList.value.map((card) => {
      if (card.id !== cardId) return card
      const exists = card.filled.some((f) => f.field === field)
      const filled = exists
        ? card.filled.map((f) => (f.field === field ? { ...f, text } : f))
        : [...card.filled, { field, text }]
      return { ...card, filled }
    })
  },
  addSection: (name) => {
    nextCardNum += 1
    deckSectionsList.value = [
      ...deckSectionsList.value,
      { id: `sec-${nextCardNum}`, name: name.trim() || 'New section' },
    ]
  },
  renameSection: (sectionId, name) => {
    deckSectionsList.value = deckSectionsList.value.map((s) =>
      s.id === sectionId ? { ...s, name } : s,
    )
  },
  removeSection: (sectionId) => {
    deckSectionsList.value = deckSectionsList.value.filter((s) => s.id !== sectionId)
    deckCardsList.value = deckCardsList.value.filter((c) => c.section !== sectionId)
  },
  keepMine: () => {},
  takeFile: () => {},
  close: () => {},
}

const PRIMARY_NOTE_TAB = 'note:1'
let nextNoteTabCount = 1
const noteTabPaths = reactive<Record<string, string>>({
  [PRIMARY_NOTE_TAB]: 'physics/Entropy.md',
})

function findNoteTabAt(path: string): string | undefined {
  return openTabIds.value.find((id) => id.startsWith('note:') && noteTabPaths[id] === path)
}

function createNoteTabForPath(path: string): string {
  nextNoteTabCount += 1
  const id = `note:${nextNoteTabCount}`
  noteTabPaths[id] = path
  return id
}

function getPdfTabId(path: string): string {
  return `reader:pdf:${path}`
}

function getRecordingTabId(path: string): string {
  return `recording:${path}`
}

function getLinkTabId(path: string): string {
  return `link:${path}`
}

function getDeckTabId(path: string): string {
  return `deck:${path}`
}

const openTabIds = ref<string[]>([
  FILES_TAB,
  PLEX_TAB,
  AGENT_TAB,
  PRIMARY_NOTE_TAB,
  getPdfTabId('reading/Boltzmann 1877.pdf'),
  getRecordingTabId(RECORDING_PATH),
  getLinkTabId(YOUTUBE_LINK_PATH),
  getDeckTabId(DECK_PATH),
])

const draggedPaths = ref<readonly string[]>([])

let nextPaneCount = 1
function createPaneId(): string {
  nextPaneCount += 1
  return `pane-${nextPaneCount}`
}

function createInitialLayout(): State {
  const entropyTab = PRIMARY_NOTE_TAB
  const boltzmannTab = getPdfTabId('reading/Boltzmann 1877.pdf')
  const recordingTab = getRecordingTabId(RECORDING_PATH)
  const youtubeTab = getLinkTabId(YOUTUBE_LINK_PATH)
  const deckTab = getDeckTabId(DECK_PATH)
  return {
    root: branch(
      'root',
      [
        pane('main', [FILES_TAB]),
        branch(
          'middle',
          [
            pane('map', [PLEX_TAB]),
            pane(
              'note',
              [entropyTab, boltzmannTab, recordingTab, youtubeTab, deckTab],
              entropyTab,
            ),
          ],
          [0.52, 0.48],
        ),
        pane('aside', [AGENT_TAB]),
      ],
      [0.22, 0.5, 0.28],
    ),
    axis: 'horizontal',
    focus: 'map',
  }
}

const held = ref<State>(createInitialLayout())

const editorRefs = new Map<string, InstanceType<typeof Editor>>()
const readerRefs = new Map<string, InstanceType<typeof Reader>>()
const agentRef = ref<InstanceType<typeof Agent> | null>(null)

function setEditorRef(tabId: string) {
  return (el: unknown) => {
    if (el) editorRefs.set(tabId, el as InstanceType<typeof Editor>)
    else editorRefs.delete(tabId)
  }
}

function setReaderRef(tabId: string) {
  return (el: unknown) => {
    if (el) readerRefs.set(tabId, el as InstanceType<typeof Reader>)
    else readerRefs.delete(tabId)
  }
}

/* ------------------- Real Tab Linking (`useTabLinks`) --------------------- */
const tabLinks = useTabLinks()
// Pre-link Files, Plex, and the primary Note tab into one synchronized group.
tabLinks.linkTabs(FILES_TAB, PLEX_TAB)
tabLinks.linkTabs(PLEX_TAB, PRIMARY_NOTE_TAB)

function getKindOfTab(tabId: string): string | undefined {
  if (tabId === FILES_TAB) return 'files'
  if (tabId === PLEX_TAB) return 'plex'
  if (tabId === AGENT_TAB) return 'agent'
  if (tabId.startsWith('note:')) return 'note'
  if (tabId.startsWith('reader:pdf:')) return 'reader:pdf'
  if (tabId.startsWith('recording:')) return 'recording'
  if (tabId.startsWith('link:')) return 'link'
  if (tabId.startsWith('deck:')) return 'deck'
  return undefined
}

function isTabLinkable(tabId: string): boolean {
  return tabId === FILES_TAB || tabId === PLEX_TAB || tabId.startsWith('note:')
}

const tabs = computed<readonly Tab[]>(() =>
  openTabIds.value.map((id) => {
    if (id === FILES_TAB) return { id, title: 'Files' }
    if (id === PLEX_TAB) {
      const focusTitle = plexState.view.neighbourhood.value?.focus.title
      return { id, title: focusTitle || 'Plex' }
    }
    if (id === AGENT_TAB) return { id, title: 'Agent' }
    if (id.startsWith('note:')) {
      const path = noteTabPaths[id] ?? 'physics/Entropy.md'
      return {
        id,
        title: notesByPath[path]?.title ?? path.split('/').pop()?.replace(/\.md$/, '') ?? path,
      }
    }
    if (id.startsWith('reader:pdf:')) {
      const path = id.slice(11)
      return { id, title: PDF_DOCS[path]?.title ?? path.split('/').pop() ?? path }
    }
    if (id.startsWith('recording:')) {
      return { id, title: 'Lecture 4' }
    }
    if (id.startsWith('link:')) {
      return { id, title: 'Feynman 1964' }
    }
    if (id.startsWith('deck:')) {
      return { id, title: 'Thermodynamics' }
    }
    return { id, title: id }
  }),
)

const linkMenu = useTabLinkMenu({
  tabLinks,
  getTabs: () => tabs.value,
  getKind: getKindOfTab,
  isLinkable: isTabLinkable,
})

let isSyncingTabs = false

function syncPathFromTab(sourceTabId: string, path: string, _title = ''): boolean {
  if (!sourceTabId || !path || isSyncingTabs) return false
  const targets = tabLinks.getLinkedTargets(sourceTabId)
  if (targets.length === 0) return false

  isSyncingTabs = true
  try {
    for (const targetId of targets) {
      if (targetId === PLEX_TAB && isNotePath(path) && notesByPath[path]) {
        if (plexState.view.here.value !== path) {
          void plexState.view.go(path)
        }
      } else if (targetId === FILES_TAB) {
        void fileTree.revealPath(path)
      } else if (targetId.startsWith('note:') && isNotePath(path) && notesByPath[path]) {
        navigateNoteTab(targetId, path)
      }
    }
    return true
  } finally {
    isSyncingTabs = false
  }
}

function navigateNoteTab(targetTabId: string, nextPath: string) {
  if (noteTabPaths[targetTabId] === nextPath) return
  noteTabPaths[targetTabId] = nextPath
  onShowWorkspaceTab(targetTabId)
}

/* --------------------- Real `FilesTab` & `useFileTree` -------------------- */
const COMMAND_ICONS: Readonly<Record<string, Component>> = {
  read: FileText,
  beside: FileText,
  reveal: FolderOpen,
  travel: Waypoints,
  child: CornerDownRight,
  parent: CornerLeftUp,
  jump: ArrowRightLeft,
  title: Type,
  copy: Copy,
  remove: Trash2,
  new: Plus,
  relation: Waypoints,
  newNote: FilePlus,
  newFolder: FolderPlus,
  rename: PenLine,
  ask: Bot,
}

function listFolderEntries(folder: string): readonly VaultEntry[] {
  const result: VaultEntry[] = []

  if (folder === ROOT) {
    for (const f of customFolders.value) {
      if (!f.includes('/')) {
        const displayName = f.charAt(0).toUpperCase() + f.slice(1)
        result.push({
          path: f,
          name: displayName,
          isFolder: true,
          isNote: false,
          type: NOTE_TYPE,
        })
      }
    }
  } else {
    const prefix = `${folder}/`
    for (const f of customFolders.value) {
      if (f.startsWith(prefix) && !f.slice(prefix.length).includes('/')) {
        result.push({
          path: f,
          name: f.slice(prefix.length),
          isFolder: true,
          isNote: false,
          type: NOTE_TYPE,
        })
      }
    }
  }

  for (const note of Object.values(notesByPath)) {
    if (note.folder === folder && !note.isHiddenFromFolderList) {
      result.push({
        path: note.path,
        name: note.path.split('/').pop() ?? `${note.title}.md`,
        isFolder: false,
        isNote: true,
        type: NOTE_TYPE,
      })
    }
  }

  if (folder === 'reading') {
    for (const pdfPath of Object.keys(PDF_DOCS)) {
      result.push({
        path: pdfPath,
        name: pdfPath.split('/').pop() ?? pdfPath,
        isFolder: false,
        isNote: false,
        type: NOTE_TYPE,
      })
    }
  }

  if (folder === 'lectures') {
    result.push({
      path: RECORDING_PATH,
      name: 'Lecture 4.mp3',
      isFolder: false,
      isNote: false,
      type: NOTE_TYPE,
    })
    result.push({
      path: YOUTUBE_LINK_PATH,
      name: 'Feynman 1964.md',
      isFolder: false,
      isNote: true,
      type: 'link',
    })
  }

  if (folder === 'decks') {
    result.push({
      path: DECK_PATH,
      name: 'Thermodynamics.md',
      isFolder: false,
      isNote: true,
      type: 'deck',
    })
  }

  return result
}

function getPathCommandsForRow({ entry, isSeveral }: PathMenuRow): readonly MenuEntry[] {
  const canRemoveEntry = Boolean(notesByPath[entry.path])
  const removeItem: MenuEntry = {
    id: 'remove',
    text: entry.isNote && !isSeveral ? 'Remove note' : 'Remove',
    group: 'remove',
  }
  const copyItem: MenuEntry = { id: 'copy', text: 'Copy path', group: 'file' }
  if (isSeveral) {
    const selected = fileTree.selectedPaths.value
    const allRemovable = selected.length > 0 && selected.every((p) => Boolean(notesByPath[p]))
    return allRemovable ? [removeItem] : []
  }
  if (entry.isFolder) return [copyItem]
  if (entry.isNote) {
    const items: MenuEntry[] = [{ id: 'read', text: 'Open the note', group: 'open' }]
    if (notesByPath[entry.path]) {
      items.push({ id: 'travel', text: 'Show in plex', group: 'open' })
    }
    items.push(
      copyItem,
      {
        id: 'relation',
        text: 'Relation',
        group: 'relation',
        children: [
          { id: 'child', text: 'New child note', group: 'relation' },
          { id: 'parent', text: 'New parent note', group: 'relation' },
          { id: 'jump', text: 'New jump note', group: 'relation' },
        ],
      },
      { id: 'ask', text: 'Ask the agent', group: 'agent' },
    )
    if (canRemoveEntry) items.push(removeItem)
    return items
  }
  return [
    { id: 'read', text: 'Open the file', group: 'open' },
    copyItem,
  ]
}

const fileTree = createFileTree({
  list: async (folder: string) => listFolderEntries(folder),
})

let untitledCounter = 0

function getNextUntitledTitle(folder: string): string {
  for (let i = 1; i < 1000; i += 1) {
    const candidate = i === 1 ? 'Untitled note' : `Untitled note ${i}`
    const candidatePath = folder ? `${folder}/${candidate}.md` : `${candidate}.md`
    if (!notesByPath[candidatePath]) return candidate
  }
  untitledCounter += 1
  return `Untitled note ${untitledCounter}`
}

function rekeySingleNotePath(from: string, to: string) {
  const note = notesByPath[from]
  if (!note) return
  const parts = to.split('/')
  const filename = parts.pop() ?? to
  const folder = parts.join('/')
  const newTitle = filename.replace(/\.md$/, '')
  delete notesByPath[from]
  notesByPath[to] = {
    ...note,
    path: to,
    title: newTitle,
    folder,
  }
  for (const other of Object.values(notesByPath)) {
    other.parents = other.parents.map((p) => (p === from ? to : p))
    other.jumps = other.jumps.map((j) => (j === from ? to : j))
  }
  for (const [edgeKey, label] of Object.entries(EDGE_LABELS)) {
    const [src, dst] = edgeKey.split('->')
    if (src === from || dst === from) {
      delete EDGE_LABELS[edgeKey]
      const nextSrc = src === from ? to : src
      const nextDst = dst === from ? to : dst
      if (nextSrc && nextDst) {
        EDGE_LABELS[`${nextSrc}->${nextDst}`] = label
      }
    }
  }
  for (const [tabId, tabPath] of Object.entries(noteTabPaths)) {
    if (tabPath === from) {
      noteTabPaths[tabId] = to
    }
  }
  if (plexState.view.here.value === from) {
    void plexState.view.go(to)
  }
}

const filesState: FilesTabState = useFilesTab(fileTree, {
  openEntry: ({ path }) => {
    openVaultFile(path, 'current')
  },
  onSelectPath: (path, state) => {
    const entry = state.list.getEntryAt(path)
    if (entry && !entry.isFolder && entry.type === NOTE_TYPE && (entry.isNote || isNotePath(path))) {
      syncPathFromTab(FILES_TAB, path, entry.name)
    }
  },
  runCommand: (command, paths) => {
    if (command === 'remove') {
      for (const p of paths) {
        runVaultCommand(command, p, 'files')
      }
      return
    }
    const first = paths[0]
    if (!first) return
    runVaultCommand(command, first, 'files')
  },
  movePath: async (from, to) => {
    if (notesByPath[from]) {
      rekeySingleNotePath(from, to)
      return
    }
    const prefix = `${from}/`
    customFolders.value = customFolders.value.map((f) => {
      if (f === from) return to
      if (f.startsWith(prefix)) return `${to}/${f.slice(prefix.length)}`
      return f
    })
    for (const oldPath of Object.keys(notesByPath)) {
      if (oldPath.startsWith(prefix)) {
        const newPath = `${to}/${oldPath.slice(prefix.length)}`
        rekeySingleNotePath(oldPath, newPath)
      }
    }
  },
  setDraggedPaths: (paths) => {
    draggedPaths.value = paths
  },
  createFolder: async (folderPath) => {
    if (!customFolders.value.includes(folderPath)) {
      customFolders.value = [...customFolders.value, folderPath]
    }
  },
  createNote: async (folder) => {
    const title = getNextUntitledTitle(folder)
    const path = folder ? `${folder}/${title}.md` : `${title}.md`
    notesByPath[path] = {
      path,
      title,
      folder,
      parents: [],
      jumps: [],
      body: `Write Markdown here — headings, lists, math, and [[wikilinks]] format live as you move the caret.\n`,
    }
    await fileTree.refresh()
    return path
  },
  canDropData: () => false,
  dropData: async () => [],
  showError: () => {},
  getPathCommands: getPathCommandsForRow,
  getCommandIcon: (id) => COMMAND_ICONS[id] ?? null,
  getEntryIcon: (entry, isOpen) => {
    if (entry.isFolder) return isOpen ? FolderOpen : Folder
    if (entry.type === 'deck') return Layers
    if (entry.type === 'link') return Link
    if (entry.path.endsWith('.mp3')) return AudioLines
    if (entry.path.endsWith('.pdf')) return Book
    return FileText
  },
})

void (async () => {
  await fileTree.openFolder(ROOT)
  await fileTree.openFolder('physics')
  await fileTree.openFolder('computation')
  await fileTree.openFolder('reading')
  await fileTree.openFolder('lectures')
  await fileTree.openFolder('decks')
  fileTree.selectPaths(['physics/Entropy.md'])
})()

watch(fileTree.selectedPaths, (paths) => {
  const first = paths[0]
  if (!first) return
  const entry = fileTree.getEntryAt(first)
  if (entry && !entry.isFolder && entry.type === NOTE_TYPE && (entry.isNote || isNotePath(first))) {
    syncPathFromTab(FILES_TAB, first, entry.name)
  }
})

/* ------------------- Real `PlexGraphView` & `usePlexTab` ------------------ */
function getParentsOf(path: string): readonly string[] {
  return notesByPath[path]?.parents ?? []
}

function getChildrenOf(path: string): string[] {
  return Object.keys(notesByPath).filter((other) => getParentsOf(other).includes(path))
}

function getJumpsOf(path: string): string[] {
  const mine = notesByPath[path]?.jumps ?? []
  const theirs = Object.keys(notesByPath).filter((other) =>
    (notesByPath[other]?.jumps ?? []).includes(path),
  )
  return [...new Set([...mine, ...theirs])]
}

function getSiblingsOf(path: string): { path: string; through: string }[] {
  const result: { path: string; through: string }[] = []
  const seen = new Set<string>([
    path,
    ...getParentsOf(path),
    ...getChildrenOf(path),
    ...getJumpsOf(path),
  ])
  for (const parent of getParentsOf(path)) {
    for (const child of getChildrenOf(parent)) {
      if (!seen.has(child)) {
        seen.add(child)
        result.push({ path: child, through: parent })
      }
    }
  }
  return result
}

function buildVaultNeighbourhood(focusPath: string): Neighbourhood {
  const resolvedPath = notesByPath[focusPath] ? focusPath : 'physics/Entropy.md'
  const focusNote = notesByPath[resolvedPath]!
  const related: Neighbour[] = []
  const connections: Connection[] = []
  const seen = new Set<string>([resolvedPath])

  const pushNeighbour = (path: string, relation: PlexRelation, through = '') => {
    if (seen.has(path) || !notesByPath[path]) return
    seen.add(path)
    const note = notesByPath[path]!
    const label =
      relation === 'parent' || relation === 'jump'
        ? (EDGE_LABELS[`${path}->${resolvedPath}`] ?? EDGE_LABELS[`${resolvedPath}->${path}`] ?? '')
        : relation === 'child'
          ? (EDGE_LABELS[`${resolvedPath}->${path}`] ?? '')
          : ''
    related.push({
      path,
      title: note.title,
      relation,
      type: NOTE_TYPE,
    })
    const isNamedFirst = relation === 'child' || relation === 'sibling'
    const named = relation === 'sibling' ? through : resolvedPath
    const side = { label, marker: 'arrow' as const, reason: '' }
    connections.push({
      first: isNamedFirst ? named : path,
      second: isNamedFirst ? path : named,
      role: relation === 'jump' ? 'jump' : 'child',
      type: '',
      firstSide: isNamedFirst ? side : null,
      secondSide: isNamedFirst ? null : side,
    })
  }

  for (const p of getParentsOf(resolvedPath)) pushNeighbour(p, 'parent')
  for (const c of getChildrenOf(resolvedPath)) pushNeighbour(c, 'child')
  for (const j of getJumpsOf(resolvedPath)) pushNeighbour(j, 'jump')
  for (const s of getSiblingsOf(resolvedPath).slice(0, 2)) pushNeighbour(s.path, 'sibling', s.through)

  return {
    focus: { path: resolvedPath, title: focusNote.title },
    focusType: NOTE_TYPE,
    related,
    connections,
  }
}

const plexView = usePlexView({
  neighbourhood: async (path: string) => buildVaultNeighbourhood(path),
})

function joinNotesInVault(
  fromPath: string,
  toPath: string,
  relation: PlexRelation,
  label?: string,
): boolean {
  const fromNote = notesByPath[fromPath]
  const toNote = notesByPath[toPath]
  if (!fromNote || !toNote || fromPath === toPath) return false

  if (relation === 'child') {
    if (!toNote.parents.includes(fromPath)) toNote.parents = [...toNote.parents, fromPath]
    if (label) EDGE_LABELS[`${fromPath}->${toPath}`] = label
  } else if (relation === 'parent') {
    if (!fromNote.parents.includes(toPath)) fromNote.parents = [...fromNote.parents, toPath]
    if (label) EDGE_LABELS[`${toPath}->${fromPath}`] = label
  } else if (relation === 'jump') {
    if (!fromNote.jumps.includes(toPath)) fromNote.jumps = [...fromNote.jumps, toPath]
    if (label) EDGE_LABELS[`${fromPath}->${toPath}`] = label
  }
  return true
}

function createNoteInRelationInVault(
  title: string,
  fromPath: string,
  relation: PlexRelation,
): { path: string; title: string } {
  const fromNote = notesByPath[fromPath] ?? notesByPath['physics/Entropy.md']!
  const folder = fromNote.folder || 'physics'
  const cleanTitle = title.trim() || getNextUntitledTitle(folder)
  const path = folder ? `${folder}/${cleanTitle}.md` : `${cleanTitle}.md`

  notesByPath[path] = {
    path,
    title: cleanTitle,
    folder,
    parents: relation === 'child' ? [fromPath] : [],
    jumps: relation === 'jump' ? [fromPath] : [],
    body: `Created as a **${relation}** of [[${fromNote.title}]].\n\nWrite Markdown here — headings, lists, math, and [[wikilinks]] format live as you move the caret.\n`,
  }

  if (relation === 'parent') {
    fromNote.parents = [...fromNote.parents, path]
  }

  void fileTree.refresh()
  if (plexView.here.value) {
    void plexView.go(plexView.here.value)
  }
  return { path, title: cleanTitle }
}

const plexState = createPlexTab(plexView, {
  editor: {
    createRelated: async ({ from, role, title }) => {
      const relation: PlexRelation = role === 'parent' || role === 'jump' ? role : 'child'
      const created = createNoteInRelationInVault(title ?? '', from, relation)
      return asValue(created.path)
    },
    join: async (fromPath, toPath, side) => {
      const relation: PlexRelation =
        side.role === 'parent' || side.role === 'jump' ? side.role : 'child'
      return joinNotesInVault(fromPath, toPath, relation, side.label)
        ? asValue(undefined)
        : asFailure('missing')
    },
    getConnection: async (first, second, type = '') => {
      const firstLabel = EDGE_LABELS[`${first}->${second}`]
      const secondLabel = EDGE_LABELS[`${second}->${first}`]
      const isFirstParent = getParentsOf(second).includes(first)
      const isSecondParent = getParentsOf(first).includes(second)
      const isJump =
        (notesByPath[first]?.jumps ?? []).includes(second) ||
        (notesByPath[second]?.jumps ?? []).includes(first)
      const role = isFirstParent
        ? 'child'
        : isSecondParent
          ? 'parent'
          : isJump
            ? 'jump'
            : null
      return asValue({
        first,
        second,
        role,
        type,
        firstSide:
          isFirstParent || isJump
            ? { label: firstLabel ?? '', marker: 'arrow', reason: '' }
            : null,
        secondSide: isSecondParent
          ? { label: secondLabel ?? '', marker: 'arrow', reason: '' }
          : null,
      })
    },
    saveConnection: async (connection) => {
      const { first, second } = connection
      const noteA = notesByPath[first]
      const noteB = notesByPath[second]
      if (!noteA || !noteB) return asFailure('missing')
      noteA.parents = noteA.parents.filter((p) => p !== second)
      noteA.jumps = noteA.jumps.filter((j) => j !== second)
      noteB.parents = noteB.parents.filter((p) => p !== first)
      noteB.jumps = noteB.jumps.filter((j) => j !== first)
      delete EDGE_LABELS[`${first}->${second}`]
      delete EDGE_LABELS[`${second}->${first}`]
      if (connection.role === 'child') {
        joinNotesInVault(first, second, 'child', connection.firstSide?.label)
      } else if (connection.role === 'parent') {
        joinNotesInVault(first, second, 'parent', connection.secondSide?.label)
      } else if (connection.role === 'jump') {
        joinNotesInVault(first, second, 'jump', connection.firstSide?.label)
      }
      return asValue(undefined)
    },
    removeConnection: async (first, second) => {
      const noteA = notesByPath[first]
      const noteB = notesByPath[second]
      if (!noteA || !noteB) return asFailure('missing')
      noteA.parents = noteA.parents.filter((p) => p !== second)
      noteA.jumps = noteA.jumps.filter((j) => j !== second)
      noteB.parents = noteB.parents.filter((p) => p !== first)
      noteB.jumps = noteB.jumps.filter((j) => j !== first)
      delete EDGE_LABELS[`${first}->${second}`]
      delete EDGE_LABELS[`${second}->${first}`]
      return asValue(undefined)
    },
  },
  isReady: ref(true),
  isNesting: ref(true),
  headings: ref(6),
  openNote: (path, options) => {
    const title = options?.title ?? notesByPath[path]?.title ?? ''
    if (!options?.showing && options?.line === undefined && syncPathFromTab(PLEX_TAB, path, title)) {
      return
    }
    const placement = options?.showing === 'split' ? 'split' : 'current'
    const tabId = openVaultFile(path, placement)
    if (options?.line !== undefined && tabId) {
      const line = options.line
      void nextTick(() => {
        editorRefs.get(tabId)?.reveal(line)
      })
    }
  },
  onSelectPath: (path, title) => {
    syncPathFromTab(PLEX_TAB, path, title ?? '')
  },
  readHeadings: async (paths) => {
    const map = new Map<string, readonly NoteHeading[]>()
    for (const p of paths) {
      const h = notesByPath[p]?.headings
      if (h) map.set(p, h)
    }
    return map
  },
  runCommand: (command, path) => {
    runVaultCommand(command, path, 'plex')
  },
  getNoteCommands: () => [
    { id: 'reveal', text: 'Show in files', group: 'note' },
    { id: 'travel', text: 'Show in plex', group: 'note' },
    { id: 'ask', text: 'Ask the agent', group: 'note' },
  ],
  getCommandIcon: (id) => COMMAND_ICONS[id] ?? null,
  getNoteIcon: (type) => (type === 'deck' ? Layers : FileText),
  openingPath: ref('physics/Entropy.md'),
  readOpeningPath: async () => 'physics/Entropy.md',
  dragged: draggedPaths,
  showMessage: () => {},
  createUntitledNote: async () => {
    const created = createNoteInRelationInVault(
      '',
      plexView.here.value || 'physics/Entropy.md',
      'child',
    )
    return created.path
  },
  creatable: ['parent', 'child', 'jump'],
  searchNames: async (query) => {
    const q = query.trim().toLowerCase()
    return Object.values(notesByPath)
      .filter((n) => !q || n.title.toLowerCase().includes(q))
      .map((n) => ({ path: n.path, title: n.title }))
  },
})

void plexView.go('physics/Entropy.md')

watch(
  () => plexView.here.value,
  (path) => {
    if (path) {
      syncPathFromTab(PLEX_TAB, path, notesByPath[path]?.title ?? '')
    }
  },
)

/* ------------------------ Real `Agent` Chat State ------------------------- */
const createAsked = (id: string, text: string): Turn => ({ id, voice: 'prompt', text })
const createDid = (id: string, text: string, subject: string, aside: string): Turn => ({
  id,
  voice: 'doing',
  text,
  subject,
  aside,
  canOpen: true,
})
const createAnswered = (id: string, text: string): Turn => ({ id, voice: 'response', text })

const turns = ref<Turn[]>([
  createAsked('1', 'What does this note leave out?'),
  createDid('2', 'note neighbourhood', 'Entropy', '4 links'),
  createAnswered(
    '3',
    'Two things. The note says the demon has to pay, but not where the payment ' +
      'lands — that is Landauer, and you have no note for him. And p. 373 counts ' +
      'arrangements at **fixed** energy, an assumption the free-energy note drops.',
  ),
  createAsked('4', 'Then make the Landauer note and put it under Entropy.'),
  createDid('5', 'note create', "Landauer's principle", 'under Entropy'),
  createAnswered(
    '6',
    'Made and filed under Entropy from p. 373 and the 1961 paper: what clearing ' +
      "one bit costs, and why the sorting pays it. Linked across to Maxwell's demon.",
  ),
])

const asking = ref('')
const isAgentWorking = ref(false)
const selectedAgentId = ref('claude')
const selectedModelId = ref('claude-3-7-sonnet')

const AGENT_OPTIONS: readonly AgentModelOption[] = [
  {
    agentId: 'claude',
    agentTitle: 'Claude',
    modelId: 'claude-3-7-sonnet',
    modelTitle: 'Sonnet 3.7',
    isAvailable: true,
    isDefault: true,
  },
  {
    agentId: 'gemini',
    agentTitle: 'Gemini',
    modelId: 'gemini-2.5-pro',
    modelTitle: '2.5 Pro',
    isAvailable: true,
  },
]

/* ------------------------ PDF & Search / Palette -------------------------- */
const PAPER = { wide: 612, high: 792 }
const MARGIN = 84
const FIRST = 132
const LEADING = 27

function getPdfPages(pdfPath: string) {
  const doc = PDF_DOCS[pdfPath] ?? PDF_DOCS['reading/Boltzmann 1877.pdf']!
  return Array.from({ length: doc.leaves }, () => PAPER)
}

function escapeHtml(line: string): string {
  return line.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

function renderPdfPage(pdfPath: string, pageNum: number): string {
  const doc = PDF_DOCS[pdfPath] ?? PDF_DOCS['reading/Boltzmann 1877.pdf']!
  const leaf =
    (((pageNum - doc.firstPage) % doc.pages.length) + doc.pages.length) % doc.pages.length
  const lines = doc.pages[leaf] ?? []
  const activeSet = lines
    .map((line, index) => {
      const y = FIRST + index * LEADING
      const weight = index === 0 && leaf === 0 ? ' font-weight="600" letter-spacing="1.2"' : ''
      return `<text x="${MARGIN}" y="${y}" font-family="Georgia, serif" font-size="16"${weight} fill="#1b1b1b">${escapeHtml(line)}</text>`
    })
    .join('')
  const number = `<text x="${PAPER.wide / 2}" y="${PAPER.high - 54}" text-anchor="middle" font-family="Georgia, serif" font-size="13" fill="#5a5a5a">${pageNum + 1}</text>`
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${PAPER.wide}" height="${PAPER.high}"><rect width="100%" height="100%" fill="#fbfaf7"/>${activeSet}${number}</svg>`
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
}

function overLines(from: number, to: number, ends: number) {
  return {
    minX: (MARGIN - 4) / PAPER.wide,
    maxX: (MARGIN + ends) / PAPER.wide,
    minY: (FIRST + from * LEADING - 18) / PAPER.high,
    maxY: (FIRST + to * LEADING + 6) / PAPER.high,
  }
}

const HIGHLIGHTS = [overLines(8, 9, 420), overLines(10, 11, 360)]

function getHighlightsOn(pdfPath: string, pageIndex: number) {
  const doc = PDF_DOCS[pdfPath] ?? PDF_DOCS['reading/Boltzmann 1877.pdf']!
  return pageIndex === doc.firstPage ? HIGHLIGHTS : []
}

const demoWindowRef = ref<HTMLElement | null>(null)
const activePanel = ref<'search' | 'commands' | 'naming' | null>(null)
const searchTyped = ref('entrop')
const pendingNaming = ref<{
  command: 'child' | 'parent' | 'jump' | 'title'
  targetPath: string
} | null>(null)

const NAMING_LABELS: Readonly<Record<'child' | 'parent' | 'jump' | 'title', string>> = {
  child: 'New child note',
  parent: 'New parent note',
  jump: 'New jump note',
  title: 'Change title',
}

function openNamingStep(command: 'child' | 'parent' | 'jump' | 'title', targetPath: string) {
  pendingNaming.value = { command, targetPath }
  searchTyped.value = command === 'title' ? (notesByPath[targetPath]?.title ?? '') : ''
  activePanel.value = 'naming'
}

function getMarks(title: string, word: string): TextRange[] {
  const q = word.trim().toLowerCase()
  if (!q) return []
  const spans: TextRange[] = []
  const lower = title.toLowerCase()
  for (let pos = lower.indexOf(q); pos >= 0; pos = lower.indexOf(q, pos + 1)) {
    spans.push({ start: pos, end: pos + q.length })
  }
  return spans
}

const TRAVEL_ACTION = { id: 'travel', text: 'Show in plex' }
const READ_ACTION = { id: 'read', text: 'Open note' }
const READ_DOC_ACTION = { id: 'readDocument', text: 'Open the document here' }

const searchGroups = computed<readonly PaletteGroup[]>(() => {
  const q = searchTyped.value.trim().toLowerCase() || 'entrop'
  const matchingNotes = Object.values(notesByPath).filter((n) =>
    n.title.toLowerCase().includes(q),
  )

  const fixedPassages = [
    {
      id: 'reading/Boltzmann 1877.pdf',
      title: 'Boltzmann 1877',
      detail:
        'the entropy of a state is the logarithm of the number of arrangements it could have been made of',
      actions: [READ_DOC_ACTION],
    },
    {
      id: 'physics/The second law.md',
      title: 'The second law',
      detail:
        'entropy never falls in an isolated system, which is the whole law stated in one line',
      actions: [READ_ACTION, TRAVEL_ACTION],
    },
  ]

  const matchingPassages = fixedPassages
    .filter((p) => p.title.toLowerCase().includes(q) || p.detail.toLowerCase().includes(q))
    .map((p) => ({
      ...p,
      detailAt: getMarks(p.detail, q),
    }))

  const groups: PaletteGroup[] = []
  if (matchingNotes.length > 0) {
    groups.push({
      id: 'names',
      title: 'Names',
      items: matchingNotes.map((n) => ({
        id: n.path,
        title: n.title,
        at: getMarks(n.title, q),
        actions: [TRAVEL_ACTION, READ_ACTION],
      })),
    })
  }
  if (matchingPassages.length > 0) {
    groups.push({
      id: 'text',
      title: 'Text',
      items: matchingPassages,
    })
  }
  return groups
})

const KEYBOARD = 'Macintosh'
const PALETTE_ACTION_KEY = commandKeyChord(KEYBOARD)
const PALETTE_ACTION_WORDS: ActionWords = {
  name: 'Actions',
  placeholder: 'Search actions',
  silence: 'No actions found',
}

interface RawCommandItem {
  readonly id: string
  readonly title: string
  readonly keys?: PaletteKeys
  readonly also?: { readonly id: string; readonly text: string }
}

const RAW_COMMAND_GROUPS: readonly {
  readonly id: string
  readonly title: string
  readonly items: readonly RawCommandItem[]
}[] = [
  {
    id: 'note',
    title: 'Note',
    items: [
      { id: 'read', title: 'Open the note', also: { id: 'beside', text: 'Open beside' } },
      { id: 'travel', title: 'Show the note', keys: keyChord('p', KEYBOARD, true) },
      { id: 'child', title: 'New child note', keys: keyChord('c', KEYBOARD, true) },
      { id: 'parent', title: 'New parent note' },
      { id: 'jump', title: 'New jump note' },
      { id: 'title', title: 'Change title' },
      { id: 'ask', title: 'Ask the agent' },
      { id: 'copy', title: 'Copy path' },
      { id: 'remove', title: 'Remove note' },
    ],
  },
  {
    id: 'window',
    title: 'General',
    items: [
      { id: 'note', title: 'New note', keys: keyChord('n', KEYBOARD) },
      { id: 'plex', title: 'Open plex tab' },
      { id: 'files', title: 'Open files tab' },
      { id: 'agent', title: 'Open agent tab', keys: keyChord('a', KEYBOARD, true) },
    ],
  },
]

const commandGroups = computed<readonly PaletteGroup[]>(() => {
  const q = searchTyped.value.trim().toLowerCase()
  const groups: PaletteGroup[] = []
  for (const group of RAW_COMMAND_GROUPS) {
    const items: PaletteItem[] = []
    for (const one of group.items) {
      if (q && !one.title.toLowerCase().includes(q)) continue
      const at = getMarks(one.title, q)
      items.push({
        id: one.id,
        title: one.title,
        ...(at.length > 0 ? { at } : {}),
        ...(one.keys ? { keys: one.keys } : {}),
        actions: [
          { id: one.id, text: one.title },
          ...(one.also ? [one.also] : []),
        ],
      })
    }
    if (items.length > 0) {
      groups.push({ id: group.id, title: group.title, items })
    }
  }
  return groups
})

const namingGroups = computed<readonly PaletteGroup[]>(() => {
  const trimmed = searchTyped.value.trim()
  return [
    {
      id: 'naming',
      title: 'Name',
      items: trimmed
        ? [
            {
              id: 'name',
              title: `Name it “${trimmed}”`,
              actions: [{ id: 'name', text: 'Name it' }],
            },
          ]
        : [],
      silence: 'Name',
    },
  ]
})

const paletteGroups = computed(() => {
  if (activePanel.value === 'naming') return namingGroups.value
  if (activePanel.value === 'commands') return commandGroups.value
  return searchGroups.value
})
const palettePlaceholder = computed(() => {
  if (activePanel.value === 'naming') return 'Name'
  if (activePanel.value === 'commands') return 'Type a command'
  return 'Search the vault'
})
const paletteCrumb = computed(() => {
  if (activePanel.value === 'naming' && pendingNaming.value) {
    return NAMING_LABELS[pendingNaming.value.command]
  }
  if (activePanel.value === 'commands') return 'Command'
  return ''
})
const paletteStep = computed(() => {
  if (activePanel.value === 'naming' && pendingNaming.value) {
    return `1:${pendingNaming.value.command}:naming`
  }
  if (activePanel.value === 'commands') return '0::commands'
  return ''
})

function getPaletteRowIcon(id: string): Component | null {
  if (activePanel.value === 'naming') {
    return pendingNaming.value ? (COMMAND_ICONS[pendingNaming.value.command] ?? FileText) : FileText
  }
  if (activePanel.value === 'commands') {
    if (id === 'note') return FilePlus
    if (id === 'plex') return Waypoints
    if (id === 'files') return FolderTree
    if (id === 'agent') return Bot
    return COMMAND_ICONS[id] ?? null
  }
  if (PDF_DOCS[id]) return BookOpen
  if (id === RECORDING_PATH || id.endsWith('.mp3')) return AudioLines
  if (id === YOUTUBE_LINK_PATH) return Link
  if (id === DECK_PATH) return Layers
  if (notesByPath[id]) return FileText
  return null
}

/* --------------------------------- Hooks ---------------------------------- */
function resetAgentScrollToTop() {
  const el = (agentRef.value?.$el as HTMLElement | undefined)?.querySelector<HTMLElement>('.thread')
  if (el) el.scrollTop = 0
}

watch(
  agentRef,
  (val) => {
    if (val) {
      void nextTick(() => {
        resetAgentScrollToTop()
      })
    }
  },
  { immediate: true },
)

onMounted(() => {
  void nextTick(() => {
    resetAgentScrollToTop()
  })
})

onBeforeUnmount(() => {
  demoAudioPlayer.pause()
})

/* -------------------------------- Handlers -------------------------------- */
function onWindowKeydown(event: KeyboardEvent) {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
    event.preventDefault()
    if (activePanel.value === 'commands') {
      activePanel.value = null
    } else {
      searchTyped.value = ''
      activePanel.value = 'commands'
    }
  } else if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'p') {
    event.preventDefault()
    if (activePanel.value === 'search') {
      activePanel.value = null
    } else {
      searchTyped.value = 'entrop'
      activePanel.value = 'search'
    }
  }
}

function onShowWorkspaceTab(tabId: string) {
  void nextTick(() => {
    editorRefs.get(tabId)?.measure()
    readerRefs.get(tabId)?.measure()
  })
}

function onActivateWorkspaceTab(tabId: string) {
  onShowWorkspaceTab(tabId)
  if (tabId.startsWith('note:')) {
    const path = noteTabPaths[tabId]
    if (path && notesByPath[path]) {
      syncPathFromTab(tabId, path, notesByPath[path]!.title)
    }
  }
}

function onCloseWorkspaceTab(tabId: string) {
  tabLinks.removeClosedTab(tabId)
  held.value = closeTab(held.value, tabId)
  openTabIds.value = openTabIds.value.filter((id) => id !== tabId)
  delete noteTabPaths[tabId]
}

function onTabContextMenu(tabId: string, pos: { x: number; y: number }) {
  linkMenu.open(tabId, pos)
}

function onGoBookPage(pdfPath: string, targetPage: number) {
  const doc = PDF_DOCS[pdfPath] ?? PDF_DOCS['reading/Boltzmann 1877.pdf']!
  pdfPagesState[pdfPath] = Math.min(Math.max(targetPage, 0), doc.leaves - 1)
}

function onOpenEditorLink(address: string, sourceTabId?: string) {
  const cleaned = address
    .replace(/^\[\[|\]\]$/g, '')
    .split('#')[0]
    ?.trim()
    .toLowerCase()
  if (!cleaned) return
  const match = Object.values(notesByPath).find(
    (n) => n.path.toLowerCase() === cleaned || n.title.toLowerCase() === cleaned,
  )
  if (match) {
    if (sourceTabId && sourceTabId.startsWith('note:')) {
      navigateNoteTab(sourceTabId, match.path)
      syncPathFromTab(sourceTabId, match.path, match.title)
      return
    }
    openVaultFile(match.path, 'current')
  }
}

function onOpenTurnSource(turn: Turn) {
  if (turn.subject === "Landauer's principle") {
    openVaultFile("computation/Landauer's principle.md", 'current')
  } else {
    openVaultFile('physics/Entropy.md', 'current')
  }
}

function onSelectModel(option: AgentModelOption) {
  selectedAgentId.value = option.agentId
  selectedModelId.value = option.modelId
}

function onSubmitAgent(prompt: string) {
  const trimmed = prompt.trim()
  if (!trimmed || isAgentWorking.value) return
  asking.value = ''
  const activePath = plexView.here.value || 'physics/Entropy.md'
  const activeNote = notesByPath[activePath] ?? notesByPath['physics/Entropy.md']!
  const turnBase = Date.now()
  turns.value = [
    ...turns.value,
    createAsked(`${turnBase}-q`, trimmed),
    createDid(`${turnBase}-tool`, 'note search', activeNote.title, '3 passages'),
  ]
  isAgentWorking.value = true
  window.setTimeout(() => {
    turns.value = [
      ...turns.value,
      createAnswered(
        `${turnBase}-a`,
        `In **${activeNote.title}**, Boltzmann's count of microstates ($S = k_B \\ln W$) and ` +
          `Landauer's erasure bound ($k_B T \\ln 2$ per bit) describe the same constraint from ` +
          `two sides. *(Interactive browser demo — download Numen to run local models and agents on your own vault.)*`,
      ),
    ]
    isAgentWorking.value = false
  }, 450)
}

function onPaletteDismiss() {
  activePanel.value = null
  pendingNaming.value = null
}

function onPaletteChoose(itemId: string, actionId?: string) {
  if (activePanel.value === 'naming' && pendingNaming.value) {
    const { command, targetPath } = pendingNaming.value
    const trimmed = searchTyped.value.trim()
    activePanel.value = null
    pendingNaming.value = null
    if (!trimmed) return
    if (command === 'title') {
      void filesState.rename(targetPath, trimmed)
    } else {
      const created = createNoteInRelationInVault(trimmed, targetPath, command)
      void plexView.go(created.path)
    }
    return
  }
  activePanel.value = null
  const activePath = plexView.here.value || 'physics/Entropy.md'
  if (
    itemId === 'read' ||
    itemId === 'beside' ||
    itemId === 'reveal' ||
    itemId === 'travel' ||
    itemId === 'child' ||
    itemId === 'parent' ||
    itemId === 'jump' ||
    itemId === 'title' ||
    itemId === 'ask' ||
    itemId === 'note' ||
    itemId === 'plex' ||
    itemId === 'files' ||
    itemId === 'agent'
  ) {
    runVaultCommand(itemId, activePath, 'plex')
    return
  }
  if (actionId === 'travel' && notesByPath[itemId]) {
    void plexView.go(itemId)
    ensureToolTabOpen(PLEX_TAB, 'map')
    return
  }
  openVaultFile(itemId, 'current')
}

/* -------------------------------- Helpers --------------------------------- */
function findContentPaneId(state: State): string | null {
  const panes = getPanes(state.root)
  const focused = paneById(state.root, state.focus)
  const isContentPane = (p: { id: string; tabs: readonly string[] }) =>
    p.tabs.some(
      (t) =>
        t.startsWith('note:') ||
        t.startsWith('reader:') ||
        t.startsWith('recording:') ||
        t.startsWith('link:') ||
        t.startsWith('deck:'),
    ) ||
    (p.id !== 'main' && p.id !== 'map' && p.id !== 'aside' && p.tabs.length === 0)

  if (focused && isContentPane(focused)) return focused.id
  const existing = panes.find(isContentPane)
  return existing?.id ?? null
}

function ensureTabRegistered(tabId: string) {
  if (!openTabIds.value.includes(tabId)) {
    openTabIds.value = [...openTabIds.value, tabId]
  }
}

function openContentTab(tabId: string, placement: 'current' | 'split' = 'current'): string {
  ensureTabRegistered(tabId)
  const contentPaneId = findContentPaneId(held.value)

  if (placement === 'split' && contentPaneId) {
    held.value = openTabAdjacent(held.value, tabId, {
      side: 'right',
      onto: contentPaneId,
      createId: createPaneId,
    })
  } else if (contentPaneId) {
    held.value = openTab(held.value, tabId, contentPaneId)
  } else {
    const anchor = paneById(held.value.root, 'map')?.id ?? getPanes(held.value.root)[0]?.id
    if (anchor) {
      held.value = openTabAdjacent(held.value, tabId, {
        side: 'bottom',
        onto: anchor,
        createId: createPaneId,
      })
    } else {
      held.value = openTab(held.value, tabId)
    }
  }

  onShowWorkspaceTab(tabId)
  return tabId
}

function openVaultFile(path: string, placement: 'current' | 'split' = 'current'): string | null {
  if (path === RECORDING_PATH) {
    return openContentTab(getRecordingTabId(RECORDING_PATH), placement)
  }
  if (path === YOUTUBE_LINK_PATH) {
    return openContentTab(getLinkTabId(YOUTUBE_LINK_PATH), placement)
  }
  if (path === DECK_PATH) {
    return openContentTab(getDeckTabId(DECK_PATH), placement)
  }
  if (PDF_DOCS[path]) {
    return openContentTab(getPdfTabId(path), placement)
  }
  if (notesByPath[path]) {
    if (placement === 'current') {
      const existingAtPath = findNoteTabAt(path)
      if (existingAtPath) {
        const tabId = openContentTab(existingAtPath, 'current')
        syncPathFromTab(tabId, path, notesByPath[path]!.title)
        return tabId
      }
      const linkedNoteTab = openTabIds.value.find(
        (id) => id.startsWith('note:') && tabLinks.getGroupOf(id) !== null,
      )
      if (linkedNoteTab) {
        navigateNoteTab(linkedNoteTab, path)
        const tabId = openContentTab(linkedNoteTab, 'current')
        syncPathFromTab(tabId, path, notesByPath[path]!.title)
        return tabId
      }
    }
    const tabId = openContentTab(createNoteTabForPath(path), placement)
    syncPathFromTab(tabId, path, notesByPath[path]!.title)
    return tabId
  }
  return null
}

function ensureToolTabOpen(tabId: string, preferredPaneId: string) {
  ensureTabRegistered(tabId)
  if (paneById(held.value.root, preferredPaneId)) {
    held.value = openTab(held.value, tabId, preferredPaneId)
  } else {
    held.value = openTab(held.value, tabId)
  }
  onShowWorkspaceTab(tabId)
}

function runVaultCommand(
  command: string,
  targetPath: string,
  source: 'files' | 'plex' = 'plex',
) {
  if (command === 'read') {
    openVaultFile(targetPath, 'current')
  } else if (command === 'beside') {
    openVaultFile(targetPath, 'split')
  } else if (command === 'reveal') {
    ensureToolTabOpen(FILES_TAB, 'main')
    void fileTree.revealPath(targetPath)
    fileTree.selectPaths([targetPath])
  } else if (command === 'travel') {
    if (notesByPath[targetPath]) {
      void plexView.go(targetPath)
      ensureToolTabOpen(PLEX_TAB, 'map')
    }
  } else if (command === 'child' || command === 'parent' || command === 'jump') {
    if (source === 'files') {
      createNoteInRelationInVault('', targetPath, command)
    } else {
      openNamingStep(command, targetPath)
    }
  } else if (command === 'note') {
    void filesState.createNote('physics')
  } else if (command === 'title') {
    if (source === 'files') {
      if (notesByPath[targetPath]) {
        filesState.setRenamingPath(targetPath)
      }
    } else {
      openNamingStep('title', targetPath)
    }
  } else if (command === 'copy') {
    void navigator.clipboard?.writeText(targetPath).catch(() => {})
  } else if (command === 'remove') {
    if (notesByPath[targetPath] && Object.keys(notesByPath).length > 1) {
      delete notesByPath[targetPath]
      for (const other of Object.values(notesByPath)) {
        other.parents = other.parents.filter((p) => p !== targetPath)
        other.jumps = other.jumps.filter((j) => j !== targetPath)
      }
      for (const [tabId, tabPath] of Object.entries(noteTabPaths)) {
        if (tabPath === targetPath) {
          onCloseWorkspaceTab(tabId)
        }
      }
      void fileTree.refresh()
      if (plexView.here.value === targetPath) {
        const fallback = Object.keys(notesByPath)[0] ?? 'physics/Entropy.md'
        void plexView.go(fallback)
      } else if (plexView.here.value) {
        void plexView.go(plexView.here.value)
      }
    }
  } else if (command === 'ask') {
    ensureToolTabOpen(AGENT_TAB, 'aside')
    asking.value = `${targetPath} — `
    void nextTick(() => {
      agentRef.value?.focus()
    })
  } else if (command === 'plex') {
    ensureToolTabOpen(PLEX_TAB, 'map')
  } else if (command === 'files') {
    ensureToolTabOpen(FILES_TAB, 'main')
  } else if (command === 'agent') {
    ensureToolTabOpen(AGENT_TAB, 'aside')
  }
}

function getTabIcon(id: string): Component | null {
  if (id === FILES_TAB) return FolderTree
  if (id === PLEX_TAB) return Waypoints
  if (id === AGENT_TAB) return Bot
  if (id.startsWith('reader:pdf:')) return BookOpen
  if (id.startsWith('recording:')) return AudioLines
  if (id.startsWith('link:')) return Link
  if (id.startsWith('deck:')) return Layers
  return FileText
}
</script>

<template>
  <div ref="demoWindowRef" class="demo-window numen" tabindex="-1" @keydown="onWindowKeydown" @contextmenu.prevent>
    <WorkspaceLayout
      v-model="held"
      :tabs="tabs"
      :minimum="140"
      :create-id="createPaneId"
      @activate="onActivateWorkspaceTab"
      @close="onCloseWorkspaceTab"
      @show="onShowWorkspaceTab"
      @tab-context-menu="onTabContextMenu"
    >
      <template #icon="{ id }">
        <component
          :is="getTabIcon(id)"
          v-if="getTabIcon(id)"
          class="tab-icon"
        />
        <TabLinkIndicator
          :is-linked="tabLinks.getGroupOf(id) !== null"
          :color-index="tabLinks.getGroupOf(id)?.colorIndex"
        />
      </template>

      <template #tab="{ id }">
        <PlexGraphView
          v-if="id === PLEX_TAB"
          :state="plexState"
        />
        <Editor
          v-else-if="id.startsWith('note:') && noteTabPaths[id] && notesByPath[noteTabPaths[id]!]"
          :ref="setEditorRef(id)"
          v-model="notesByPath[noteTabPaths[id]!]!.body"
          @open="(address: string) => onOpenEditorLink(address, id)"
        />
        <Reader
          v-else-if="id.startsWith('reader:pdf:') && PDF_DOCS[id.slice(11)]"
          :ref="setReaderRef(id)"
          class="h-full"
          :pages="getPdfPages(id.slice(11))"
          :at="pdfPagesState[id.slice(11)] ?? 0"
          :picture="(pageNum: number) => renderPdfPage(id.slice(11), pageNum)"
          :highlights="(pageNum: number) => getHighlightsOn(id.slice(11), pageNum)"
          @go="(pageNum: number) => onGoBookPage(id.slice(11), pageNum)"
        />
        <RecordingTab
          v-else-if="id.startsWith('recording:')"
          class="h-full"
          :path="RECORDING_PATH"
          :recordings="demoRecordings"
          :how="recordingHow"
        />
        <UrlTab
          v-else-if="id.startsWith('link:')"
          class="h-full"
          :path="YOUTUBE_LINK_PATH"
          :recordings="demoRecordings"
        />
        <DeckTab
          v-else-if="id.startsWith('deck:')"
          class="h-full"
          :state="deckState"
        />
        <Agent
          v-else-if="id === AGENT_TAB"
          ref="agentRef"
          v-model="asking"
          class="h-full"
          :turns="turns"
          :is-working="isAgentWorking"
          :options="AGENT_OPTIONS"
          :selected-agent-id="selectedAgentId"
          :selected-model-id="selectedModelId"
          placeholder="Ask about the vault"
          @submit="onSubmitAgent"
          @open="onOpenTurnSource"
          @select-model="onSelectModel"
        />
        <FilesTab
          v-else-if="id === FILES_TAB"
          :state="filesState"
        />
      </template>
    </WorkspaceLayout>

    <TabLinkMenu
      v-if="linkMenu.tabId.value"
      :tab-id="linkMenu.tabId.value"
      :eligible-tabs="linkMenu.eligibleTabs.value"
      :at="linkMenu.at.value"
      :is-linked="linkMenu.isLinked.value"
      :linked-target-title="linkMenu.linkedTargetTitle.value"
      :get-tab-icon="getTabIcon"
      @link="linkMenu.link"
      @unlink="linkMenu.unlink"
      @close="linkMenu.close"
    />

    <Palette
      v-if="activePanel"
      v-model="searchTyped"
      :groups="paletteGroups"
      :from="demoWindowRef"
      :to="demoWindowRef ?? 'body'"
      :crumb="paletteCrumb"
      :step="paletteStep"
      is-open
      :placeholder="palettePlaceholder"
      name="Search the vault"
      :action-words="PALETTE_ACTION_WORDS"
      :action-key="PALETTE_ACTION_KEY"
      @choose="onPaletteChoose"
      @back="onPaletteDismiss"
      @dismiss="onPaletteDismiss"
    >
      <template #icon="{ id }">
        <component :is="getPaletteRowIcon(id)" v-if="getPaletteRowIcon(id)" class="command-icon" />
      </template>
    </Palette>
  </div>
</template>

<style scoped>
.demo-window {
  position: relative;
  width: 100%;
  height: 100%;
  border-radius: 0 0 0.45rem 0.45rem;
  overflow: hidden;
  background: var(--numen-surface);
  color: var(--numen-ink);
  font-family: var(--numen-font-sans);
  text-align: left;
  user-select: none;
  -webkit-user-select: none;
}

.demo-window :deep(.palette) {
  position: absolute;
  --drop: 4.5rem;
  --tallest: 20rem;
}

.command-icon {
  inline-size: 100%;
  block-size: 100%;
  stroke-width: 1.75;
  opacity: 0.75;
}

.tab-icon {
  inline-size: 0.9rem;
  block-size: 0.9rem;
  flex-shrink: 0;
  stroke-width: 1.75;
  opacity: 0.75;
}

.demo-window :deep(.tab__icon) {
  inline-size: auto;
  min-inline-size: 0.9rem;
}

.demo-window :deep(.workspace__dragged) {
  z-index: 60;
}
</style>
