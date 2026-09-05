<script setup lang="ts">
/**
 * The window is going and these notes are not written.
 *
 * Each stands with the three ways out of it, and the window waits until every
 * one of them has been answered or put off.
 */
import type { DrawnQuestion } from './leaving'
import { WORDS as note } from './note/words'
import { WORDS as words } from './words'

const props = defineProps<{
  questions: readonly DrawnQuestion[]
  /** What each note is called, for a person to tell them apart by. */
  called: (note: string) => string
}>()
</script>

<template>
  <section v-if="props.questions.length" role="alertdialog" class="leaving">
    <p class="leaving__says">{{ words.going }}</p>
    <ul class="leaving__notes">
      <li v-for="one in props.questions" :key="one.note" class="leaving__note">
        <span class="leaving__title">{{ props.called(one.note) }}</span>
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
.leaving {
  position: fixed;
  inset-block-end: 1rem;
  inset-inline: 1rem;
  z-index: var(--numen-lift-leaving);
  padding: 0.8rem 1rem;
  border-radius: var(--numen-radius);
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  box-shadow: var(--numen-panel-shadow);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
}

.leaving__says {
  margin: 0 0 0.5rem;
}

.leaving__notes {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.leaving__note {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.leaving__title {
  flex: 1;
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

</style>
