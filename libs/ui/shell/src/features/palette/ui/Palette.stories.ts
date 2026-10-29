/**
 * Palette stories. Also the test corpus: each story runs in a browser through
 * `@storybook/addon-vitest`, the only place the behaviors this component exists
 * for can fail: a scrolling list, scrolling the active item into view, a line
 * too long for the panel, and a panel stacked above all other content.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { onMounted, onUnmounted, ref, type Component } from 'vue'
import { AudioLines, BookOpen, FileText, Gauge, Layers, LayoutTemplate } from '@lucide/vue'
import Palette from './Palette.vue'
import { keyChord } from '@numen/ui'
import type { PaletteGroup } from '../lib/item'
import type { TextRange } from '@numen/ui'
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
  { id: 'read', text: 'Open note' },
]
const READ = [{ id: 'read', text: 'Open note' }]

/** What a command offers: doing it. */
const RUN = [{ id: 'run', text: 'Run' }]

/** An item with a primary and a secondary action; the keys reach only the primary. */
const EVERYTHING = [
  { id: 'travel', text: 'Show in plex' },
  { id: 'read', text: 'Open note' },
  { id: 'beside', text: 'Open beside' },
  { id: 'rename', text: 'Rename' },
  { id: 'copy', text: 'Copy the link' },
  { id: 'reveal', text: 'Show in the folder' },
  { id: 'pin', text: 'Keep at the top' },
  { id: 'remove', text: 'Move to trash' },
]

/**
 * Builds highlight ranges for each occurrence of `word` in `text`. The palette
 * receives the ranges as given and does no matching of its own.
 */
const marks = (text: string, word: string): TextRange[] => {
  if (!word) return []
  const out: TextRange[] = []
  const haystack = text.toLowerCase()
  const needle = word.toLowerCase()
  for (let pos = haystack.indexOf(needle); pos >= 0; pos = haystack.indexOf(needle, pos + 1)) {
    out.push({ start: pos, end: pos + needle.length })
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

const ALL: PaletteGroup[] = [NAMES, TEXT, { ...MEANING, isWorking: true }]

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
    const searchQuery = ref('ent')
    return { args, open, typed: searchQuery }
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
        :is-open="open"
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
const activeItem = () => document.body.querySelector<HTMLElement>('[data-here]')
const options = () =>
  Array.from(document.body.querySelectorAll<HTMLElement>('[data-palette="list"] [role="option"]'))
const field = () => document.body.querySelector<HTMLInputElement>('[data-palette="field"]')
const sheet = () => document.body.querySelector<HTMLElement>('[data-actions="panel"]')
const searchInput = () => document.body.querySelector<HTMLInputElement>('[data-actions="search"]')
const actions = () =>
  Array.from(document.body.querySelectorAll<HTMLElement>('[data-actions="list"] [role="option"]'))

/** The text of a line, with its highlighted runs concatenated. */
const getText = (of: Element | null | undefined): string =>
  (of?.textContent ?? '').replace(/\s+/g, ' ').trim()

/** The keystroke announced for a key cap, which is the text a screen reader receives. */
const getSpokenKey = (cap: Element | null | undefined): string =>
  getText(cap?.querySelector('.sr-only'))

/** The two keyboards a keystroke is written for. */
const APPLE = 'MacIntel'
const OTHER = 'Linux x86_64'

/** Every setting, live: three groups, one of them still on its way. */
export const Playground: Story = {}

/**
 * Choosing an item. Both identifiers are returned as given, and Shift selects
 * the item's secondary action.
 */
export const Choosing: Story = {
  play: async ({ args }) => {
    await waitFor(() => expect(activeItem()).not.toBeNull())

    await userEvent.keyboard('{Enter}')
    await expect(args.onChoose).toHaveBeenCalledWith('entropy', 'travel')

    await userEvent.keyboard('{Shift>}{Enter}{/Shift}')
    await expect(args.onChoose).toHaveBeenCalledWith('entropy', 'read')
  },
}

/**
 * Content behind the palette is unreachable while it is active.
 * Tab does not move focus out of the palette, and the action panel applies the
 * same rule to the palette beneath it.
 */
export const NothingBelowIt: Story = {
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
    await waitFor(() => expect(document.activeElement).toBe(searchInput()))
    await expect(sheet()).toHaveAttribute('aria-modal', 'true')

    await userEvent.tab()
    await expect(document.activeElement).toBe(searchInput())
  },
}

/**
 * The active item is announced on every keyboard move. A caller that previews
 * the active item, such as a theme, reads it from this announcement alone.
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
 * A list of values opens with the current value active, wherever it is in the
 * list. The caller acts on the active item without first navigating to it.
 */
export const OpensOnAValue: Story = {
  args: { opensOn: 'gibbs' },
  play: async ({ args }) => {
    await waitFor(() => expect(activeItem()).not.toBeNull())

    await expect(activeItem()?.textContent).toContain('Gibbs free energy')
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
    await waitFor(() => expect(activeItem()).not.toBeNull())

    await expect(activeItem()?.textContent).toContain('Entropy')
    await expect(args.onLight).toHaveBeenLastCalledWith('entropy')
  },
}

/**
 * A single item is active on open. Navigating a one-item list does not move the
 * active item, so it is announced once.
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
    groups: [{ id: 'names', title: 'Names', items: [], silence: 'Search failed' }],
  },
  play: async ({ args }) => {
    await waitFor(() => expect(palette()).not.toBeNull())

    await userEvent.keyboard('{ArrowDown}')
    await expect(args.onLight).not.toHaveBeenCalled()
  },
}

/** Many items, navigated to the end: every row crossed is announced, in order. */
export const LitFarDown: Story = {
  args: {
    groups: [
      {
        id: 'names',
        title: 'Names',
        items: Array.from({ length: 60 }, (_, index) =>
          createItem(`note-${index}`, `Entropy in ${index + 1} dimensions`, 'ent'),
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
 * Announcements that do not occur: a row that cannot be chosen, and any row
 * while the action panel is open. The panel applies to the item that was active
 * when it opened, and the keyboard operates a separate list.
 */
export const NotLit: Story = {
  args: {
    groups: [
      {
        id: 'names',
        title: 'Names',
        items: [
          { ...createItem('entropy', 'Entropy', 'ent'), actions: EVERYTHING },
          { id: 'stuck', title: 'Enthalpy — this file cannot be read', isDisabled: true },
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
      const searchQuery = ref('ent')
      const groups = ref<PaletteGroup[]>([{ ...NAMES, items: [], isWorking: true }, TEXT])

      let timeoutId: ReturnType<typeof setTimeout> | undefined
      onMounted(() => {
        timeoutId = setTimeout(() => (groups.value = [NAMES, TEXT]), 700)
      })
      onUnmounted(() => clearTimeout(timeoutId))

      return { args, open, typed: searchQuery, groups }
    },
    template: `
      <div class="numen" style="height:100vh;background:var(--numen-surface)">
        <Palette
          v-model="typed"
          :groups="groups"
          :is-open="open"
          @choose="args.onChoose"
          @dismiss="args.onDismiss"
        />
      </div>
    `,
  }),
  play: async () => {
    // The only item there is to begin with is in the second group.
    await waitFor(() => expect(activeItem()?.textContent).toContain('Heat engines'))
    await waitFor(() => expect(options().length).toBe(6), { timeout: 3000 })
    await expect(activeItem()?.textContent).toContain('Heat engines')
  },
}

/**
 * A search that returned no results, and what a screen reader announces for it.
 *
 * A group with rows is announced through the row the keyboard lands on. A group
 * with no rows has no such row, so its empty-state text is announced in place
 * of one. A group that is still loading has no result yet and announces nothing.
 */
export const NothingHeard: Story = {
  render: (args) => ({
    components: { Palette },
    setup() {
      const open = ref(true)
      const searchQuery = ref('ent')
      const groups = ref<PaletteGroup[]>([{ ...NAMES, items: [], isWorking: true }])

      let timeoutId: ReturnType<typeof setTimeout> | undefined
      onMounted(() => {
        timeoutId = setTimeout(
          () => (groups.value = [{ ...NAMES, items: [], silence: 'No note answers to that' }]),
          700,
        )
      })
      onUnmounted(() => clearTimeout(timeoutId))

      return { args, open, typed: searchQuery, groups }
    },
    template: `
      <div class="numen" style="height:100vh;background:var(--numen-surface)">
        <Palette
          v-model="typed"
          :groups="groups"
          :is-open="open"
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
 * Every group returned no results. A group with no results has no heading, so
 * none is drawn and the caller's empty-state content replaces the list.
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
 * One group returned no results and another failed to load. The empty group is
 * not drawn; the failed group shows its error text.
 */
export const CouldNotBeQueried: Story = {
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
  sample: Layers,
  form: LayoutTemplate,
  sheet: Gauge,
  book: BookOpen,
  recording: AudioLines,
}

/** A palette whose rows carry an icon, which is the caller filling the icon slot. */
const renderWithIcons = (args: Knobs) => ({
  components: { Palette },
  setup() {
    const open = ref(true)
    const searchQuery = ref('ent')
    return { args, open, typed: searchQuery, icons: ICONS }
  },
  template: `
    <div class="numen" style="height:100vh;background:var(--numen-surface)">
      <Palette
        v-model="typed"
        :groups="args.groups"
        :is-open="open"
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

const getIcon = (row: Element | null | undefined): string =>
  /(?:^|\s)lucide-([a-z-]+?)(?:-icon)?(?:\s|$)/.exec(
    row?.querySelector('svg')?.getAttribute('class') ?? '',
  )?.[1] ?? ''

/**
 * An icon before every row, supplied by the caller: here a note, a sample, a
 * form, a sheet, a book and a recording. A row without an icon reserves the
 * icon column, so the names stay aligned.
 */
export const Icons: Story = {
  args: {
    groups: [
      {
        id: 'names',
        title: 'Names',
        items: [
          createItem('note', 'Entropy', 'ent'),
          createItem('sample', 'Words to learn', ''),
          createItem('form', 'Animal', ''),
          createItem('sheet', 'Every day', ''),
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

    await expect(options().map(getIcon)).toEqual([
      'file-text',
      'layers',
      'layout-template',
      'gauge',
      'book-open',
      'audio-lines',
      '',
    ])
    // The row without an icon reserves the icon column, so the names stay aligned.
    await expect(options()[6]?.querySelector('[data-palette="icon"]')).not.toBeNull()
  },
}

/**
 * Where an icon relies on a row carrying two lines: on the name, not between
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
          passage('sample', 'Words to learn', LONG, 'the'),
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
        items: Array.from({ length: 60 }, (_, index) =>
          createItem(`note-${index}`, `Entropy in ${index + 1} dimensions`, 'ent'),
        ),
      },
    ],
  },
  play: async () => {
    await waitFor(() => expect(activeItem()).not.toBeNull())

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
 * panel and not named.
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
    const overElement = document.body.querySelector<HTMLElement>('[data-palette="ground"]')!
    const clear = Number.parseFloat(getComputedStyle(overElement).paddingInlineStart)
    const bounds = panel.getBoundingClientRect()
    const room = overElement.getBoundingClientRect()

    await expect(bounds.left).toBeGreaterThanOrEqual(room.left + clear - 1)
    await expect(bounds.right).toBeLessThanOrEqual(room.right - clear + 1)
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
          { id: 'whole', title: GRAPHEMES, at: [{ start: 0, end: 1 }], actions: TRAVEL },
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
          { id: 'stuck', title: 'Enthalpy — this file cannot be read', isDisabled: true },
          createItem('gibbs', 'Gibbs free energy', 'e'),
        ],
      },
    ],
  },
  play: async ({ args }) => {
    await waitFor(() => expect(activeItem()).not.toBeNull())

    await userEvent.keyboard('{ArrowDown}')
    await expect(activeItem()?.textContent).toContain('Gibbs')

    await userEvent.click(within(document.body).getByText(/cannot be read/))
    await expect(args.onChoose).not.toHaveBeenCalled()
  },
}

/**
 * One group returned results, another is still loading, and a third failed to
 * load.
 */
export const SomeCameBackEmpty: Story = {
  args: {
    groups: [
      { id: 'names', title: 'Names', items: [], silence: 'Search failed' },
      TEXT,
      { ...MEANING, isWorking: true },
    ],
  },
  play: async () => {
    await waitFor(() => expect(activeItem()).not.toBeNull())

    const viewElement = Array.from(document.body.querySelectorAll('[data-palette="title"]')).map(
      (group) => group.textContent?.trim(),
    )
    await expect(viewElement).toEqual(['Text', 'Meaning', 'Names'])
    await expect(activeItem()?.textContent).toContain('Heat engines')
  },
}

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
    await waitFor(() => expect(activeItem()).not.toBeNull())

    const reach = Array.from(document.body.querySelectorAll('[data-palette="key"]')).map(getText)
    await expect(reach).toEqual(['Return Show in plex', 'Shift Return Open note'])
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
    await waitFor(() => expect(activeItem()).not.toBeNull())

    await userEvent.keyboard('{Control>}k{/Control}')
    await waitFor(() => expect(sheet()).not.toBeNull())

    await expect(actions()).toHaveLength(EVERYTHING.length)
    await expect(document.activeElement).toBe(searchInput())
    await expect(field()?.getAttribute('aria-activedescendant')).toBeNull()
    await expect(searchInput()?.getAttribute('aria-activedescendant')).toBe(actions()[0]?.id)

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

    await userEvent.clear(searchInput()!)
    await waitFor(() => expect(actions()).toHaveLength(EVERYTHING.length))
  },
}

/**
 * Two steps: pick an item, then name it. The palette draws one step at a time;
 * the caller owns the step stack and swaps the groups, the field text and the
 * chip that identifies the current step.
 */
export const Steps: Story = {
  render: (args) => ({
    components: { Palette },
    setup() {
      const open = ref(true)
      const step = ref('find')
      const searchQuery = ref('')
      const crumb = ref('')

      const getTitle = (id: string) => NAMED.items.find((one) => one.id === id)?.title ?? ''

      const choose = (item: string, action: string) => {
        args.onChoose(item, action)
        if (action !== 'rename') return
        crumb.value = `New name for «${getTitle(item)}»`
        searchQuery.value = getTitle(item)
        step.value = 'name'
      }

      const goBack = () => {
        args.onBack()
        crumb.value = ''
        searchQuery.value = ''
        step.value = 'find'
      }

      return { args, open, step, typed: searchQuery, crumb, choose, goBack, NAMED }
    },
    template: `
      <div class="numen" style="height:100vh;background:var(--numen-surface)">
        <Palette
          v-model="typed"
          :groups="step === 'find' ? [NAMED] : []"
          :is-open="open"
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
    await waitFor(() => expect(activeItem()).not.toBeNull())

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
 * Items with a keystroke hint shown outside the palette: an Apple keyboard
 * layout, the layout of other keyboards, a key cap with one icon and a key cap
 * with four.
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
          { id: 'close', title: 'Close tab', keys: keyChord('w', OTHER, true), actions: RUN },
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
    await waitFor(() => expect(activeItem()).not.toBeNull())

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
