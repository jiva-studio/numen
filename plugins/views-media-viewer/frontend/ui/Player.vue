<script setup lang="ts">
/**
 * Playback controls: start, stop, show the current position and seek.
 *
 * It does not play audio itself. Playback runs outside the rendered view, and
 * this component only emits the user's actions.
 */
import { computed, ref } from 'vue'
import { Pause, Play } from '@lucide/vue'
import { clock } from '@numen/ui'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    at: number
    length: number
    isPlaying: boolean
    label?: string
    play?: string
    pause?: string
  }>(),
  { label: 'The recording', play: 'Play', pause: 'Pause' },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  play: []
  pause: []
  seek: [ms: number]
}>()

/* --------------------------------- State ---------------------------------- */
/** The duration is at least the furthest position reached. */
const runs = computed(() => Math.max(props.length, props.at, 0))

/**
 * While the bar is dragged, the dragged position is shown in place of the
 * playback position, until the seek takes effect.
 */
const dragging = ref<number | null>(null)

const position = computed(() => dragging.value ?? Math.min(props.at, runs.value))

/* -------------------------------- Handlers -------------------------------- */
function onSeekInput(event: Event) {
  const ms = Number((event.target as HTMLInputElement).value)
  dragging.value = ms
  emit('seek', ms)
}

function onSeekChange() {
  dragging.value = null
}
</script>

<template>
  <div class="player" role="group" :aria-label="props.label">
    <button
      type="button"
      class="player__sound"
      :aria-label="props.isPlaying ? props.pause : props.play"
      :aria-pressed="props.isPlaying ? 'true' : 'false'"
      @click="props.isPlaying ? emit('pause') : emit('play')"
    >
      <Pause v-if="props.isPlaying" />
      <Play v-else />
    </button>

    <span class="player__at">{{ clock(position) }}</span>

    <input
      type="range"
      class="player__bar"
      min="0"
      :max="runs"
      :disabled="runs === 0"
      step="1000"
      :value="position"
      :aria-label="props.label"
      :aria-valuetext="clock(position)"
      @input="onSeekInput"
      @change="onSeekChange"
    />

    <span class="player__at">{{ clock(runs) }}</span>
  </div>
</template>

<style scoped>
/* The controls are flat and inherit the surrounding background. */
.player {
  --player-gap: 0.5rem;
  display: flex;
  align-items: center;
  gap: var(--player-gap);
  inline-size: 100%;
}

/* A disc one row across, which is the height every control is drawn at. */
.player__sound {
  display: grid;
  place-items: center;
  inline-size: var(--numen-action-size);
  block-size: var(--numen-action-size);
  flex: none;
  padding: 0;
  border: 0;
  border-radius: var(--numen-radius-pill);
  background: none;
  color: inherit;
  cursor: pointer;
}

.player__sound:hover {
  background: var(--numen-field-bg);
}

.player__sound svg {
  --sound-inline-size: 0.9rem;
  --sound-block-size: 0.9rem;
  inline-size: var(--sound-inline-size);
  block-size: var(--sound-block-size);
  fill: currentColor;
}

/* The two times share the width of the longer one, so the bar does not
   move as the seconds count up. */
.player__at {
  --at-min-inline-size: 2.4em;
  flex: none;
  min-inline-size: var(--at-min-inline-size);
  color: var(--numen-hushed);
  font-variant-numeric: tabular-nums;
  font-size: var(--numen-text-1);
  text-align: center;
}

.player__bar {
  flex: 1;
  min-inline-size: 0;
  accent-color: var(--numen-accent);
}
</style>
