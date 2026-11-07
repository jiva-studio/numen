<script setup lang="ts">
/**
 * A url tab: what is at the address at the top of the pane, and under it the
 * text fetched from it.
 *
 * What is at an address is a copy on this disk where one was fetched, and the
 * page itself in a frame where none was: a video plays in it and a page is
 * read in it. The text is a transcript where what is there was spoken and
 * prose where it was written.
 */
import { computed, onMounted, onUnmounted, readonly, ref, useTemplateRef, watch } from 'vue'
import MediaEmbed from './embed/MediaEmbed.vue'
import MediaLayout from './MediaLayout.vue'
import {
  useTranscript,
  type TranscriptOptions,
} from '@plugins/views-media-viewer/frontend/model/transcript'
import type { Player } from '@plugins/views-media-viewer/frontend/model/player'
import type { ReaderTabState, Selector } from '@numen/plugin-host'
import type { Recordings, TextRange } from '@plugins/views-media-viewer/frontend/types'
import {
  createTimeSelector,
  getSelectorStart,
} from '@plugins/views-media-viewer/frontend/lib/place'
import { WORDS as words } from './words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  readonly path?: string
  readonly file?: string
  readonly state?: Partial<ReaderTabState> & { file?: string; [key: string]: unknown }
  /** What the tab reads the media through. The window's own by default. */
  readonly recordings?: Recordings
  readonly how?: TranscriptOptions
}>()

/* --------------------------------- State ---------------------------------- */
const targetPath = props.path || props.file || props.state?.path || props.state?.file || ''
const player = useTemplateRef<{ seek(ms: number): void }>('player')

const frameUrl = ref('')
const framePosition = ref(0)
const frameDuration = ref(0)
const frameIsPlaying = ref(false)
const frameError = ref('')

const framePlayer: Player = {
  url: readonly(frameUrl),
  position: readonly(framePosition),
  duration: readonly(frameDuration),
  isPlaying: readonly(frameIsPlaying),
  error: readonly(frameError),
  load(next: string) {
    frameUrl.value = next
  },
  play(next: string) {
    frameUrl.value = next
    frameIsPlaying.value = true
  },
  pause() {
    frameIsPlaying.value = false
  },
  seek(next: string, ms: number) {
    frameUrl.value = next
    framePosition.value = Math.max(0, Math.round(ms))
    player.value?.seek(framePosition.value)
  },
  getDuration: () => Promise.resolve(0),
  attach: () => undefined,
  release: () => undefined,
}

const tab = useTranscript(props.recordings, targetPath, {
  ...props.how,
  audioPlayer: framePlayer,
  plays: () => true,
})
const { url, now, isPlaying } = tab.playback
const menuItems = tab.offers.items

const isFramedPage = computed(() => url.value !== '' && !url.value.endsWith('.mp4'))

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  if (props.state) {
    const shared = props.state as Record<string, unknown>
    shared.focusSpans = (...spans: readonly TextRange[]) => tab.navigation.reach(...spans)
  }
})

watch(
  () => props.state?.selector,
  (selector: Selector | null | undefined) => {
    const start = getSelectorStart(selector)
    if (start !== null) tab.playback.controls.go(start)
  },
  { immediate: true },
)

const savePlace = () => props.state?.savePlace?.(createTimeSelector(now.value))

watch(isPlaying, (playing) => {
  if (!playing) savePlace()
})

onUnmounted(() => {
  savePlace()
  tab.close()
})

/* -------------------------------- Handlers -------------------------------- */
function onChoose(id: string) {
  void tab.offers.run(id)
}

function onTimeUpdate(ms: number) {
  framePosition.value = ms
}

function onPlayState(playing: boolean) {
  frameIsPlaying.value = playing
}
</script>

<template>
  <MediaLayout :state="tab" is-framed :menu-items="menuItems" @choose="onChoose">
    <template #player>
      <MediaEmbed
        ref="player"
        :embed="isFramedPage ? url : ''"
        :copy="!isFramedPage ? url : ''"
        :words="words"
        @time-update="onTimeUpdate"
        @play-state="onPlayState"
      />
    </template>
  </MediaLayout>
</template>
