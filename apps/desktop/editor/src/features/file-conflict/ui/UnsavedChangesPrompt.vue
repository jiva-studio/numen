<script setup lang="ts">
/**
 * Shown when the window is closing with unsaved notes.
 *
 * Each note offers three resolutions, and the window waits until every note is
 * resolved or deferred.
 */
import type { ConflictPrompt } from '../model/flush'

/** The labels shown for the unsaved notes. */
export interface UnsavedChangesWords {
  readonly going: string
  readonly keep: string
  readonly take: string
  readonly later: string
}

// --- Props & Emits ---
const props = defineProps<{
  conflicts: readonly ConflictPrompt[]
  /** The display name of a note, which distinguishes the notes. */
  called: (note: string) => string
  words: UnsavedChangesWords
}>()

// --- State ---

// --- Handlers ---
async function onKeep(conflict: ConflictPrompt) {
  await conflict.keep()
}

async function onTake(conflict: ConflictPrompt) {
  await conflict.take()
}

function onLater(conflict: ConflictPrompt) {
  conflict.later()
}

// --- Helpers ---
</script>

<template>
  <section v-if="props.conflicts.length" role="alertdialog" class="unsaved">
    <p class="unsaved__says">{{ props.words.going }}</p>
    <ul class="unsaved__notes">
      <li v-for="one in props.conflicts" :key="one.note" class="unsaved__note">
        <span class="unsaved__title">{{ props.called(one.note) }}</span>
        <button type="button" class="answer" @click="onKeep(one)">
          {{ props.words.keep }}
        </button>
        <button type="button" class="answer" @click="onTake(one)">
          {{ props.words.take }}
        </button>
        <button type="button" class="answer" @click="onLater(one)">
          {{ props.words.later }}
        </button>
      </li>
    </ul>
  </section>
</template>

<style scoped>
/* Overlays the content, because the prompt must be resolved before anything else. */
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
