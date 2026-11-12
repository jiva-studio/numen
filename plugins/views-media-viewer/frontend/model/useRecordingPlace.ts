import { onUnmounted, watch } from 'vue'
import type { ReaderTabState, Selector } from '@numen/plugin-sdk'
import { getSelectorStart, getTimeSelector } from '../lib/place'
import type { TextRange } from '../types'
import type { TranscriptState } from './transcript'

/**
 * Keeps the place of a recording tab: seeks to the selector and ranges the window sets, and reports
 * and saves the playback position.
 */
export function useRecordingPlace(
  tab: TranscriptState,
  getState: () => Partial<ReaderTabState> | undefined,
) {
  const { now, isPlaying } = tab.playback.progress
  const { isLoaded } = tab.playback.source

  const seekToSelector = (selector: Selector | null | undefined): number | null => {
    const start = getSelectorStart(selector)
    if (start !== null) tab.playback.controls.go(start)
    return start
  }

  const saveTime = (ms: number) => {
    getState()?.savePlace?.(getTimeSelector(ms))
    getState()?.reportPlace?.({ positionSeconds: Math.max(0, ms) / 1000 })
  }

  const seekToSpans = async (...spans: readonly TextRange[]) => {
    const ms = await tab.navigation.seekToSpans(...spans)
    if (ms !== null) saveTime(ms)
  }

  const savePlace = () => {
    if (isLoaded.value) saveTime(now.value)
  }

  // The window sets a selector for a search hit and for the place saved before a restart. The one the
  // tab opens with is already stored by the window, so it is reported and only a later one is saved.
  const initialTime = seekToSelector(getState()?.selector)
  if (initialTime !== null) getState()?.reportPlace?.({ positionSeconds: initialTime / 1000 })

  watch(
    () => getState()?.selector,
    (selector: Selector | null | undefined) => {
      const start = seekToSelector(selector)
      if (start !== null) saveTime(start)
    },
  )

  // The window sets a new array on each request, so playback seeks to the same ranges again.
  watch(
    () => getState()?.ranges,
    (ranges) => {
      if (ranges && ranges.length > 0) seekToSpans(...ranges).catch(tab.lifecycle.reportFailure)
    },
    { immediate: true },
  )

  // Saved when playback stops and every ten seconds of it, so a restart loses at most that much.
  watch(isPlaying, (playing) => {
    if (!playing) savePlace()
  })
  watch(
    () => Math.floor(now.value / 10_000),
    () => {
      if (isPlaying.value) savePlace()
    },
  )

  onUnmounted(() => {
    savePlace()
    tab.lifecycle.close()
  })

  const seekTo = (ms: number) => {
    tab.playback.controls.go(ms)
    saveTime(ms)
  }

  return { seekTo }
}
