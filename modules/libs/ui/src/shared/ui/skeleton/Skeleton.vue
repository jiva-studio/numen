<script setup lang="ts">
/**
 * The shape a figure will take, standing in the room it will take, while it is
 * still being worked out.
 *
 * It says that something is on its way and never that there is nothing:
 * whoever draws it draws the value itself the moment there is one. The fill is
 * taken from the text of whatever holds it.
 */
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    /** How wide it stands, as a length. */
    wide?: string
    /** How tall it stands, as a length. */
    high?: string
    /** Rounded to its own ends, where what is awaited is drawn as a pill. */
    isPill?: boolean
  }>(),
  { wide: '100%', high: '1em', isPill: false },
)

const skeletonStyle = computed(() => ({
  inlineSize: props.wide,
  blockSize: props.high,
}))
</script>

<template>
  <span
    class="skeleton numen"
    :class="{ 'skeleton--pill': isPill }"
    :style="skeletonStyle"
    aria-hidden="true"
  />
</template>

<style scoped>
.skeleton {
  --cycle: 1600ms;

  display: inline-block;
  flex: none;
  vertical-align: middle;
  border-radius: var(--numen-radius);
  background: color-mix(in srgb, currentcolor 14%, transparent);
  animation: skeleton-breathe var(--cycle) ease-in-out infinite;
}

.skeleton--pill {
  border-radius: var(--numen-radius-pill);
}

/* Between a fill and half of one, and no faster than a person reading the row
   it stands in. */
@keyframes skeleton-breathe {
  50% {
    opacity: 0.5;
  }
}

/* Still, it is a filled shape where the figure will be, which is the whole of
   what it has to say. */
@media (prefers-reduced-motion: reduce) {
  .skeleton {
    animation: none;
  }
}
</style>
