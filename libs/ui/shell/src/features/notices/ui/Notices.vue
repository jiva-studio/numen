<script setup lang="ts">
/**
 * What the window has to say, as cards in its bottom corner.
 *
 * Work appears once it has lasted and goes when the work does; something that is
 * so stands while it is so; something that happened stands to be read and then
 * goes, unless it is trouble, which stands until it is put away.
 *
 * It takes no room from what it covers, and what each card says is the caller's.
 */
import { useTemplateRef } from 'vue'
import { getNow, isDocumentHidden } from '@/shared/lib/dom'
import { LiveRegions } from './live-regions'
import NoticeCard from './NoticeCard.vue'
import { CARD_DELAY } from '../lib/duration'
import { ROOM } from '../lib/fold'
import { useAnnouncer } from '../model/announcer'
import { useNoticeCards } from '../model/cards'
import type { NoticesProps } from './Notices.types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<NoticesProps>(), {
  notices: () => [],
  name: 'Background work',
  dismiss: 'Dismiss',
  more: 'more',
  wait: CARD_DELAY,
  room: ROOM,
  clock: () => getNow(),
  hidden: () => isDocumentHidden(),
})

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** A card is finished with: read long enough, or put away. */
  (event: 'dismiss', id: string): void
}>()

/* ---------------------------------- State --------------------------------- */
const stack = useTemplateRef<HTMLElement>('stack')

const {
  notices,
  folds,
  isOpened,
  leftOn,
  holdCard,
  dismissByHand,
  onPointerOver,
  onPointerOut,
  onFocusIn,
  onFocusOut,
} = useNoticeCards({
  stack,
  getNotices: () => props.notices,
  getWait: () => props.wait,
  getRoom: () => props.room,
  getNow: () => props.clock(),
  isHidden: () => props.hidden(),
  dismiss: (id) => emit('dismiss', id),
})

/** What the corner is read out through. */
const { polite, assertive } = useAnnouncer(() => notices.value)
</script>

<template>
  <div class="notices numen text-small font-sans">
    <LiveRegions :told="polite" :cried="assertive" />

    <aside
      v-if="folds.visible.length || folds.overflowCount"
      ref="stack"
      class="notices__stack flex flex-col"
      :aria-label="name"
      @pointerout="onPointerOut"
      @focusin="onFocusIn"
      @focusout="onFocusOut"
    >
      <TransitionGroup name="notice">
        <button
          v-if="folds.overflowCount"
          key="folded"
          type="button"
          class="notice notice__folded"
          @pointerover="onPointerOver"
          @click="isOpened = true"
        >
          {{ folds.overflowCount }} {{ more }}
        </button>

        <NoticeCard
          v-for="one in folds.visible"
          :key="one.id"
          :ref="(card) => holdCard(one.id, card)"
          :one="one"
          :dismiss="dismiss"
          :left="leftOn(one)"
          @pointerover="onPointerOver"
          @dismiss="dismissByHand(one.id)"
        />
      </TransitionGroup>
    </aside>
  </div>
</template>

<style scoped src="./Notices.css"></style>
