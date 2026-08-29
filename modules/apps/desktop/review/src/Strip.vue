<script setup lang="ts">
/**
 * The line a card is headed by: what is true of it on the left, and what can be
 * done to it on the right.
 *
 * Both the card being answered and the boxes it is put right in are headed by
 * one of these, so a mark is always in the same place whichever of them is in
 * front of the person.
 */
withDefaults(
  defineProps<{
    /** What it is cut by, where that is worth saying. */
    stencil?: string
    /** Nobody has answered it, which is what a person means by a new card. */
    fresh?: boolean
  }>(),
  { stencil: '', fresh: false },
)
</script>

<template>
  <header class="strip">
    <span v-if="fresh" class="strip__new">new</span>
    <span v-if="stencil" class="strip__cut">{{ stencil }}</span>
    <span class="strip__deeds"><slot /></span>
  </header>
</template>

<style scoped>
.strip {
  display: flex;
  flex: none;
  align-items: center;
  min-block-size: 1.75rem;
  gap: var(--numen-inset);
}

.strip__new {
  padding: 0.0625rem 0.375rem;
  border-radius: var(--numen-radius-pill);
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  font-size: var(--numen-edge-label-size);
}

.strip__cut {
  overflow: hidden;
  color: var(--numen-edge-label);
  font-size: var(--numen-edge-label-size);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.strip__deeds {
  display: flex;
  align-items: center;
  margin-inline-start: auto;
  gap: 0.125rem;
}
</style>
