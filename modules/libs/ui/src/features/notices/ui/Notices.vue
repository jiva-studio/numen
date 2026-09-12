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
import { computed, nextTick, ref, shallowRef, useTemplateRef, watch, watchEffect } from 'vue'
import { LiveRegions } from './live-regions'
import NoticeCard from './NoticeCard.vue'
import { arrivals, dwellOf, getFinishedNotices, getShownNotices, WAIT } from '../lib/dwell'
import { foldNotices, ROOM } from '../lib/fold'
import { measured, type Movement } from '../lib/movement'
import { readable, remembered, tallyOf, type Notice } from '../lib/notice'
import { getRemainingWord } from '../lib/tally'
import { useAnnouncer } from '../model/announcer'
import { useNoticeStack } from '../model/stack'

const props = withDefaults(
  defineProps<{
    /** What the window has to say, in the order it is drawn. */
    notices?: readonly Notice[]
    /** What the corner is announced as. */
    name?: string
    /** What the way to put one away is called. */
    putAway?: string
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
    putAway: 'Put away',
    more: 'more',
    wait: WAIT,
    room: ROOM,
    clock: () => Date.now(),
    hidden: () => document.hidden,
  },
)

const emit = defineEmits<{
  /** A card is finished with: read long enough, or put away. */
  (event: 'gone', id: string): void
}>()

/** How fast each count is moving. This is the clock the rate is read against. */
const moving = shallowRef<ReadonlyMap<string, Movement>>(new Map())

/** The ones a person has put away, and when each of the rest arrived. */
const away = shallowRef<ReadonlySet<string>>(new Set())
const arrived = shallowRef<ReadonlyMap<string, number>>(new Map())

const stack = useTemplateRef<HTMLElement>('stack')

/** How long the corner has been held for, and the moment a card is read against. */
const { now, read, beat, onPointerOver, onPointerOut, onFocusIn, onFocusOut } = useNoticeStack(
  stack,
  () => props.clock(),
  () => props.hidden(),
)

/** The ones whose caller has already been told they are finished with. */
const forgotten = new Set<string>()

/**
 * The clock forward, and every count read against it.
 *
 * A count is read on the clock rather than as it arrives, so how fast it is
 * moving is measured over stretches of time and not over however often the work
 * behind it happens to speak.
 */
const sample = (): void => {
  beat()
  moving.value = measured(moving.value, props.notices, now.value)
}

watch(
  () => props.notices,
  (all) => {
    sample()
    arrived.value = arrivals(arrived.value, all, read.value)
    away.value = remembered(away.value, all)
    const here = new Set(readable(all).map((one) => one.id))
    for (const id of [...forgotten]) if (!here.has(id)) forgotten.delete(id)
  },
  { immediate: true },
)

const leftOn = (one: Notice): string => {
  const tally = tallyOf(one)
  if (tally === undefined) return ''
  return getRemainingWord(tally.total - tally.done, moving.value.get(one.id)?.rate ?? 0)
}

const drawn = computed(() =>
  getShownNotices(props.notices, arrived.value, away.value, read.value, props.wait),
)
/** Whether a person has asked to see what is folded away behind the rest. */
const opened = ref(false)
const folds = computed(() =>
  foldNotices(drawn.value, opened.value ? drawn.value.length : props.room),
)

// Asking to see what is behind the rest is asked about what stands then. Once
// it all fits again, the next stack over the room folds as any other would.
watch(drawn, (all) => {
  if (all.length <= props.room) opened.value = false
})

/** Whether anything readable has not yet lasted long enough to be drawn. */
const coming = computed(() => {
  const shown = new Set(drawn.value.map((one) => one.id))
  return readable(props.notices).some(
    (one) => one.stay !== 'read' && !away.value.has(one.id) && !shown.has(one.id),
  )
})

/** Whether anything drawn is going to go by itself. */
const dwelling = computed(() =>
  drawn.value.some(
    (one) => one.stay === 'read' && dwellOf(one.says, one.about) !== Infinity,
  ),
)

/** Whether anything drawn is counting, and so has a rate to be read. */
const counting = computed(() => drawn.value.some((one) => tallyOf(one) !== undefined))

// The corner changes by itself while nothing else changes, so the moment is
// watched for as long as something is waiting on it.
watchEffect((clean) => {
  if (!coming.value && !dwelling.value && !counting.value) return
  const tick = setInterval(sample, 250)
  clean(() => clearInterval(tick))
})

watchEffect(() => {
  for (const id of getFinishedNotices(props.notices, arrived.value, read.value)) {
    if (forgotten.has(id)) continue
    forgotten.add(id)
    emit('gone', id)
  }
})

/** A card standing now, which carries the way away under it. */
interface CardHandle {
  readonly way: HTMLElement | null
}

/** The cards drawn, each under the notice it stands for. */
const cards = new Map<string, CardHandle>()

const holdCard = (id: string, card: unknown): void => {
  if (card) cards.set(id, card as CardHandle)
  else cards.delete(id)
}

/**
 * A card put away, and the keyboard left where it can go on putting them away:
 * on the card that takes the place of the one that went, or on the last.
 */
const put = async (id: string) => {
  const at = folds.value.shown.findIndex((one) => one.id === id)
  const held = cards.get(id)?.way === document.activeElement
  forgotten.add(id)
  away.value = new Set([...away.value, id])
  emit('gone', id)
  if (!held || at < 0) return
  await nextTick()
  const left = folds.value.shown
  const next = left[Math.min(at, left.length - 1)]
  if (next) cards.get(next.id)?.way?.focus()
}

/** What the corner is read out through. */
const { told, cried } = useAnnouncer(() => drawn.value)

</script>

<template>
  <div class="notices numen font-sans text-small">
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
          :put-away="putAway"
          :left="leftOn(one)"
          @pointerover="onPointerOver"
          @put="put(one.id)"
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
