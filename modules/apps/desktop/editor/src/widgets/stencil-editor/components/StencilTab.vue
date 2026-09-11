<script setup lang="ts">
/**
 * Displays flashcard stencil editor and handles field and face definitions.
 */
import { computed } from 'vue'
import { StencilEditor } from '@numen/ui'
import type { InsertionPoint, Half } from '@numen/ui'
import FileConflictPrompt from '../../../features/file-conflict/FileConflictPrompt.vue'
import { conflictIn } from '../../../features/file-conflict/flushing'
import type { StencilTabState } from '../types'
import { WORDS as words } from '../../../entities/deck/words'

// --- Props & Emits ---
const props = defineProps<{ state: StencilTabState }>()

// --- State ---
const stencil = computed(() => props.state.stencil.value)
const marks = computed(() => props.state.marks.value)

// --- Handlers ---
function onKeep() {
  props.state.keepMine()
}

function onTake() {
  props.state.takeFile()
}

function onAddField(name: string) {
  props.state.addField(name)
}

function onRenameField(field: string, name: string) {
  props.state.renameField(field, name)
}

function onRemoveField(field: string) {
  props.state.removeField(field)
}

function onMoveField(field: string, at: InsertionPoint) {
  props.state.moveField(field, at)
}

function onAddFace(name: string) {
  props.state.addFace(name)
}

function onRenameFace(id: string, name: string) {
  props.state.renameFace(id, name)
}

function onRemoveFace(id: string) {
  props.state.removeFace(id)
}

function onMoveFace(id: string, at: InsertionPoint) {
  props.state.moveFace(id, at)
}

function onWriteFace(id: string, half: Half, text: string) {
  props.state.writeFaceHalf(id, half, text)
}

// --- Helpers ---
</script>

<template>
  <div class="stencil-tab">
    <FileConflictPrompt
      :errorMessage="props.state.errorMessage.value"
      :conflict="conflictIn(props.state.shown.value.state)"
      :words="words"
      @keep="onKeep"
      @take="onTake"
    />

    <ul v-if="marks.whole.length" class="caution wrong" :aria-label="words.problems">
      <li v-for="(text, at) in marks.whole" :key="at">{{ text }}</li>
    </ul>

    <StencilEditor
      class="stencil-tab__body"
      :fields="stencil.fields"
      :faces="stencil.faces"
      :wrong="marks"
      :name="words.stencil"
      @add-field="onAddField"
      @rename-field="onRenameField"
      @remove-field="onRemoveField"
      @move-field="onMoveField"
      @add-face="onAddFace"
      @rename-face="onRenameFace"
      @remove-face="onRemoveFace"
      @move-face="onMoveFace"
      @write="onWriteFace"
    />
  </div>
</template>

<style scoped>
/* The editor takes what the rows above it leave, and scrolls inside itself. */
.stencil-tab {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
}

.stencil-tab__body {
  flex: 1;
  min-block-size: 0;
}

.wrong {
  margin: 0;
  padding-inline-start: 1.1rem;
  list-style: disc;
}
</style>
