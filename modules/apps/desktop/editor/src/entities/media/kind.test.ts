/**
 * What the window says about a recording tab, asked without a screen.
 *
 * How far a recording has been written down and how long it runs are both
 * milliseconds, and both are what is reported of the tab.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { createRecordingKind, useTranscriptTab, type MediaTabState, type Medium } from './kind'
import type { TranscriptState } from './model/transcript'
import type { FileOpeners, SourceReader, WindowHandle } from '@/entities/tab/@x/media'
import type { Span } from '@/shared/span'

/** A medium drawn by nothing, which is as much of one as a kind is asked for. */
const played: Medium = {
  tab: 'recording',
  source: 'recording',
  pane: {},
  register: (puts, read) => puts.registerReader({ kind: played.source }, read),
}

/** A window holding no tabs. What is asked here is over the tab in hand. */
const tabs = { openTab: async () => '', getTabState: () => null } as unknown as WindowHandle

/** Openers that take the reader the kind hands over and ask nothing of it. */
const openers = { registerReader: () => {} } as unknown as FileOpeners

/** A recording open in a tab, as far as the window reads one. */
const recording = (path: string, transcript: number, duration: number) =>
  ({
    path,
    called: path.split('/').pop(),
    transcribedDuration: ref(transcript),
    duration: ref(duration),
  }) as unknown as MediaTabState

/** The kind, made over the one recording tab this test hands it. */
const kind = (state: MediaTabState) =>
  createRecordingKind(
    tabs,
    () => state as unknown as TranscriptState,
    { runCommand: () => {} },
    openers,
    played,
  ).kind

describe('what a command asked over a recording tab is over', () => {
  it('is the file it plays, which is what a run is asked over', () => {
    const held = recording('talks/Ants.mp3', 4000, 9000)

    expect(kind(held).getTarget!(held)).toStrictEqual({
      file: 'talks/Ants.mp3',
      source: 'recording',
    })
  })
})

describe('what a recording tab holds, as whoever answers for the person is told it', () => {
  it('is the file, how far it is written down, and how long it runs', () => {
    const held = recording('talks/Ants.mp3', 4000, 9000)

    expect(kind(held).getOpenTab!(held)).toStrictEqual({
      path: 'talks/Ants.mp3',
      recording: { transcribedDurationMs: 4000, durationMs: 9000 },
    })
  })
})

/**
 * A recording as the tab reads it: what it was found to hold, and whether the
 * finding has landed.
 */
const createReading = (how: { isLoading: boolean; times: readonly unknown[] }) =>
  ({
    path: 'talks/Ants.mp3',
    times: ref(how.times),
    points: ref(false),
    isWorking: ref(false),
    isLoading: ref(how.isLoading),
  }) as unknown as TranscriptState

describe('a recording whose words have not been read yet', () => {
  it('offers none of the runs over them', () => {
    const tab = useTranscriptTab(createReading({ isLoading: true, times: [] }), {
      runCommand: () => {},
    })

    expect(tab.transcribable.value).toBe(false)
    expect(tab.proofreadable.value).toBe(false)
    expect(tab.deletable.value).toBe(false)
  })

  it('offers writing them down once the reading has landed and found none', () => {
    const tab = useTranscriptTab(createReading({ isLoading: false, times: [] }), {
      runCommand: () => {},
    })

    expect(tab.transcribable.value).toBe(true)
  })

  it('offers putting them right once the reading has landed and found some', () => {
    const tab = useTranscriptTab(createReading({ isLoading: false, times: [{}] }), {
      runCommand: () => {},
    })

    expect(tab.transcribable.value).toBe(false)
    expect(tab.proofreadable.value).toBe(true)
    expect(tab.deletable.value).toBe(true)
  })
})

describe('how a recording tab is named and found again', () => {
  it('is called what the tab calls the recording', () => {
    const held = recording('talks/Ants.mp3', 0, 0)

    expect(kind(held).getTitle!(held)).toBe('Ants.mp3')
  })

  it('is found again by the file it plays', () => {
    const held = recording('talks/Ants.mp3', 0, 0)

    expect(kind(held).identity!('talks/Ants.mp3')).toBe('talks/Ants.mp3')
  })
})

describe('a recording tab being closed', () => {
  it('closes what it holds, and may close', () => {
    const close = vi.fn<() => void>()
    const held = { ...recording('talks/Ants.mp3', 0, 0), close } as unknown as MediaTabState

    expect(kind(held).onClose!(held, 'tab-1')).toBe(true)
    expect(close).toHaveBeenCalledTimes(1)
  })
})

describe('a hit in the words of a recording', () => {
  /** The reader the kind handed the openers, over a window whose tabs the test names. */
  const createReader = (tab: { reach: (...spans: readonly Span[]) => void } | null) => {
    let reader: SourceReader | undefined
    const opened: [string, string][] = []
    const window = {
      openTab: async (kindName: string, path: string) => {
        opened.push([kindName, path])
        return 'tab-1'
      },
      getTabState: () => tab,
    } as unknown as WindowHandle
    const medium: Medium = {
      ...played,
      register: (_puts, read) => {
        reader = read
      },
    }
    createRecordingKind(
      window,
      () => ({}) as unknown as TranscriptState,
      { runCommand: () => {} },
      openers,
      medium,
    )
    return { read: reader!, opened }
  }

  it('opens the recording and takes the person to the spans asked for', async () => {
    const reach = vi.fn<(...spans: Span[]) => void>()
    const { read, opened } = createReader({ reach })
    const spans: readonly Span[] = [{ from: 3, to: 9 }]

    read('talks/Ants.mp3', spans)
    await vi.waitFor(() => expect(reach).toHaveBeenCalledWith(...spans))

    expect(opened).toStrictEqual([['recording', 'talks/Ants.mp3']])
  })

  it('opens the recording and goes nowhere where the tab is gone', async () => {
    const { read, opened } = createReader(null)

    read('talks/Ants.mp3', [{ from: 3, to: 9 }])
    await vi.waitFor(() => expect(opened).toHaveLength(1))
    await Promise.resolve()
    await Promise.resolve()

    expect(opened).toStrictEqual([['recording', 'talks/Ants.mp3']])
  })
})

describe('the recording tabs a window holds, as the application reports on them', () => {
  /** Two recording tabs, of which the test sees what was asked. */
  const createTabs = () => {
    const make = (path: string) => ({
      path,
      setWorking: vi.fn<(next: boolean) => void>(),
      reload: vi.fn<() => void>(),
    })
    const ants = make('talks/Ants.mp3')
    const bees = make('talks/Bees.mp3')
    const window = {
      openTab: async () => '',
      getTabState: () => null,
      each: () => [{ state: ants }, { state: bees }],
    } as unknown as WindowHandle
    const made = createRecordingKind(
      window,
      () => ({}) as unknown as TranscriptState,
      { runCommand: () => {} },
      openers,
      played,
    )
    return { made, ants, bees }
  }

  it('tells a tab it is working when a task is about its recording, and the others it is not', () => {
    const { made, ants, bees } = createTabs()

    made.updateTasks([{ about: 'talks/Ants.mp3' }, { about: 'talks/Other.mp3' }] as never)

    expect(ants.setWorking).toHaveBeenCalledWith(true)
    expect(bees.setWorking).toHaveBeenCalledWith(false)
  })

  it('tells every tab it is not working where there are no tasks', () => {
    const { made, ants, bees } = createTabs()

    made.updateTasks([])

    expect(ants.setWorking).toHaveBeenCalledWith(false)
    expect(bees.setWorking).toHaveBeenCalledWith(false)
  })

  it('has the tabs standing on a recording whose transcript went read it again', () => {
    const { made, ants, bees } = createTabs()

    made.reloadTranscript('talks/Ants.mp3')

    expect(ants.reload).toHaveBeenCalledTimes(1)
    expect(bees.reload).not.toHaveBeenCalled()
  })
})
