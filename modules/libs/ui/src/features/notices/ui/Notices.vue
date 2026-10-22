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
import { LiveRegions } from './live-regions'
import NoticeCard from './NoticeCard.vue'
import { CARD_DELAY } from '../lib/duration'
import { ROOM } from '../lib/fold'
import { type Notice } from '../lib/notice'
import { useAnnouncer } from '../model/announcer'
import { useNoticeCards } from '../model/cards'

const props = withDefaults(
  defineProps<{
    /** What the window has to say, in the order it is drawn. */
    notices?: readonly Notice[]
    /** What the corner is announced as. */
    name?: string
    /** What the way to dismiss one is called. */
    dismiss?: string
    /** What the ones folded away behind the rest are counted as. */
    more?: string
    /** How long work runs before it is worth a card. */
    wait?: number
    /** How many cards stand at once. */
    room?: number
    /** What the moment is. The window's clock by default. */
    clock?: () => number
    /** Whether nobody is looking. The window's own answer by default. */
    hidden?: () => boolean
  }>(),
  {
    notices: () => [],
    name: 'Background work',
    dismiss: 'Put away',
    more: 'more',
    wait: CARD_DELAY,
    room: ROOM,
    clock: () => Date.now(),
    hidden: () => document.hidden,
  },
)

const emit = defineEmits<{
  /** A card is finished with: read long enough, or put away. */
  (event: 'dismiss', id: string): void
}>()

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
