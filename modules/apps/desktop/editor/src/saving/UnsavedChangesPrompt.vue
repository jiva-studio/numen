<script setup lang="ts">
/**
 * The window is going and these notes are not written.
 *
 * Each stands with the three ways out of it, and the window waits until every
 * one of them has been settled or put off.
 */
import type { ConflictPrompt } from './flushing'
import { WORDS as note } from '../note/words'
import { WORDS as words } from '../words'

const props = defineProps<{
  conflicts: readonly ConflictPrompt[]
  /** What each note is called, for a person to tell them apart by. */
  called: (note: string) => string
}>()
</script>

<template>
  <section v-if="props.conflicts.length" role="alertdialog" class="unsaved">
    <p class="unsaved__says">{{ words.going }}</p>
    <ul class="unsaved__notes">
      <li v-for="one in props.conflicts" :key="one.note" class="unsaved__note">
        <span class="unsaved__title">{{ props.called(one.note) }}</span>
        <button type="button" class="answer" @click="void one.keep()">
          {{ note.keep }}
        </button>
        <button type="button" class="answer" @click="void one.take()">
          {{ note.take }}
        </button>
        <button type="button" class="answer" @click="one.later()">
          {{ words.later }}
        </button>
      </li>
    </ul>
  </section>
</template>

<style scoped>
/* It sits over the work because nothing else the person does can end it. */
.unsaved {
  position: fixed;
  inset-block-end: 1rem;
  inset-inline: 1rem;
  z-index: var(--numen-lift-unsaved);
  padding: 0.8rem 1rem;
  border-radius: var(--numen-radius);
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  box-shadow: var(--numen-panel-shadow);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
}

.unsaved__says {
  margin: 0 0 0.5rem;
}

.unsaved__notes {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.unsaved__note {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.unsaved__title {
  flex: 1;
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

</style>
