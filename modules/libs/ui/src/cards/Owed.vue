<script setup lang="ts">
/**
 * How many cards a collection has waiting today, wherever one is named: a vault
 * on the screen a window opens on, a deck in the list under it, or the whole of
 * a vault on the button that runs it.
 *
 * One number, and the word for what it counts. What is owed and what has never
 * been asked are both cards to sit down to, and a person choosing where to
 * start is not choosing between them.
 */
withDefaults(
  defineProps<{
    /** Cards waiting today: owed, and never asked. */
    waiting: number
    /**
     * The number alone. Where a list is long and the room is short, the word is
     * said once above the list and not on every row of it.
     */
    bare?: boolean
    /** Drawn on a ground of its own, where it stands on a filled button. */
    over?: boolean
  }>(),
  { bare: false, over: false },
)
</script>

<template>
  <span class="owed" :class="{ 'owed--over': over }">
    {{ waiting }}<template v-if="!bare"> to review</template>
  </span>
</template>

<style scoped>
.owed {
  flex: none;
  min-inline-size: 1.5rem;
  padding: 0.0625rem 0.4rem;
  text-align: center;
  border-radius: var(--numen-radius-pill);
  background: var(--numen-highlight);
  color: var(--numen-caution-fg);
  font-size: var(--numen-edge-label-size);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

/* On a filled button the ground is the button's own, lightened. */
.owed--over {
  background: #ffffff2e;
  color: inherit;
}
</style>
