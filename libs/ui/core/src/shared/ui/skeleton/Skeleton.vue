<script setup lang="ts">
/**
 * Placeholder with the shape of a figure, occupying the space it will take while its
 * value is still being computed.
 *
 * It indicates that a value is pending and never that there is none: the consumer
 * renders the value itself as soon as it is available. The fill is taken from the
 * text colour of the parent.
 */
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    /** Width, as a length. */
    wide?: string
    /** Height, as a length. */
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
  --cycle: var(--numen-wait-shimmer);

  display: inline-block;
  flex: none;
  vertical-align: middle;
  border-radius: var(--numen-radius);
  background: color-mix(in srgb, currentcolor 14%, transparent);
  animation: skeleton-breathe var(--cycle) ease-in-out infinite;
}

.skeleton--pill {
  border-radius: var(--numen-radius-isPill);
}

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
