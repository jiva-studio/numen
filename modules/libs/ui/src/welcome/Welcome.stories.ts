/**
 * The screen a window opens on, drawn in windows of the shapes it has to
 * survive.
 *
 * The claim is the arrangement: while the whole of it fits by height it is one
 * column in the middle, and where it does not the ways in and the vaults stand
 * side by side. Each story is a window of a fixed size, because that size is
 * what the claim is about.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { Bot, FolderPlus, Settings, SquarePen, Waypoints } from '@lucide/vue'
import Welcome from './Welcome.vue'
import type { Held, Offer, Way } from './welcome'

/** The ways in the editor's window offers, in the order it offers them. */
const WAYS: readonly Way[] = [
  { id: 'find', text: 'Search the vault', keys: { marks: ['control'], letter: 'K' } },
  { id: 'commands', text: 'Show the commands', keys: { marks: ['control', 'shift'], letter: 'P' } },
  { id: 'note', text: 'New note', icon: SquarePen, keys: { marks: ['control'], letter: 'N' } },
  { id: 'plex', text: 'Open plex', icon: Waypoints, keys: { marks: ['control'], letter: 'G' } },
  { id: 'agent', text: 'New agent', icon: Bot, keys: { marks: ['control', 'shift'], letter: 'A' } },
  { id: 'settings', text: 'Settings', icon: Settings, keys: { marks: ['control'], letter: ',' } },
]

const OFFER: Offer = {
  text: 'New vault',
  detail: 'Choose a folder',
  icon: FolderPlus,
  keys: { marks: ['control', 'shift'], letter: 'N' },
}

const held = (name: string, path: string, detail?: string): Held => ({
  id: name.toLowerCase(),
  name,
  path,
  ...(detail ? { detail } : {}),
})

const VAULTS: readonly Held[] = [
  held('Studies', '/home/rowan/vaults/studies', 'Open'),
  held('Sanskrit', '/home/rowan/vaults/sanskrit'),
  held('Fieldwork', '/home/rowan/Documents/fieldwork'),
]

/** More vaults than a short window has room for, which is where the list scrolls. */
const MANY: readonly Held[] = [
  ...VAULTS,
  held('Птицы', '/home/rowan/vaults/birds'),
  held('Boltzmann', '/home/rowan/vaults/boltzmann'),
  held('Allotments', '/home/rowan/vaults/allotments'),
  held('Letters', '/home/rowan/Documents/letters'),
  held('Recipes', '/home/rowan/vaults/recipes'),
  held('The rota', '/home/rowan/vaults/rota'),
  held('Weather', '/home/rowan/vaults/weather'),
  held('Hedgerow', '/home/rowan/vaults/hedgerow'),
  held('Marrowfield', '/home/rowan/vaults/marrowfield'),
]

interface Knobs {
  /** How wide the window is, in pixels. */
  wide: number
  /** How tall the window is, in pixels. */
  high: number
  ways: readonly Way[]
  vaults: readonly Held[]
}

const meta: Meta<Knobs> = {
  title: 'Application/Welcome',
  parameters: { layout: 'fullscreen' },
  argTypes: {
    wide: { control: { type: 'range', min: 320, max: 1600, step: 20 } },
    high: { control: { type: 'range', min: 240, max: 1000, step: 20 } },
  },
  args: { wide: 900, high: 640, ways: WAYS, vaults: VAULTS },
  render: (args) => ({
    components: { Welcome },
    setup: () => ({ args, OFFER }),
    template: `
      <div
        :style="{
          inlineSize: args.wide + 'px',
          blockSize: args.high + 'px',
          background: 'var(--numen-surface)',
          fontFamily: 'var(--numen-font-sans)',
          fontSize: 'var(--numen-font-size)',
          lineHeight: 'var(--numen-line-height)',
        }"
      >
        <Welcome
          :ways="args.ways"
          :vaults="args.vaults"
          heading="Vaults"
          :offer="OFFER"
          version="0.4.1"
        />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

/** A window with the height for all of it: one column in the middle. */
export const Opening: Story = {}

/** Wide and not tall: the ways in on the left, the vaults on the right. */
export const ShortAndWide: Story = {
  args: { wide: 1120, high: 420 },
}

/** The same window with more vaults than it can hold, where the list scrolls. */
export const ShortAndWideWithManyVaults: Story = {
  args: { wide: 1120, high: 420, vaults: MANY },
}

/** Short and narrow at once: no width to stand in two, so the whole of it scrolls. */
export const ShortAndNarrow: Story = {
  args: { wide: 420, high: 420 },
}

/** The flashcards window, which offers no way in and opens on the list alone. */
export const Flashcards: Story = {
  args: { wide: 1120, high: 420, ways: [], vaults: MANY },
  render: (args) => ({
    components: { Welcome },
    setup: () => ({ args }),
    template: `
      <div
        :style="{
          inlineSize: args.wide + 'px',
          blockSize: args.high + 'px',
          background: 'var(--numen-surface)',
          fontFamily: 'var(--numen-font-sans)',
          fontSize: 'var(--numen-font-size)',
          lineHeight: 'var(--numen-line-height)',
        }"
      >
        <Welcome name="flashcards" :vaults="args.vaults" heading="Vaults" version="0.4.1" />
      </div>
    `,
  }),
}
