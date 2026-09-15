<script setup lang="ts">
/**
 * Displays flashcard review preset settings and target curve slider.
 */
import PresetSettings from './preset-settings/PresetSettings.vue'
import { PresetGoal } from './preset-goal'
import type { PresetTabState } from '../types'
import { WORDS as words } from '../words'

// --- Props & Emits ---
const props = defineProps<{ state: PresetTabState }>()

// --- State ---
// The tab's state outlives this component, so what it holds is bound once here
// and the template unwraps it.
const { hasChanged, problems, errorMessage } = props.state

// --- Handlers ---
function onRefresh() {
  props.state.reload()
}

// --- Helpers ---
</script>

<template>
  <div class="preset">
    <!-- Reading the file again is the way out of anything the tab has to say,
         and it waits on nothing in the vault. -->
    <p v-if="errorMessage" role="alert" class="preset__warning preset__answering">
      {{ errorMessage }}
      <button type="button" class="answer" @click="onRefresh">
        {{ words.reads }}
      </button>
    </p>

    <p v-if="hasChanged" role="status" class="preset__warning preset__answering">
      {{ words.changed }}
      <button type="button" class="answer" @click="onRefresh">
        {{ words.reads }}
      </button>
    </p>

    <ul
      v-if="problems.length"
      class="preset__warning preset__problems"
      :aria-label="words.problems"
    >
      <li v-for="(text, at) in problems" :key="at">{{ text }}</li>
    </ul>

    <div class="preset__page">
      <div class="preset__column">
        <PresetGoal :state="props.state" />

        <PresetSettings :state="props.state" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.preset {
  /* The measure a preset is read at, which the goal and the settings share. */
  --preset-measure: 46rem;
  /* The step every gap down the column is set by. */
  --preset-step: 1.25rem;
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
}

.preset__page {
  flex: 1;
  min-block-size: 0;
  overflow-y: auto;
  padding: var(--numen-gutter);
}

/*
 * The column the tab is read in, centred in whatever room the pane has. Its
 * two edges are the lines this tab is read against: a block that draws a box
 * puts the box's edge on them, and a block that draws none puts its text
 * there. Nothing in the column is inset from them, and nothing is pulled out.
 */
.preset__column {
  display: flex;
  flex-direction: column;
  gap: var(--preset-step);
  inline-size: 100%;
  max-inline-size: var(--preset-measure);
  margin-inline: auto;
}

.preset__warning {
  margin: 0;
  padding: var(--numen-inset) var(--numen-gutter);
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  overflow-wrap: break-word;
}

.preset__answering {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0 var(--numen-panel-gap);
}

.preset__problems {
  padding-inline-start: calc(var(--numen-gutter) * 2);
  list-style: disc;
}
</style>
