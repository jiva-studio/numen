<script setup lang="ts">
/**
 * The words of a recording, one line to a moment. The line being spoken is marked, and a press on a line
 * goes to its moment.
 */
import { useTemplateRef } from 'vue'
import { clock } from '@numen/ui'
import type { Cue } from '../lib/cues'
import { useFollowedLine } from '../model/useFollowedLine'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ cues: readonly Cue[]; active: number; isFollowing: boolean }>()

const emit = defineEmits<{ seek: [ms: number] }>()

/* --------------------------------- State ---------------------------------- */
const list = useTemplateRef<HTMLElement>('list')

/* --------------------------------- Hooks ---------------------------------- */
useFollowedLine(
  list,
  () => props.active,
  () => props.isFollowing,
)
</script>

<template>
  <ol ref="list" class="transcript">
    <li
      v-for="(cue, index) in cues"
      :key="cue.from"
      :data-active="index === active ? '' : undefined"
    >
      <button type="button" class="transcript__line" @click="emit('seek', cue.from)">
        <time class="transcript__time">{{ clock(cue.from) }}</time>
        <span class="transcript__text">{{ cue.text }}</span>
      </button>
    </li>
  </ol>
</template>

<style scoped>
.transcript {
  flex: 1;
  min-block-size: 0;
  margin: 0;
  padding: 0.75rem var(--numen-gutter);
  overflow-y: auto;
  list-style: none;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-reading-size);
  line-height: var(--numen-line-height);
  color: var(--numen-ink);
  user-select: text;
  -webkit-user-select: text;
  scrollbar-width: none;
}

.transcript::-webkit-scrollbar {
  display: none;
}

.transcript__line {
  display: grid;
  grid-template-columns: 3.5rem 1fr;
  inline-size: 100%;
  padding: 0.15rem 0;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  text-align: start;
  cursor: pointer;
}

.transcript__time {
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  font-variant-numeric: tabular-nums;
  padding-block-start: 0.1rem;
}

[data-active] .transcript__line,
[data-active] .transcript__time {
  color: var(--numen-accent);
}
</style>
