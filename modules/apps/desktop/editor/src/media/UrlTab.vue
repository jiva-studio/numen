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
import { computed, useTemplateRef, watchPostEffect } from 'vue'
import MediaTab from './MediaTab.vue'
import Embed from './Embed.vue'
import { DELETE_TEXT, WORDS as words } from '../recording/words'
import type { MediaTabState } from '../recording/kind'

const props = defineProps<{ state: MediaTabState }>()

const { address, deletable, embed, playable, points } = props.state

/** Where the frame stands: the copy on this disk, or the address itself. */
const shown = computed(() => (playable.value ? '' : embed.value || points.value))

// What is at the address plays where it is drawn, so a moment chosen in the
// words is seeked there and the line being said follows it.
const player = useTemplateRef<{ seeks(ms: number): void }>('player')
watchPostEffect(() => props.state.playsIn(player.value))

const offered = computed(() =>
  deletable.value ? [{ id: DELETE_TEXT, text: words.deleteText }] : [],
)

const chose = (id: string) => {
  if (id === DELETE_TEXT) props.state.deletes()
}
</script>

<template>
  <MediaTab :state="props.state" framed :offered="offered" @choose="chose">
    <template #player>
      <Embed
        ref="player"
        :embed="shown"
        :copy="playable ? address : ''"
        :words="words"
        @time-update="(ms: number) => props.state.reached(ms)"
      />
    </template>
  </MediaTab>
</template>
