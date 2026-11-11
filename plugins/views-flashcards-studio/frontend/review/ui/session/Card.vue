<script setup lang="ts">
/**
 * One card of the session: the front, and the back once the answer is requested.
 */
import { computed, ref } from 'vue'
import { findLinkHref, renderCardHtml, vMarkup } from '../../../api/render'
import { scheme } from '../../../lib/cards'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  front: string
  back: string
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
const frontHtml = computed(() => renderCardHtml(props.front))
const backHtml = computed(() => renderCardHtml(props.back))

/* -------------------------------- Handlers -------------------------------- */
const onPointerDown = (press: PointerEvent) => {
  from.value = press.clientX
}

const onClick = (press: MouseEvent) => {
  const href = findLinkHref(press)
  if (href) {
    press.preventDefault()
    // A name carrying no scheme points inside the vault, which is where the
    // reading beside the card is.
    if (scheme(href) === null) emit('read', decodeHref(href))
    return
  }
  if (press.defaultPrevented) return
  // A press that did not drag requests the answer.
  const distanceX = from.value === null ? 0 : Math.abs(press.clientX - from.value)
  from.value = null
  if (distanceX <= MAX_TAP_DISTANCE && !props.isShown) emit('show')
}

/* -------------------------------- Helpers --------------------------------- */
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
    <div v-markup="frontHtml" class="prose prose-sm prose-numen numen max-w-none break-words"></div>
    <div v-if="isShown" class="card__rule" />
    <div
      v-if="isShown"
      v-markup="backHtml"
      class="prose prose-sm prose-numen numen max-w-none break-words"
    ></div>
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
  --rule-block-size: 0.0625rem;
  flex: none;
  block-size: var(--rule-block-size);
  background: var(--numen-edge);
}

.prose {
  font-size: var(--local-prose-size);
  user-select: text;
  -webkit-user-select: text;

  /* A line written with no enclosing tag is not wrapped, so the breaks
     between such lines are kept and each reads as the line it is. */
  white-space: pre-wrap;
}

/* Inside a user-written tag the whitespace belongs to the markup: a list is not
   indented by the spaces the file was laid out with. Preformatted text keeps
   its own. */
.prose :deep(*:not(pre, pre *)) {
  white-space: normal;
}

.prose :deep(table) {
  display: block;
  overflow-x: auto;
}

/* A slot naming no field is read at its current position. */
.prose :deep(mark) {
  --prose-padding-inline: 0.25em;
  border-radius: var(--numen-radius);
  padding-inline: var(--prose-padding-inline);
  background: var(--numen-alarm-bg);
  color: var(--numen-alarm);
}
</style>
