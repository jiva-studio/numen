/**
 * A recording tab, drawn: the player over the transcript, and the run offered
 * where nothing has written the recording down yet.
 *
 * The recording and the window's player are made up here, so nothing plays and
 * nothing is read from a vault.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor, within } from 'storybook/test'
import { ref } from 'vue'
import RecordingTab from './RecordingTab.vue'
import type { Cue } from './lib/cues'
import type { Player } from './model/player'
import type { MenuEntry, Recordings } from './types'
import { WORDS } from './words'

const MEDIA = 'http://127.0.0.1:1/token/v/Lectures/Lecture 4.mp3'
const PATH = 'Lectures/Lecture 4.mp3'
const RUNS = 1_842_000
const NOW = 47_200

/** What the lecturer says, one paragraph to a cue. */
const SAID: readonly string[] = [
  'Last week we left the demon at the door, letting the fast molecules through one way and the slow ones the other.',
  'After a while one side is hot and the other is cold, and nothing was pushed and nothing was heated. That is the claim.',
  'For seventy years the answers to it were bad. People said the demon must warm up, or that seeing a molecule costs a photon.',
  'The point is that the demon has to know which molecule is which, and it has to hold that knowledge somewhere.',
  'The memory fills. And a finite memory has to be cleared before it can be used again.',
  'Clearing is where the bill arrives: at the moment the record of a molecule is destroyed. kT ln 2 for one bit.',
  'So what the second law forbids is not a clever machine. It forbids a description that improves itself for nothing.',
  'Shannon counts the messages that could have been sent; we count the arrangements the molecules could have been in.',
  'Entropy is not a property of a gas. It is a property of what you know about a gas.',
  'Read Landauer for next week — the 1961 paper, it is nine pages — and have a look at Bennett as well.',
]

/** One cue to a paragraph, at the pace a person is talked at. */
const SPOKEN: readonly Cue[] = SAID.reduce<Cue[]>((cues, text) => {
  const from = cues.length === 0 ? 0 : (cues.at(-1)?.to ?? 0)
  return [...cues, { text, from, to: from + 2_400 + text.length * 62 }]
}, [])

const TRANSCRIBE: MenuEntry = {
  action: 'createText',
  pluginId: 'md.numen.engines.transcription',
  versionId: '',
  isLive: false,
}

const TRANSCRIBED: MenuEntry = {
  action: 'deleteVersion',
  pluginId: 'md.numen.engines.transcription',
  versionId: 'md.numen.engines.transcription/lecture-4',
  isLive: true,
}

/** A recording that answers with these words and this menu. */
const createRecordings = (cues: readonly Cue[], menu: readonly MenuEntry[]): Recordings => ({
  getSource: async () => ({ mediaUrl: MEDIA, mediaType: 'audio/mpeg' }),
  readTranscript: async () => cues,
  writeTranscript: async () => {},
  findCueTime: async () => null,
  getMenu: async () => menu,
  runMenuEntry: async () => {},
  watchWork: (_, onChange) => {
    onChange(false)
    return () => {}
  },
})

/** The window's one player, standing at a moment of the recording and making no sound. */
const createPlayer = (isPlaying: boolean): Player => ({
  url: ref(MEDIA),
  position: ref(NOW),
  duration: ref(RUNS),
  isPlaying: ref(isPlaying),
  error: ref(''),
  load: () => {},
  play: () => {},
  pause: () => {},
  seek: () => {},
  getDuration: async () => RUNS,
})

const meta: Meta = {
  title: 'Recording/Tab',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

/** The tab filling the pane a window gives it. */
const render = (cues: readonly Cue[], menu: readonly MenuEntry[], isPlaying: boolean) => () => ({
  components: { RecordingTab },
  setup: () => ({
    path: PATH,
    recordings: createRecordings(cues, menu),
    how: { audioPlayer: createPlayer(isPlaying), plays: () => true },
  }),
  template: `<div class="numen h-screen bg-surface"><RecordingTab :path="path" :recordings="recordings" :how="how" /></div>`,
})

/** A lecture playing, with the words heard in it under the player. */
export const Playing: Story = {
  parameters: { reach: { keeps: 'Escape hands Tab back to the page' } },
  render: render(SPOKEN, [TRANSCRIBED], true),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(() => expect(canvas.getByText(SAID[0]!)).toBeVisible())
    // The line being said at the player's moment is the one drawn as current.
    const current = canvasElement.querySelector('.cm-line.cm-current')
    await expect(current?.textContent).toBe(SAID[5])
  },
}

/** A recording nothing has written down, and the run that would. */
export const NoTranscript: Story = {
  render: render([], [TRANSCRIBE], false),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(() =>
      expect(canvas.getByRole('button', { name: WORDS.transcribe })).toBeVisible(),
    )
  },
}
