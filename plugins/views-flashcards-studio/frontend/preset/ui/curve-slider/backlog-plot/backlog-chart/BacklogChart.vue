<script setup lang="ts">
/**
 * The backlog run, drawn against the rules it is read off. It carries
 * `data-backlog` for its two parts: `picture` and `line`.
 */
import {
  BACKLOG_PLOT_HEIGHT,
  BACKLOG_PLOT,
  PLOT_LEFT,
  PLOT_RIGHT,
  PLOT_WIDTH,
} from '../../../../lib/plot'
import '../../curve-slider.css'

// --- Props & Emits ---
defineProps<{
  /** The run of the backlog, as a path. */
  line: string
}>()

// --- State ---

// --- Handlers ---

// --- Helpers ---
</script>

<template>
  <svg
    class="curve-slider__picture backlog__chart"
    data-backlog="picture"
    aria-hidden="true"
    :viewBox="`0 0 ${PLOT_WIDTH} ${BACKLOG_PLOT_HEIGHT}`"
  >
    <!-- The foot is nothing overdue, which is what the extent is read up from. -->
    <line
      class="curve-slider__rule"
      data-control="rule"
      :x1="PLOT_LEFT"
      :x2="PLOT_LEFT"
      :y1="BACKLOG_PLOT.top"
      :y2="BACKLOG_PLOT.foot"
    />
    <line
      class="curve-slider__rule"
      data-control="rule"
      :x1="PLOT_LEFT"
      :x2="PLOT_RIGHT"
      :y1="BACKLOG_PLOT.foot"
      :y2="BACKLOG_PLOT.foot"
    />

    <path class="backlog__line" data-backlog="line" :d="line" />
  </svg>
</template>

<style scoped>
/* The backlog is read and not dragged, so no pointer is offered over it. */
.backlog__chart {
  cursor: default;
}

.backlog__line {
  fill: none;
  stroke: var(--numen-caution-fg);
  stroke-width: 1.75;
  stroke-linecap: round;
  stroke-linejoin: round;
}
</style>
