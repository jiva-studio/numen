<script setup lang="ts">
/**
 * One card, as it stands in front of a person: the front, and the back once
 * they have said they are ready for it.
 *
 * A face is markup, and it is drawn as the markup it is. It cannot run: the
 * window serves itself under a policy that allows no script it did not serve
 * and no handler written in an attribute.
 */
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
    <!-- eslint-disable-next-line vue/no-v-html -->
    <div class="card__side" v-html="front" />
    <div v-if="shown" class="card__rule" />
    <!-- eslint-disable-next-line vue/no-v-html -->
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
