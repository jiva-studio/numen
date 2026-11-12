<script setup lang="ts">
/**
 * The transcript's words as plain text, edited in place.
 */
import { useTemplateRef } from 'vue'
import type { Extension } from '@codemirror/state'
import { useTranscriptEditor } from '../model/useTranscriptEditor'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  modelValue: string
  isEditable: boolean
  label: string
  extensions: Extension
}>()

const emit = defineEmits<{
  'update:modelValue': [text: string]
  save: []
}>()

/* --------------------------------- State ---------------------------------- */
const host = useTemplateRef<HTMLElement>('host')
useTranscriptEditor(host, props, {
  change: (text) => emit('update:modelValue', text),
  save: () => emit('save'),
})
</script>

<template>
  <div ref="host" class="transcript-text" />
</template>

<style scoped>
.transcript-text {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  overflow: hidden;
  user-select: text;
  -webkit-user-select: text;
}

.transcript-text :deep(.cm-editor) {
  flex: 1 1 auto;
  min-block-size: 0;
  height: 100%;
}

.transcript-text :deep(.cm-scroller) {
  overflow-y: auto;
}
</style>
