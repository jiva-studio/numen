<script setup lang="ts">
/**
 * What a file that moved out from under a tab puts to the person holding it.
 *
 * A file that moved past what the tab read stops saving and asks which of the
 * two is theirs. A file that is gone keeps what is on screen and offers to make
 * it again. What either is said in is the tab's own.
 */
import type { FileConflict } from '../lib/states'

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
  /** What could not be read or written, in words a person reads. */
  errorMessage: string
  /** Which conflict the file the tab holds stands in, if it stands in one. */
  conflict: FileConflict
  words: FileConflictWords
}>()

defineEmits<{
  /** The person keeps what they have written, over whatever the file holds. */
  keep: []
  /** The person takes what the file holds. */
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
