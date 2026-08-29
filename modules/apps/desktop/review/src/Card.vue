<script setup lang="ts">
/**
 * One card, as it stands in front of a person: the front, and the back once
 * they have said they are ready for it.
 *
 * A face is markup, and it is drawn as the markup it is. It cannot run: the
 * window serves itself under a policy that allows no script it did not serve
 * and no handler written in an attribute.
 */
withDefaults(
  defineProps<{
    front: string
    back: string
    /** Whether the answer is being shown. */
    shown: boolean
    /** Nobody has answered this card, which is what a person means by a new one. */
    fresh?: boolean
  }>(),
  { fresh: false },
)

defineEmits<{ (event: 'show'): void }>()
</script>

<template>
  <article class="card" @click="!shown && $emit('show')">
    <!-- What is true of this card and not of the one before it, in its own
         corner: a card nobody has answered. -->
    <span v-if="fresh" class="card__new">new</span>
    <!-- eslint-disable-next-line vue/no-v-html -->
    <div class="card__side" v-html="front" />
    <div v-if="shown" class="card__rule" />
    <!-- eslint-disable-next-line vue/no-v-html -->
    <div v-if="shown" class="card__side" v-html="back" />
  </article>
</template>

<style scoped>
.card {
  position: relative;
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

.card__new {
  position: absolute;
  inset-block-start: var(--numen-inset);
  inset-inline-end: var(--numen-inset);
  padding: 0.0625rem 0.375rem;
  border-radius: var(--numen-radius-pill);
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  font-size: var(--numen-edge-label-size);
}

.card__rule {
  flex: none;
  block-size: 1px;
  background: var(--numen-edge);
}
</style>
