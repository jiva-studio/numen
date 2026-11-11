import { computed, watch } from 'vue'
import { createTimeGutter } from './timeGutter'
import type { TranscriptState } from './transcript'

/** The times in the editor's gutter, kept on the line being played. */
export function useTranscriptGutter(
  state: TranscriptState,
  getIsPlaying: () => boolean | undefined,
) {
  const { isPlaying: playbackIsPlaying } = state.playback.progress
  const { current, following } = state.content
  const { typing } = state.draft
  const { times: cueTimes } = state.lines
  const isPlaying = computed(() => getIsPlaying() ?? playbackIsPlaying.value)

  const times = createTimeGutter((line) => state.navigation.goToLine(line))

  watch(
    [cueTimes, current, following, isPlaying, typing],
    ([timesValue, currentLine, isFollowing, isPlayingValue, isTyping]) => {
      times.show({
        times: timesValue,
        current: currentLine,
        isFollowing: isFollowing && isPlayingValue && !isTyping,
      })
    },
    { immediate: true, flush: 'post' },
  )

  watch(
    following,
    (isFollowing, wasFollowing) => {
      if (isFollowing && !wasFollowing && !isPlaying.value && !typing.value) {
        times.centerCurrent()
      }
    },
    { flush: 'post' },
  )

  return { extension: times.extension }
}
