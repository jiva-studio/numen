/**
 * Every situation the palette has to survive. Also the test corpus: each story
 * is run in a browser by `@storybook/addon-vitest`, which is the only place the
 * things this component exists for can fail — a list that scrolls, an item
 * brought into sight, a line too long for the panel, and a panel that stands
 * over everything the window holds.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { onMounted, onUnmounted, ref } from 'vue'
import Palette from './Palette.vue'
import type { PaletteBand, PaletteSpan } from './model'
import {
  ARABIC,
  DEVANAGARI,
  GRAPHEMES,
  LINK,
  LONG,
  RUSSIAN,
  UNBREAKABLE,
} from '@/fixtures/prose'

interface Knobs {
  bands: readonly PaletteBand[]
  placeholder: string
  crumb: string
  step: string
  name: string
  onChoose: (item: string, action: string) => void
  onLit: (item: string) => void
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
const marks = (text: string, word: string): PaletteSpan[] => {
  if (!word) return []
  const out: PaletteSpan[] = []
  const haystack = text.toLowerCase()
  const needle = word.toLowerCase()
  for (let at = haystack.indexOf(needle); at >= 0; at = haystack.indexOf(needle, at + 1)) {
    out.push({ from: at, to: at + needle.length })
  }
  return out
}

const named = (id: string, title: string, word: string) => ({
  id,
  title,
  at: marks(title, word),
  actions: TRAVEL,
})

const inside = (id: string, title: string, heading: string, word: string) => ({
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
const NAMES: PaletteBand = {
  id: 'names',
  title: 'Names',
  items: [
    named('entropy', 'Entropy', 'ent'),
    named('enthalpy', 'Enthalpy of formation', 'ent'),
    inside('carnot', 'The Carnot cycle', 'Entropy over one cycle', 'ent'),
    inside('gibbs', 'Gibbs free energy', 'Entropy and the second law', 'ent'),
  ],
}

const TEXT: PaletteBand = {
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

const MEANING: PaletteBand = {
  id: 'meaning',
  title: 'Meaning',
  items: [
    // Nothing is marked: a hit by meaning has no run to point at, and saying so
    // by drawing none is the honest answer.
    passage('arrow', 'The arrow of time', 'why the past and the future are not alike', ''),
  ],
}

const ALL: PaletteBand[] = [NAMES, TEXT, { ...MEANING, working: true }]

/** The same names, each offering more than two keys can reach. */
const NAMED: PaletteBand = {
  ...NAMES,
  items: NAMES.items.map((one) => ({ ...one, actions: EVERYTHING.slice(0, 5) })),
}

/** A window with something in it, and a palette standing over the lot. */
const over = (args: Knobs) => ({
  components: { Palette },
  setup() {
    const open = ref(true)
    const typed = ref('ent')
    return { args, open, typed }
  },
  template: `
    <div class="numen" style="height:100vh;padding:24px;background:var(--numen-surface);color:var(--numen-node-fg);font-family:var(--numen-font-sans)">
      <p style="margin:0 0 12px">The window, with the palette standing over it.</p>
      <button
        type="button"
        style="padding:8px 14px;border-radius:6px;border:1px solid var(--numen-node-border);background:var(--numen-node-bg);color:inherit;font:inherit"
        @click="open = true"
      >Open the palette</button>
      <Palette
        v-model="typed"
        :bands="args.bands"
        :open="open"
        :placeholder="args.placeholder"
        :crumb="args.crumb"
        :step="args.step"
        :name="args.name"
        @choose="args.onChoose"
        @lit="args.onLit"
        @back="args.onBack"
        @dismiss="open = false; args.onDismiss()"
      >
        <template #silence>Type to look for something</template>
      </Palette>
    </div>
  `,
})

const meta = {
  title: 'Generic/Palette',
  component: Palette,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'A field, and everything the words in it turned up, in bands. It ' +
          'stands over the whole window and takes bands of items, and says ' +
          'which item was chosen and what was asked of it. What the bands ' +
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
    bands: { table: { disable: true } },
    step: { table: { disable: true } },
    onChoose: { table: { disable: true } },
    onLit: { table: { disable: true } },
    onDismiss: { table: { disable: true } },
    onBack: { table: { disable: true } },
  },
  args: {
    bands: ALL,
    placeholder: 'Search',
    crumb: '',
    step: '',
    name: 'Palette',
    onChoose: fn(),
    onLit: fn(),
    onDismiss: fn(),
    onBack: fn(),
  },
  render: over,
} satisfies Meta<Knobs>

export default meta
type Story = StoryObj<typeof meta>

const palette = () => document.body.querySelector<HTMLElement>('.palette')
const lit = () => document.body.querySelector<HTMLElement>('[data-here]')
const options = () => Array.from(document.body.querySelectorAll<HTMLElement>('.palette__item'))
const field = () => document.body.querySelector<HTMLInputElement>('.palette__field')
const sheet = () => document.body.querySelector<HTMLElement>('.palette__actions')
const hunt = () => document.body.querySelector<HTMLInputElement>('.palette__hunt')
const deeds = () => Array.from(document.body.querySelectorAll<HTMLElement>('.palette__deed'))

/** Every setting, live: three bands, one of them still on its way. */
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
 * Where the keyboard is standing, said as it moves. A caller showing what is
 * lit — a theme worn while it is walked past — draws from this alone.
 */
export const Lit: Story = {
  play: async ({ args }) => {
    await waitFor(() => expect(args.onLit).toHaveBeenCalledWith('entropy'))

    await userEvent.keyboard('{ArrowDown}')
    await waitFor(() => expect(args.onLit).toHaveBeenCalledWith('enthalpy'))

    await userEvent.keyboard('{ArrowUp}')
    await waitFor(() => expect(args.onLit).toHaveBeenLastCalledWith('entropy'))
  },
}

/**
 * One item, which is where it opens. Walking a list of one moves nothing, so
 * it is said once and not again.
 */
export const LitAlone: Story = {
  args: {
    bands: [{ id: 'names', title: 'Names', items: [named('entropy', 'Entropy', 'ent')] }],
  },
  play: async ({ args }) => {
    await waitFor(() => expect(args.onLit).toHaveBeenCalledWith('entropy'))

    await userEvent.keyboard('{ArrowDown}')
    await userEvent.keyboard('{End}')
    await expect(args.onLit).toHaveBeenCalledTimes(1)
  },
}

/** Every band came back with nothing, so there is nowhere to stand. */
export const LitNothing: Story = {
  args: {
    bands: [{ id: 'names', title: 'Names', items: [], silence: 'No name holds those words' }],
  },
  play: async ({ args }) => {
    await waitFor(() => expect(palette()).not.toBeNull())

    await userEvent.keyboard('{ArrowDown}')
    await expect(args.onLit).not.toHaveBeenCalled()
  },
}

/** Far too many, walked to the end: every row it crosses is said, in order. */
export const LitFarDown: Story = {
  args: {
    bands: [
      {
        id: 'names',
        title: 'Names',
        items: Array.from({ length: 60 }, (_, at) =>
          named(`note-${at}`, `Entropy in ${at + 1} dimensions`, 'ent'),
        ),
      },
    ],
  },
  play: async ({ args }) => {
    await waitFor(() => expect(args.onLit).toHaveBeenCalledWith('note-0'))

    for (let step = 0; step < 59; step += 1) await userEvent.keyboard('{ArrowDown}')

    await waitFor(() => expect(args.onLit).toHaveBeenLastCalledWith('note-59'))
    await expect(args.onLit).toHaveBeenCalledTimes(60)
  },
}

/**
 * What is never said: a row that cannot be chosen, and anything at all while
 * the action panel stands. The panel is about the item that was lit when it
 * opened, and the keyboard is in a list of its own.
 */
export const NotLit: Story = {
  args: {
    bands: [
      {
        id: 'names',
        title: 'Names',
        items: [
          { ...named('entropy', 'Entropy', 'ent'), actions: EVERYTHING },
          { id: 'stuck', title: 'Enthalpy — this file cannot be read', disabled: true },
          named('gibbs', 'Gibbs free energy', 'e'),
        ],
      },
    ],
  },
  play: async ({ args }) => {
    await waitFor(() => expect(args.onLit).toHaveBeenCalledWith('entropy'))

    await userEvent.keyboard('{ArrowDown}')
    await waitFor(() => expect(args.onLit).toHaveBeenLastCalledWith('gibbs'))
    await expect(args.onLit).not.toHaveBeenCalledWith('stuck')

    await userEvent.click(within(document.body).getByText(/cannot be read/))
    await expect(args.onLit).toHaveBeenLastCalledWith('gibbs')

    await userEvent.keyboard('{ArrowUp}')
    await waitFor(() => expect(args.onLit).toHaveBeenCalledTimes(3))

    await userEvent.keyboard('{Control>}k{/Control}')
    await waitFor(() => expect(sheet()).not.toBeNull())
    await userEvent.keyboard('{ArrowDown}')
    await expect(args.onLit).toHaveBeenCalledTimes(3)
  },
}

/**
 * A band arriving while it is being read.
 *
 * The band on its way lands above the one the keyboard is in, so every number
 * in the list moves. What is lit is the item, not the number, and it stays put.
 */
export const Filling: Story = {
  render: (args) => ({
    components: { Palette },
    setup() {
      const open = ref(true)
      const typed = ref('ent')
      const bands = ref<PaletteBand[]>([
        { ...NAMES, items: [], working: true },
        TEXT,
      ])

      let waiting: ReturnType<typeof setTimeout> | undefined
      onMounted(() => {
        waiting = setTimeout(() => (bands.value = [NAMES, TEXT]), 700)
      })
      onUnmounted(() => clearTimeout(waiting))

      return { args, open, typed, bands }
    },
    template: `
      <div class="numen" style="height:100vh;background:var(--numen-surface)">
        <Palette
          v-model="typed"
          :bands="bands"
          :open="open"
          @choose="args.onChoose"
          @dismiss="args.onDismiss"
        />
      </div>
    `,
  }),
  play: async () => {
    // The only item there is to begin with is in the second band.
    await waitFor(() => expect(lit()?.textContent).toContain('Heat engines'))
    await waitFor(() => expect(options().length).toBe(6), { timeout: 3000 })
    await expect(lit()?.textContent).toContain('Heat engines')
  },
}

/** Every band came back with nothing, each saying so in its own words. */
export const Nothing: Story = {
  args: {
    bands: [
      { id: 'names', title: 'Names', items: [], silence: 'No name holds those words' },
      { id: 'text', title: 'Text', items: [], silence: 'Nothing is written with them' },
      { id: 'meaning', title: 'Meaning', items: [], silence: 'No model is set' },
    ],
  },
}

/** Nothing has been typed yet, so there is no band to draw at all. */
export const Unasked: Story = {
  args: { bands: [] },
}

/** One band, one item, one thing to do with it. */
export const Alone: Story = {
  args: {
    bands: [{ id: 'names', title: 'Names', items: [named('entropy', 'Entropy', 'ent')] }],
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
    bands: [
      {
        id: 'names',
        title: 'Names',
        items: Array.from({ length: 60 }, (_, at) =>
          named(`note-${at}`, `Entropy in ${at + 1} dimensions`, 'ent'),
        ),
      },
    ],
  },
  play: async () => {
    await waitFor(() => expect(lit()).not.toBeNull())

    for (let step = 0; step < 59; step += 1) await userEvent.keyboard('{ArrowDown}')

    const last = options().at(-1)!
    await expect(last.getAttribute('aria-selected')).toBe('true')

    const list = document.body.querySelector<HTMLElement>('.palette__list')!
    const inside = last.getBoundingClientRect()
    const room = list.getBoundingClientRect()
    await expect(inside.bottom).toBeLessThanOrEqual(Math.ceil(room.bottom))
    await expect(inside.top).toBeGreaterThanOrEqual(Math.floor(room.top))
  },
}

/** Scripts that are not Latin, one of which runs the other way. */
export const NotLatin: Story = {
  args: {
    bands: [
      {
        id: 'names',
        title: 'Названия',
        items: [
          named('ru', RUSSIAN, 'заметки'),
          named('sa', DEVANAGARI, 'वेद'),
          named('ar', ARABIC, 'السطر'),
        ],
      },
    ],
  },
}

/**
 * Lines past any width, and words with nowhere to break.
 *
 * Each is one line and then an ellipsis, and none of them makes the panel any
 * wider than it is.
 */
export const TooLong: Story = {
  args: {
    bands: [
      {
        id: 'names',
        title: 'Names',
        items: [
          named('long', LONG, 'component'),
          named('word', UNBREAKABLE, 'schiff'),
          passage('link', LINK, LINK, 'query'),
        ],
      },
    ],
  },
  play: async () => {
    await waitFor(() => expect(palette()).not.toBeNull())

    const panel = document.body.querySelector<HTMLElement>('.palette__panel')!
    await expect(panel.getBoundingClientRect().width).toBeLessThanOrEqual(640)
    for (const option of options()) {
      await expect(option.scrollWidth).toBeLessThanOrEqual(option.clientWidth + 1)
    }
  },
}

/** Characters that are several code units each, with a run landing inside one. */
export const Graphemes: Story = {
  args: {
    bands: [
      {
        id: 'names',
        title: 'Names',
        items: [
          { id: 'whole', title: GRAPHEMES, at: [{ from: 0, to: 1 }], actions: TRAVEL },
          named('mixed', `Waving 👋🏽 at the reader`, 'reader'),
        ],
      },
    ],
  },
}

/** An item that is drawn and cannot be chosen, beside ones that can. */
export const NotToBeChosen: Story = {
  args: {
    bands: [
      {
        id: 'names',
        title: 'Names',
        items: [
          named('entropy', 'Entropy', 'ent'),
          { id: 'stuck', title: 'Enthalpy — this file cannot be read', disabled: true },
          named('gibbs', 'Gibbs free energy', 'e'),
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
 * One band answered and another came back with nothing.
 *
 * The empty one is still drawn — it says the question was asked — and it stands
 * at the foot, out of the way of what a person is actually reading.
 */
export const SomeCameBackEmpty: Story = {
  args: {
    bands: [
      { id: 'names', title: 'Names', items: [], silence: 'No name holds those words' },
      TEXT,
      { ...MEANING, working: true },
    ],
  },
  play: async () => {
    await waitFor(() => expect(lit()).not.toBeNull())

    const drawn = Array.from(document.body.querySelectorAll('.palette__title')).map((band) =>
      band.textContent?.trim(),
    )
    await expect(drawn).toEqual(['Text', 'Meaning', 'Names'])
    await expect(lit()?.textContent).toContain('Heat engines')
  },
}

/** One item offering five things, where two of them have a key. */
export const FiveActions: Story = {
  args: {
    bands: [
      {
        id: 'names',
        title: 'Names',
        items: [
          { ...named('entropy', 'Entropy', 'ent'), actions: EVERYTHING.slice(0, 5) },
          named('enthalpy', 'Enthalpy of formation', 'ent'),
        ],
      },
    ],
  },
  play: async ({ args }) => {
    await waitFor(() => expect(lit()).not.toBeNull())

    const said = Array.from(document.body.querySelectorAll('.palette__key')).map((one) =>
      one.textContent?.trim(),
    )
    await expect(said).toEqual(['↵ Show in plex', '⇧↵ Open the note'])
    await expect(document.body.querySelector('.palette__more')?.textContent).toContain('Actions')

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
    bands: [
      {
        id: 'names',
        title: 'Names',
        items: [
          { ...named('entropy', 'Entropy', 'ent'), actions: EVERYTHING },
          named('enthalpy', 'Enthalpy of formation', 'ent'),
        ],
      },
    ],
  },
  play: async () => {
    await waitFor(() => expect(lit()).not.toBeNull())

    await userEvent.keyboard('{Control>}k{/Control}')
    await waitFor(() => expect(sheet()).not.toBeNull())

    await expect(deeds()).toHaveLength(EVERYTHING.length)
    await expect(document.activeElement).toBe(hunt())
    await expect(field()?.getAttribute('aria-activedescendant')).toBeNull()
    await expect(hunt()?.getAttribute('aria-activedescendant')).toBe(deeds()[0]?.id)

    // Walking back from the first brings the last row into sight.
    await userEvent.keyboard('{ArrowUp}')
    const last = deeds().at(-1)!
    const inside = last.getBoundingClientRect()
    const room = document.body.querySelector<HTMLElement>('.palette__deeds')!.getBoundingClientRect()
    await expect(inside.bottom).toBeLessThanOrEqual(Math.ceil(room.bottom))
    await expect(inside.top).toBeGreaterThanOrEqual(Math.floor(room.top))

    await userEvent.keyboard('{Home}')
    await userEvent.keyboard('name')
    await waitFor(() => expect(deeds()).toHaveLength(1))
    await expect(deeds()[0]?.textContent).toContain('Rename')

    await userEvent.clear(hunt()!)
    await waitFor(() => expect(deeds()).toHaveLength(EVERYTHING.length))
  },
}

/**
 * Two steps: pick a thing, then name it. The palette draws one step at a time;
 * the caller keeps the stack and swaps the bands, the words in the field and
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

      const titleOf = (id: string) =>
        NAMED.items.find((one) => one.id === id)?.title ?? ''

      const choose = (item: string, action: string) => {
        args.onChoose(item, action)
        if (action !== 'rename') return
        crumb.value = `New name for «${titleOf(item)}»`
        typed.value = titleOf(item)
        step.value = 'name'
      }

      const back = () => {
        args.onBack()
        crumb.value = ''
        typed.value = ''
        step.value = 'find'
      }

      return { args, open, step, typed, crumb, choose, back, NAMED }
    },
    template: `
      <div class="numen" style="height:100vh;background:var(--numen-surface)">
        <Palette
          v-model="typed"
          :bands="step === 'find' ? [NAMED] : []"
          :open="open"
          :step="step"
          :crumb="crumb"
          :placeholder="step === 'find' ? 'Search' : 'The new name'"
          @choose="choose"
          @back="back"
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
    await waitFor(() => expect(deeds()).toHaveLength(1))
    await userEvent.keyboard('{Enter}')

    await waitFor(() =>
      expect(document.body.querySelector('.palette__crumb')?.textContent).toBe(
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

/** Items carrying the keystroke that reaches them away from the palette. */
export const KeyHints: Story = {
  args: {
    bands: [
      {
        id: 'commands',
        title: 'Commands',
        items: [
          { id: 'new', title: 'New note', keys: '⌘N', actions: RUN },
          { id: 'plex', title: 'Show the plex', keys: '⌘⇧P', actions: RUN },
          { id: 'save', title: 'Save', keys: '⌘S', actions: RUN },
          { id: 'long', title: LONG, keys: '⌘⌥⇧L', actions: RUN },
          { id: 'none', title: 'Reload the vault', actions: RUN },
        ],
      },
    ],
  },
  play: async () => {
    await waitFor(() => expect(lit()).not.toBeNull())

    const hints = Array.from(document.body.querySelectorAll('.palette__hint')).map((one) =>
      one.textContent?.trim(),
    )
    await expect(hints).toEqual(['⌘N', '⌘⇧P', '⌘S', '⌘⌥⇧L'])

    // A title too long for the row gives way to the key, and neither wraps.
    for (const option of options()) {
      await expect(option.scrollWidth).toBeLessThanOrEqual(option.clientWidth + 1)
    }
  },
}
