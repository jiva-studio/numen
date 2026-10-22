<script setup lang="ts">
/**
 * Standalone Markdown and plain text editor tab component using CodeMirror 6.
 */
import { computed, useTemplateRef } from 'vue'
import { Editor } from '@numen/ui'
import type { Extension } from '@codemirror/state'
import { useMarkdownDocument, type EditorHandle } from './useMarkdownDocument'

/* --------------------------------- Props ---------------------------------- */
interface Props {
  readonly modelValue?: string
  readonly path?: string
  readonly filePath?: string
  readonly errorMessage?: string
  readonly isReadOnly?: boolean
  readonly extensions?: Extension | readonly Extension[]
  readonly state?: Record<string, unknown>
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  path: '',
  filePath: '',
  errorMessage: '',
  isReadOnly: false,
})

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  'update:modelValue': [value: string]
  save: []
  open: [url: string]
  change: [value: string]
}>()

/* --------------------------------- State ---------------------------------- */
const editorRef = useTemplateRef<EditorHandle>('editorRef')

const {
  docContent,
  isDirty,
  frontmatter,
  isLoading,
  displayErrorMessage,
  onUpdateModelValue,
  onSave,
  setContent,
} = useMarkdownDocument(props, emit, editorRef)

const editorLanguage = computed(() => {
  const p =
    props.filePath ||
    props.path ||
    (props.state?.path as string) ||
    (props.state?.file as string) ||
    (props.state?.at as string) ||
    ''
  if (p.endsWith('.txt')) return 'plaintext'
  if (p.endsWith('.json')) return 'json'
  return 'markdown'
})

/* -------------------------------- Handlers -------------------------------- */
function onOpen(url: string) {
  emit('open', url)
}

defineExpose({
  focus: () => editorRef.value?.focus?.() ?? false,
  measure: () => editorRef.value?.measure?.(),
  reveal: (line: number) => editorRef.value?.reveal?.(line) ?? false,
  isDirty,
  frontmatter,
  getContent: () => docContent.value,
  setContent,
})
</script>

<template>
  <div class="markdown-note-editor note">
    <p v-if="displayErrorMessage" class="caution" role="alert">{{ displayErrorMessage }}</p>
    <Editor
      ref="editorRef"
      :model-value="docContent"
      :language="editorLanguage"
      is-live
      :readonly="isReadOnly || isLoading"
      v-bind="props.extensions ? { extensions: props.extensions as Extension } : {}"
      class="markdown-note-editor__pane note__text"
      @update:model-value="onUpdateModelValue"
      @save="onSave"
      @open="onOpen"
    />
  </div>
</template>

<style scoped>
.markdown-note-editor {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
}

.caution {
  margin: 0;
  padding: 0.5rem 1rem;
  background-color: var(--numen-warning-subtle, #fef3c7);
  color: var(--numen-warning, #b45309);
  font-size: 0.875rem;
}

.markdown-note-editor__pane {
  flex: 1;
  min-block-size: 0;
  overflow: auto;
}
</style>
