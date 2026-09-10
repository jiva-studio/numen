<script setup lang="ts">
/**
 * Displays flashcard stencil editor and handles field and face definitions.
 */
import { computed } from 'vue'
import { StencilEditor } from '@numen/ui'
import type { InsertionPoint, Half } from '@numen/ui'
import FileConflictPrompt from '../shared/saving/FileConflictPrompt.vue'
import { conflictIn } from '../shared/saving/flushing'
import type { StencilTabState } from './types'
import { WORDS as words } from '../shared/flashcards/words'

// --- Props & Emits ---
const props = defineProps<{ state: StencilTabState }>()

// --- State ---
const stencil = computed(() => props.state.stencil.value)
const marks = computed(() => props.state.marks.value)

// --- Handlers ---
function onKeep() {
  const keep = props.state.keepMine ?? props.state.keep
  keep()
}

function onTake() {
  const take = props.state.takeFile ?? props.state.take
  take()
}

function onAddField(name: string) {
  const add = props.state.addField ?? props.state.addsField
  add(name)
}

function onRenameField(field: string, name: string) {
  const rename = props.state.renameField ?? props.state.namesField
  rename(field, name)
}

function onRemoveField(field: string) {
  const remove = props.state.removeField ?? props.state.removesField
  remove(field)
}

function onMoveField(field: string, at: InsertionPoint) {
  const move = props.state.moveField ?? props.state.movesField
  move(field, at)
}

function onAddFace(name: string) {
  const add = props.state.addFace ?? props.state.addsFace
  add(name)
}

function onRenameFace(id: string, name: string) {
  const rename = props.state.renameFace ?? props.state.namesFace
  rename(id, name)
}

function onRemoveFace(id: string) {
  const remove = props.state.removeFace ?? props.state.removesFace
  remove(id)
}

function onMoveFace(id: string, at: InsertionPoint) {
  const move = props.state.moveFace ?? props.state.movesFace
  move(id, at)
}

function onWriteFace(id: string, half: Half, text: string) {
  const write = props.state.writeFaceHalf ?? props.state.writes
  write(id, half, text)
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
