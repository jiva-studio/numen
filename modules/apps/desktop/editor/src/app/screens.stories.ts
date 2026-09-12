/**
 * The window drawn around the tabs this application adds to what the library
 * draws: the settings, a preset, a deck and the stencil that cuts it, a
 * recording and its transcript, and the runs a row of the tree can be put
 * through.
 *
 * Nothing here reaches a vault. Each tab is handed what it would hold, so every
 * piece is drawn in the state it settles in and a picture can be taken of it.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { StopReason } from '@numen/protocol'
import { WorkspaceLayout, branch, pane } from '@numen/ui'
import type { Tab, Workspace } from '@numen/ui'
import { computed, nextTick, onMounted, ref, shallowRef, type Component } from 'vue'

import { SettingsTab, type Installation } from '@/pages/settings'
import {
  DEFAULTS,
  PresetTab,
  type Curve,
  type Point,
  type PresetCounts,
  type PresetTabState,
  type Settings as Scheduling,
  type SettingsBounds,
} from '@/pages/preset-editor'
import { DeckTab, type DeckTabState } from '@/pages/deck-editor'
import { StencilTab, type StencilTabState } from '@/pages/stencil-editor'
import type { Marks } from '@/entities/deck'
import { RecordingTab } from '@/pages/media-recording'
import { useTranscript, type Recordings } from '@/entities/media'
import type { Cue } from '@/entities/media'
import type { Player } from '@/entities/media'
import { useTranscriptTab } from '@/entities/media'
import {
  DocumentTab,
  useDocumentReader,
  useDocumentTab,
  type Documents,
} from '@/pages/document-viewer'
import { FilesTab, ROOT, useFilesTab, useFileTree } from '@/pages/file-manager'
import type { Entry } from '@/shared/file'
import { iconOfKind } from '@/entities/tab'
import { DECK, DOCUMENT, FILES, PRESET, RECORDING, SETTINGS, STENCIL } from '@/entities/tab'

/** Every tab this file draws, under the identity the window opens it at. */
const TABS: readonly Tab[] = [
  { id: FILES, title: 'Files' },
  { id: SETTINGS, title: 'Settings' },
  { id: `${PRESET}:sanskrit`, title: 'Sanskrit' },
  { id: `${DECK}:roots`, title: 'Roots' },
  { id: `${STENCIL}:animal`, title: 'Word' },
  { id: `${RECORDING}:lecture`, title: 'Lecture 4.mp3' },
  { id: `${DOCUMENT}:boltzmann`, title: 'Boltzmann 1877.pdf' },
]

/** A tab is drawn as its kind, which is what stands before the identity. */
const iconOfTab = (id: string) => iconOfKind(id.split(':')[0] ?? id)

/** One tab filling the window, drawn in the chrome the window draws it in. */
const window = (tab: string, tabPane: Component, state: unknown) => ({
  components: { WorkspaceLayout },
  setup() {
    const layout = ref<Workspace>({
      root: pane('main', TABS.map((one) => one.id), tab),
      axis: 'horizontal',
      focus: 'main',
    })
    return { layout, TABS, tab, tabPane, state, iconOfTab }
  },
  template: `
    <div style="height: 100vh">
      <WorkspaceLayout v-model="layout" :tabs="TABS">
        <!-- Lucide draws on a 24 grid, and the stroke is given in those units. -->
        <template #icon="{ id }">
          <component
            :is="iconOfTab(id)"
            v-if="iconOfTab(id)"
            style="inline-size: 100%; block-size: 100%; stroke-width: 1.75; opacity: 0.75"
          />
        </template>

        <template #tab="{ id }">
          <component :is="tabPane" v-if="id === tab" :state="state" />
          <div v-else />
        </template>
      </WorkspaceLayout>
    </div>
  `,
})

const meta: Meta = {
  title: 'Desktop/Window',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

/** A window holding a note or a transcript: Tab indents until Escape hands it back. */
const WRITING = { reach: { keeps: 'Escape hands Tab back to the page' } }

/* The settings. ------------------------------------------------------------ */

const INSTALLATION: Installation = {
  themes: ref([
    { name: 'preset:numen', title: 'numen', isBuiltIn: true, isPinned: false },
    { name: 'preset:dracula', title: 'dracula', isBuiltIn: true, isPinned: true },
    { name: 'preset:solarized', title: 'solarized', isBuiltIn: true, isPinned: false },
    { name: 'mine:sea', title: 'sea', isBuiltIn: false, isPinned: false },
  ]),
  applied: ref('preset:numen'),
  mode: ref('system'),
  pinned: ref(false),
  sizes: ref({ interfaceScale: 1, textScale: 1.25 }),
  bounds: ref({
    interfaceScale: { least: 0.8, most: 2 },
    textScale: { least: 0.8, most: 1.75 },
  }),
  choose: () => {},
  syncing: ref(true),
  hangs: ref(true),
  parts: ref(6),
  partsBounds: ref({ least: 1, most: 12 }),
  chooseParts: () => {},
  dayStarts: ref('04:00'),
  latestDayStarts: ref('12:00'),
  chooseDayStarts: () => {},
  setting: () => undefined,
  models: () => [],
  write: () => {},
  file: ref('/numen.json'),
  openFile: () => {},
}

/** Everything in numen.json a person can change, in the groups the file keeps. */
export const Settings: Story = {
  render: () => window(SETTINGS, SettingsTab, { installation: INSTALLATION }),
}

/* A preset. ---------------------------------------------------------------- */

/** One place on the curve: what a day there costs, and what it comes to. */
const point = (over: Partial<Point> = {}): Point => ({
  reviews: 0,
  minutes: 0,
  retained: 0,
  owed: 0,
  through: 0,
  enough: true,
  closed: [],
  clears: 0,
  learned: 0,
  short: 0,
  backlog: [],
  ...over,
})

const GRID = [5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60]

/**
 * What stands overdue at the end of each day, one entry a day. A longer day
 * pays the debt off sooner, and a day too short never pays it off at all.
 */
const backlogAt = (rate: number): readonly number[] =>
  Array.from({ length: 60 }, (_, day) => Math.max(0, Math.round(96 - day * rate + day * day * rate * 0.004)))

const AT: readonly Point[] = GRID.map((minutes, at) =>
  point({
    minutes,
    // A longer day buys fewer extra cards than the one before it.
    reviews: Math.round(28 + 44 * Math.sqrt(at)),
    retained: 0.62 + at * 0.026,
    learned: 140 + Math.round(210 * Math.sqrt(at)),
    clears: at < 2 ? -1 : Math.max(4, Math.round(140 / at)),
    through: Math.min(1, 0.18 + at * 0.075),
    backlog: backlogAt(0.4 + at * 0.55),
  }),
)

const CURVE: Curve = {
  goal: 'minutes',
  grid: GRID,
  days: [],
  at: AT,
  now: { at: 4, value: 25, day: '' },
  suggested: { at: 7, value: 40, day: '' },
  decks: 3,
  cards: 1_240,
  overdue: 96,
  unbegun: 410,
  honest: true,
}

const SETTINGS_OF_PRESET: Scheduling = {
  ...DEFAULTS,
  minutesADay: 25,
  newADay: 12,
  reviewsADay: 140,
}

/** How far each setting goes, as the application answers a read. */
const BOUNDS: SettingsBounds = {
  minutesADay: { least: 0, most: 24 * 60 },
  newADay: { least: 0, most: 9999 },
  reviewsADay: { least: 0, most: 9999 },
  retention: { least: 0.7, most: 0.99 },
  backlog: { least: 0, most: 100 },
  interval: { least: 1, most: 365 },
}

const PRESET_STATE: PresetTabState = {
  id: 'Sanskrit.md',
  settings: shallowRef(SETTINGS_OF_PRESET),
  curve: shallowRef(CURVE),
  material: shallowRef({ decks: 3, cards: 1_240, overdue: 96, unbegun: 410 } as PresetCounts),
  place: ref(4),
  waiting: ref(false),
  bounds: shallowRef(BOUNDS),
  problems: shallowRef([]),
  stopped: ref(StopReason.NOTHING),
  errorMessage: ref(''),
  changed: ref(false),
  again: () => {},
  chooseGoal: () => {},
  moveSlider: () => {},
  settle: () => {},
  updateSetting: () => {},
  close: () => {},
}

/** The one control a preset is steered by, and the settings under it. */
export const Preset: Story = {
  render: () => window(`${PRESET}:sanskrit`, PresetTab, PRESET_STATE),
}

/* A deck and the stencil that cuts it. ------------------------------------- */

const NO_MARKS: Marks = {
  at: new Map(),
  under: new Map(),
  fields: new Map(),
  whole: [],
}

const WORD = { name: 'Word', fields: ['Word', 'Reading', 'Meaning', 'In a sentence'] }
const ROOT_CUT = { name: 'Root', fields: ['Root', 'Class', 'Present', 'Meaning'] }

const card = (
  id: string,
  section: string,
  stencilLink: string,
  values: readonly { field: string; text: string }[],
) => ({ id, section, stencil: stencilLink, filled: values })

const CARDS = [
  card('gam', 'going', 'Root', [
    { field: 'Root', text: 'gam' },
    { field: 'Class', text: '1, thematic' },
    { field: 'Present', text: 'gacchati' },
    { field: 'Meaning', text: 'to go, to move' },
  ]),
  card('car', 'going', 'Root', [
    { field: 'Root', text: 'car' },
    { field: 'Class', text: '1' },
    { field: 'Present', text: 'carati' },
    { field: 'Meaning', text: 'to walk, to wander' },
  ]),
  card('sthā', 'going', 'Root', [
    { field: 'Root', text: 'sthā' },
    { field: 'Class', text: '1' },
    { field: 'Present', text: 'tiṣṭhati' },
    { field: 'Meaning', text: 'to stand, to stay' },
  ]),
  card('i', 'going', 'Root', [
    { field: 'Root', text: 'i' },
    { field: 'Class', text: '2' },
    { field: 'Present', text: 'eti' },
    { field: 'Meaning', text: 'to go' },
  ]),
  card('vid', 'knowing', 'Root', [
    { field: 'Root', text: 'vid' },
    { field: 'Class', text: '2' },
    { field: 'Present', text: 'vetti' },
    { field: 'Meaning', text: 'to know' },
  ]),
  card('jñā', 'knowing', 'Root', [
    { field: 'Root', text: 'jñā' },
    { field: 'Class', text: '9' },
    { field: 'Present', text: 'jānāti' },
    { field: 'Meaning', text: 'to know, to recognise' },
  ]),
  card('smṛ', 'knowing', 'Root', [
    { field: 'Root', text: 'smṛ' },
    { field: 'Class', text: '1' },
    { field: 'Present', text: 'smarati' },
    { field: 'Meaning', text: 'to remember' },
  ]),
  card('mārga', 'nouns', 'Word', [
    { field: 'Word', text: 'mārga' },
    { field: 'Reading', text: 'मार्ग' },
    { field: 'Meaning', text: 'a road, a way' },
    { field: 'In a sentence', text: 'sa mārgaṃ gacchati' },
  ]),
]

const SECTIONS = [
  { id: 'going', name: 'Verbs of going' },
  { id: 'knowing', name: 'Verbs of knowing' },
  { id: 'nouns', name: 'Nouns off them' },
]

const DECK_STATE: DeckTabState = {
  id: 'Sanskrit/Roots.md',
  note: computed(() => ({ path: 'Sanskrit/Roots.md', body: '', state: 'clean', error: null })),
  deck: computed(() => ({ preamble: '', cards: [], sections: [], tail: '' })),
  drawn: computed(() => CARDS),
  sections: computed(() => SECTIONS),
  stencils: computed(() => [ROOT_CUT, WORD]),
  marks: computed(() => NO_MARKS),
  errorMessage: computed(() => ''),
  scheduled: computed(() => ({ path: 'Sanskrit.md', name: 'Sanskrit', errorMessage: '' })),
  choices: computed(() => [
    { path: '', name: 'The defaults' },
    { path: 'Sanskrit.md', name: 'Sanskrit' },
    { path: 'Anatomy.md', name: 'Anatomy' },
  ]),
  setSchedule: () => {},
  addCard: () => {},
  removeCard: () => {},
  moveCard: () => {},
  writeCardField: () => {},
  addSection: () => {},
  renameSection: () => {},
  removeSection: () => {},
  keepMine: () => {},
  takeFile: () => {},
  close: () => {},
}

/** The cards of a deck, under the sections they stand in. */
export const Deck: Story = {
  render: () => window(`${DECK}:roots`, DeckTab, DECK_STATE),
}

const STENCIL_STATE: StencilTabState = {
  id: 'Sanskrit/Word.md',
  note: computed(() => ({ path: 'Sanskrit/Word.md', body: '', state: 'clean', error: null })),
  stencil: computed(() => ({
    fields: WORD.fields,
    preamble: '',
    faces: [
      {
        id: 'recognise',
        name: 'Recognise',
        preamble: '',
        front: '<p>{{Word}} — <i>{{Reading}}</i></p>',
        back: '<p><b>Meaning:</b> {{Meaning}}</p>\n<p><b>In a sentence:</b> {{In a sentence}}</p>',
      },
      {
        id: 'produce',
        name: 'Produce',
        preamble: '',
        front: '<p>Which word means <i>{{Meaning}}</i>?</p>',
        back: '<p>{{Word}} — <i>{{Reading}}</i></p>\n<p>{{In a sentence}}</p>',
      },
      {
        id: 'read-it',
        name: 'Read it',
        preamble: '',
        front: '<p><i>{{Reading}}</i></p>',
        back: '<p>{{Word}}</p>\n<p><b>Meaning:</b> {{Meaning}}</p>',
      },
    ],
    tail: '',
  })),
  marks: computed(() => NO_MARKS),
  errorMessage: computed(() => ''),
  addField: () => {},
  renameField: () => {},
  removeField: () => {},
  moveField: () => {},
  addFace: () => {},
  renameFace: () => {},
  removeFace: () => {},
  moveFace: () => {},
  writeFaceHalf: () => {},
  keepMine: () => {},
  takeFile: () => {},
  close: () => {},
}

/** The fields a stencil names, and the faces that show them. */
export const Stencil: Story = {
  render: () => window(`${STENCIL}:animal`, StencilTab, STENCIL_STATE),
}

/* A recording, and the words heard in it. ---------------------------------- */

const cue = (from: number, to: number, text: string): Cue => ({ text, from, to })

/**
 * What was said, line by line. Enough of it to fill the pane it is read in,
 * because a transcript is what a person scrolls through rather than glances at.
 */
const SAID: readonly string[] = [
  'Right — so where we left off was the counting argument, and I want to do it once ' +
    'more slowly, because everybody nods at it and then uses it wrongly ten minutes later.',
  'You fix the energy of the gas. That is the whole of the setup. Fixing the energy does ' +
    'not fix the arrangement: there are enormously many ways the molecules can be put that ' +
    'all come to the same total, and no measurement you make on the gas as a whole tells ' +
    'one of them from another.',
  'Now — the number of those ways is not a property of any one arrangement. It is a ' +
    'property of the description. It says how much the description has left open. If you ' +
    'knew everything, the number would be one, and its logarithm would be nothing at all.',
  'That is the piece people drop. They talk as though disorder were something the gas has. ' +
    'It is not. It is something your account of the gas has. Which is why the same ' +
    'expression turns up in Shannon, about messages, and in Landauer, about a memory that ' +
    'is going to be cleared. Three subjects, one logarithm, and only the noun changes.',
  'Let me do the demon, because that is where this stops being bookkeeping.',
  'Maxwell asks you to imagine a being small enough to see single molecules. It stands at a ' +
    'door between two halves of a box and lets the fast ones through one way and the slow ' +
    'ones the other. After a while one side is hot and the other is cold, and nothing was ' +
    'pushed and nothing was heated. The gas has been sorted for free — that is the claim.',
  'And for seventy years the answers to it were bad. People said the demon must warm up, or ' +
    'that seeing a molecule costs a photon. Both are true and neither is the point, because ' +
    'you can always make the lamp dimmer.',
  'The point is that the demon has to know which molecule is which, and it has to hold that ' +
    'knowledge somewhere. The memory fills. And a finite memory has to be cleared before it ' +
    'can be used again.',
  'Clearing is where the bill arrives. Not at the door — at the moment the record of a ' +
    'molecule is destroyed, and at exactly the rate at which the sorting reduced the count. ' +
    'kT ln 2 for one bit. The books balance, and the law stands.',
  'So what the second law forbids is not a clever machine. It forbids a description that ' +
    'improves itself for nothing.',
  'Now, before anybody asks: yes, this is the same argument as the one about a channel. ' +
    'Shannon counts the messages that could have been sent; we count the arrangements the ' +
    'molecules could have been in. The counting does not care what is being counted, which ' +
    'is why the same logarithm turns up in both places and why people keep discovering it ' +
    'and thinking it is a coincidence.',
  'The thing to hold on to is that none of this is a statement about heat. Heat is where ' +
    'we met it, because that is where the nineteenth century was looking, but the measure ' +
    'is about descriptions. A description that leaves a great deal open has a large number ' +
    'under it; one that leaves little open has a small one.',
  'Which means — and this is the part I want you to take away — that entropy is not a ' +
    'property of a gas. It is a property of what you know about a gas. Two people with ' +
    'different information about the same box write down different numbers, and neither of ' +
    'them is wrong.',
  'That sounds like it should break something. It does not, and working out why it does ' +
    'not is most of what the subject has been doing since. The short version is that the ' +
    'information has to be held somewhere, and holding it is physical.',
  'One more thing and then I will let you go. When you read the papers you will find people ' +
    'talking about the demon as though it were a puzzle that was solved once. It was not. ' +
    'It was answered three or four times, badly, before Landauer said the thing that stuck, ' +
    'and the bad answers are worth reading because they are all reasonable.',
  'Read Landauer for next week — the 1961 paper, it is nine pages — and have a look at ' +
    'Bennett as well, because he is the one who worked out that the measuring is free and ' +
    'the forgetting is not.',
]

/** One stretch of speech to a paragraph, at the pace a person is talked at. */
const SPOKEN: readonly Cue[] = SAID.reduce<Cue[]>((cues, text) => {
  const from = cues.length === 0 ? 0 : (cues[cues.length - 1]?.to ?? 0)
  return [...cues, cue(from, from + 2_400 + text.length * 62, text)]
}, [])

const MEDIA = 'vault://Lectures/Lecture 4.mp3'
const RUNS = 1_842_000

const PLAYER: Player = {
  url: ref(MEDIA),
  at: ref(47_200),
  duration: ref(RUNS),
  playing: ref(true),
  error: ref(''),
  load: () => {},
  play: () => {},
  pause: () => {},
  seek: () => {},
}

const createRecordings = (cues: readonly Cue[]): Recordings => ({
  getSummary: async () => ({
    duration: RUNS,
    mediaUrl: MEDIA,
    mediaType: 'audio/mpeg',
    url: '',
  }),
  getTaskStates: async () => ({ transcript: 'done' }),
  readTranscript: async () => ({ cues, editable: true, prose: '' }),
  readArticle: async () => ({ cues: [], editable: true, prose: '' }),
  writeTranscript: async () => {},
  findCueTime: async () => null,
})

/** The words heard in a recording, against the moment each was said. */
export const Recording: Story = {
  parameters: WRITING,
  render: () =>
    window(
      `${RECORDING}:lecture`,
      RecordingTab,
      useTranscriptTab(useTranscript(createRecordings(SPOKEN), 'Lectures/Lecture 4.mp3', { through: PLAYER }), { runs: () => {} }),
    ),
}

/** A recording nothing has written down, and the run that would. */
export const NoTranscript: Story = {
  render: () =>
    window(
      `${RECORDING}:lecture`,
      RecordingTab,
      useTranscriptTab(useTranscript(createRecordings([]), 'Lectures/Lecture 4.mp3', { through: PLAYER }), { runs: () => {} }),
    ),
}

/**
 * The same recording with the run going. The player stands where it stands in
 * every other recording, and only what is below it says a run is on.
 */
export const Transcribing: Story = {
  render: () => {
    const state = useTranscriptTab(useTranscript(createRecordings([]), 'Lectures/Lecture 4.mp3', { through: PLAYER }), {
      runs: () => {},
    })
    state.setWorking(true)
    return window(`${RECORDING}:lecture`, RecordingTab, state)
  },
}

/* The tree, and the runs a row can be put through. ------------------------- */

const entry = (path: string, over: Partial<Entry> = {}): Entry => ({
  path,
  name: path.split('/').pop() ?? path,
  folder: false,
  kind: 'note',
  type: 'note',
  ...over,
})

const other = (path: string, kind: Entry['kind']): Entry =>
  entry(path, { kind, type: 'note' })

/** The folders of the vault the pictures in this file are taken of. */
const VAULT: Record<string, readonly Entry[]> = {
  [ROOT]: [
    entry('Lectures', { folder: true, kind: 'other' }),
    entry('Physics', { folder: true, kind: 'other' }),
    entry('Reading', { folder: true, kind: 'other' }),
    entry('Sanskrit', { folder: true, kind: 'other' }),
    entry('Entropy.md'),
    entry('Inbox.md'),
    entry('Reading list.md'),
    entry('Sanskrit.md', { type: 'preset' }),
  ],
  Lectures: [
    other('Lectures/Lecture 3.mp3', 'recording'),
    other('Lectures/Lecture 4.mp3', 'recording'),
    other('Lectures/Lecture 5.mp3', 'recording'),
    other('Lectures/Seminar, May.wav', 'recording'),
    other('Lectures/Kolb, an interview.flac', 'recording'),
    entry('Lectures/What the demon costs.md'),
  ],
  Physics: [
    entry('Physics/Entropy of mixing.md'),
    entry('Physics/Free energy.md'),
    entry("Physics/Landauer's principle.md"),
    entry("Physics/Maxwell's demon.md"),
    entry('Physics/The second law.md'),
  ],
  Reading: [
    other('Reading/Bennett 1982.epub', 'book'),
    other('Reading/Boltzmann 1877.pdf', 'book'),
    other('Reading/Fermi, Thermodynamics.pdf', 'book'),
    other('Reading/Landauer 1961.pdf', 'book'),
    other('Reading/Shannon 1948.pdf', 'book'),
  ],
  Sanskrit: [
    entry('Sanskrit/Declensions.md', { type: 'deck' }),
    entry('Sanskrit/Roots.md', { type: 'deck' }),
    entry('Sanskrit/Root.md', { type: 'stencil' }),
    entry('Sanskrit/Sandhi.md', { type: 'deck' }),
    entry('Sanskrit/Word.md', { type: 'stencil' }),
  ],
}

/** A files tab over that vault, and the reading of the folders named in it. */
const files = (open: readonly string[]) => {
  const list = useFileTree({ list: async (at: string) => VAULT[at] ?? [] })
  const state = useFilesTab(list, {
    openDestination: () => {},
    runCommand: () => {},
    movePath: async () => {},
    setDraggedPaths: () => {},
    createFolder: async () => {},
    createNote: async () => '',
    createDeck: async () => '',
    createStencil: async () => '',
    createPreset: async () => '',
    importAddress: async () => '',
    showError: () => {},
  })
  const read = (async () => {
    await list.openFolder(ROOT)
    for (const at of open) await list.openFolder(at)
  })()
  return { state, read }
}

/** A frame drawn, and the layout it has settled into. */
const waitForFrame = () => new Promise((then) => requestAnimationFrame(() => then(null)))

/* A scanned book, for the run that reads one. --------------------------------
 *
 * The pages are drawn here rather than fetched, so the picture of a book is a
 * book. They are broken into lines by hand, because what draws a page cannot
 * break one.
 */

const PAGE = { width: 612, height: 792 }
const MARGIN = 72
const FIRST = 132
const LEADING = 36
const SET = 21
const LEAVES = 412
const OPENS_AT = 372

const LINES: readonly (readonly string[])[] = [
  [
    '§ 6.  THE COUNT OF ARRANGEMENTS',
    '',
    'Let a quantity of gas be enclosed, and let',
    'the total energy it holds be fixed. The',
    'molecules may then be arranged in a great',
    'many ways without that total being',
    'disturbed, and no measurement made upon',
    'the gas as a whole distinguishes one such',
    'arrangement from another.',
    '',
    'The number of these arrangements is not a',
    'property of any one of them. It belongs to',
    'the description: it says how much that',
    'description has left open. Where the',
    'description is exact, the number is one.',
  ],
  [
    'It is this logarithm, and not the heat,',
    'that the second law is a statement about.',
    'A quantity of gas passes from an',
    'arrangement that could have been reached',
    'in few ways to one that could have been',
    'reached in many, because there are more of',
    'the latter to be reached.',
    '',
    'The same counting serves wherever a',
    'description leaves something open. Nothing',
    'in the argument is peculiar to molecules:',
    'what is counted are the possibilities a',
    'statement admits, and the measure of them',
    'is the same whether the statement is made',
    'about a gas or about a message.',
  ],
]

const escapeXml = (line: string) =>
  line.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

const pageAt = (page: number): string => {
  const lines = LINES[page - OPENS_AT] ?? []
  const set = lines
    .map((line, at) => {
      const y = FIRST + at * LEADING
      const weight = at === 0 ? ' font-weight="600" letter-spacing="1.2"' : ''
      return `<text x="${MARGIN}" y="${y}" font-family="Georgia, serif" font-size="${SET}"${weight} fill="#1b1b1b">${escapeXml(line)}</text>`
    })
    .join('')
  const number = `<text x="${PAGE.width / 2}" y="${PAGE.height - 54}" text-anchor="middle" font-family="Georgia, serif" font-size="16" fill="#5a5a5a">${page + 1}</text>`
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${PAGE.width}" height="${PAGE.height}"><rect width="100%" height="100%" fill="#fbfaf7"/>${set}${number}</svg>`
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
}

/**
 * Where one line of the drawn page stands, in fractions of it. What was read
 * off a scan is highlighted where it sits on the paper, so a passage is a
 * rectangle a line at a time and not a block.
 */
const overLine = (at: number, ends: number) => ({
  minX: (MARGIN - 4) / PAGE.width,
  maxX: (MARGIN + ends) / PAGE.width,
  minY: (FIRST + at * LEADING - 22) / PAGE.height,
  maxY: (FIRST + at * LEADING + 7) / PAGE.height,
})

/** The passage a search found: the sentence the count is defined by. */
const HIGHLIGHTS = [overLine(10, 396), overLine(11, 412), overLine(12, 372)]

const BOOK: Documents = {
  getDocumentLayout: async () => ({
    pages: Array.from({ length: LEAVES }, () => PAGE),
    fingerprint: '1024 1700000000000000000 book.pdf',
  }),
  getPageUrl: (_path, at) => pageAt(at),
  getHighlights: async () => [[{ page: OPENS_AT, rects: HIGHLIGHTS }]],
}

/**
 * A files tab with the menu standing open on one row, as a right-click leaves
 * it. Where the row is drawn is measured, so the menu stands under the name it
 * was asked for on however the tree happens to be laid out.
 */
const createMenuStory = (
  path: string,
  open: readonly string[],
  file: { tab: string; pane: Component; state: unknown; opens?: () => Promise<void> },
) => ({
  components: { WorkspaceLayout, FilesTab },
  setup() {
    const { state, read } = files(open)
    const layout = ref<Workspace>({
      root: branch('root', [pane('files', [FILES]), pane('main', [file.tab])], [0.3, 0.7]),
      axis: 'horizontal',
      focus: 'files',
    })
    // The tree has to stand on the page before a row has a place on it, and
    // the menu is asked for where that row is.
    // What the pane beside the tree is turned to settles first: a page turning
    // under an open menu is what takes the menu away again.
    onMounted(async () => {
      await file.opens?.()
      await read
      await nextTick()
      await waitForFrame()
      await waitForFrame()
      const row = document.querySelector(`[data-tree-row="${CSS.escape(path)}"]`)
      const box = row?.getBoundingClientRect()
      state.openMenu({ path, at: { x: (box?.left ?? 0) + 24, y: box?.bottom ?? 0 } })
    })
    return { layout, TABS, state, file, FILES, iconOfTab }
  },
  template: `
    <div style="height: 100vh">
      <WorkspaceLayout v-model="layout" :tabs="TABS">
        <template #icon="{ id }">
          <component
            :is="iconOfTab(id)"
            v-if="iconOfTab(id)"
            style="inline-size: 100%; block-size: 100%; stroke-width: 1.75; opacity: 0.75"
          />
        </template>

        <template #tab="{ id }">
          <FilesTab v-if="id === FILES" :state="state" />
          <component :is="file.pane" v-else :state="file.state" />
        </template>
      </WorkspaceLayout>
    </div>
  `,
})

/** The run a recording can be put through, offered on the row it stands at. */
export const Transcribed: Story = {
  parameters: WRITING,
  render: () =>
    createMenuStory('Lectures/Lecture 4.mp3', ['Lectures', 'Physics', 'Reading', 'Sanskrit'], {
      tab: `${RECORDING}:lecture`,
      pane: RecordingTab,
      state: useTranscriptTab(useTranscript(createRecordings(SPOKEN), 'Lectures/Lecture 4.mp3', { through: PLAYER }), {
        runs: () => {},
      }),
    }),
}

/** The run a scanned document can be put through, on the row it stands at. */
export const Recognised: Story = {
  render: () => {
    const state = useDocumentTab(useDocumentReader(BOOK, 'Reading/Boltzmann 1877.pdf'))
    return createMenuStory(
      'Reading/Boltzmann 1877.pdf',
      ['Lectures', 'Physics', 'Reading', 'Sanskrit'],
      {
        tab: `${DOCUMENT}:boltzmann`,
        pane: DocumentTab,
        state,
        // The reading a search sent a person into: the book turns to the page
        // the passage stands on, and the passage is highlighted where it stands.
        opens: () => state.focusSpans({ from: 0, to: 1 }),
      },
    )
  },
}
