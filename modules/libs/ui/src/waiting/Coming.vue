<script setup lang="ts">
/**
 * The shape a figure will take, standing in the room it will take, while it is
 * still being worked out.
 *
 * It says that something is coming and never that there is nothing: whoever
 * draws it draws the value itself the moment there is one. The fill is taken
 * from the text of whatever holds it.
 */
withDefaults(
  defineProps<{
    /** How wide it stands, as a length. */
    wide?: string
    /** How tall it stands, as a length. */
    high?: string
    /** Rounded to its own ends, where what is coming is drawn as a pill. */
    pill?: boolean
  }>(),
  { wide: '100%', high: '1em', pill: false },
)
</script>

<template>
  <span
    class="coming numen"
    :class="{ 'coming--pill': pill }"
    :style="{ inlineSize: wide, blockSize: high }"
    aria-hidden="true"
  />
</template>

<style scoped>
.coming {
  --cycle: 1600ms;

  display: inline-block;
  flex: none;
  vertical-align: middle;
  border-radius: var(--numen-radius);
  background: color-mix(in srgb, currentcolor 14%, transparent);
  animation: coming-breathe var(--cycle) ease-in-out infinite;
}

.coming--pill {
  border-radius: var(--numen-radius-pill);
}

/* Between a fill and half of one, and no faster than a person reading the row
   it stands in. */
@keyframes coming-breathe {
  50% {
    opacity: 0.5;
  }
}

/* Still, it is a filled shape where the figure will be, which is the whole of
   what it has to say. */
@media (prefers-reduced-motion: reduce) {
  .coming {
    animation: none;
  }
}
</style>
