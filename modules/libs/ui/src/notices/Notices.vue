<script setup lang="ts">
/**
 * What the window has to say, as cards in its bottom corner.
 *
 * Work appears once it has lasted and goes when the work does. Something that
 * is so stands while it is so. Something that happened stands to be read and
 * then goes, unless it is trouble, which stands until it is put away.
 *
 * It stands over what it covers and takes no room from it. What each card is
 * and what it is called belong to whoever draws this.
 */
import { computed, nextTick, onMounted, ref, useTemplateRef, watch, watchEffect } from 'vue'
import Activity from '../activity/Activity.vue'
import { remainingWord } from '../activity/model'
import {
  arrivals,
  dwellOf,
  finished,
  folded,
  measured,
  remembered,
  ROOM,
  showing,
  standing,
  tallyOf,
  WAIT,
  type Movement,
  type Notice,
} from './model'

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
    /** What the moment is. The window's, unless a story hands in its own. */
    clock?: () => number
    /** Whether nobody is looking. The window's, unless a story hands in its own. */
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

/** The ones a person has put away, and when each of the rest arrived. */
const away = ref<ReadonlySet<string>>(new Set())
const arrived = ref<ReadonlyMap<string, number>>(new Map())
const now = ref(props.clock())

/** How fast each count is moving. This is the clock the rate is read against. */
const moving = ref<ReadonlyMap<string, Movement>>(new Map())

const stack = useTemplateRef<HTMLElement>('stack')
/** Whether a pointer is on the stack, and whether the keyboard is in it. */
const pointed = ref(false)
const focused = ref(false)
/** Whether the corner is being held. */
const holding = (): boolean => pointed.value || focused.value || props.hidden()
/** How long it has been held for. */
const heldFor = ref(0)

/**
 * The moment a card is read against.
 *
 * Time a person spent with the corner under their pointer, or away from the
 * window entirely, is not time they spent reading it.
 */
const read = computed(() => now.value - heldFor.value)

/** The card a pointer is on, for as long as that card is still there. */
let on: Element | null = null

/**
 * Takes the clock forward, and the held time with it.
 *
 * A card is taken out from under whatever was on it, and a browser owes nothing
 * about the boundary event for one that has gone, so what holds the corner is
 * asked of the page each time.
 */
const beat = () => {
  if (pointed.value && on !== null && !on.isConnected) {
    pointed.value = false
    on = null
  }
  if (focused.value && !stack.value?.contains(document.activeElement)) focused.value = false
  const at = props.clock()
  if (holding()) heldFor.value += at - now.value
  now.value = at
}

/** The ones whose caller has already been told they are finished with. */
const forgotten = new Set<string>()

watch(
  () => props.notices,
  (all) => {
    beat()
    arrived.value = arrivals(arrived.value, all, read.value)
    away.value = remembered(away.value, all)
    moving.value = measured(moving.value, all, now.value)
    const here = new Set(standing(all).map((one) => one.id))
    for (const id of [...forgotten]) if (!here.has(id)) forgotten.delete(id)
  },
  { immediate: true },
)

const leftOn = (one: Notice): string => {
  const tally = tallyOf(one)
  if (tally === undefined) return ''
  return remainingWord(tally.total - tally.done, moving.value.get(one.id)?.rate ?? 0)
}

const drawn = computed(() =>
  showing(props.notices, arrived.value, away.value, read.value, props.wait),
)
/** Whether a person has asked to see what is folded away behind the rest. */
const opened = ref(false)
const folds = computed(() =>
  folded(drawn.value, opened.value ? drawn.value.length : props.room),
)

// Asking to see what is behind the rest is asked about what stands then. Once
// it all fits again, the next stack over the room folds as any other would.
watch(drawn, (all) => {
  if (all.length <= props.room) opened.value = false
})

/** Whether anything standing has not yet lasted long enough to be drawn. */
const coming = computed(() => {
  const shown = new Set(drawn.value.map((one) => one.id))
  return standing(props.notices).some(
    (one) => one.stay !== 'read' && !away.value.has(one.id) && !shown.has(one.id),
  )
})

/** Whether anything drawn is going to go by itself. */
const dwelling = computed(() =>
  drawn.value.some(
    (one) => one.stay === 'read' && dwellOf(one.says, one.about) !== Infinity,
  ),
)

// The corner changes by itself while nothing else changes, so the moment is
// watched for as long as something is waiting on it.
watchEffect((clean) => {
  if (!coming.value && !dwelling.value) return
  const tick = setInterval(beat, 250)
  clean(() => clearInterval(tick))
})

watchEffect(() => {
  for (const id of finished(props.notices, arrived.value, read.value)) {
    if (forgotten.has(id)) continue
    forgotten.add(id)
    emit('gone', id)
  }
})

/** The ways away, in the order they stand. */
const ways = (): readonly HTMLElement[] => [
  ...(stack.value?.querySelectorAll<HTMLElement>('.notice__away') ?? []),
]

/**
 * A card put away, and the keyboard left where it can go on putting them away.
 */
const put = async (id: string) => {
  const at = ways().indexOf(document.activeElement as HTMLElement)
  forgotten.add(id)
  away.value = new Set([...away.value, id])
  emit('gone', id)
  if (at < 0) return
  await nextTick()
  const left = ways()
  left[Math.min(at, left.length - 1)]?.focus()
}

/** What each card reads out as. */
const wordsOf = (one: Notice): string => (one.about ? `${one.says} — ${one.about}` : one.says)

/** What is read out, and what is read out over whatever else is being read. */
const told = ref('')
const cried = ref('')
/** The words each card was last read out by. */
const announced = ref<ReadonlyMap<string, string>>(new Map())
/** Whether both regions have stood empty, which is what makes them read. */
let listening = false
/**
 * Which reading is the one in hand, and what is waiting to be read out.
 *
 * Two changes can land inside one tick, and what the second reads out is
 * everything neither of them has read out yet.
 */
let reading = 0
let waiting: readonly Notice[] = []

const reads = async (all: readonly Notice[]) => {
  if (!listening) return
  const fresh = all.filter((one) => announced.value.get(one.id) !== wordsOf(one))
  announced.value = new Map(all.map((one) => [one.id, wordsOf(one)]))
  if (fresh.length === 0) return
  waiting = [...waiting, ...fresh]

  // A region holding the words already is a region that reads out nothing.
  const mine = ++reading
  told.value = ''
  cried.value = ''
  await nextTick()
  if (mine !== reading) return

  const said = waiting
  waiting = []
  const loud = said.filter((one) => one.tone === 'alarm')
  const quiet = said.filter((one) => one.tone !== 'alarm')
  if (loud.length) cried.value = loud.map(wordsOf).join('. ')
  if (quiet.length) told.value = quiet.map(wordsOf).join('. ')
}

watch(drawn, (all) => void reads(all))

onMounted(async () => {
  await nextTick()
  listening = true
  void reads(drawn.value)
})

const enters = (event: PointerEvent) => {
  const target = event.target
  on = target instanceof Element ? target.closest('.notice') : null
  pointed.value = true
}

const leaves = (event: PointerEvent) => {
  const to = event.relatedTarget
  if (to instanceof Node && stack.value?.contains(to)) return
  pointed.value = false
  on = null
}

const holds = () => {
  focused.value = true
}

const lets = (event: FocusEvent) => {
  const to = event.relatedTarget
  if (to instanceof Node && stack.value?.contains(to)) return
  focused.value = false
}
</script>

<template>
  <div class="notices numen font-sans text-small">
    <span class="sr-only" aria-live="polite">{{ told }}</span>
    <span class="sr-only" aria-live="assertive">{{ cried }}</span>

    <aside
      v-if="folds.shown.length || folds.over"
      ref="stack"
      class="notices__stack flex flex-col"
      :aria-label="name"
      @pointerover="enters"
      @pointerout="leaves"
      @focusin="holds"
      @focusout="lets"
    >
      <TransitionGroup name="notice">
        <button
          v-if="folds.over"
          key="folded"
          type="button"
          class="notice notice__folded"
          @click="opened = true"
        >
          {{ folds.over }} {{ more }}
        </button>

        <article
          v-for="one in folds.shown"
          :key="one.id"
          class="notice"
          :data-tone="one.tone ?? 'plain'"
        >
          <Activity
            class="notice__work"
            :says="one.says"
            :about="one.about ?? ''"
            :tally="tallyOf(one)"
            :working="one.working ?? false"
            :left="leftOn(one)"
            :tone="one.tone ?? 'plain'"
          />
          <button
            type="button"
            class="notice__away focus-visible:ring-ring focus-visible:ring-(length:--numen-ring-width)"
            :aria-label="`${putAway}: ${one.says}`"
            @click="put(one.id)"
          >
            <svg viewBox="0 0 12 12" aria-hidden="true" focusable="false">
              <path d="M3 3 L9 9 M9 3 L3 9" />
            </svg>
          </button>
        </article>
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

/* As wide as it needs, and never wider than a narrow window. */
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
  border: var(--numen-stroke) solid var(--numen-node-border);
  border-radius: var(--numen-radius-panel);
  background: var(--numen-node-bg);
  color: var(--numen-node-fg);
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

.notice__work {
  min-inline-size: 0;
  flex: 1;
}

.notice__away {
  flex: none;
  display: grid;
  place-items: center;
  inline-size: 1.25rem;
  block-size: 1.25rem;
  border-radius: var(--numen-radius-pill);
  color: inherit;
  opacity: 0.5;
}

.notice__away:hover,
.notice__away:focus-visible {
  opacity: 1;
  background: color-mix(in oklab, currentColor 12%, transparent);
}

.notice__away svg {
  inline-size: 0.75rem;
  block-size: 0.75rem;
  stroke: currentColor;
  stroke-width: 1.5;
  stroke-linecap: round;
  fill: none;
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
