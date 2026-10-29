<script setup lang="ts">
import { measureRect } from './dom'
/**
 * A recording and its transcript: the player at the top of the pane, and under it the words.
 *
 * The player comes in through the slot, and so does whatever the tab lets a person ask over it.
 */
import { computed, ref } from 'vue'
import { Menu, NamedIcon } from '@numen/ui'
import type { MenuItem, Position } from '@numen/ui'
import MediaActions from './MediaActions.vue'
import Transcript from './Transcript.vue'
import { WORDS as words } from './words'
import type { TranscriptState } from './model/transcript'

/* ----------------------------- Props & Emits ------------------------------ */
const props = withDefaults(
  defineProps<{
    state: TranscriptState
    /** What the menu at the end of the strip offers, and nothing where it offers none. */
    menuItems?: readonly MenuItem[]
    isPlaying?: boolean | undefined
  }>(),
  { isPlaying: undefined },
)

const emit = defineEmits<{ choose: [id: string] }>()

/* --------------------------------- State ---------------------------------- */
const { following } = props.state.content
const { timed } = props.state.lines
const { isPlaying: playbackIsPlaying } = props.state.playback

const isFollowing = computed(() => following.value)
const isPlaying = computed(() => props.isPlaying ?? playbackIsPlaying.value)

const items = computed(() => props.menuItems ?? [])

/** Where the menu was asked for, and nothing while it is not open. */
const menu = ref<{ at: Position; from: HTMLElement } | null>(null)

/* -------------------------------- Handlers -------------------------------- */
function onOpenMenu(event: Event) {
  const button = event.currentTarget
  if (!(button instanceof HTMLElement)) return
  const box = measureRect(button)
  menu.value = { at: { x: box.left, y: box.bottom }, from: button }
}

function onChooseMenuItem(id: string) {
  menu.value = null
  emit('choose', id)
}

function onDismissMenu() {
  menu.value = null
}

function onToggleFollow() {
  props.state.content.setFollowing(!isFollowing.value)
}
</script>

<template>
  <div class="media">
    <div class="media__head">
      <slot name="player" />

      <MediaActions
        v-if="timed || items.length"
        :is-timed="timed"
        :is-following="isFollowing"
        :has-menu="items.length > 0"
        @toggle-follow="onToggleFollow"
        @open-menu="onOpenMenu"
      />
    </div>

    <Transcript :state="props.state" :is-playing="isPlaying" />

    <Menu
      v-if="menu"
      :items="items"
      :at="menu.at"
      :from="menu.from"
      is-open
      :name="words.more"
      @choose="onChooseMenuItem"
      @dismiss="onDismissMenu"
    >
      <template #icon="{ id }">
        <NamedIcon
          v-if="props.state.offers.getIcon(id)"
          :name="props.state.offers.getIcon(id)"
          class="media__menu-icon"
        />
      </template>
    </Menu>
  </div>
</template>

<style scoped>
.media {
  --media-apart: 1rem;
  --media-close: 0.25rem;
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
}

/* The player heads the pane at its full width, on the rule that separates it from the words. */
.media__head {
  display: flex;
  align-items: center;
  flex: none;
  gap: var(--media-apart);
  inline-size: 100%;
  min-block-size: var(--numen-action-size);
  padding-inline: var(--numen-gutter);
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

.media__menu-icon {
  inline-size: 0.875rem;
  block-size: 0.875rem;
  stroke-width: 1.875;
  opacity: 0.75;
}
</style>
