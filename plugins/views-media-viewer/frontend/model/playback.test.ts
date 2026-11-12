/**
 * Playback state of one recording in the shared audio player.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { usePlayback } from './playback'
import type { Player } from './player'
import type { RecordingSource } from '../types'

const TALK = 'http://127.0.0.1:1/token/v/talk.mp3'
const OTHER = 'http://127.0.0.1:1/token/v/other.mp3'
const flushPromises = () => new Promise((resolve) => setTimeout(resolve, 0))

const FILE: RecordingSource = { mediaUrl: TALK, mediaType: 'audio/mpeg', pageUrl: '' }

const createPlayer = () => {
  const url = ref('')
  const position = ref(0)
  const duration = ref(0)
  const isPlaying = ref(false)
  const error = ref('')
  const load = vi.fn<(url: string) => void>()
  const play = vi.fn<(url: string) => void>()
  const pause = vi.fn<() => void>()
  const seek = vi.fn<(url: string, ms: number) => void>()
  const getDuration = vi.fn<(url: string) => Promise<number>>(async () => 0)
  const attach = vi.fn<(container: HTMLElement) => void>()
  const release = vi.fn<(container: HTMLElement) => void>()
  const audioPlayer: Player = {
    state: { url, position, duration, isPlaying, error },
    load,
    play,
    pause,
    seek,
    getDuration,
    surface: { attach, release },
  }
  return {
    audioPlayer,
    url,
    position,
    duration,
    isPlaying,
    error,
    load,
    play,
    pause,
    seek,
    getDuration,
  }
}

const createPlayback = (how: { isPlayable?: boolean } = {}) => {
  const player = createPlayer()
  const state = { isOpen: true }
  const playback = usePlayback(
    player.audioPlayer,
    () => how.isPlayable ?? true,
    () => state.isOpen,
  )
  return { playback, state, ...player }
}

describe('a recording that does not know its address yet', () => {
  it('has none, runs nowhere, and is not called unplayable before it is read', () => {
    const { playback } = createPlayback({ isPlayable: false })

    expect(playback.source.url.value).toBe('')
    expect(playback.progress.runs.value).toBe(0)
    expect(playback.source.isUnplayable.value).toBe(false)
  })
})

describe('a recording the player holds', () => {
  it('reports the player position, duration and error', () => {
    const { playback, url, position, duration, isPlaying, error } = createPlayback()
    playback.source.setSource(FILE)
    url.value = TALK
    position.value = 4_000
    duration.value = 6_000
    isPlaying.value = true
    error.value = 'broken'

    expect(playback.progress.now.value).toBe(4_000)
    expect(playback.progress.runs.value).toBe(6_000)
    expect(playback.progress.isPlaying.value).toBe(true)
    expect(playback.progress.failure.value).toBe('broken')
  })

  it('is paused through the player only while it is the one playing', () => {
    const { playback, url, isPlaying, pause } = createPlayback()
    playback.source.setSource(FILE)
    url.value = OTHER
    isPlaying.value = true
    playback.controls.pause()
    expect(pause).not.toHaveBeenCalled()

    url.value = TALK
    playback.controls.pause()
    expect(pause).toHaveBeenCalledOnce()
  })
})

describe('a recording another holds the player for', () => {
  it('starts at zero and paused', () => {
    const { playback, url, position, isPlaying } = createPlayback()
    playback.source.setSource(FILE)
    url.value = OTHER
    position.value = 7_000
    isPlaying.value = true

    expect(playback.progress.now.value).toBe(0)
    expect(playback.progress.isPlaying.value).toBe(false)
  })
})

describe('whether the window can play a recording', () => {
  it('is unplayable where the application named no address', () => {
    const { playback } = createPlayback()
    playback.source.setSource({ ...FILE, mediaUrl: '' })

    expect(playback.source.isUnplayable.value).toBe(true)
  })

  it('follows what the window says of the type', () => {
    const playable = createPlayback()
    playable.playback.source.setSource(FILE)
    const unplayable = createPlayback({ isPlayable: false })
    unplayable.playback.source.setSource(FILE)

    expect(playable.playback.source.isUnplayable.value).toBe(false)
    expect(unplayable.playback.source.isUnplayable.value).toBe(true)
  })
})

describe('the source given to a player holding nothing', () => {
  it('is loaded, so the controls read how long it runs', () => {
    const { playback, load } = createPlayback()
    playback.source.setSource(FILE)
    expect(load).toHaveBeenCalledWith(TALK)
  })

  it('leaves a player holding another recording alone', () => {
    const { playback, url, load } = createPlayback()
    url.value = OTHER
    playback.source.setSource(FILE)
    expect(load).not.toHaveBeenCalled()
  })
})

describe('a recording while another one holds the player', () => {
  it('shows its own length, read without taking the player, at position 0', async () => {
    const { playback, url, duration, position, getDuration, load, seek } = createPlayback()
    url.value = OTHER
    duration.value = 6_000
    position.value = 2_000
    getDuration.mockResolvedValue(187_000)

    playback.source.setSource(FILE)
    await flushPromises()

    expect(getDuration).toHaveBeenCalledWith(TALK)
    expect(playback.progress.runs.value).toBe(187_000)
    expect(playback.progress.now.value).toBe(0)
    expect(load).not.toHaveBeenCalled()
    expect(seek).not.toHaveBeenCalled()
  })

  it('keeps the length the player read once another recording takes the player', async () => {
    const { playback, url, duration, getDuration } = createPlayback()
    playback.source.setSource(FILE)
    url.value = TALK
    duration.value = 187_000
    await nextTick()

    url.value = OTHER
    duration.value = 6_000
    await nextTick()

    expect(playback.progress.runs.value).toBe(187_000)
    expect(getDuration).not.toHaveBeenCalled()
  })

  it('reads nothing of its own where the player is free and loads it', () => {
    const { playback, getDuration, load } = createPlayback()
    playback.source.setSource(FILE)

    expect(load).toHaveBeenCalledWith(TALK)
    expect(getDuration).not.toHaveBeenCalled()
  })
})

describe('a moment gone to', () => {
  it('is played from, to the millisecond and no earlier than the beginning', () => {
    const { playback, seek } = createPlayback()
    playback.source.setSource(FILE)
    playback.controls.go(1_234.4)
    playback.controls.go(-50)
    expect(seek.mock.calls).toStrictEqual([
      [TALK, 1_234],
      [TALK, 0],
    ])
  })

  it('is played from once the recording knows its address', () => {
    const { playback, seek, load } = createPlayback()
    playback.controls.go(3_000)
    expect(seek).not.toHaveBeenCalled()

    playback.source.setSource(FILE)
    expect(seek).toHaveBeenCalledWith(TALK, 3_000)
    expect(load).not.toHaveBeenCalled()
  })

  it('is not gone to once the tab is closed', () => {
    const { playback, state, seek } = createPlayback()
    playback.source.setSource(FILE)
    state.isOpen = false
    playback.controls.go(1_000)
    expect(seek).not.toHaveBeenCalled()
  })
})

describe('a recording played', () => {
  it('is played through the player at its address, and not once the tab is closed', () => {
    const { playback, state, play } = createPlayback()
    playback.controls.play()
    expect(play).not.toHaveBeenCalled()

    playback.source.setSource(FILE)
    playback.controls.play()
    expect(play).toHaveBeenCalledWith(TALK)

    state.isOpen = false
    playback.controls.play()
    expect(play).toHaveBeenCalledOnce()
  })
})
