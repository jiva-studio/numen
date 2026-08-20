<script setup lang="ts">
/**
 * A document read: one page of it drawn, with rectangles lit over it.
 *
 * The page arrives drawn, and this says how wide it wants it in device pixels.
 * A rectangle is a fraction of the page, so it is placed in per cent and the
 * zoom carries it along.
 *
 * It fills whatever it is put in, and says nothing about where that is.
 */
import { computed, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from 'vue'
import { Button } from '@/components/ui/button'

/** Where something sits on the page, in fractions of it. */
interface Lit {
  readonly minX: number
  readonly minY: number
  readonly maxX: number
  readonly maxY: number
}

const props = withDefaults(
  defineProps<{
    /** The page in front, drawn, as an address a picture is pointed at. */
    picture?: string
    /** How many pages the document has. */
    pages?: number
    /** Which page is in front, counted from the first. */
    at?: number
    /** What is lit on the page in front, in fractions of it. */
    lit?: readonly Lit[]
    /** What turning back a page is called, and turning on. */
    back?: string
    next?: string
    /** What the field the page is typed in is called. */
    page?: string
    /** What drawing the page larger is called, and smaller. */
    closer?: string
    further?: string
    /** What is said where the page would not come. */
    undrawn?: string
  }>(),
  {
    picture: '',
    pages: 0,
    at: 0,
    lit: () => [],
    back: 'Previous page',
    next: 'Next page',
    page: 'Page',
    closer: 'Closer',
    further: 'Further',
    undrawn: 'This page would not come.',
  },
)

const emit = defineEmits<{
  /** The page to turn to, counted from the first. */
  (event: 'go', page: number): void
  /** The page is wanted this many device pixels across. */
  (event: 'wide', pixels: number): void
}>()

/** How close the page is drawn. One is the width of the room it is read in. */
const zoom = ref(1)
const CLOSEST = 4
const FURTHEST = 0.5
const NEARER = 1.25

/** The room the page is read in, in CSS pixels. */
const room = ref(0)
/** How wide the page is drawn, in CSS pixels. */
const wide = computed(() => Math.round(room.value * zoom.value))

/**
 * The widths a page is asked for. Dragging the edge of a pane crosses a few of
 * them, and a page drawn wider than its box is drawn down into it.
 */
const STAGE = 128
const staged = (pixels: number) => Math.ceil(pixels / STAGE) * STAGE

/**
 * How long the width has to have stood still before the page is asked for at
 * it. A pane edge dragged across a screen crosses a dozen widths, and each one
 * is a page drawn and thrown away.
 */
const SETTLED = 150
let settling: ReturnType<typeof setTimeout> | undefined

// The first width is asked for at once: a document opening has nothing drawn
// and nothing to wait for.
watch(wide, (pixels, before) => {
  if (!before) {
    if (pixels > 0) emit('wide', staged(pixels * devicePixelRatio))
    return
  }
  clearTimeout(settling)
  settling = setTimeout(() => {
    if (pixels > 0) emit('wide', staged(pixels * devicePixelRatio))
  }, SETTLED)
})

const area = useTemplateRef<HTMLElement>('area')
let watching: ResizeObserver | undefined

const measure = () => {
  if (area.value) room.value = area.value.clientWidth
}

onMounted(() => {
  measure()
  if (!area.value || typeof ResizeObserver === 'undefined') return
  watching = new ResizeObserver(measure)
  watching.observe(area.value)
})

onBeforeUnmount(() => {
  watching?.disconnect()
  clearTimeout(settling)
})

/** What is typed in the field, which follows the page in front. */
const typed = ref(String(props.at + 1))
watch(
  () => props.at,
  (page) => {
    typed.value = String(page + 1)
  },
)

/** A page asked for by its place in the document. The field shows the page in front. */
const turn = () => {
  const page = Number(typed.value)
  if (Number.isFinite(page)) emit('go', Math.round(page) - 1)
  typed.value = String(props.at + 1)
}

/**
 * The page that would not draw, and how many times it has been asked for.
 *
 * A document is busy while another page of it is drawing, so a page that did
 * not come is asked for again a few times before the pane says it is not
 * coming. Each ask carries a number the last one did not, because a picture at
 * an address the browser already refused is not asked for again.
 */
const missing = ref('')
const tries = ref(0)

/** The address the page in front is asked for at, this attempt. */
const drawing = computed(() =>
  !props.picture || props.picture === missing.value
    ? ''
    : tries.value > 0
      ? `${props.picture}&again=${tries.value}`
      : props.picture,
)

const failed = () => {
  if (tries.value >= ATTEMPTS) {
    missing.value = props.picture ?? ''
    tries.value = 0
    return
  }
  tries.value += 1
}

/** How many times a page that did not come is asked for again. */
const ATTEMPTS = 3

// A page turned to is asked for afresh. What would not come is what would not
// come at that moment, and turning away and back is a person asking again.
watch(
  () => props.picture,
  () => {
    missing.value = ''
    tries.value = 0
  },
)

const drawTo = (how: number) => {
  zoom.value = Math.min(Math.max(zoom.value * how, FURTHEST), CLOSEST)
}

/** One lit rectangle, as a share of the page it is drawn over. */
const boxOf = (one: Lit) => ({
  insetInlineStart: `${one.minX * 100}%`,
  insetBlockStart: `${one.minY * 100}%`,
  inlineSize: `${(one.maxX - one.minX) * 100}%`,
  blockSize: `${(one.maxY - one.minY) * 100}%`,
})

defineExpose({
  /**
   * Take the room again. A reader drawn out of sight has none, and the caller
   * says when it is on screen.
   */
  measure,
})
</script>

<template>
  <div class="reader numen flex h-full min-h-0 flex-col font-sans text-base text-ink">
    <div class="reader__turning flex shrink-0 items-center gap-1 px-inset py-inset">
      <Button
        variant="ghost"
        size="icon-small"
        :aria-label="back"
        :disabled="at <= 0"
        @click="emit('go', at - 1)"
      >
        <svg
          viewBox="0 0 16 16"
          class="size-4"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M10 3 5 8l5 5" />
        </svg>
      </Button>
      <Button
        variant="ghost"
        size="icon-small"
        :aria-label="next"
        :disabled="at >= pages - 1"
        @click="emit('go', at + 1)"
      >
        <svg
          viewBox="0 0 16 16"
          class="size-4"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="m6 3 5 5-5 5" />
        </svg>
      </Button>

      <input
        v-model="typed"
        class="reader__at w-10 rounded-node border border-field-rule bg-field px-1 text-center text-small text-ink outline-none focus-visible:ring-(length:--numen-ring-width) focus-visible:ring-ring"
        type="number"
        min="1"
        :max="pages"
        :aria-label="page"
        @change="turn"
        @keydown.enter="turn"
      />
      <span class="text-small text-hushed">/ {{ pages }}</span>

      <Button
        variant="ghost"
        size="icon-small"
        class="ml-auto"
        :aria-label="further"
        :disabled="zoom <= FURTHEST"
        @click="drawTo(1 / NEARER)"
      >
        <svg
          viewBox="0 0 16 16"
          class="size-4"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
        >
          <path d="M3.5 8h9" />
        </svg>
      </Button>
      <Button
        variant="ghost"
        size="icon-small"
        :aria-label="closer"
        :disabled="zoom >= CLOSEST"
        @click="drawTo(NEARER)"
      >
        <svg
          viewBox="0 0 16 16"
          class="size-4"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
        >
          <path d="M8 3.5v9M3.5 8h9" />
        </svg>
      </Button>
    </div>

    <div ref="area" class="reader__room min-h-0 flex-1 overflow-auto">
      <div v-if="drawing" class="flex min-w-max justify-center">
        <figure
          class="reader__page relative my-inset overflow-hidden rounded-node"
          :style="{ inlineSize: `${wide}px` }"
        >
          <img
            class="reader__picture block w-full"
            :src="drawing"
            :alt="`${page} ${at + 1}`"
            @error="failed"
            @load="tries = 0"
          />
          <div
            v-for="(one, index) in lit"
            :key="index"
            class="reader__lit pointer-events-none absolute rounded-[2px] bg-(--numen-highlight)"
            :style="boxOf(one)"
          />
        </figure>
      </div>
      <p v-else class="reader__silence grid h-full place-items-center text-small text-hushed">
        <template v-if="missing">{{ undrawn }}</template>
        <slot v-else name="silence" />
      </p>
    </div>
  </div>
</template>

<style scoped>
/* The page stands on the surface, and its own edge is what tells it from it.
   The edge is the page's, so what is lit is placed inside it. */
.reader__page {
  border: var(--numen-stroke) solid var(--numen-node-border);
  background: var(--numen-node-bg);
}

/* The field carries the page and nothing else: a number field's own arrows are
   not drawn. */
.reader__at::-webkit-inner-spin-button,
.reader__at::-webkit-outer-spin-button {
  appearance: none;
  margin: 0;
}

.reader__at {
  appearance: textfield;
}
</style>
