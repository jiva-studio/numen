<script setup lang="ts">
/**
 * Displays flashcard card model editor and handles field and template definitions.
 */
import { computed } from 'vue'
import { CardModelEditor, type InsertionPoint, type Half } from '../../ui/cards'
import { parseFileConflict } from '../../buffer'
import FileConflictPrompt from '../../ui/FileConflictPrompt.vue'
import type { CardModelTabState } from '../types'
import { WORDS as words } from '../../cards'

// --- Props & Emits ---
const props = defineProps<{ state: CardModelTabState }>()

// --- State ---
const cardModel = computed(() => props.state.cardModel.value)
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

function onMoveField(field: string, target: InsertionPoint) {
  props.state.moveField(field, target)
}

function onAddTemplate(name: string) {
  props.state.addTemplate(name)
}

function onRenameTemplate(id: string, name: string) {
  props.state.renameTemplate(id, name)
}

function onRemoveTemplate(id: string) {
  props.state.removeTemplate(id)
}

function onMoveTemplate(id: string, target: InsertionPoint) {
  props.state.moveTemplate(id, target)
}

function onWriteTemplate(id: string, half: Half, text: string) {
  props.state.writeTemplateHalf(id, half, text)
}

// --- Helpers ---
</script>

<template>
  <div class="card-model-tab">
    <FileConflictPrompt
      :error-message="props.state.errorMessage.value"
      :conflict="parseFileConflict(props.state.note.value.state)"
      :words="words"
      @keep="onKeep"
      @take="onTake"
    />

    <ul v-if="marks.whole.length" class="caution wrong" :aria-label="words.problems">
      <li v-for="(text, at) in marks.whole" :key="at">{{ text }}</li>
    </ul>

    <CardModelEditor
      class="card-model-tab__body"
      :fields="cardModel.fields"
      :templates="cardModel.templates"
      :wrong="marks"
      :name="words.cardModel"
      @add-field="onAddField"
      @rename-field="onRenameField"
      @remove-field="onRemoveField"
      @move-field="onMoveField"
      @add-template="onAddTemplate"
      @rename-template="onRenameTemplate"
      @remove-template="onRemoveTemplate"
      @move-template="onMoveTemplate"
      @write="onWriteTemplate"
    />
  </div>
</template>

<style scoped>
/* The editor takes what the rows above it leave, and scrolls inside itself. */
.card-model-tab {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
}

.card-model-tab__body {
  flex: 1;
  min-block-size: 0;
}

.wrong {
  margin: 0;
  padding-inline-start: 1.1rem;
  list-style: disc;
}
</style>
