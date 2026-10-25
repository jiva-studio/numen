/**
 * Playback state of one recording in the shared audio player.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { usePlayback } from './playback'
import type { Player } from './player'
import type { RecordingSource } from '../types'

const TALK = 'http://127.0.0.1:1/token/v/talk.mp3'
const OTHER = 'http://127.0.0.1:1/token/v/other.mp3'
const FILE: RecordingSource = { mediaUrl: TALK, mediaType: 'audio/mpeg' }

/** The one player of the window, with what is asked of it recorded. */
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
  const audioPlayer: Player = { url, position, duration, isPlaying, error, load, play, pause, seek }
  return { audioPlayer, url, position, duration, isPlaying, error, load, play, pause, seek }
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
  it('has none, runs nowhere, and cannot be played', () => {
    const { playback } = createPlayback()

    expect(playback.url.value).toBe('')
    expect(playback.runs.value).toBe(0)
    expect(playback.isPlayable.value).toBe(false)
  })
})

describe('a recording the player holds', () => {
  it('reports the player position, duration and error', () => {
    const { playback, url, position, duration, isPlaying, error } = createPlayback()
    playback.setSource(FILE)
    url.value = TALK
    position.value = 4_000
    duration.value = 6_000
    isPlaying.value = true
    error.value = 'broken'

    expect(playback.now.value).toBe(4_000)
    expect(playback.runs.value).toBe(6_000)
    expect(playback.isPlaying.value).toBe(true)
    expect(playback.failure.value).toBe('broken')
  })

  it('is paused through the player only while it is the one playing', () => {
    const { playback, url, isPlaying, pause } = createPlayback()
    playback.setSource(FILE)
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
    playback.setSource(FILE)
    url.value = OTHER
    position.value = 7_000
    isPlaying.value = true

    expect(playback.now.value).toBe(0)
    expect(playback.isPlaying.value).toBe(false)
  })
})

describe('whether the window can play a recording', () => {
  it('follows what the window says of the type', () => {
    const playable = createPlayback()
    playable.playback.setSource(FILE)
    const unplayable = createPlayback({ isPlayable: false })
    unplayable.playback.setSource(FILE)

    expect(playable.playback.isPlayable.value).toBe(true)
    expect(unplayable.playback.isPlayable.value).toBe(false)
  })
})

describe('the source given to a player holding nothing', () => {
  it('is loaded, so the controls read how long it runs', () => {
    const { playback, load } = createPlayback()
    playback.setSource(FILE)
    expect(load).toHaveBeenCalledWith(TALK)
  })

  it('leaves a player holding another recording alone', () => {
    const { playback, url, load } = createPlayback()
    url.value = OTHER
    playback.setSource(FILE)
    expect(load).not.toHaveBeenCalled()
  })
})

describe('a moment gone to', () => {
  it('is played from, to the millisecond and no earlier than the beginning', () => {
    const { playback, seek } = createPlayback()
    playback.setSource(FILE)
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

    playback.setSource(FILE)
    expect(seek).toHaveBeenCalledWith(TALK, 3_000)
    expect(load).not.toHaveBeenCalled()
  })

  it('is not gone to once the tab is closed', () => {
    const { playback, state, seek } = createPlayback()
    playback.setSource(FILE)
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

    playback.setSource(FILE)
    playback.controls.play()
    expect(play).toHaveBeenCalledWith(TALK)

    state.isOpen = false
    playback.controls.play()
    expect(play).toHaveBeenCalledOnce()
  })
})
