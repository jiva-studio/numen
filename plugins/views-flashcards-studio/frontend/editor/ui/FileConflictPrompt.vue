<script setup lang="ts">
/**
 * The prompt for a file that changed or disappeared under an open tab.
 *
 * A file that changed after the tab read it stops saving and prompts for which of the
 * two versions to retain. A file that is gone keeps the content on screen and
 * offers to recreate it. The wording of both is supplied by the tab.
 */
import { Button } from '@numen/ui'
import type { FileConflict } from '@numen/plugin-sdk'

// --- Props & Emits ---
/** The words the two conflicts are put in. */
interface FileConflictWords {
  readonly gone: string
  readonly makeAgain: string
  readonly stale: string
  readonly keep: string
  readonly take: string
}

defineProps<{
  errorMessage: string
  conflict: FileConflict
  words: FileConflictWords
}>()

defineEmits<{
  keep: []
  take: []
}>()
</script>

<template>
  <p v-if="errorMessage" class="caution" role="alert">{{ errorMessage }}</p>

  <p v-if="conflict === 'gone'" class="caution caution--conflict" role="status">
    {{ words.gone }}
    <Button size="small" @click="$emit('keep')">{{ words.makeAgain }}</Button>
  </p>

  <p v-if="conflict === 'stale'" class="caution caution--conflict" role="status">
    {{ words.stale }}
    <Button size="small" @click="$emit('keep')">{{ words.keep }}</Button>
    <Button variant="outline" size="small" @click="$emit('take')">{{ words.take }}</Button>
  </p>
</template>
