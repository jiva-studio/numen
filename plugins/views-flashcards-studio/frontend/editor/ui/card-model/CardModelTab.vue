<script setup lang="ts">
/**
 * Displays flashcard card model editor and handles field and template definitions.
 */
import { computed } from 'vue'
import { type InsertionPoint, type Half } from '../../../lib/cards'
import { CardModelEditor } from '../cards/card-model-editor'
import { parseFileConflict } from '../../lib/buffer/words'
import FileConflictPrompt from '../FileConflictPrompt.vue'
import { WORDS as words } from '../../../words'
import type { CardModelTabState } from '../../model/card-model/types'

// --- Props & Emits ---
const props = defineProps<{ state: CardModelTabState }>()

// --- State ---
const cardModel = computed(() => props.state.data.cardModel.value)
const marks = computed(() => props.state.data.marks.value)
const isRead = computed(() => props.state.data.note.value.isRead)

function onKeep() {
  props.state.file.keepMine()
}

function onTake() {
  props.state.file.takeFile()
}

function onAddField(name: string) {
  props.state.fields.addField(name)
}

function onRenameField(field: string, name: string) {
  props.state.fields.renameField(field, name)
}

function onRemoveField(field: string) {
  props.state.fields.removeField(field)
}

function onMoveField(field: string, target: InsertionPoint) {
  props.state.fields.moveField(field, target)
}

function onAddTemplate(name: string) {
  props.state.templates.addTemplate(name)
}

function onRenameTemplate(id: string, name: string) {
  props.state.templates.renameTemplate(id, name)
}

function onRemoveTemplate(id: string) {
  props.state.templates.removeTemplate(id)
}

function onMoveTemplate(id: string, target: InsertionPoint) {
  props.state.templates.moveTemplate(id, target)
}

function onWriteTemplate(id: string, half: Half, text: string) {
  props.state.templates.writeTemplateHalf(id, half, text)
}

// --- Helpers ---
</script>

<template>
  <div class="card-model-tab">
    <FileConflictPrompt
      :error-message="props.state.data.errorMessage.value"
      :conflict="parseFileConflict(props.state.data.note.value.state)"
      :words="words"
      @keep="onKeep"
      @take="onTake"
    />

    <ul v-if="marks.whole.length" class="caution wrong" :aria-label="words.problems">
      <li v-for="(text, at) in marks.whole" :key="at">{{ text }}</li>
    </ul>

    <!-- The editor takes no input until the file is read. -->
    <fieldset class="card-model-tab__body" :disabled="!isRead">
      <CardModelEditor
        class="card-model-tab__editor"
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
    </fieldset>
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
  display: flex;
  flex: 1;
  flex-direction: column;
  min-block-size: 0;
  min-inline-size: 0;
  margin: 0;
  padding: 0;
  border: 0;
}

.card-model-tab__editor {
  flex: 1;
  min-block-size: 0;
}

.wrong {
  --wrong-padding-inline-start: 1.1rem;
  margin: 0;
  padding-inline-start: var(--wrong-padding-inline-start);
  list-style: disc;
}
</style>
