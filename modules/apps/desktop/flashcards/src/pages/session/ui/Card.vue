<script setup lang="ts">
/**
 * One card, as it stands in front of a person: the front, and the back once
 * they have said they are ready for it.
 *
 * A face is drawn by the one component every face is drawn by, so a card reads
 * here the way it reads where it was written.
 */
import { ref } from 'vue'
import { CardProse, scheme } from '@numen/ui'

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
    // Nothing is wrong with the card; only with an escape in it, and the
    // characters as written are the closest thing to what was meant.
    return named
  }
}

/**
 * A link inside a card is never followed — this window has one page — but a
 * link into the vault opens the reading beside the card on the note it names.
 */
const handleFollow = (href: string, press: MouseEvent) => {
  press.preventDefault()
  // A name carrying no scheme points inside the vault, which is where the
  // reading beside the card is.
  if (scheme(href) === null) emit('read', plain(href))
}

/** The card is turned over by pressing it, a press spent on a link aside. */
const handleClick = (press: MouseEvent) => {
  if (press.defaultPrevented) return
  // A hand that took the card across was moving the panel into view, and a
  // press that went nowhere is a person asking for the answer.
  const went = from.value === null ? 0 : Math.abs(press.clientX - from.value)
  from.value = null
  if (went <= STILL && !props.shown) emit('show')
}
</script>

<template>
  <article class="card" @click="handleClick" @pointerdown="took">
    <CardProse :text="front" @follow="handleFollow" />
    <div v-if="shown" class="card__rule" />
    <CardProse v-if="shown" :text="back" @follow="handleFollow" />
  </article>
</template>

<style scoped>
.card {
  /* A card is read, not scanned, so it is set at the size reading is set at,
     and so is the prose inside it. */
  --numen-prose-size: var(--numen-reading-size);

  display: flex;
  flex: 1;
  flex-direction: column;
  min-block-size: 0;
  padding: var(--numen-inset-wide);
  overflow-y: auto;
  border: 1px solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background: var(--numen-raised);
  font-size: var(--numen-reading-size);
  gap: var(--numen-inset-wide);
}

.card__rule {
  flex: none;
  block-size: 1px;
  background: var(--numen-edge);
}
</style>
