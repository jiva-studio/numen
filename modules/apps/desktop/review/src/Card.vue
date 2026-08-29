<script setup lang="ts">
/**
 * One card, as it stands in front of a person: the front, and the back once
 * they have said they are ready for it.
 *
 * A card is markdown the person wrote, so it is drawn as markdown — a picture
 * in a card is a picture, and a list is a list.
 */
import { Prose } from '@numen/ui'

defineProps<{
  front: string
  back: string
  /** Whether the answer is being shown. */
  shown: boolean
}>()

defineEmits<{ (event: 'show'): void }>()
</script>

<template>
  <article class="card" @click="!shown && $emit('show')">
    <div class="card__side card__side--front">
      <Prose :text="front" />
    </div>
    <div v-if="shown" class="card__rule" />
    <div v-if="shown" class="card__side card__side--back">
      <Prose :text="back" />
    </div>
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
  border-radius: var(--numen-radius-panel);
  background: var(--numen-node-bg);
  gap: var(--numen-inset-wide);
}

/* What a person reads off a card is theirs to carry out of the window, so the
   text takes a selection back. */
.card__side {
  user-select: text;
  -webkit-user-select: text;
}

.card__side--front {
  font-size: 1.0625rem;
}

.card__rule {
  flex: none;
  block-size: 1px;
  background: var(--numen-edge);
}
</style>
