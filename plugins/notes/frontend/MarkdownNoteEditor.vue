<script setup lang="ts">
/**
 * Standalone Markdown and plain text editor tab component using CodeMirror 6.
 */
import { computed, ref, useTemplateRef, watch } from 'vue'
import { Editor } from './editor'
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
const isDismissed = ref(false)

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

watch(displayErrorMessage, () => {
  isDismissed.value = false
})

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
function onDismiss() {
  isDismissed.value = true
}

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
    <div v-if="displayErrorMessage && !isDismissed" class="caution" role="alert">
      <span class="caution__text">{{ displayErrorMessage }}</span>
      <button type="button" class="caution__dismiss" aria-label="Dismiss banner" @click="onDismiss">
        &times;
      </button>
    </div>
    <Editor
      ref="editorRef"
      :model-value="docContent"
      :language="editorLanguage"
      is-live
      :is-readonly="isReadOnly || isLoading"
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
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  margin: 0;
  padding: 0.4rem 1rem;
  font-family: var(--numen-font-sans, inherit);
  font-size: var(--numen-text-2, 0.875rem);
  background-color: var(--numen-caution-bg, light-dark(#f7e9c8, #42321c));
  color: var(--numen-caution-fg, light-dark(#8a5410, #eec26a));
  overflow-wrap: break-word;
}

.caution__text {
  flex: 1;
}

.caution__dismiss {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.125rem 0.375rem;
  background: transparent;
  border: none;
  color: inherit;
  font-size: 1.125rem;
  line-height: 1;
  cursor: pointer;
  border-radius: var(--numen-radius-pill, 9999px);
  opacity: 0.8;
}

.caution__dismiss:hover {
  opacity: 1;
  background-color: var(--numen-highlight, rgba(0, 0, 0, 0.08));
}

.markdown-note-editor__pane {
  flex: 1;
  min-block-size: 0;
  overflow: auto;
}
</style>
