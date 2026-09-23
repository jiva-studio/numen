<script setup lang="ts">
/**
 * A piece of media and the text of it: the media at the top of the pane, and
 * under it the transcript or the prose.
 *
 * What the media is belongs to the tab that draws this — a recording is played
 * by the window, a url is played or framed where it is drawn — so it comes in
 * through the slot, and so does whatever the tab lets a person ask over it.
 */
import { computed, ref } from 'vue'
import { Menu } from '@numen/ui'
import type { MenuItem, Position } from '@numen/ui'
import { iconFor } from '@/shared/icons'
import { MediaActions } from './media-actions'
import Transcript from './Transcript.vue'
import { WORDS as words } from '../words'
import type { MediaTabState } from '../kind'

// --- Props & Emits ---
const props = defineProps<{
  state: MediaTabState
  /**
   * Whether the media is the whole of the strip. A picture is: it is as wide as
   * the pane and carries no controls beside it.
   */
  isFramed?: boolean
  /** What the menu at the end of the strip offers, and nothing where it offers none. */
  offered?: readonly MenuItem[]
}>()

const emit = defineEmits<{ choose: [id: string] }>()

// --- State ---
const { following, timed } = props.state

/** Whether the view keeps the line being said in sight. */
const isFollowing = computed(() => following.value)

const offered = computed(() => props.offered ?? [])

/** Where the menu was asked for, and nothing while it is not open. */
const menu = ref<{ at: Position; from: HTMLElement } | null>(null)

// --- Handlers ---
function onOpenMenu(event: Event) {
  const button = event.currentTarget
  if (!(button instanceof HTMLElement)) return
  const box = button.getBoundingClientRect()
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
  props.state.setFollowing(!isFollowing.value)
}

// --- Helpers ---
</script>

<template>
  <div class="media">
    <div class="media__head" :data-framed="props.isFramed ? '' : undefined">
      <slot name="player" />

      <!-- Following is only for text that carries times, and a framed media is
           the whole strip and carries no controls at all. -->
      <MediaActions
        v-if="!props.isFramed && (timed || offered.length)"
        :is-timed="timed"
        :is-following="isFollowing"
        :has-menu="offered.length > 0"
        @toggle-follow="onToggleFollow"
        @open-menu="onOpenMenu"
      />
    </div>

    <Transcript :state="props.state" />

    <Menu
      v-if="menu"
      :items="offered"
      :at="menu.at"
      :from="menu.from"
      open
      :name="words.more"
      @choose="onChooseMenuItem"
      @dismiss="onDismissMenu"
    >
      <template #icon="{ id }">
        <component :is="iconFor(id)" v-if="iconFor(id)" class="media__mark" aria-hidden="true" />
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

/* The media heads the pane at its full width, on the rule that separates it
   from what stands below. The strip is one row of controls tall whatever it
   holds, and each control carries its own air around the mark on it. */
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

/* A picture is as wide as the pane and stands on nothing: it is the strip. The
   height a person drew is theirs until the pane is shorter than it. */
.media__head[data-framed] {
  display: block;
  flex: 0 1 auto;
  min-block-size: 0;
  overflow: hidden;
  padding-inline: 0;
  border-block-end: 0;
}

/* The room the menu keeps beside an item for the mark of what it asks for. */
.media__mark {
  inline-size: 0.875rem;
  block-size: 0.875rem;
}
</style>
