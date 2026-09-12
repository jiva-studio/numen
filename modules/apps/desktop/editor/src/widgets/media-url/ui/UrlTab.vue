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
import { MediaLayout } from '@/entities/media'
import Embed from './embed/Embed.vue'
import { DELETE_TEXT, WORDS as words } from '@/entities/media'
import type { MediaTabState } from '@/entities/media'

// --- Props & Emits ---
const props = defineProps<{ state: MediaTabState }>()

// --- State ---
const { address, deletable, framing, playable } = props.state

// What is at the address plays where it is drawn, so a moment chosen in the
// words is seeked there and the line being said follows it.
const player = useTemplateRef<{ seeks(ms: number): void }>('player')
watchPostEffect(() => props.state.playsIn(player.value))

const offered = computed(() =>
  deletable.value ? [{ id: DELETE_TEXT, text: words.deleteText }] : [],
)

// --- Handlers ---
function onChoose(id: string) {
  if (id === DELETE_TEXT) props.state.deletes()
}

function onTimeUpdate(ms: number) {
  props.state.reached(ms)
}

// --- Helpers ---
</script>

<template>
  <MediaLayout :state="props.state" framed :offered="offered" @choose="onChoose">
    <template #player>
      <Embed
        ref="player"
        :embed="framing ? address : ''"
        :copy="playable ? address : ''"
        :words="words"
        @time-update="onTimeUpdate"
      />
    </template>
  </MediaLayout>
</template>
