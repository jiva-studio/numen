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
import type { PaletteSection, PaletteSpan } from './model'
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
  sections: readonly PaletteSection[]
  placeholder: string
  name: string
  onChoose: (item: string, action: string) => void
  onDismiss: () => void
}

/** What a plex can do with a name, and what a note can do with a place in it. */
const TRAVEL = [
  { id: 'travel', text: 'Show in plex' },
  { id: 'read', text: 'Open the note' },
]
const READ = [{ id: 'read', text: 'Open the note' }]

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
const NAMES: PaletteSection = {
  id: 'names',
  title: 'Names',
  items: [
    named('entropy', 'Entropy', 'ent'),
    named('enthalpy', 'Enthalpy of formation', 'ent'),
    inside('carnot', 'The Carnot cycle', 'Entropy over one cycle', 'ent'),
    inside('gibbs', 'Gibbs free energy', 'Entropy and the second law', 'ent'),
  ],
}

const TEXT: PaletteSection = {
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

const MEANING: PaletteSection = {
  id: 'meaning',
  title: 'Meaning',
  items: [
    // Nothing is marked: a hit by meaning has no run to point at, and saying so
    // by drawing none is the honest answer.
    passage('arrow', 'The arrow of time', 'why the past and the future are not alike', ''),
  ],
}

const ALL: PaletteSection[] = [NAMES, TEXT, { ...MEANING, working: true }]

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
        :sections="args.sections"
        :open="open"
        :placeholder="args.placeholder"
        :name="args.name"
        @choose="args.onChoose"
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
    name: { control: 'text' },
    sections: { table: { disable: true } },
    onChoose: { table: { disable: true } },
    onDismiss: { table: { disable: true } },
  },
  args: {
    sections: ALL,
    placeholder: 'Search',
    name: 'Palette',
    onChoose: fn(),
    onDismiss: fn(),
  },
  render: over,
} satisfies Meta<Knobs>

export default meta
type Story = StoryObj<typeof meta>

const palette = () => document.body.querySelector<HTMLElement>('.palette')
const lit = () => document.body.querySelector<HTMLElement>('[data-here]')
const options = () => Array.from(document.body.querySelectorAll<HTMLElement>('[role="option"]'))

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
      const sections = ref<PaletteSection[]>([
        { ...NAMES, items: [], working: true },
        TEXT,
      ])

      let waiting: ReturnType<typeof setTimeout> | undefined
      onMounted(() => {
        waiting = setTimeout(() => (sections.value = [NAMES, TEXT]), 700)
      })
      onUnmounted(() => clearTimeout(waiting))

      return { args, open, typed, sections }
    },
    template: `
      <div class="numen" style="height:100vh;background:var(--numen-surface)">
        <Palette
          v-model="typed"
          :sections="sections"
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
    sections: [
      { id: 'names', title: 'Names', items: [], silence: 'No name holds those words' },
      { id: 'text', title: 'Text', items: [], silence: 'Nothing is written with them' },
      { id: 'meaning', title: 'Meaning', items: [], silence: 'No model is set' },
    ],
  },
}

/** Nothing has been typed yet, so there is no band to draw at all. */
export const Unasked: Story = {
  args: { sections: [] },
}

/** One band, one item, one thing to do with it. */
export const Alone: Story = {
  args: {
    sections: [{ id: 'names', title: 'Names', items: [named('entropy', 'Entropy', 'ent')] }],
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
    sections: [
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
    sections: [
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
    sections: [
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
    sections: [
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
    sections: [
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
    sections: [
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
