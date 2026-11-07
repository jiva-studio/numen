<script setup lang="ts">
/**
 * The transcript's words as plain text, edited in place.
 */
import { onBeforeUnmount, onMounted, useTemplateRef, watch } from 'vue'
import type { Extension } from '@codemirror/state'
import { createTranscriptView, type TranscriptView } from './lib/transcriptView'

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
let view: TranscriptView | null = null

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  if (!host.value) return
  view = createTranscriptView({
    parent: host.value,
    text: props.modelValue,
    isEditable: props.isEditable,
    label: props.label,
    extensions: props.extensions,
    onChange: (text) => emit('update:modelValue', text),
    onSave: () => emit('save'),
  })
})

onBeforeUnmount(() => {
  view?.destroy()
  view = null
})

watch(
  () => props.modelValue,
  (text) => view?.setText(text),
)
watch(
  () => props.isEditable,
  (isEditable) => view?.setEditable(isEditable),
)
watch(
  () => props.extensions,
  (extensions) => view?.setExtensions(extensions),
)
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
