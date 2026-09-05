<script setup lang="ts">
/**
 * The questions a file puts to the person holding it open.
 *
 * A file that moved past what the tab read stops saving and asks which of the
 * two is theirs. A file that is gone keeps what is on screen and offers to make
 * it again. What either is said in is the tab's own.
 */
import type { State } from './note/tab'

/** The words the two questions are put in. */
export interface Words {
  readonly gone: string
  readonly makeAgain: string
  readonly overtaken: string
  readonly keep: string
  readonly take: string
}

defineProps<{
  /** What could not be read or written, in words a person reads. */
  saying: string
  /** The state the file the tab holds is in. */
  state: State
  words: Words
}>()

defineEmits<{
  /** The person keeps what they have written, over whatever the file holds. */
  keep: []
  /** The person takes what the file holds. */
  take: []
}>()
</script>

<template>
  <p v-if="saying" class="caution" role="alert">{{ saying }}</p>

  <p v-if="state === 'gone'" class="caution caution--answering" role="status">
    {{ words.gone }}
    <button type="button" class="answer" @click="$emit('keep')">{{ words.makeAgain }}</button>
  </p>

  <p v-if="state === 'overtaken'" class="caution caution--answering" role="status">
    {{ words.overtaken }}
    <button type="button" class="answer" @click="$emit('keep')">{{ words.keep }}</button>
    <button type="button" class="answer" @click="$emit('take')">{{ words.take }}</button>
  </p>
</template>
