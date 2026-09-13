/**
 * Every situation the palette has to survive. Also the test corpus: each story
 * is run in a browser by `@storybook/addon-vitest`, which is the only place the
 * things this component exists for can fail — a list that scrolls, an item
 * brought into sight, a line too long for the panel, and a panel that stands
 * over everything the window holds.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { onMounted, onUnmounted, ref, type Component } from 'vue'
import { AudioLines, BookOpen, FileText, Gauge, Layers, LayoutTemplate } from '@lucide/vue'
import Palette from './Palette.vue'
import { keyChord } from '../lib/keys'
import type { PaletteGroup } from '../lib/item'
import type { Span } from '@/shared/lib/span'
import {
  ARABIC,
  DEVANAGARI,
  GRAPHEMES,
  LINK,
  LONG,
  RUSSIAN,
  UNBREAKABLE,
} from '@/shared/fixtures/prose'

interface Knobs {
  groups: readonly PaletteGroup[]
  placeholder: string
  crumb: string
  step: string
  opensOn: string
  name: string
  onChoose: (item: string, action: string) => void
  onLight: (item: string) => void
  onDismiss: () => void
  onBack: () => void
}

/** What a plex can do with a name, and what a note can do with a place in it. */
const TRAVEL = [
  { id: 'travel', text: 'Show in plex' },
  { id: 'read', text: 'Open the note' },
]
const READ = [{ id: 'read', text: 'Open the note' }]

/** What a command offers: doing it. */
const RUN = [{ id: 'run', text: 'Run' }]

/** Everything one thing can be asked, which is more than the keys reach. */
const EVERYTHING = [
  { id: 'travel', text: 'Show in plex' },
  { id: 'read', text: 'Open the note' },
  { id: 'beside', text: 'Open beside' },
  { id: 'rename', text: 'Rename' },
  { id: 'copy', text: 'Copy the link' },
  { id: 'reveal', text: 'Show in the folder' },
  { id: 'pin', text: 'Keep at the top' },
  { id: 'remove', text: 'Move to trash' },
]

/**
 * Where a word stands in a line, every time it stands there. This is the
 * fixture's own arithmetic: the palette is told where the runs are.
 */
const marks = (text: string, word: string): Span[] => {
  if (!word) return []
  const out: Span[] = []
  const haystack = text.toLowerCase()
  const needle = word.toLowerCase()
  for (let at = haystack.indexOf(needle); at >= 0; at = haystack.indexOf(needle, at + 1)) {
    out.push({ from: at, to: at + needle.length })
  }
  return out
}

const createItem = (id: string, title: string, word: string) => ({
  id,
  title,
  at: marks(title, word),
  actions: TRAVEL,
})

const createDetailItem = (id: string, title: string, heading: string, word: string) => ({
  id,
  title,
  detail: heading,
  detailAt: marks(heading, word),
  actions: TRAVEL,
})

const passage = (id: string, title: string, text: string, word: string) => ({
  id,
  title,
  detail: text,
  detailAt: marks(text, word),
  actions: READ,
})

/** A vault with something in it, answering the word "ent". */
const NAMES: PaletteGroup = {
  id: 'names',
  title: 'Names',
  items: [
    createItem('entropy', 'Entropy', 'ent'),
    createItem('enthalpy', 'Enthalpy of formation', 'ent'),
    createDetailItem('carnot', 'The Carnot cycle', 'Entropy over one cycle', 'ent'),
    createDetailItem('gibbs', 'Gibbs free energy', 'Entropy and the second law', 'ent'),
  ],
}

const TEXT: PaletteGroup = {
  id: 'text',
  title: 'Text',
  items: [
    passage(
      'heat',
      'Heat engines',
      'no engine working between two reservoirs can be more efficient than a reversible one',
      'engine',
    ),
    passage('maxwell', 'Maxwell’s demon', 'the demon sorts fast molecules from slow', 'demon'),
  ],
}

const MEANING: PaletteGroup = {
  id: 'meaning',
  title: 'Meaning',
  items: [
    // Nothing is marked: a hit by meaning has no run to point at, and saying so
    // by drawing none is the honest answer.
    passage('arrow', 'The arrow of time', 'why the past and the future are not alike', ''),
  ],
}

const ALL: PaletteGroup[] = [NAMES, TEXT, { ...MEANING, working: true }]

/** The same names, each offering more than two keys can reach. */
const NAMED: PaletteGroup = {
  ...NAMES,
  items: NAMES.items.map((one) => ({ ...one, actions: EVERYTHING.slice(0, 5) })),
}

/** A window with something in it, and a palette standing over the lot. */
const renderOverWindow = (args: Knobs) => ({
  components: { Palette },
  setup() {
    const open = ref(true)
    const typed = ref('ent')
    return { args, open, typed }
  },
  template: `
    <div class="numen" style="height:100vh;padding:24px;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans)">
      <p style="margin:0 0 12px">The window, with the palette standing over it.</p>
      <button
        type="button"
        style="padding:8px 14px;border-radius:6px;border:1px solid var(--numen-rule);background:var(--numen-raised);color:inherit;font:inherit"
        @click="open = true"
      >Open the palette</button>
      <Palette
        v-model="typed"
        :groups="args.groups"
        :open="open"
        :placeholder="args.placeholder"
        :crumb="args.crumb"
        :step="args.step"
        :opens-on="args.opensOn"
        :name="args.name"
        @choose="args.onChoose"
        @light="args.onLight"
        @back="args.onBack"
        @dismiss="open = false; args.onDismiss()"
      >
        <template #silence>Type to look for something</template>
      </Palette>
    </div>
  `,
})

const meta = {
  title: 'Application/Palette',
  component: Palette,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'A field, and everything the words in it turned up, in groups. It ' +
          'stands over the whole window and takes groups of items, and says ' +
          'which item was chosen and what was asked of it. What the groups ' +
          'are, what an item addresses and what choosing one does are the ' +
          'caller’s. The keyboard stays in the field the whole time, because ' +
          'a person is still typing.',
      },
    },
  },
  argTypes: {
    placeholder: { control: 'text' },
    crumb: { control: 'text' },
    name: { control: 'text' },
    opensOn: { control: 'text' },
    groups: { table: { disable: true } },
    step: { table: { disable: true } },
    onChoose: { table: { disable: true } },
    onLight: { table: { disable: true } },
    onDismiss: { table: { disable: true } },
    onBack: { table: { disable: true } },
  },
  args: {
    groups: ALL,
    placeholder: 'Search',
    crumb: '',
    step: '',
    opensOn: '',
    name: 'Palette',
    onChoose: fn(),
    onLight: fn(),
    onDismiss: fn(),
    onBack: fn(),
  },
  render: renderOverWindow,
} satisfies Meta<Knobs>

export default meta
type Story = StoryObj<typeof meta>

const palette = () => document.body.querySelector<HTMLElement>('[data-palette="ground"]')
const lit = () => document.body.querySelector<HTMLElement>('[data-here]')
const options = () =>
  Array.from(document.body.querySelectorAll<HTMLElement>('[data-palette="list"] [role="option"]'))
const field = () => document.body.querySelector<HTMLInputElement>('[data-palette="field"]')
const sheet = () => document.body.querySelector<HTMLElement>('[data-actions="panel"]')
const hunt = () => document.body.querySelector<HTMLInputElement>('[data-actions="hunt"]')
const actions = () =>
  Array.from(document.body.querySelectorAll<HTMLElement>('[data-actions="list"] [role="option"]'))

/** What a line says, with the runs it is written in run together. */
const getText = (of: Element | null | undefined): string =>
  (of?.textContent ?? '').replace(/\s+/g, ' ').trim()

/** The keystroke a cap is announced as, which is all of it a reader hears. */
const getSpokenKey = (cap: Element | null | undefined): string =>
  getText(cap?.querySelector('.sr-only'))

/** The two keyboards a keystroke is written for. */
const APPLE = 'MacIntel'
const OTHER = 'Linux x86_64'

/** Every setting, live: three groups, one of them still on its way. */
export const Playground: Story = {}

/**
 * Choosing one. Both identifiers are handed back as given, and Shift reaches
 * the second thing the item offers.
 */
export const Choosing: Story = {
  play: async ({ args }) => {
    await waitFor(() => expect(lit()).not.toBeNull())

    await userEvent.keyboard('{Enter}')
    await expect(args.onChoose).toHaveBeenCalledWith('entropy', 'travel')

    await userEvent.keyboard('{Shift>}{Enter}{/Shift}')
    await expect(args.onChoose).toHaveBeenCalledWith('entropy', 'read')
  },
}

/**
 * Nothing behind it is reachable while it stands, which is what a palette is.
 * Tab is answered here and moves nowhere, and the panel over the palette says
 * the same of the palette under it.
 */
export const NothingBehindIt: Story = {
  play: async () => {
    await waitFor(() => expect(document.activeElement).toBe(field()))
    await expect(palette()?.querySelector('[data-palette="panel"]')).toHaveAttribute(
      'aria-modal',
      'true',
    )

    await userEvent.tab()
    await expect(document.activeElement).toBe(field())
    await userEvent.tab({ shift: true })
    await expect(document.activeElement).toBe(field())

    await userEvent.keyboard('{Control>}k{/Control}')
    await waitFor(() => expect(document.activeElement).toBe(hunt()))
    await expect(sheet()).toHaveAttribute('aria-modal', 'true')

    await userEvent.tab()
    await expect(document.activeElement).toBe(hunt())
  },
}

/**
 * Where the keyboard is standing, said as it moves. A caller showing what is
 * lit — a theme worn while it is walked past — draws from this alone.
 */
export const Lit: Story = {
  play: async ({ args }) => {
    await waitFor(() => expect(args.onLight).toHaveBeenCalledWith('entropy'))

    await userEvent.keyboard('{ArrowDown}')
    await waitFor(() => expect(args.onLight).toHaveBeenCalledWith('enthalpy'))

    await userEvent.keyboard('{ArrowUp}')
    await waitFor(() => expect(args.onLight).toHaveBeenLastCalledWith('entropy'))
  },
}

/**
 * A list of values opens standing on the value in force, wherever in the list
 * it sits. Nothing has to be walked back, and a caller acting on what is lit
 * acts on what is already so.
 */
export const OpensOnAValue: Story = {
  args: { opensOn: 'gibbs' },
  play: async ({ args }) => {
    await waitFor(() => expect(lit()).not.toBeNull())

    await expect(lit()?.textContent).toContain('Gibbs free energy')
    await expect(args.onLight).toHaveBeenLastCalledWith('gibbs')
  },
}

/**
 * A value with no item to stand on: the list holds none of it, and the first
 * item is where it opens.
 */
export const OpensOnNothingThere: Story = {
  args: { opensOn: 'nowhere' },
  play: async ({ args }) => {
    await waitFor(() => expect(lit()).not.toBeNull())

    await expect(lit()?.textContent).toContain('Entropy')
    await expect(args.onLight).toHaveBeenLastCalledWith('entropy')
  },
}

/**
 * One item, which is where it opens. Walking a list of one moves nothing, so
 * it is said once and not again.
 */
export const LitAlone: Story = {
  args: {
    groups: [{ id: 'names', title: 'Names', items: [createItem('entropy', 'Entropy', 'ent')] }],
  },
  play: async ({ args }) => {
    await waitFor(() => expect(args.onLight).toHaveBeenCalledWith('entropy'))

    await userEvent.keyboard('{ArrowDown}')
    await userEvent.keyboard('{End}')
    await expect(args.onLight).toHaveBeenCalledTimes(1)
  },
}

/** Every group came back with nothing, so there is nowhere to stand. */
export const LitNothing: Story = {
  args: {
    groups: [{ id: 'names', title: 'Names', items: [], silence: 'The vault could not answer' }],
  },
  play: async ({ args }) => {
    await waitFor(() => expect(palette()).not.toBeNull())

    await userEvent.keyboard('{ArrowDown}')
    await expect(args.onLight).not.toHaveBeenCalled()
  },
}

/** Far too many, walked to the end: every row it crosses is said, in order. */
export const LitFarDown: Story = {
  args: {
    groups: [
      {
        id: 'names',
        title: 'Names',
        items: Array.from({ length: 60 }, (_, at) =>
          createItem(`note-${at}`, `Entropy in ${at + 1} dimensions`, 'ent'),
        ),
      },
    ],
  },
  play: async ({ args }) => {
    await waitFor(() => expect(args.onLight).toHaveBeenCalledWith('note-0'))

    for (let step = 0; step < 59; step += 1) await userEvent.keyboard('{ArrowDown}')

    await waitFor(() => expect(args.onLight).toHaveBeenLastCalledWith('note-59'))
    await expect(args.onLight).toHaveBeenCalledTimes(60)
  },
}

/**
 * What is never said: a row that cannot be chosen, and anything at all while
 * the action panel stands. The panel is about the item that was lit when it
 * opened, and the keyboard is in a list of its own.
 */
export const NotLit: Story = {
  args: {
    groups: [
      {
        id: 'names',
        title: 'Names',
        items: [
          { ...createItem('entropy', 'Entropy', 'ent'), actions: EVERYTHING },
          { id: 'stuck', title: 'Enthalpy — this file cannot be read', disabled: true },
          createItem('gibbs', 'Gibbs free energy', 'e'),
        ],
      },
    ],
  },
  play: async ({ args }) => {
    await waitFor(() => expect(args.onLight).toHaveBeenCalledWith('entropy'))

    await userEvent.keyboard('{ArrowDown}')
    await waitFor(() => expect(args.onLight).toHaveBeenLastCalledWith('gibbs'))
    await expect(args.onLight).not.toHaveBeenCalledWith('stuck')

    await userEvent.click(within(document.body).getByText(/cannot be read/))
    await expect(args.onLight).toHaveBeenLastCalledWith('gibbs')

    await userEvent.keyboard('{ArrowUp}')
    await waitFor(() => expect(args.onLight).toHaveBeenCalledTimes(3))

    await userEvent.keyboard('{Control>}k{/Control}')
    await waitFor(() => expect(sheet()).not.toBeNull())
    await userEvent.keyboard('{ArrowDown}')
    await expect(args.onLight).toHaveBeenCalledTimes(3)
  },
}

/**
 * A group arriving while it is being read.
 *
 * The group on its way lands above the one the keyboard is in, so every number
 * in the list moves. What is lit is the item, not the number, and it stays put.
 */
export const Filling: Story = {
  render: (args) => ({
    components: { Palette },
    setup() {
      const open = ref(true)
      const typed = ref('ent')
      const groups = ref<PaletteGroup[]>([{ ...NAMES, items: [], working: true }, TEXT])

      let waiting: ReturnType<typeof setTimeout> | undefined
      onMounted(() => {
        waiting = setTimeout(() => (groups.value = [NAMES, TEXT]), 700)
      })
      onUnmounted(() => clearTimeout(waiting))

      return { args, open, typed, groups }
    },
    template: `
      <div class="numen" style="height:100vh;background:var(--numen-surface)">
        <Palette
          v-model="typed"
          :groups="groups"
          :open="open"
          @choose="args.onChoose"
          @dismiss="args.onDismiss"
        />
      </div>
    `,
  }),
  play: async () => {
    // The only item there is to begin with is in the second group.
    await waitFor(() => expect(lit()?.textContent).toContain('Heat engines'))
    await waitFor(() => expect(options().length).toBe(6), { timeout: 3000 })
    await expect(lit()?.textContent).toContain('Heat engines')
  },
}

/**
 * A search that came back with nothing, and what a reader is told of it.
 *
 * A group that answered with rows is read out by the row the keyboard lands
 * on. A group that answered with none has no row to land on, so what it says
 * in place of one is read out where a sighted person reads it. While it is
 * still working it has answered nothing, and nothing is said.
 */
export const NothingHeard: Story = {
  render: (args) => ({
    components: { Palette },
    setup() {
      const open = ref(true)
      const typed = ref('ent')
      const groups = ref<PaletteGroup[]>([{ ...NAMES, items: [], working: true }])

      let waiting: ReturnType<typeof setTimeout> | undefined
      onMounted(() => {
        waiting = setTimeout(
          () => (groups.value = [{ ...NAMES, items: [], silence: 'No note answers to that' }]),
          700,
        )
      })
      onUnmounted(() => clearTimeout(waiting))

      return { args, open, typed, groups }
    },
    template: `
      <div class="numen" style="height:100vh;background:var(--numen-surface)">
        <Palette
          v-model="typed"
          :groups="groups"
          :open="open"
          @choose="args.onChoose"
          @dismiss="args.onDismiss"
        />
      </div>
    `,
  }),
  play: async () => {
    const region = () => document.body.querySelector('[data-palette="said"]')

    await waitFor(() => expect(region()).not.toBeNull())
    await expect(region()).toHaveAttribute('aria-live', 'polite')
    await expect(getText(region())).toBe('')

    await waitFor(() => expect(getText(region())).toBe('No note answers to that'), {
      timeout: 3000,
    })
  },
}

/**
 * Every group was asked and answered with nothing. A group that has answered and
 * has nothing to say is worth no heading of its own, so none of them is drawn
 * and what stands there is what the caller says in place of a list.
 */
export const Nothing: Story = {
  args: {
    groups: [
      { id: 'names', title: 'Names', items: [] },
      { id: 'text', title: 'Text', items: [] },
      { id: 'meaning', title: 'Meaning', items: [] },
    ],
  },
  play: async () => {
    await waitFor(() =>
      expect(document.body.querySelectorAll('[data-palette="title"]')).toHaveLength(0),
    )
    await expect(getText(document.body.querySelector('[data-palette="nothing"]'))).toBe(
      'Type to look for something',
    )
  },
}

/**
 * One group answered with nothing and another could not be asked at all. The
 * first is drawn nowhere; the second says why in its own words, which is
 * something a person needs to be told.
 */
export const CouldNotBeAsked: Story = {
  args: {
    groups: [
      { id: 'names', title: 'Names', items: [] },
      { id: 'text', title: 'Text', items: [] },
      { id: 'meaning', title: 'Meaning', items: [], silence: 'No model is set' },
    ],
  },
  play: async () => {
    const getDrawnTitles = () =>
      Array.from(document.body.querySelectorAll('[data-palette="title"]')).map((group) =>
        group.textContent?.trim(),
      )
    await waitFor(() => expect(getDrawnTitles()).toEqual(['Meaning']))
    await expect(getText(document.body.querySelector('[data-palette="silence"]'))).toBe(
      'No model is set',
    )
  },
}

/** Nothing has been typed yet, so there is no group to draw at all. */
export const Unasked: Story = {
  args: { groups: [] },
}

/** One group, one item, one thing to do with it. */
export const Alone: Story = {
  args: {
    groups: [{ id: 'names', title: 'Names', items: [createItem('entropy', 'Entropy', 'ent')] }],
  },
}

/** What each row of the icons story is drawn as. The map is the caller's. */
const ICONS: Record<string, Component> = {
  note: FileText,
  deck: Layers,
  stencil: LayoutTemplate,
  preset: Gauge,
  book: BookOpen,
  recording: AudioLines,
}

/** A palette whose rows carry an icon, which is the caller filling the icon slot. */
const renderWithIcons = (args: Knobs) => ({
  components: { Palette },
  setup() {
    const open = ref(true)
    const typed = ref('ent')
    return { args, open, typed, icons: ICONS }
  },
  template: `
    <div class="numen" style="height:100vh;background:var(--numen-surface)">
      <Palette
        v-model="typed"
        :groups="args.groups"
        :open="open"
        @choose="args.onChoose"
        @dismiss="args.onDismiss"
      >
        <template #icon="{ id }">
          <component :is="icons[id]" v-if="icons[id]" style="inline-size:100%;block-size:100%" />
        </template>
      </Palette>
    </div>
  `,
})

/** What an icon stands in, and what it draws, as the drawn rows report it. */
const iconOf = (row: Element | null | undefined): string =>
  /lucide-([a-z-]+)-icon/.exec(row?.querySelector('svg')?.getAttribute('class') ?? '')?.[1] ?? ''

/**
 * An icon before every row, drawn by whoever offered the row: here a note, a
 * deck, a stencil, a preset, a book and a recording, each drawn as itself. A
 * row the caller has no icon for keeps the room, so the names line up down the
 * list.
 */
export const Icons: Story = {
  args: {
    groups: [
      {
        id: 'names',
        title: 'Names',
        items: [
          createItem('note', 'Entropy', 'ent'),
          createItem('deck', 'Words to learn', ''),
          createItem('stencil', 'Animal', ''),
          createItem('preset', 'Every day', ''),
        ],
      },
      {
        id: 'text',
        title: 'Text',
        items: [
          passage('book', 'The Mahabharata', 'the war of the two houses', 'war'),
          passage('recording', '730709BG.LON.mp3', 'what was said that morning', 'said'),
          createItem('nothing', 'A file of no kind', ''),
        ],
      },
    ],
  },
  render: renderWithIcons,
  play: async () => {
    await waitFor(() => expect(options()).toHaveLength(7))

    await expect(options().map(iconOf)).toEqual([
      'file-text',
      'layers',
      'layout-template',
      'gauge',
      'book-open',
      'audio-lines',
      '',
    ])
    // The row with no icon keeps the room for one, so the names line up.
    await expect(options()[6]?.querySelector('[data-palette="icon"]')).not.toBeNull()
  },
}

/**
 * Where an icon stands on a row carrying two lines: on the name, not between
 * the two lines, so the icons read down the list beside the names.
 */
export const IconsOnTheName: Story = {
  args: {
    groups: [
      {
        id: 'names',
        title: 'Names',
        items: [
          createItem('note', 'Entropy', 'ent'),
          passage('deck', 'Words to learn', LONG, 'the'),
        ],
      },
    ],
  },
  render: renderWithIcons,
  play: async () => {
    await waitFor(() => expect(options()).toHaveLength(2))

    for (const row of options()) {
      const icon = row.querySelector('[data-palette="icon"]')!.getBoundingClientRect()
      const name = row.querySelector('[data-palette="name"]')!.getBoundingClientRect()
      const middle = (box: DOMRect) => box.top + box.height / 2
      // Centred on the name's own line, within the rounding a layout leaves.
      await expect(Math.abs(middle(icon) - middle(name))).toBeLessThan(1.5)
    }

    // The second row is the tall one, so an icon centred on the row would sit
    // well below the name.
    const rows = options().map((row) => row.getBoundingClientRect().height)
    await expect(rows[1]).toBeGreaterThan(rows[0]! + 8)
  },
}

/**
 * Far too many, so the list scrolls.
 *
 * Walking to the end has to bring the last item into sight without moving the
 * window under it. Only a browser can answer that.
 */
export const FarTooMany: Story = {
  args: {
    groups: [
      {
        id: 'names',
        title: 'Names',
        items: Array.from({ length: 60 }, (_, at) =>
          createItem(`note-${at}`, `Entropy in ${at + 1} dimensions`, 'ent'),
        ),
      },
    ],
  },
  play: async () => {
    await waitFor(() => expect(lit()).not.toBeNull())

    for (let step = 0; step < 59; step += 1) await userEvent.keyboard('{ArrowDown}')

    const last = options().at(-1)!
    await expect(last.getAttribute('aria-selected')).toBe('true')

    // A row is as tall as the type it is set in, so it lands on a fraction of
    // a pixel and is brought into sight to within one.
    const list = document.body.querySelector<HTMLElement>('[data-palette="list"]')!
    const inside = last.getBoundingClientRect()
    const room = list.getBoundingClientRect()
    await expect(inside.bottom).toBeLessThanOrEqual(room.bottom + 1)
    await expect(inside.top).toBeGreaterThanOrEqual(room.top - 1)
  },
}

/** Scripts that are not Latin, one of which runs the other way. */
export const NotLatin: Story = {
  args: {
    groups: [
      {
        id: 'names',
        title: 'Названия',
        items: [
          createItem('ru', RUSSIAN, 'заметки'),
          createItem('sa', DEVANAGARI, 'बगीचा'),
          createItem('ar', ARABIC, 'السطر'),
        ],
      },
    ],
  },
}

/**
 * Lines past any width, and words with nowhere to break.
 *
 * Each is one line and then an ellipsis. The panel keeps the clearance it is
 * drawn with on either side, and nothing in it pushes the window wider. The
 * clearance is a length the interface multiplier moves, so it is read from the
 * panel rather than named.
 */
export const TooLong: Story = {
  args: {
    groups: [
      {
        id: 'names',
        title: 'Names',
        items: [
          createItem('long', LONG, 'component'),
          createItem('word', UNBREAKABLE, 'schiff'),
          passage('link', LINK, LINK, 'query'),
        ],
      },
    ],
  },
  play: async () => {
    await waitFor(() => expect(palette()).not.toBeNull())

    const panel = document.body.querySelector<HTMLElement>('[data-palette="panel"]')!
    const over = document.body.querySelector<HTMLElement>('[data-palette="ground"]')!
    const clear = parseFloat(getComputedStyle(over).paddingInlineStart)
    const drawn = panel.getBoundingClientRect()
    const room = over.getBoundingClientRect()

    await expect(drawn.left).toBeGreaterThanOrEqual(room.left + clear - 1)
    await expect(drawn.right).toBeLessThanOrEqual(room.right - clear + 1)
    for (const option of options()) {
      await expect(option.scrollWidth).toBeLessThanOrEqual(option.clientWidth + 1)
    }
    await expect(document.documentElement.scrollWidth).toBeLessThanOrEqual(
      document.documentElement.clientWidth + 1,
    )
  },
}

/** Characters that are several code units each, with a run landing inside one. */
export const Graphemes: Story = {
  args: {
    groups: [
      {
        id: 'names',
        title: 'Names',
        items: [
          { id: 'whole', title: GRAPHEMES, at: [{ from: 0, to: 1 }], actions: TRAVEL },
          createItem('mixed', `Waving 👋🏽 at the reader`, 'reader'),
        ],
      },
    ],
  },
}

/** An item that is drawn and cannot be chosen, beside ones that can. */
export const NotToBeChosen: Story = {
  args: {
    groups: [
      {
        id: 'names',
        title: 'Names',
        items: [
          createItem('entropy', 'Entropy', 'ent'),
          { id: 'stuck', title: 'Enthalpy — this file cannot be read', disabled: true },
          createItem('gibbs', 'Gibbs free energy', 'e'),
        ],
      },
    ],
  },
  play: async ({ args }) => {
    await waitFor(() => expect(lit()).not.toBeNull())

    await userEvent.keyboard('{ArrowDown}')
    await expect(lit()?.textContent).toContain('Gibbs')

    await userEvent.click(within(document.body).getByText(/cannot be read/))
    await expect(args.onChoose).not.toHaveBeenCalled()
  },
}

/**
 * One group answered, another is still filling, and a third could not be asked.
 *
 * The one with something to say is still drawn, and it stands at the foot, out
 * of the way of what a person is actually reading.
 */
export const SomeCameBackEmpty: Story = {
  args: {
    groups: [
      { id: 'names', title: 'Names', items: [], silence: 'The vault could not answer' },
      TEXT,
      { ...MEANING, working: true },
    ],
  },
  play: async () => {
    await waitFor(() => expect(lit()).not.toBeNull())

    const drawn = Array.from(document.body.querySelectorAll('[data-palette="title"]')).map(
      (group) => group.textContent?.trim(),
    )
    await expect(drawn).toEqual(['Text', 'Meaning', 'Names'])
    await expect(lit()?.textContent).toContain('Heat engines')
  },
}

/** One item offering five things, where two of them have a key. */
export const FiveActions: Story = {
  args: {
    groups: [
      {
        id: 'names',
        title: 'Names',
        items: [
          { ...createItem('entropy', 'Entropy', 'ent'), actions: EVERYTHING.slice(0, 5) },
          createItem('enthalpy', 'Enthalpy of formation', 'ent'),
        ],
      },
    ],
  },
  play: async ({ args }) => {
    await waitFor(() => expect(lit()).not.toBeNull())

    const reach = Array.from(document.body.querySelectorAll('[data-palette="key"]')).map(getText)
    await expect(reach).toEqual(['Return Show in plex', 'Shift Return Open the note'])
    await expect(document.body.querySelector('[data-palette="more"]')?.textContent).toContain(
      'Actions',
    )

    await userEvent.keyboard('{Enter}')
    await expect(args.onChoose).toHaveBeenCalledWith('entropy', 'travel')
  },
}

/**
 * The action panel, open over the item it is about.
 *
 * Eight things can be done and there are two keys, so the rest are reached by
 * name. The panel has a field of its own and a list of its own, and the list
 * underneath stays exactly where it was.
 */
export const ActionPanel: Story = {
  args: {
    groups: [
      {
        id: 'names',
        title: 'Names',
        items: [
          { ...createItem('entropy', 'Entropy', 'ent'), actions: EVERYTHING },
          createItem('enthalpy', 'Enthalpy of formation', 'ent'),
        ],
      },
    ],
  },
  play: async () => {
    await waitFor(() => expect(lit()).not.toBeNull())

    await userEvent.keyboard('{Control>}k{/Control}')
    await waitFor(() => expect(sheet()).not.toBeNull())

    await expect(actions()).toHaveLength(EVERYTHING.length)
    await expect(document.activeElement).toBe(hunt())
    await expect(field()?.getAttribute('aria-activedescendant')).toBeNull()
    await expect(hunt()?.getAttribute('aria-activedescendant')).toBe(actions()[0]?.id)

    // Walking back from the first brings the last row into sight.
    await userEvent.keyboard('{ArrowUp}')
    const last = actions().at(-1)!
    const inside = last.getBoundingClientRect()
    const room = document.body
      .querySelector<HTMLElement>('[data-actions="list"]')!
      .getBoundingClientRect()
    await expect(inside.bottom).toBeLessThanOrEqual(Math.ceil(room.bottom))
    await expect(inside.top).toBeGreaterThanOrEqual(Math.floor(room.top))

    await userEvent.keyboard('{Home}')
    await userEvent.keyboard('name')
    await waitFor(() => expect(actions()).toHaveLength(1))
    await expect(actions()[0]?.textContent).toContain('Rename')

    await userEvent.clear(hunt()!)
    await waitFor(() => expect(actions()).toHaveLength(EVERYTHING.length))
  },
}

/**
 * Two steps: pick a thing, then name it. The palette draws one step at a time;
 * the caller keeps the stack and swaps the groups, the words in the field and
 * the chip that says which step this is.
 *
 * On the second step the current name stands in the field and is selected, so
 * typing replaces it.
 */
export const Steps: Story = {
  render: (args) => ({
    components: { Palette },
    setup() {
      const open = ref(true)
      const step = ref('find')
      const typed = ref('')
      const crumb = ref('')

      const titleOf = (id: string) => NAMED.items.find((one) => one.id === id)?.title ?? ''

      const choose = (item: string, action: string) => {
        args.onChoose(item, action)
        if (action !== 'rename') return
        crumb.value = `New name for «${titleOf(item)}»`
        typed.value = titleOf(item)
        step.value = 'name'
      }

      const goBack = () => {
        args.onBack()
        crumb.value = ''
        typed.value = ''
        step.value = 'find'
      }

      return { args, open, step, typed, crumb, choose, goBack, NAMED }
    },
    template: `
      <div class="numen" style="height:100vh;background:var(--numen-surface)">
        <Palette
          v-model="typed"
          :groups="step === 'find' ? [NAMED] : []"
          :open="open"
          :step="step"
          :crumb="crumb"
          :placeholder="step === 'find' ? 'Search' : 'The new name'"
          @choose="choose"
          @back="goBack"
          @dismiss="args.onDismiss"
        />
      </div>
    `,
  }),
  play: async ({ args }) => {
    await waitFor(() => expect(lit()).not.toBeNull())

    // Backspace in an empty field is the gesture for the step before, at every
    // step there is.
    await userEvent.keyboard('{Backspace}')
    await expect(args.onBack).toHaveBeenCalled()

    await userEvent.keyboard('{Control>}k{/Control}')
    await waitFor(() => expect(sheet()).not.toBeNull())
    await userEvent.keyboard('name')
    await waitFor(() => expect(actions()).toHaveLength(1))
    await userEvent.keyboard('{Enter}')

    await waitFor(() =>
      expect(document.body.querySelector('[data-palette="crumb"]')?.textContent).toBe(
        'New name for «Entropy»',
      ),
    )

    const input = field()!
    await waitFor(() => expect(input.value).toBe('Entropy'))
    await expect(document.activeElement).toBe(input)
    await expect(input.selectionStart).toBe(0)
    await expect(input.selectionEnd).toBe('Entropy'.length)
  },
}

/**
 * Items carrying the keystroke that reaches them away from the palette: an
 * Apple keyboard's row, the row of every other keyboard, and a cap holding one
 * icon and a cap holding four.
 */
export const KeyHints: Story = {
  args: {
    groups: [
      {
        id: 'commands',
        title: 'Commands',
        items: [
          { id: 'new', title: 'New note', keys: keyChord('n', APPLE), actions: RUN },
          { id: 'plex', title: 'Show the plex', keys: keyChord('p', APPLE, true), actions: RUN },
          { id: 'close', title: 'Close this tab', keys: keyChord('w', OTHER, true), actions: RUN },
          { id: 'goto', title: 'Go to a note', keys: keyChord('g', OTHER), actions: RUN },
          { id: 'run', title: 'Run it', keys: { icons: ['return'], letter: '' }, actions: RUN },
          {
            id: 'long',
            title: LONG,
            keys: { icons: ['command', 'option', 'shift'], letter: 'L' },
            actions: RUN,
          },
          { id: 'none', title: 'Reload the vault', actions: RUN },
        ],
      },
    ],
  },
  play: async () => {
    await waitFor(() => expect(lit()).not.toBeNull())

    const hints = Array.from(document.body.querySelectorAll('[data-palette="hint"]'))
    await expect(hints.map(getSpokenKey)).toEqual([
      'Command N',
      'Command Shift P',
      'Control Shift W',
      'Control G',
      'Return',
      'Command Option Shift L',
    ])

    // A cap holding one icon is as tall as a cap holding three.
    const heights = new Set(hints.map((cap) => Math.round(cap.getBoundingClientRect().height)))
    await expect(heights.size).toBe(1)

    // A title too long for the row gives way to the key, and neither wraps.
    for (const option of options()) {
      await expect(option.scrollWidth).toBeLessThanOrEqual(option.clientWidth + 1)
    }
  },
}
