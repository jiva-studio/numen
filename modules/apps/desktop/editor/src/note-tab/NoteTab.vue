<script setup lang="ts">
/**
 * A note tab: the text, and the questions the file puts to the person.
 *
 * A note whose file moved past what was read stops saving and asks which of
 * the two is theirs. A note whose file is gone keeps what is on screen and
 * offers to make it again.
 *
 * A note pointing at an address is a tab of its own: what is at the address is
 * played and what was fetched for it is written, and none of that is what an
 * ordinary note is.
 */
import { watch } from 'vue'
import { Editor } from '@numen/ui'
import FileConflictPrompt from '../shared/saving/FileConflictPrompt.vue'
import { conflictIn } from '../shared/saving/flushing'
import { WORDS as words } from './words'
import type { NoteTabState } from './kind'

// --- Props & Emits ---
const props = defineProps<{ state: NoteTabState }>()

// --- State ---
// The prose of a note arrives after the tab it is drawn in. The editor takes
// the keyboard it is owed once there are lines for a caret to stand on.
watch(
  () => props.state.shown.value.body,
  (body, was) => {
    if (!was && body) props.state.measure()
  },
)

// --- Handlers ---
function onKeep() {
  props.state.keep()
}

function onTake() {
  props.state.take()
}

function onSetEditor(editor: unknown) {
  props.state.drew(editor)
}

function onUpdateModelValue(body: string) {
  props.state.typed(body)
}

function onSave() {
  props.state.save()
}

function onOpen(address: string) {
  props.state.follows(address)
}

// --- Helpers ---
</script>

<template>
  <div class="note">
    <FileConflictPrompt
      :saying="props.state.saying.value"
      :conflict="conflictIn(props.state.shown.value.state)"
      :words="words"
      @keep="onKeep"
      @take="onTake"
    />

    <Editor
      :ref="onSetEditor"
      :model-value="props.state.shown.value.body"
      :change="props.state.change.value"
      class="note__text"
      @update:model-value="onUpdateModelValue"
      @save="onSave"
      @open="onOpen"
    />
  </div>
</template>

<style scoped>
/* A note fills the pane it is in: the editor scrolls, and the line it says
   something is wrong on stays where it is. */
.note {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
}

.note__text {
  flex: 1;
  min-block-size: 0;
}
</style>
