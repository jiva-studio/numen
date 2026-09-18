<script setup lang="ts">
/**
 * The settings file, whole, in the editor this window edits everything in.
 *
 * It is written as JSON, so it is read as JSON and set in the face code is set
 * in. What is typed is kept the way a note is kept, and a file the settings
 * cannot be read out of is refused with what is wrong with it. A file that
 * moved past what was read puts the person the two answers a note puts.
 */
import { Editor, Waiting } from '@numen/ui'
import type { TextEditorTabState } from '../model/useTextEditor'
import { WORDS as words } from '../words'

// --- Props & Emits ---
const props = defineProps<{ state: TextEditorTabState }>()

// --- State ---
// The tab's state outlives this component, so what it holds is bound once here
// and the template unwraps it.
const { errorMessage, isStale, isRead, text } = props.state

// --- Handlers ---
function onKeep() {
  void props.state.keep()
}

function onTake() {
  void props.state.take()
}

function onUpdateModelValue(text: string) {
  props.state.type(text)
}

function onSave() {
  void props.state.save()
}

// --- Helpers ---
</script>

<template>
  <div class="settings-file">
    <p v-if="errorMessage" role="alert" class="settings-file__wrong">
      {{ errorMessage }}
    </p>

    <p v-if="isStale" class="caution caution--conflict" role="status">
      {{ words.stale }}
      <button type="button" class="answer" @click="onKeep">{{ words.keep }}</button>
      <button type="button" class="answer" @click="onTake">{{ words.take }}</button>
    </p>

    <Editor
      v-if="isRead"
      :model-value="text"
      :is-live="false"
      language="json"
      class="settings-file__editor"
      :aria-label="words.called"
      @update:model-value="onUpdateModelValue"
      @save="onSave"
    />
    <Waiting v-else :label="words.loading" />
  </div>
</template>

<style scoped>
/* The file fills the pane it is in: the editor scrolls, and the line saying
   something is wrong stays where it is. */
.settings-file {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
}

/* What is wrong with what was typed, which is the one thing to catch the eye. */
.settings-file__wrong {
  flex: none;
  margin: 0;
  padding: var(--numen-inset) var(--numen-gutter);
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  overflow-wrap: break-word;
}

.settings-file__editor {
  flex: 1;
  min-block-size: 0;
  overflow: auto;
}
</style>
