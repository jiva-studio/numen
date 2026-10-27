<script setup lang="ts">
/**
 * Standalone Markdown and plain text editor tab component using CodeMirror 6.
 */
import { computed } from 'vue'
import { Editor } from '@numen/ui'
import type { Extension } from '@codemirror/state'

/* --------------------------------- Props ---------------------------------- */
interface Props {
  readonly modelValue?: string
  readonly filePath?: string
  readonly isReadOnly?: boolean
  readonly extensions?: Extension | readonly Extension[]
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  filePath: '',
  isReadOnly: false,
})

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  'update:modelValue': [value: string]
  save: []
  open: [url: string]
}>()

/* --------------------------------- State ---------------------------------- */
const editorLanguage = computed(() => {
  if (props.filePath.endsWith('.txt')) {
    return 'plaintext'
  }
  return 'markdown'
})

/* -------------------------------- Handlers -------------------------------- */
function onUpdateModelValue(value: string) {
  emit('update:modelValue', value)
}

function onSave() {
  emit('save')
}

function onOpen(url: string) {
  emit('open', url)
}
</script>

<template>
  <div class="markdown-editor">
    <Editor
      :model-value="props.modelValue"
      :language="editorLanguage"
      is-live
      v-bind="props.extensions ? { extensions: props.extensions as Extension } : {}"
      class="markdown-editor__pane"
      @update:model-value="onUpdateModelValue"
      @save="onSave"
      @open="onOpen"
    />
  </div>
</template>

<style scoped>
.markdown-editor {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
}

.markdown-editor__pane {
  flex: 1;
  min-block-size: 0;
  overflow: auto;
}
</style>
