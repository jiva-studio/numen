<script setup lang="ts">
/**
 * Displays note editor and handles file conflict resolution prompts.
 */
import { watch } from 'vue'
import { Editor } from '@numen/ui'
import { conflictIn, FileConflictPrompt } from '@/features/file-conflict'
import { WORDS as words } from '@/entities/note'
import type { NoteTabState } from '../types'

// --- Props & Emits ---
const props = defineProps<{ state: NoteTabState }>()

// --- State ---
watch(
  () => props.state.shown.value.body,
  (body, was) => {
    if (!was && body) props.state.measure()
  },
)

// --- Handlers ---
function onKeep() {
  props.state.keepMine()
}

function onTake() {
  props.state.takeFile()
}

function onSetEditor(editor: unknown) {
  props.state.setEditor(editor)
}

function onUpdateModelValue(body: string) {
  props.state.updateBody(body)
}

function onSave() {
  props.state.save()
}

function onOpen(url: string) {
  props.state.followLink(url)
}

// --- Helpers ---
</script>

<template>
  <div class="note">
    <FileConflictPrompt
      :errorMessage="props.state.errorMessage.value"
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
