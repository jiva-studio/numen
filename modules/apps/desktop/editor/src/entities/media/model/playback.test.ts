/**
 * Where one recording stands: in the single player the window makes sound
 * with, or in the frame a url draws for itself.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { usePlayback, type TabPlayer } from './playback'
import type { Player } from './player'
import type { RecordingSummary } from '../types'

const TALK = 'http://127.0.0.1:1/files/w/v/talk.mp3'
const OTHER = 'http://127.0.0.1:1/files/w/v/other.mp3'

/** The recording as the application answers for a file on this disk. */
const FILE: RecordingSummary = { duration: 9_000, mediaUrl: TALK, mediaType: 'audio/mpeg', url: '' }
/** A url with no copy on this disk: a page read in a frame. */
const PAGE: RecordingSummary = {
  duration: 0,
  mediaUrl: 'http://127.0.0.1:1/files/w/v/page.html',
  mediaType: 'text/html',
  url: 'https://example.org/talk',
}

/** The one player of the window, with what is asked of it written down. */
const createPlayer = () => {
  const url = ref('')
  const at = ref(0)
  const duration = ref(0)
  const isPlaying = ref(false)
  const error = ref('')
  const load = vi.fn<(url: string) => void>()
  const play = vi.fn<(url: string) => void>()
  const pause = vi.fn<() => void>()
  const seek = vi.fn<(url: string, ms: number) => void>()
  const through: Player = { url, at, duration, isPlaying, error, load, play, pause, seek }
  return { through, url, at, duration, isPlaying, error, load, play, pause, seek }
}

/** The player a tab draws for a url. */
const createFrame = () => {
  const seek = vi.fn<(ms: number) => void>()
  const frame: TabPlayer = { seek }
  return { frame, seek }
}

/** A recording that is open, played through a window that can play what it is asked. */
const createPlayback = (how: { open?: boolean; plays?: boolean } = {}) => {
  const standing = createPlayer()
  const state = { open: how.open ?? true }
  const playback = usePlayback(
    standing.through,
    () => how.plays ?? true,
    () => state.open,
  )
  return { playback, state, ...standing }
}

describe('a recording nothing has been said of', () => {
  it('has no address, no type, no web address, and does not run', () => {
    const { playback } = createPlayback()

    expect(playback.url.value).toBe('')
    expect(playback.type.value).toBe('')
    expect(playback.points.value).toBe('')
    expect(playback.runs.value).toBe(0)
    expect(playback.now.value).toBe(0)
    expect(playback.hasTime.value).toBe(true)
    expect(playback.playable.value).toBe(false)
    expect(playback.framing.value).toBe(false)
  })

  it('is not the one the player holds, even where the player holds nothing', () => {
    const { playback, isPlaying, at, error } = createPlayback()
    isPlaying.value = true
    at.value = 4_000
    error.value = 'The recording could not be played.'

    expect(playback.isPlaying.value).toBe(false)
    expect(playback.now.value).toBe(0)
    expect(playback.broken.value).toBe('')
  })
})

describe('a recording the player holds', () => {
  it('stands where the player stands, and runs as long as the longer of the two says', () => {
    const { playback, url, at, duration } = createPlayback()
    playback.setSource(FILE)
    url.value = TALK
    at.value = 4_000
    duration.value = 12_000

    expect(playback.now.value).toBe(4_000)
    expect(playback.runs.value).toBe(12_000)
    duration.value = 3_000
    expect(playback.runs.value).toBe(9_000)
  })

  it('is playing while the player is, and says what the player could not do', () => {
    const { playback, url, isPlaying, error } = createPlayback()
    playback.setSource(FILE)
    url.value = TALK
    isPlaying.value = true
    error.value = 'The recording could not be played.'

    expect(playback.isPlaying.value).toBe(true)
    expect(playback.broken.value).toBe('The recording could not be played.')
  })

  it('is paused through the player only while it is the one playing', () => {
    const { playback, url, isPlaying, pause } = createPlayback()
    playback.setSource(FILE)
    playback.pause()
    expect(pause).not.toHaveBeenCalled()

    url.value = TALK
    isPlaying.value = true
    playback.pause()

    expect(pause).toHaveBeenCalledTimes(1)
  })
})

describe('a recording another holds the player for', () => {
  it('stands at its beginning, runs as the application says, and is not playing', () => {
    const { playback, url, at, duration, isPlaying, error } = createPlayback()
    playback.setSource(FILE)
    url.value = OTHER
    at.value = 4_000
    duration.value = 12_000
    isPlaying.value = true
    error.value = 'The recording could not be played.'

    expect(playback.now.value).toBe(0)
    expect(playback.runs.value).toBe(9_000)
    expect(playback.isPlaying.value).toBe(false)
    expect(playback.broken.value).toBe('')
  })
})

describe('whether the window can play a recording', () => {
  it('is so for an address of a kind the window plays', () => {
    const { playback } = createPlayback({ plays: true })

    playback.setSource(FILE)

    expect(playback.playable.value).toBe(true)
  })

  it('is not so for a kind the window does not play', () => {
    const { playback } = createPlayback({ plays: false })

    playback.setSource(FILE)

    expect(playback.playable.value).toBe(false)
  })

  it('is not so where the recording has no address', () => {
    const { playback } = createPlayback({ plays: true })

    playback.setSource({ ...FILE, mediaUrl: '' })

    expect(playback.playable.value).toBe(false)
  })
})

describe('whether a recording is a page', () => {
  it('is so for an address that holds a page', () => {
    const { playback } = createPlayback()

    playback.setSource(PAGE)

    expect(playback.framing.value).toBe(true)
  })

  it('is not so for an address that holds sound', () => {
    const { playback } = createPlayback()

    playback.setSource(FILE)

    expect(playback.framing.value).toBe(false)
  })

  it('is not so where the page has no address', () => {
    const { playback } = createPlayback()

    playback.setSource({ ...PAGE, mediaUrl: '' })

    expect(playback.framing.value).toBe(false)
  })
})

describe('the player the player takes for a recording nothing holds', () => {
  it('is loaded with a recording on this disk, so the controls read how long it runs', () => {
    const { playback, load } = createPlayback()

    playback.setSource(FILE)

    expect(load).toHaveBeenCalledWith(TALK)
  })

  it('is left alone where it holds another recording', () => {
    const { playback, url, load } = createPlayback()
    url.value = OTHER

    playback.setSource(FILE)

    expect(load).not.toHaveBeenCalled()
  })

  it('is left alone for a url, which plays where it is drawn', () => {
    const { playback, load } = createPlayback()

    playback.setSource(PAGE)

    expect(load).not.toHaveBeenCalled()
  })

  it('is left alone for a recording with no address', () => {
    const { playback, load } = createPlayback()

    playback.setSource({ ...FILE, mediaUrl: '' })

    expect(load).not.toHaveBeenCalled()
  })
})

describe('a moment gone to in a recording', () => {
  it('is played from in the player, to the millisecond and no earlier than the beginning', () => {
    const { playback, seek } = createPlayback()
    playback.setSource(FILE)

    playback.go(1_234.6)
    playback.go(-50)

    expect(seek.mock.calls).toStrictEqual([
      [TALK, 1_235],
      [TALK, 0],
    ])
  })

  it('is not gone to once the tab is closed', () => {
    const { playback, state, seek } = createPlayback()
    playback.setSource(FILE)
    state.open = false

    playback.go(1_000)

    expect(seek).not.toHaveBeenCalled()
  })

  it('is played from once the recording knows its own address', () => {
    const { playback, seek } = createPlayback()

    playback.go(0)
    expect(seek).not.toHaveBeenCalled()
    playback.setSource(FILE)
    expect(seek.mock.calls).toStrictEqual([[TALK, 0]])

    playback.setSource(FILE)
    expect(seek).toHaveBeenCalledTimes(1)
  })

  it('is kept for a url, which is not played through the player', () => {
    const { playback, seek } = createPlayback()
    playback.setSource({ ...PAGE, mediaType: 'audio/mpeg' })

    playback.go(5_000)

    expect(seek).not.toHaveBeenCalled()
  })

  it('is sent to the frame a url is drawn in, and the frame reads where it stands', () => {
    const { playback, seek } = createPlayback()
    const { frame, seek: seekFrame } = createFrame()
    playback.setSource(PAGE)
    playback.setFramePlayer(frame)

    playback.go(1_234.6)

    expect(seekFrame).toHaveBeenCalledWith(1_235)
    expect(seek).not.toHaveBeenCalled()
    expect(playback.now.value).toBe(1_235)
  })
})

describe('a moment gone to before the frame is drawn', () => {
  it('is gone to in the frame once it is handed over, and once only', () => {
    const { playback } = createPlayback()
    const { frame, seek } = createFrame()
    playback.setSource(PAGE)

    playback.go(0)
    playback.setFramePlayer(frame)
    expect(seek.mock.calls).toStrictEqual([[0]])
    expect(playback.hasTime.value).toBe(true)

    playback.setFramePlayer(null)
    playback.setFramePlayer(frame)
    expect(seek).toHaveBeenCalledTimes(1)
  })
})

describe('the frame a url is drawn in', () => {
  it('has said nothing of where it stands until it does', () => {
    const { playback } = createPlayback()
    const { frame } = createFrame()
    playback.setSource(PAGE)

    playback.setFramePlayer(frame)
    expect(playback.hasTime.value).toBe(false)
    expect(playback.now.value).toBe(0)

    playback.setFrameTime(500)
    expect(playback.hasTime.value).toBe(true)
    expect(playback.now.value).toBe(500)
  })

  it('is not sent to a moment when none was gone to before it was handed over', () => {
    const { playback } = createPlayback()
    const { frame, seek } = createFrame()
    playback.setSource(PAGE)

    playback.setFramePlayer(frame)

    expect(seek).not.toHaveBeenCalled()
  })

  it('has said nothing again once another frame is handed over', () => {
    const { playback } = createPlayback()
    const { frame } = createFrame()
    playback.setSource(PAGE)
    playback.setFramePlayer(frame)
    playback.setFrameTime(500)

    playback.setFramePlayer(createFrame().frame)

    expect(playback.hasTime.value).toBe(false)
    expect(playback.now.value).toBe(0)
  })
})

describe('a recording played', () => {
  it('is played through the player at its address', () => {
    const { playback, play } = createPlayback()
    playback.setSource(FILE)

    playback.play()

    expect(play).toHaveBeenCalledWith(TALK)
  })

  it('is not played once the tab is closed', () => {
    const { playback, state, play } = createPlayback()
    playback.setSource(FILE)
    state.open = false

    playback.play()

    expect(play).not.toHaveBeenCalled()
  })

  it('is not played where it has no address', () => {
    const { playback, play } = createPlayback()

    playback.play()

    expect(play).not.toHaveBeenCalled()
  })
})
