<script setup lang="ts">
/**
 * How many cards a collection has waiting today, wherever one is named: a vault
 * on the screen a window opens on, a deck in the list under it, or the whole of
 * a vault on the button that runs it.
 *
 * One number, and the word for what it counts. What is owed and what has never
 * been asked are both cards to sit down to, and a person choosing where to
 * start is not choosing between them.
 *
 * A count that has not been worked out yet is drawn as the shape of the figure
 * it will be, in the same box, so the row it stands in does not move when the
 * figure lands.
 */
import Coming from '../waiting/Coming.vue'

withDefaults(
  defineProps<{
    /** Cards waiting today: owed, and never asked. Nothing until it is counted. */
    waiting: number | null
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
  <span
    class="owed"
    :class="{ 'owed--over': over }"
    :aria-label="waiting === null ? 'still being counted' : undefined"
  >
    <!-- Two figures wide, which is what a day's cards come to for most
         collections, and the box keeps the height of the line either way. -->
    <Coming v-if="waiting === null" wide="0.8rem" high="0.7em" pill />
    <template v-else>{{ waiting }}<template v-if="!bare"> to review</template></template>
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

/* On a filled button the ground is the button's own, lightened by the text that
   stands on it, so the pill follows whatever the button is painted. */
.owed--over {
  background: color-mix(in srgb, currentcolor 18%, transparent);
  color: inherit;
}
</style>
