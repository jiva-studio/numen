<script setup lang="ts">
/**
 * Shown when the window is closing with unsaved notes.
 *
 * Each note offers three resolutions, and the window waits until every note is
 * resolved or deferred.
 */
import { Button } from '@numen/ui'
import type { ConflictPrompt } from '../model/flush'
import { keepConflict, postponeConflict, takeConflict } from '../model/resolveConflict'

/** The labels shown for the unsaved notes. */
interface UnsavedChangesWords {
  readonly going: string
  readonly keep: string
  readonly take: string
  readonly later: string
}

// --- Props & Emits ---
const props = defineProps<{
  conflicts: readonly ConflictPrompt[]
  called: (note: string) => string
  words: UnsavedChangesWords
}>()

// --- State ---
const onKeep = keepConflict
const onTake = takeConflict
const onLater = postponeConflict

// --- Helpers ---
</script>

<template>
  <section v-if="props.conflicts.length" role="alertdialog" class="unsaved">
    <p class="unsaved__says">{{ props.words.going }}</p>
    <ul class="unsaved__notes">
      <li v-for="one in props.conflicts" :key="one.note" class="unsaved__note">
        <span class="unsaved__title">{{ props.called(one.note) }}</span>
        <Button size="small" @click="onKeep(one)">
          {{ props.words.keep }}
        </Button>
        <Button variant="outline" size="small" @click="onTake(one)">
          {{ props.words.take }}
        </Button>
        <Button variant="ghost" size="small" @click="onLater(one)">
          {{ props.words.later }}
        </Button>
      </li>
    </ul>
  </section>
</template>

<style scoped>
/* Overlays the content, because the prompt must be resolved before anything else. */
.unsaved {
  --inset-end: 1rem;
  --inset-inline: 1rem;
  --pad: 0.8rem 1rem;
  position: fixed;
  inset-block-end: var(--inset-end);
  inset-inline: var(--inset-inline);
  z-index: var(--numen-lift-unsaved);
  padding: var(--pad);
  border-radius: var(--numen-radius);
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  box-shadow: var(--numen-panel-shadow);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
}

.unsaved__says {
  --margin: 0 0 0.5rem;
  margin: var(--margin);
}

.unsaved__notes {
  --gap: 0.35rem;
  display: flex;
  flex-direction: column;
  gap: var(--gap);
  margin: 0;
  padding: 0;
  list-style: none;
}

.unsaved__note {
  --gap: 0.5rem;
  display: flex;
  align-items: center;
  gap: var(--gap);
}

.unsaved__title {
  flex: 1;
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
