<script setup lang="ts">
/**
 * A stencil tab: the fields it names, the faces that show them, and the
 * questions the file puts.
 *
 * What is wrong with the faces and the fields is handed to the editor, which
 * stands a face's mark under that face's name and a field's under that field's
 * row. What stands against neither is said above the editor.
 */
import { computed } from 'vue'
import { StencilEditor } from '@numen/ui'
import type { InsertionPoint, Half } from '@numen/ui'
import FileConflictPrompt from '../shared/saving/FileConflictPrompt.vue'
import { conflictIn } from '../shared/saving/flushing'
import type { StencilTabState } from './stencilTabs'
import { WORDS as words } from '../shared/flashcards/words'

// --- Props & Emits ---
const props = defineProps<{ state: StencilTabState }>()

// --- State ---
const stencil = computed(() => props.state.stencil.value)
const marks = computed(() => props.state.marks.value)

// --- Handlers ---
function onKeep() {
  props.state.keep()
}

function onTake() {
  props.state.take()
}

function onAddField(name: string) {
  props.state.addsField(name)
}

function onRenameField(field: string, name: string) {
  props.state.namesField(field, name)
}

function onRemoveField(field: string) {
  props.state.removesField(field)
}

function onMoveField(field: string, at: InsertionPoint) {
  props.state.movesField(field, at)
}

function onAddFace(name: string) {
  props.state.addsFace(name)
}

function onRenameFace(id: string, name: string) {
  props.state.namesFace(id, name)
}

function onRemoveFace(id: string) {
  props.state.removesFace(id)
}

function onMoveFace(id: string, at: InsertionPoint) {
  props.state.movesFace(id, at)
}

function onWriteFace(id: string, half: Half, text: string) {
  props.state.writes(id, half, text)
}

// --- Helpers ---
</script>

<template>
  <div class="stencil-tab">
    <FileConflictPrompt
      :saying="props.state.saying.value"
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
