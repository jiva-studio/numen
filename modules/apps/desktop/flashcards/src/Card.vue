<script setup lang="ts">
/**
 * One card, as it stands in front of a person: the front, and the back once
 * they have said they are ready for it.
 *
 * A face is HTML and is drawn as the HTML it is, measured first against what a
 * card may be drawn with: a deck may have come from another person, and what
 * they wrote is not this window's to run, style or navigate with.
 */
import { computed, ref } from 'vue'
import { safe } from '@numen/ui'

const props = defineProps<{
  front: string
  back: string
  /** Whether the answer is being shown. */
  shown: boolean
}>()

const emit = defineEmits<{ (event: 'show'): void }>()

const front = computed(() => safe(props.front))
const back = computed(() => safe(props.back))

/** A hand that moved less than this across was pressing and not dragging. */
const STILL = 4

/** Where the hand went down, for telling a press from a drag across the card. */
const from = ref<number | null>(null)

const took = (press: PointerEvent) => {
  from.value = press.clientX
}

/**
 * The card is turned over by pressing it. A link inside one is not followed:
 * this window has one page and no way back to it, and where a card points is
 * read in the editor.
 */
const pressed = (press: MouseEvent) => {
  if ((press.target as HTMLElement | null)?.closest?.('a')) {
    press.preventDefault()
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
