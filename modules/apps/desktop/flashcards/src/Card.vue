<script setup lang="ts">
/**
 * One card, as it stands in front of a person: the front, and the back once
 * they have said they are ready for it.
 *
 * A face is markdown, with the tags a person writes among the marks, and is
 * drawn through the one function that reads it. What comes out is measured
 * against what a card may be drawn with: a deck may have come from another
 * person, and what they wrote is not this window's to run, style or navigate
 * with.
 */
import { computed, ref } from 'vue'
import { drawn, scheme } from '@numen/ui'

const props = defineProps<{
  front: string
  back: string
  /** Whether the answer is being shown. */
  shown: boolean
}>()

const emit = defineEmits<{
  (event: 'show'): void
  (event: 'read', named: string): void
}>()

const front = computed(() => drawn(props.front))
const back = computed(() => drawn(props.back))

/** A hand that moved less than this across was pressing and not dragging. */
const STILL = 4

/** Where the hand went down, for telling a press from a drag across the card. */
const from = ref<number | null>(null)

const took = (press: PointerEvent) => {
  from.value = press.clientX
}

/**
 * A name written into an href is escaped, and a card comes from whoever wrote
 * it: one escaped wrongly is taken as the characters it already is.
 */
const plain = (named: string) => {
  try {
    return decodeURIComponent(named)
  } catch {
    return named
  }
}

/**
 * The card is turned over by pressing it. A link inside one is never followed —
 * this window has one page — but a link into the vault opens the reading beside
 * the card on the note it names.
 */
const pressed = (press: MouseEvent) => {
  const link = (press.target as HTMLElement | null)?.closest?.('a')
  if (link) {
    press.preventDefault()
    const named = link.getAttribute('href') ?? ''
    // A name carrying no scheme points inside the vault, which is where the
    // reading beside the card is.
    if (named && scheme(named) === null) emit('read', plain(named))
    return
  }
  // A hand that took the card across was moving the panel into view, and a
  // press that went nowhere is a person asking for the answer.
  const went = from.value === null ? 0 : Math.abs(press.clientX - from.value)
  from.value = null
  if (went <= STILL && !props.shown) emit('show')
}
</script>

<template>
  <article class="card" @click="pressed" @pointerdown="took">
    <!-- eslint-disable-next-line vue/no-v-html -- measured against what a card may be drawn with -->
    <div class="card__side" v-html="front" />
    <div v-if="shown" class="card__rule" />
    <!-- eslint-disable-next-line vue/no-v-html -- measured against what a card may be drawn with -->
    <div v-if="shown" class="card__side" v-html="back" />
  </article>
</template>

<style scoped>
.card {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-block-size: 0;
  padding: var(--numen-inset-wide);
  overflow-y: auto;
  border: 1px solid var(--numen-node-border);
  border-radius: var(--numen-radius);
  background: var(--numen-node-bg);
  /* A card is read, not scanned, so it is set at the size reading is set at. */
  font-size: var(--numen-reading-size);
  gap: var(--numen-inset-wide);
}

/* What a person reads off a card is theirs to carry out of the window, so the
   text takes a selection back. */
.card__side {
  user-select: text;
  -webkit-user-select: text;
}

.card__rule {
  flex: none;
  block-size: 1px;
  background: var(--numen-edge);
}
</style>
