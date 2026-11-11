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
import { expect, fn, userEvent, within } from 'storybook/test'
import { Bot, FolderPlus, Settings, SquarePen, Waypoints } from '@lucide/vue'
import WelcomePage from './WelcomePage.vue'
import type { FooterEntry, VaultRow, WelcomeAction } from '../model/welcome'

/** The ways in the editor's window offers, in the order it offers them. */
const WAYS: readonly WelcomeAction[] = [
  { id: 'find', text: 'Search the vault', keys: { icons: ['control'], letter: 'K' } },
  { id: 'commands', text: 'Show commands', keys: { icons: ['control', 'shift'], letter: 'P' } },
  { id: 'note', text: 'New note', icon: SquarePen, keys: { icons: ['control'], letter: 'N' } },
  { id: 'plex', text: 'Open plex', icon: Waypoints, keys: { icons: ['control'], letter: 'G' } },
  { id: 'agent', text: 'New agent', icon: Bot, keys: { icons: ['control', 'shift'], letter: 'A' } },
  { id: 'settings', text: 'Settings', icon: Settings, keys: { icons: ['control'], letter: ',' } },
]

const FOOTER: FooterEntry = {
  text: 'New vault',
  detail: 'Choose a folder',
  icon: FolderPlus,
  keys: { icons: ['control', 'shift'], letter: 'N' },
}

const createVault = (name: string, path: string, detail?: string): VaultRow => ({
  id: name.toLowerCase(),
  name,
  path,
  ...(detail ? { detail } : {}),
})

const VAULTS: readonly VaultRow[] = [
  createVault('Studies', '/home/rowan/vaults/studies', 'Open'),
  createVault('Sanskrit', '/home/rowan/vaults/sanskrit'),
  createVault('Fieldwork', '/home/rowan/Documents/fieldwork'),
]

/** More vaults than fit in a short window, where the list scrolls. */
const MANY: readonly VaultRow[] = [
  ...VAULTS,
  createVault('Птицы', '/home/rowan/vaults/birds'),
  createVault('Boltzmann', '/home/rowan/vaults/boltzmann'),
  createVault('Allotments', '/home/rowan/vaults/allotments'),
  createVault('Letters', '/home/rowan/Documents/letters'),
  createVault('Recipes', '/home/rowan/vaults/recipes'),
  createVault('The rota', '/home/rowan/vaults/rota'),
  createVault('Weather', '/home/rowan/vaults/weather'),
  createVault('Hedgerow', '/home/rowan/vaults/hedgerow'),
  createVault('Marrowfield', '/home/rowan/vaults/marrowfield'),
]

interface Knobs {
  wide: number
  high: number
  ways: readonly WelcomeAction[]
  vaults: readonly VaultRow[]
  onRun?: (id: string) => void
  onOpen?: (id: string) => void
  onTakeFooter?: () => void
}

const meta: Meta<Knobs> = {
  title: 'Application/Welcome page',
  parameters: { layout: 'fullscreen' },
  argTypes: {
    wide: { control: { type: 'range', min: 320, max: 1600, step: 20 } },
    high: { control: { type: 'range', min: 240, max: 1000, step: 20 } },
  },
  args: { wide: 900, high: 640, ways: WAYS, vaults: VAULTS },
  render: (args) => ({
    components: { WelcomePage },
    setup: () => ({ args, FOOTER }),
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
        <WelcomePage
          :ways="args.ways"
          :vaults="args.vaults"
          heading="Vaults"
          :footer="FOOTER"
          version="0.4.1"
          @run="args.onRun?.($event)"
          @open="args.onOpen?.($event)"
          @take-footer="args.onTakeFooter?.()"
        />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

async function found(canvas: HTMLElement, selector: string): Promise<HTMLElement> {
  const el = canvas.querySelector<HTMLElement>(selector)
  await expect(el, `nothing here is ${selector}`).not.toBeNull()
  return el as HTMLElement
}

/** A window with the height for all of it: one column in the middle. */
export const Opening: Story = {
  play: async ({ canvasElement }) => {
    const page = await found(canvasElement, '.welcome-page')
    const mark = await found(canvasElement, '.welcome-page__glyph')

    // The mark stays a small part of the page.
    await expect(mark.getBoundingClientRect().height).toBeLessThan(
      page.getBoundingClientRect().height / 2,
    )
  },
}

/** A way in, a vault and the row below the list each emit what was pressed. */
export const EmitsWhatIsPressed: Story = {
  args: { onRun: fn(), onOpen: fn(), onTakeFooter: fn() },
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)

    await userEvent.click(canvas.getByRole('button', { name: /New note/ }))
    await expect(args.onRun).toHaveBeenCalledTimes(1)
    await expect(args.onRun).toHaveBeenCalledWith('note')

    await userEvent.click(canvas.getByRole('button', { name: /Sanskrit/ }))
    await expect(args.onOpen).toHaveBeenCalledTimes(1)
    await expect(args.onOpen).toHaveBeenCalledWith('sanskrit')

    await userEvent.click(canvas.getByRole('button', { name: /New vault/ }))
    await expect(args.onTakeFooter).toHaveBeenCalledTimes(1)
  },
}

/** Wide and not tall: the ways in on the left, the vaults on the right. */
export const ShortAndWide: Story = {
  args: { wide: 1120, high: 460 },
}

/** The same window with more vaults than it can hold, where the list scrolls. */
export const ShortAndWideWithManyVaults: Story = {
  args: { wide: 1120, high: 460, vaults: MANY },
}

export const TheMarkGivesWay: Story = {
  args: { wide: 1120, high: 340, vaults: MANY },
}

export const TheMarkHasGone: Story = {
  args: { wide: 1120, high: 300, vaults: MANY },
}

/** The shortest window: the six ways in and the list, and nothing above them. */
export const TheWaysInAndTheListAlone: Story = {
  args: { wide: 1120, high: 260, vaults: MANY },
}

/** Short and narrow at once: no width to stand in two, so the whole of it scrolls. */
export const ShortAndNarrow: Story = {
  args: { wide: 420, high: 420 },
}

/** The flashcards window, which offers no way in and opens on the list alone. */
export const Flashcards: Story = {
  args: { wide: 1120, high: 420, ways: [], vaults: MANY },
  render: (args) => ({
    components: { WelcomePage },
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
        <WelcomePage name="flashcards" :vaults="args.vaults" heading="Vaults" version="0.4.1" />
      </div>
    `,
  }),
}
