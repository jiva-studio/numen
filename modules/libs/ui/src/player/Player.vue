<script setup lang="ts">
/**
 * The controls a sound is played by: start it, stop it, see where it stands
 * and put it somewhere else.
 *
 * It plays nothing of its own. What plays lives where being drawn cannot
 * reach it, and this only says what a person did.
 */
import { computed } from 'vue'
import { Pause, Play } from '@lucide/vue'
import { clock } from './clock'

const props = withDefaults(
  defineProps<{
    /** Where the sound stands, in milliseconds. */
    at: number
    /** How long it runs, in milliseconds. */
    length: number
    playing: boolean
    /** What the controls are called, where a person is read to. */
    label?: string
    play?: string
    pause?: string
  }>(),
  { label: 'The recording', play: 'Play', pause: 'Pause' },
)

const emit = defineEmits<{
  play: []
  pause: []
  seek: [ms: number]
}>()

/** A sound is at least as long as the furthest anything has stood in it. */
const runs = computed(() => Math.max(props.length, props.at, 0))

const sought = (event: Event) => emit('seek', Number((event.target as HTMLInputElement).value))
</script>

<template>
  <div class="player" role="group" :aria-label="props.label">
    <button
      type="button"
      class="player__sound"
      :aria-label="props.playing ? props.pause : props.play"
      :aria-pressed="props.playing ? 'true' : 'false'"
      @click="props.playing ? emit('pause') : emit('play')"
    >
      <Pause v-if="props.playing" />
      <Play v-else />
    </button>

    <span class="player__at">{{ clock(props.at) }}</span>

    <input
      type="range"
      class="player__bar"
      min="0"
      :max="runs"
      step="1000"
      :value="Math.min(props.at, runs)"
      :aria-label="props.label"
      :aria-valuetext="clock(props.at)"
      @input="sought"
    />

    <span class="player__at">{{ clock(runs) }}</span>
  </div>
</template>

<style scoped>
/* The controls are drawn flat, on whatever ground they stand on. */
.player {
  display: flex;
  align-items: center;
  gap: 0.5rem;
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
  inline-size: 0.9rem;
  block-size: 0.9rem;
  fill: currentColor;
}

/* The two times take the room the longer of them needs, so the bar does not
   move as the seconds count up. */
.player__at {
  flex: none;
  min-inline-size: 4ch;
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
