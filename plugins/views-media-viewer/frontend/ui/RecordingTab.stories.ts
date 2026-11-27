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
import type { Player } from '../model/mediaPlayer'
import type { Recordings, Cue } from '../types'
import { WORDS } from '../words'

const MEDIA = 'http://127.0.0.1:1/token/v/Lectures/Lecture 4.mp3'
const PATH = 'Lectures/Lecture 4.mp3'
const RUNS = 1_842_000
const NOW = 47_200

/** The lecturer's speech, one paragraph per cue. */
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
  'Let me make the demon as small as it can be. Szilard did this in 1929: one molecule, one box, nothing else.',
  'You slide a partition into the middle of the box. The molecule is now on the left or on the right, and you do not know which.',
  'Then you look. One bit: left or right. And knowing that bit, you can let the molecule push the partition like a piston.',
  'The molecule does work on the way out — kT ln 2 of it, if you let it expand slowly. Taken from the heat of the walls.',
  'Take the partition out, put it back, look again. Every cycle turns a bit of knowledge into a bit of work.',
  'Szilard thought the measurement must cost at least what it earns. He was right about the amount and wrong about the place.',
  'Yes, at the back? — No, the molecule does not need to be slow. Any molecule will do; the box only has to be at one temperature.',
  'Where the cost really sits was found much later, and not by a physicist of heat but by a man building computers at IBM.',
  'Landauer, 1961. His question was a practical one: what is the least heat a computer has to give off for each step?',
  'His answer: most steps need give off nothing at all. Copying a bit, moving a bit, combining bits so you can undo it — all free in principle.',
  'Only one step is never free. Erasing. Setting a bit to zero when you do not know what it held before.',
  'Think of what erasing does. Two different states, a one and a zero, go to the same state. The memory forgets which it was.',
  'Two arrangements become one. And by now you know what that means: the entropy of the memory falls by k ln 2.',
  'The second law does not let that entropy vanish. It goes into the surroundings as heat, at least kT ln 2 of it.',
  'So now go back to the demon. It measures, and the measuring can be free. It stores the answer, and storing is free.',
  'But its memory is finite. Sooner or later it has to make room, and making room is erasing.',
  'At that moment it pays back everything the sorting gained. Not a little more, not a little less. Exactly the bill.',
  'Bennett made this clean in 1982. He showed the measurement itself can be done reversibly, so the measurement was never the culprit.',
  'And he went further: a whole computer can be built that never erases, if it keeps a record of every step and runs back over it.',
  'You compute the answer, copy it out onto a blank page, then undo every step until the scratch memory is clean again.',
  'The copy is the only thing that is new, and writing on a blank page costs nothing. The machine gives off no heat it must give off.',
  'Why do real computers run hot, then? Because they erase all the time, and they run fast. Reversible means slow.',
  'Your laptop gives off something like a billion times the Landauer bound for each bit it clears. There is a lot of room at the bottom.',
  'In 2012 a group in Lyon measured the bound itself, with one glass bead held in a laser trap. It came out at kT ln 2, within the error.',
  'So when somebody tells you information is abstract, remember the bead. A bit is written in something, and wiping it out warms something.',
  'A word on the exam, since three of you asked. Nothing from today will be asked as a derivation. You will be asked to explain it.',
  'If you can say in two sentences why the demon fails, and where the heat goes, you have understood today.',
  'Read Landauer for next week — the 1961 paper, it is nine pages — and have a look at Bennett as well.',
]

/** One cue per paragraph, at the pace of speech. */
const SPOKEN: readonly Cue[] = SAID.reduce<Cue[]>((cues, text) => {
  const from = cues.length === 0 ? 0 : (cues.at(-1)?.to ?? 0)
  return [...cues, { text, from, to: from + 2_400 + text.length * 62 }]
}, [])

const createRecordings = (cues: readonly Cue[], canCreateText: boolean): Recordings => ({
  getSource: async () => ({ mediaUrl: MEDIA, mediaType: 'audio/mpeg', pageUrl: '' }),
  readTranscript: async () => cues,
  readArticle: async () => '',
  writeTranscript: async () => undefined,
  findCueTime: async () => null,
  canCreateText: async () => canCreateText,
  createText: async () => undefined,
  watchChanges: () => () => undefined,
  watchWork: (_, onChange) => {
    onChange(false)
    return () => undefined
  },
})

const createPlayer = (isPlaying: boolean): Player => ({
  state: {
    url: ref(MEDIA),
    position: ref(NOW),
    duration: ref(RUNS),
    isPlaying: ref(isPlaying),
    error: ref(''),
  },
  load: () => undefined,
  play: () => undefined,
  pause: () => undefined,
  seek: () => undefined,
  getDuration: async () => RUNS,
  surface: { attach: () => undefined, release: () => undefined },
})

const meta: Meta = {
  title: 'Recording/Tab',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

const render =
  (cues: readonly Cue[], canCreateText: boolean, isPlaying: boolean, height = 'h-screen') =>
  () => ({
    components: { RecordingTab },
    setup: () => ({
      path: PATH,
      recordings: createRecordings(cues, canCreateText),
      how: { audioPlayer: createPlayer(isPlaying), plays: () => true },
    }),
    template: `<div class="numen ${height} bg-surface"><RecordingTab :path="path" :recordings="recordings" :how="how" /></div>`,
  })

/** A lecture playing, with the words heard in it under the player. */
export const Playing: Story = {
  tags: ['screenshot'],
  parameters: {
    screenshot: [
      { name: 'recording', app: 'docs' },
      { name: 'window-audio', app: 'landing' },
    ],
  },
  render: render(SPOKEN, true, true),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(() => expect(canvas.getByText(SAID[0]!)).toBeVisible())
    // The line at the player's current time is drawn as current.
    const current = canvasElement.querySelector('.cm-line.cm-current')
    await expect(current?.textContent).toBe(SAID[5])
  },
}

/** A recording nothing has written down, and the run that would. */
export const NoTranscript: Story = {
  tags: ['screenshot'],
  parameters: {
    screenshot: [
      { name: 'transcribe', app: 'docs', drawn: 5_000 },
      { name: 'window-transcript', app: 'landing', drawn: 5_000 },
    ],
  },
  render: render([], true, false),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(() =>
      expect(canvas.getByRole('button', { name: WORDS.transcribe })).toBeVisible(),
    )
  },
}

const LECTURE = { title: 'Lecture 4', icon: 'audio' }

/** The lecture playing in a tab of the window, beside the note it is about. */
export const InTheWindow: Story = {
  ...Playing,
  tags: ['screenshot'],
  parameters: {
    frame: {
      place: 'tab',
      tab: LECTURE,
      note: 'after',
      share: 0.56,
      notePath: "Physics/Maxwell's demon.md",
      selected: 'Lectures/Lecture 4.mp3',
    },
    screenshot: [{ name: 'recording' }],
  },
  render: render(SPOKEN, true, true, 'h-full'),
}

/** The transcript of the lecture, a time against each paragraph, in a tab of the window. */
export const TranscriptInTheWindow: Story = {
  tags: ['screenshot'],
  parameters: {
    frame: {
      place: 'tab',
      tab: LECTURE,
      otherTabs: [{ title: 'Lecture 5', icon: 'audio' }],
      selected: 'Lectures/Lecture 4.mp3',
    },
    screenshot: [{ name: 'transcript', plugin: 'engines-transcription' }],
  },
  render: render(SPOKEN, true, false, 'h-full'),
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.querySelector('.cm-time')).not.toBeNull())
  },
}
