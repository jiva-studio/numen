<script setup lang="ts">
/**
 * One card of the session: the front, and the back once the answer is requested.
 *
 * A review card is drawn by the one component every review card is drawn by, so a card reads
 * here the way it reads where it was written.
 */
import { ref } from 'vue'
import { CardProse, scheme } from '../../../../ui/cards'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  front: string
  back: string
  /** Whether the answer is being shown. */
  isShown: boolean
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'show'): void
  (event: 'read', note: string): void
}>()

/** A pointer that moved less than this distance horizontally is a press and not a drag. */
const MAX_TAP_DISTANCE = 4

/* --------------------------------- State ---------------------------------- */
/** Horizontal position where the pointer went down, to tell a press from a drag across the card. */
const from = ref<number | null>(null)

/* -------------------------------- Handlers -------------------------------- */
const onPointerDown = (press: PointerEvent) => {
  from.value = press.clientX
}

/**
 * A link inside a card is never followed — this window has one page — but a
 * link into the vault opens the reading beside the card on the note it names.
 */
const onFollow = (href: string, press: MouseEvent) => {
  press.preventDefault()
  // A name carrying no scheme points inside the vault, which is where the
  // reading beside the card is.
  if (scheme(href) === null) emit('read', decodeHref(href))
}

/** The card is turned over by pressing it, a press spent on a link aside. */
const onClick = (press: MouseEvent) => {
  if (press.defaultPrevented) return
  // A press that did not drag requests the answer.
  const distanceX = from.value === null ? 0 : Math.abs(press.clientX - from.value)
  from.value = null
  if (distanceX <= MAX_TAP_DISTANCE && !props.isShown) emit('show')
}

/* -------------------------------- Helpers --------------------------------- */
/**
 * A name written into an href is URI-escaped. A card's content is author-supplied,
 * so a malformed escape is kept as the literal characters.
 */
function decodeHref(href: string): string {
  try {
    return decodeURIComponent(href)
  } catch {
    // Nothing is wrong with the card; only with an escape in it, and the
    // characters as written are the closest thing to what was meant.
    return href
  }
}
</script>

<template>
  <article class="card" @click="onClick" @pointerdown="onPointerDown">
    <CardProse :text="front" @follow="onFollow" />
    <div v-if="isShown" class="card__rule" />
    <CardProse v-if="isShown" :text="back" @follow="onFollow" />
  </article>
</template>

<style scoped>
.card {
  /* A card is read, not scanned, so it is set at the size reading is set at,
     and so is the prose inside it. */
  --local-prose-size: var(--numen-reading-size);

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
