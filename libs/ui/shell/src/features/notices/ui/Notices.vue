<script setup lang="ts">
/**
 * What the window reports, as cards in its bottom corner.
 *
 * Work appears once it has lasted and is removed when the work ends; a standing
 * state stays while it holds; an event stays to be read and is then removed,
 * unless it is an error, which stays until dismissed.
 *
 * It takes no space from what it covers, and each card's content is the caller's.
 */
import { useTemplateRef } from 'vue'
import { getNow } from '@numen/ui'
import { isDocumentHidden } from '@/shared/lib/dom'
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
  dismissManually,
  onPointerOver,
  onPointerOut,
  onFocusIn,
  onFocusOut,
} = useNoticeCards({
  stack,
  getNotices: () => props.notices,
  getWait: () => props.wait,
  getFreeSpace: () => props.room,
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
          @dismiss="dismissManually(one.id)"
        />
      </TransitionGroup>
    </aside>
  </div>
</template>

<style scoped>
/* Over the corner, never in the way of a pointer that is not on a card. */
.notices {
  --gap: 0.5rem;

  position: fixed;
  inset-block-end: var(--numen-inset-wide);
  inset-inline-start: var(--numen-inset-wide);
  z-index: var(--numen-lift-notice);
  pointer-events: none;
}

.notices__stack {
  gap: var(--gap);
}

.notice {
  --room: 24rem;

  pointer-events: auto;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  inline-size: var(--room);
  max-inline-size: calc(100vw - 2 * var(--numen-inset-wide));
  padding-block: 0.5rem;
  padding-inline: 0.75rem 0.5rem;
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius-panel);
  background: var(--numen-raised);
  color: var(--numen-ink);
  box-shadow: var(--numen-shadow-card);
  text-align: start;
}

.notice[data-severity='warning'] {
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  border-color: color-mix(in oklab, currentColor 25%, transparent);
}

.notice[data-severity='error'] {
  background: var(--numen-alarm-bg);
  color: var(--numen-alarm);
  border-color: color-mix(in oklab, currentColor 25%, transparent);
}

.notice__folded {
  margin: 0;
  opacity: 0.7;
}

.notice__folded:hover,
.notice__folded:focus-visible {
  opacity: 1;
}

.notice-enter-active,
.notice-leave-active,
.notice-move {
  transition:
    opacity var(--numen-motion) var(--numen-easing),
    transform var(--numen-motion) var(--numen-easing);
}

.notice-enter-from,
.notice-leave-to {
  opacity: 0;
  transform: translateY(0.5rem);
}

/* A card on its way out is out of the stack, so the ones above it come down
   while it goes. */
.notice-leave-active {
  position: absolute;
  inset-inline-start: 0;
}

/* A card is a card whether or not it arrived moving. */
@media (prefers-reduced-motion: reduce) {
  .notice-enter-active,
  .notice-leave-active,
  .notice-move {
    transition: none;
  }

  .notice-enter-from,
  .notice-leave-to {
    opacity: 1;
    transform: none;
  }
}
</style>
