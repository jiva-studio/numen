/**
 * The typing a user does before it is applied to the recording: the words appear on
 * screen as they are edited, and update the file only once typing pauses.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { formatErrorMessage } from '@numen/plugin-sdk'
import { useTranscriptDraft } from './draft'
import { getText, type Cue } from '../lib/cues'

const CUES: readonly Cue[] = [
  { text: 'The first thing said.', from: 0, to: 2_000 },
  { text: 'The second thing said.', from: 2_500, to: 5_000 },
]

/** The words have been still long enough to be written, and were, if anything was. */
const waitForStill = () => new Promise((resolve) => setTimeout(resolve, 20))

describe('a draft told nothing more is typed here', () => {
  it('writes nothing still waiting out its stillness', async () => {
    const writes: (readonly Cue[])[] = []
    const draft = useTranscriptDraft({
      cues: ref(CUES),
      isEditable: ref(true),
      error: ref(''),
      idleMs: 5,
      isOpen: () => true,
      write: async (next) => {
        writes.push(next)
      },
    })

    draft.setProse('The first thing meant.\nThe second thing said.')
    draft.stop()
    await waitForStill()

    expect(writes).toStrictEqual([])
  })
})

const failure = new Error('disk full')
const TEST_WRITE_IDLE_MS = 100
const EDITED = 'The first thing meant.\nThe second thing said.'
const EDITED_AGAIN = 'The first thing meant twice.\nThe second thing said.'

/** A write that stays pending until the test resolves it. */
const createPendingWrite = () => {
  const calls: (readonly Cue[])[] = []
  const answers: { resolve: () => void; reject: (why: Error) => void }[] = []
  const write = (next: readonly Cue[]) => {
    calls.push(next)
    return new Promise<void>((resolve, reject) => {
      answers.push({ resolve, reject })
    })
  }
  return { calls, answers, write }
}

/** A draft of the two cues on screen, over a recording that writes as the test directs. */
const createDraft = (
  how: {
    write?: (cues: readonly Cue[]) => Promise<void>
    isEditable?: boolean
    error?: string
  } = {},
) => {
  const state = { open: true }
  const cues = ref<readonly Cue[]>(CUES)
  const isEditable = ref(how.isEditable ?? true)
  const error = ref(how.error ?? '')
  const writes: (readonly Cue[])[] = []
  const draft = useTranscriptDraft({
    cues,
    isEditable,
    error,
    idleMs: TEST_WRITE_IDLE_MS,
    isOpen: () => state.open,
    write:
      how.write ??
      (async (next) => {
        writes.push(next)
      }),
  })
  draft.setText(getText(CUES))
  return { draft, state, cues, isEditable, error, writes }
}

describe('a draft whose words are typed over', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('writes the edited words once the typing has been still', async () => {
    const { draft, cues, writes } = createDraft()

    draft.setProse(EDITED)
    await vi.advanceTimersByTimeAsync(TEST_WRITE_IDLE_MS - 1)
    expect(writes).toStrictEqual([])
    await vi.advanceTimersByTimeAsync(1)

    expect(writes).toHaveLength(1)
    expect(writes[0]![0]!.text).toBe('The first thing meant.')
    expect(cues.value).toStrictEqual(writes[0])
  })

  it('waits for the last of the typing, and writes once', async () => {
    const { draft, writes } = createDraft()

    draft.setProse(EDITED)
    await vi.advanceTimersByTimeAsync(TEST_WRITE_IDLE_MS - 1)
    draft.setProse(EDITED_AGAIN)
    await vi.advanceTimersByTimeAsync(2)
    expect(writes).toStrictEqual([])
    await vi.advanceTimersByTimeAsync(TEST_WRITE_IDLE_MS)

    expect(writes).toHaveLength(1)
    expect(writes[0]![0]!.text).toBe('The first thing meant twice.')
  })

  it('keeps the view from following until the typing has been still', async () => {
    const { draft } = createDraft()

    expect(draft.typing.value).toBe(false)
    draft.setProse(EDITED)
    expect(draft.typing.value).toBe(true)
    await vi.advanceTimersByTimeAsync(TEST_WRITE_IDLE_MS - 1)
    draft.setProse(EDITED_AGAIN)
    await vi.advanceTimersByTimeAsync(2)
    expect(draft.typing.value).toBe(true)
    await vi.advanceTimersByTimeAsync(TEST_WRITE_IDLE_MS)

    expect(draft.typing.value).toBe(false)
  })

  it('takes no typing once the tab is closed', async () => {
    const { draft, state, writes } = createDraft()
    state.open = false

    draft.setProse(EDITED)
    await vi.advanceTimersByTimeAsync(TEST_WRITE_IDLE_MS)

    expect(draft.prose.value).toBe(getText(CUES))
    expect(draft.typing.value).toBe(false)
    expect(writes).toStrictEqual([])
  })

  it('takes the same words typed again as no typing at all', async () => {
    const { draft, writes } = createDraft()

    draft.setProse(getText(CUES))
    expect(draft.typing.value).toBe(false)
    await draft.keep()

    expect(writes).toStrictEqual([])
  })

  it('writes nothing for words that read as the recording has them', async () => {
    const { draft, writes } = createDraft()

    draft.setProse(EDITED)
    draft.setProse(getText(CUES))
    await draft.keep()

    expect(writes).toStrictEqual([])
  })

  it('writes nothing where there was no typing', async () => {
    const { draft, writes } = createDraft()

    await draft.keep()

    expect(writes).toStrictEqual([])
  })

  it('writes nothing while the transcript may not be written over', async () => {
    const { draft, writes } = createDraft({ isEditable: false })

    draft.setProse(EDITED)
    await draft.keep()

    expect(writes).toStrictEqual([])
  })

  it('writes nothing once the tab is closed', async () => {
    const { draft, state, writes } = createDraft()

    draft.setProse(EDITED)
    state.open = false
    await draft.keep()

    expect(writes).toStrictEqual([])
  })

  it('lets the words on screen be replaced by the recording until they are typed over', () => {
    const { draft } = createDraft()

    draft.setText('Heard again.')
    expect(draft.prose.value).toBe('Heard again.')
    draft.setProse(EDITED)
    draft.setText('Heard once more.')

    expect(draft.prose.value).toBe(EDITED)
  })

  it('lets the recording replace the words once what was typed is written', async () => {
    const { draft } = createDraft()

    draft.setProse(EDITED)
    await draft.keep()
    draft.setText('Heard again.')

    expect(draft.prose.value).toBe('Heard again.')
  })

  it('lets the recording replace the words once what was typed reads as it did', async () => {
    const { draft } = createDraft()

    draft.setProse(getText(CUES) + ' ')
    draft.setProse(getText(CUES))
    await draft.keep()
    draft.setText('Heard again.')

    expect(draft.prose.value).toBe('Heard again.')
  })

  it('clears what the recording could not do once the words are written', async () => {
    const { draft, error } = createDraft({ error: 'The write failed.' })

    draft.setProse(EDITED)
    await draft.keep()

    expect(error.value).toBe('')
  })

  it('does not write over a write that has not answered', async () => {
    const pending = createPendingWrite()
    const { draft } = createDraft({ write: pending.write })

    draft.setProse(EDITED)
    const first = draft.keep()
    draft.setProse(EDITED_AGAIN)
    await draft.keep()
    pending.answers[0]!.resolve()
    await first

    expect(pending.calls).toHaveLength(1)
  })

  it('writes the typing that landed while a write was in the air', async () => {
    const pending = createPendingWrite()
    const { draft } = createDraft({ write: pending.write })

    draft.setProse(EDITED)
    const first = draft.keep()
    draft.setProse(EDITED_AGAIN)
    await vi.advanceTimersByTimeAsync(TEST_WRITE_IDLE_MS)
    pending.answers[0]!.resolve()
    await first
    await vi.advanceTimersByTimeAsync(TEST_WRITE_IDLE_MS)

    expect(pending.calls).toHaveLength(2)
    expect(pending.calls[1]![0]!.text).toBe('The first thing meant twice.')
  })

  it('does not write the typing that landed during a write once the tab is closed', async () => {
    const pending = createPendingWrite()
    const { draft, state } = createDraft({ write: pending.write })

    draft.setProse(EDITED)
    const first = draft.keep()
    draft.setProse(EDITED_AGAIN)
    await vi.advanceTimersByTimeAsync(TEST_WRITE_IDLE_MS)
    state.open = false
    pending.answers[0]!.resolve()
    await first
    state.open = true
    await vi.advanceTimersByTimeAsync(TEST_WRITE_IDLE_MS)

    expect(pending.calls).toHaveLength(1)
  })

  it('leaves the recording as it was when the tab closed during the write', async () => {
    const pending = createPendingWrite()
    const { draft, state, cues, error } = createDraft({
      write: pending.write,
      error: 'The write failed.',
    })

    draft.setProse(EDITED)
    const first = draft.keep()
    state.open = false
    pending.answers[0]!.resolve()
    await first

    expect(cues.value).toStrictEqual(CUES)
    expect(error.value).toBe('The write failed.')
  })

  it('says what a refused write could not do, and offers the words again at the next keep', async () => {
    const pending = createPendingWrite()
    const { draft, cues, error } = createDraft({ write: pending.write })

    draft.setProse(EDITED)
    const first = draft.keep()
    pending.answers[0]!.reject(failure)
    await first

    expect(error.value).toBe(formatErrorMessage(failure))
    expect(cues.value).toStrictEqual(CUES)
    draft.setText('Heard again.')
    expect(draft.prose.value).toBe(EDITED)

    const second = draft.keep()
    pending.answers[1]!.resolve()
    await second
    expect(pending.calls).toHaveLength(2)
    expect(error.value).toBe('')
  })

  it('does not write a refused write again by itself', async () => {
    const pending = createPendingWrite()
    const { draft } = createDraft({ write: pending.write })

    draft.setProse(EDITED)
    const first = draft.keep()
    pending.answers[0]!.reject(failure)
    await first
    await vi.advanceTimersByTimeAsync(TEST_WRITE_IDLE_MS * 3)

    expect(pending.calls).toHaveLength(1)
  })

  it('says nothing of a refused write once the tab is closed', async () => {
    const pending = createPendingWrite()
    const { draft, state, error } = createDraft({ write: pending.write })

    draft.setProse(EDITED)
    const first = draft.keep()
    state.open = false
    pending.answers[0]!.reject(failure)
    await first
    state.open = true
    draft.setText('Heard again.')

    expect(error.value).toBe('')
    expect(draft.prose.value).toBe('Heard again.')
  })

  it('drops an edit waiting to be written', async () => {
    const { draft, writes } = createDraft()

    draft.setProse(EDITED)
    draft.drop()
    await vi.advanceTimersByTimeAsync(TEST_WRITE_IDLE_MS)
    await draft.keep()

    expect(writes).toStrictEqual([])
    draft.setText('Heard again.')
    expect(draft.prose.value).toBe('Heard again.')
  })

  it('offers dropped words again only after the stillness that follows a refused write', async () => {
    const pending = createPendingWrite()
    const { draft } = createDraft({ write: pending.write })

    draft.setProse(EDITED)
    const first = draft.keep()
    draft.setProse(EDITED_AGAIN)
    await vi.advanceTimersByTimeAsync(10)
    draft.drop()
    await vi.advanceTimersByTimeAsync(10)
    pending.answers[0]!.reject(failure)
    await first
    await vi.advanceTimersByTimeAsync(TEST_WRITE_IDLE_MS - 15)

    expect(pending.calls).toHaveLength(1)
  })

  it('takes the words off the screen and writes no more when stopped', async () => {
    const { draft, writes } = createDraft()

    draft.setProse(EDITED)
    draft.stop()
    await vi.advanceTimersByTimeAsync(TEST_WRITE_IDLE_MS)

    expect(draft.prose.value).toBe('')
    expect(writes).toStrictEqual([])
  })

  it('leaves the view where it was when stopped', async () => {
    const { draft } = createDraft()

    draft.setProse(EDITED)
    draft.stop()
    await vi.advanceTimersByTimeAsync(TEST_WRITE_IDLE_MS)

    expect(draft.typing.value).toBe(true)
  })
})
