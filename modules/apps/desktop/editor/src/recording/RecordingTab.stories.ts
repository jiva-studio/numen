/**
 * A recording tab: the player heading the pane, and the transcript below it.
 *
 * What is asked here is what only a browser can answer. The editor is steered
 * through two custom properties — how wide the column of words runs, and how
 * much room stands above the first line — and both are resolved by layout and
 * by nothing else. Where the first line begins is a question about where the
 * browser put two boxes.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor, within } from 'storybook/test'
import { ref } from 'vue'
import RecordingTab from './RecordingTab.vue'
import { transcribed } from './kind'
import { transcript, type Cue, type Recordings } from './transcript'
import type { MediaTypeProbe, Player } from './player'
import { WORDS as words } from './words'

/** Two lines, said a minute and a half apart. */
const CUES: readonly Cue[] = [
  { text: 'The first thing said.', from: 0, to: 2_000 },
  { text: 'The second thing said.', from: 83_000, to: 85_000 },
]

/** A line long enough to be broken by the measure and not by anything else. */
const LONG: readonly Cue[] = [
  {
    text: `${'A sentence with nowhere in particular to break and no end to it either. '.repeat(6)}`,
    from: 0,
    to: 4_000,
  },
]

/** A recording that answers what it holds and writes nothing back. */
const talk = (cues: readonly Cue[]): Recordings => ({
  listened: async () => ({
    length: 85_000,
    heard: 85_000,
    media: 'http://127.0.0.1:1/w/v/talk.mp3',
    type: 'audio/mpeg',
  }),
  cues: async () => ({ cues, editable: true }),
  writes: async () => {},
  plays: async () => null,
})

/** The one player the window has, faked: nothing here makes a sound. */
const played = (): Player => {
  const address = ref('')
  return {
    address,
    at: ref(0),
    length: ref(85_000),
    playing: ref(false),
    failed: ref(''),
    load: (wanted) => void (address.value = wanted),
    play: (wanted) => void (address.value = wanted),
    pause: () => {},
    seek: (wanted) => void (address.value = wanted),
  }
}

const holding = (cues: readonly Cue[], plays: MediaTypeProbe = () => true) =>
  transcribed(transcript(talk(cues), 'talks/Ants.mp3', { through: played(), plays }), {
    runs: () => {},
  })

interface Knobs {
  /** What the recording was transcribed as. None is a tab with no words at all. */
  cues: readonly Cue[]
  /** How wide the pane holding the tab is. */
  width: string
}

const room = (args: Knobs) => ({
  components: { RecordingTab },
  setup: () => ({ args, held: holding(args.cues) }),
  template: `
    <div class="numen" :style="{ height: '100vh', width: args.width, background: 'var(--numen-surface)' }">
      <RecordingTab :held="held" />
    </div>
  `,
})

const meta: Meta<Knobs> = {
  title: 'Window/Recording',
  component: RecordingTab,
  parameters: { layout: 'fullscreen' },
  argTypes: { width: { control: 'text' }, cues: { table: { disable: true } } },
  args: { cues: CUES, width: '100%' },
  render: room,
}

export default meta
type Story = StoryObj<Knobs>

/** One recording with its transcript under the controls. */
export const ATranscript: Story = {}

/** The editor the transcript is written in, by the name the tab gives it. */
const transcriptIn = (canvas: HTMLElement): HTMLElement => {
  const host = canvas.querySelector<HTMLElement>(`[aria-label="${words.transcript}"]`)
  if (!host) throw new Error('no transcript')
  return host
}

/**
 * The words themselves, once the editor has drawn them. They are the editable
 * part of it, which is what the two custom properties are resolved against.
 */
const wordsIn = async (canvas: HTMLElement): Promise<HTMLElement> =>
  await waitFor(() => {
    const written = transcriptIn(canvas).querySelector<HTMLElement>('[contenteditable="true"]')
    if (!written) throw new Error('the transcript has not been drawn yet')
    return written
  })

/**
 * The player stands on a rule at the top of the pane, and the words are read
 * below that rule and clear of it.
 *
 * The room above the first line is a custom property, and whether it is kept is
 * a question about two boxes a browser placed.
 */
export const TheWordsClearThePlayer: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const written = await wordsIn(canvasElement)
    const follow = canvas.getByRole('button', { name: words.follow })

    // The controls stand in the strip the player heads the pane with, and that
    // strip takes room of its own.
    const strip = canvasElement.querySelector<HTMLElement>('.recording__head')!
    expect(strip.contains(follow)).toBe(true)
    expect(strip.getBoundingClientRect().height).toBeGreaterThan(0)

    // The first line begins below the strip, and keeps room above it.
    const first = await waitFor(() => within(written).getByText(CUES[0]!.text))
    expect(first.getBoundingClientRect().top).toBeGreaterThanOrEqual(
      strip.getBoundingClientRect().bottom,
    )
    expect(Number.parseFloat(getComputedStyle(written).paddingTop)).toBeGreaterThan(0)
  },
}

/**
 * The words are read at one measure however wide the pane is, and the controls
 * over them keep to it too.
 *
 * The measure is a custom property handed to the editor. Nothing without a
 * layout engine can say what it came to, or whether a line ever runs past it.
 */
export const TheWordsKeepTheirMeasure: Story = {
  args: { cues: LONG, width: '1600px' },
  play: async ({ canvasElement }) => {
    const written = await wordsIn(canvasElement)
    const pane = canvasElement.firstElementChild as HTMLElement

    // The measure is what the tab declares, in the pixels it comes to here.
    const measure = Number.parseFloat(getComputedStyle(written).maxWidth)
    expect(measure).toBeGreaterThan(0)

    // The pane is far wider than the measure, and the words are not.
    const room = pane.getBoundingClientRect()
    const column = written.getBoundingClientRect()
    expect(room.width).toBeGreaterThan(measure * 1.5)
    expect(column.width).toBeLessThanOrEqual(measure + 1)

    // The words and the times beside them are centred as one block, so the
    // column stands the gutter's width to the trailing side of the middle.
    const gutter = written.previousElementSibling?.getBoundingClientRect()
    expect(gutter?.width).toBeGreaterThan(0)
    const before = column.left - room.left
    const after = room.right - column.right
    expect(Math.abs(before - after - gutter!.width)).toBeLessThanOrEqual(2)

    // A line with nowhere to break is broken by the measure, so it runs down
    // the column rather than out of it.
    const line = within(written).getByText(LONG[0]!.text.trim(), { exact: false })
    expect(line.getBoundingClientRect().right).toBeLessThanOrEqual(column.right + 1)
    expect(line.getBoundingClientRect().height).toBeGreaterThan(
      Number.parseFloat(getComputedStyle(written).lineHeight) * 2,
    )
  },
}

/**
 * A recording nothing has written down: the player heads the pane as it always
 * does, and what can be asked stands in the middle of the room the words would
 * have had.
 */
export const NoTranscriptYet: Story = {
  args: { cues: [] },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(() => expect(canvas.queryByRole('button', { name: words.follow })).toBeNull())
    expect(canvasElement.querySelector(`[aria-label="${words.transcript}"]`)).toBeNull()

    const pane = (canvasElement.firstElementChild as HTMLElement).getBoundingClientRect()
    const player = canvas.getByRole('group', { name: words.player }).getBoundingClientRect()
    expect(player.top - pane.top).toBeLessThan(pane.height / 4)

    // What is asked over the recording stands in the middle of what is left,
    // down the page and across it.
    const ask = canvas.getByRole('button', { name: words.transcribe }).getBoundingClientRect()
    expect(Math.abs((ask.left + ask.right) / 2 - (pane.left + pane.right) / 2)).toBeLessThanOrEqual(
      2,
    )
    expect(ask.top).toBeGreaterThan(pane.top + pane.height / 4)
    expect(ask.bottom).toBeLessThan(pane.bottom - pane.height / 4)
  },
}
