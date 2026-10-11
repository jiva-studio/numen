/**
 * The typing a person does before it reaches the recording: the words stand
 * on screen as they are edited, and reach the file only once they have been
 * still.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { useTranscriptDraft } from './draft'
import type { Cue } from '../lib/cues'

const CUES: readonly Cue[] = [
  { text: 'The first thing said.', from: 0, to: 2_000 },
  { text: 'The second thing said.', from: 2_500, to: 5_000 },
]

/** The words have been still long enough to be written, and were, if anything was. */
const waitForStill = () => new Promise((done) => setTimeout(done, 20))

describe('a draft told nothing more is typed here', () => {
  it('writes nothing still waiting out its stillness', async () => {
    const written: (readonly Cue[])[] = []
    const draft = useTranscriptDraft({
      cues: ref(CUES),
      isEditable: ref(true),
      error: ref(''),
      quiet: 5,
      isOpen: () => true,
      write: async (next) => void written.push(next),
    })

    draft.setProse('The first thing meant.\nThe second thing said.')
    draft.stop()
    await waitForStill()

    expect(written).toStrictEqual([])
  })
})
