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
import { WAIT } from '../lib/dwell'
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
    wait: WAIT,
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
  drawn,
  folds,
  opened,
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
const { told, cried } = useAnnouncer(() => drawn.value)
</script>

<template>
  <div class="notices numen text-small font-sans">
    <LiveRegions :told="told" :cried="cried" />

    <aside
      v-if="folds.shown.length || folds.over"
      ref="stack"
      class="notices__stack flex flex-col"
      :aria-label="name"
      @pointerout="onPointerOut"
      @focusin="onFocusIn"
      @focusout="onFocusOut"
    >
      <TransitionGroup name="notice">
        <button
          v-if="folds.over"
          key="folded"
          type="button"
          class="notice notice__folded"
          @pointerover="onPointerOver"
          @click="opened = true"
        >
          {{ folds.over }} {{ more }}
        </button>

        <NoticeCard
          v-for="one in folds.shown"
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

/* One width whatever it says, and never wider than a narrow window. */
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

/* A card a person has to read stands on a ground of its own. */
.notice[data-tone='caution'] {
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  border-color: color-mix(in oklab, currentColor 25%, transparent);
}

.notice[data-tone='alarm'] {
  background: var(--numen-alarm-bg);
  color: var(--numen-alarm);
  border-color: color-mix(in oklab, currentColor 25%, transparent);
}

/* What stands behind the rest is counted on a card of its own, and pressing it
   brings them out. */
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
