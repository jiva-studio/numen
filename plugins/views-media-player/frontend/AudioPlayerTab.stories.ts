/**
 * The audio player: the words heard in a recording against the moment each was said, a recording
 * nothing has written down, and one a run is writing down.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import AudioPlayerTab from './AudioPlayerTab.vue'

const meta: Meta = {
  title: 'Media/Player',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

/** A player holding a transcript: Tab indents until Escape returns focus. */
const WRITING = { reach: { keeps: 'Escape returns Tab focus to the page' } }

interface Cue {
  readonly text: string
  readonly from: number
  readonly to: number
}

const cue = (from: number, to: number, text: string): Cue => ({ text, from, to })

/**
 * the transcript, line by line. Enough of it to fill the pane it is read in,
 * because a transcript is what a person scrolls through and not glances at.
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

/** The player drawn alone, filling the page, over the state it is handed. */
const player = (state: Record<string, unknown>) => ({
  components: { AudioPlayerTab },
  setup: () => ({ state }),
  template: '<div style="height: 100vh"><AudioPlayerTab :state="state" /></div>',
})

/** The words heard in a recording, against the moment each was said. */
export const Recording: Story = {
  parameters: WRITING,
  render: () => player({ url: MEDIA, now: 47_200, runs: RUNS, isPlaying: true, cues: SPOKEN }),
}

/** A recording nothing has written down, and the run that would. */
export const NoTranscript: Story = {
  render: () => player({ url: MEDIA, runs: RUNS, cues: [] }),
}

/** A recording a run is writing down: only what is below the player says a run is on. */
export const Transcribing: Story = {
  render: () => player({ url: MEDIA, runs: RUNS, cues: [], transcribable: false }),
}
