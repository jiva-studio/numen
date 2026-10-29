<script setup lang="ts">
/**
 * The prompt for a file that changed or disappeared under an open tab.
 *
 * A file that changed after the tab read it stops saving and prompts for which of the
 * two versions to retain. A file that is gone keeps the content on screen and
 * offers to recreate it. The wording of both is supplied by the tab.
 */
import type { FileConflict } from '@numen/plugin-sdk'

// --- Props & Emits ---
/** The words the two conflicts are put in. */
export interface FileConflictWords {
  readonly gone: string
  readonly makeAgain: string
  readonly stale: string
  readonly keep: string
  readonly take: string
}

defineProps<{
  /** The read or write error, as a message for the user. */
  errorMessage: string
  conflict: FileConflict
  words: FileConflictWords
}>()

defineEmits<{
  /** The user keeps their own content over whatever the file holds. */
  keep: []
  /** The user takes the content the file holds. */
  take: []
}>()
</script>

<template>
  <p v-if="errorMessage" class="caution" role="alert">{{ errorMessage }}</p>

  <p v-if="conflict === 'gone'" class="caution caution--conflict" role="status">
    {{ words.gone }}
    <button type="button" class="answer" @click="$emit('keep')">{{ words.makeAgain }}</button>
  </p>

  <p v-if="conflict === 'stale'" class="caution caution--conflict" role="status">
    {{ words.stale }}
    <button type="button" class="answer" @click="$emit('keep')">{{ words.keep }}</button>
    <button type="button" class="answer" @click="$emit('take')">{{ words.take }}</button>
  </p>
</template>
