<script setup lang="ts">
/**
 * Standalone Markdown and plain text editor tab component using CodeMirror 6.
 */
import { computed, onMounted, ref, watch } from 'vue'
import { Editor } from '@numen/ui'
import type { Extension } from '@codemirror/state'
import { parseFrontmatter, type FrontmatterResult } from './frontmatter'

/* --------------------------------- Props ---------------------------------- */
interface Props {
  readonly modelValue?: string
  readonly path?: string
  readonly filePath?: string
  readonly isReadOnly?: boolean
  readonly extensions?: Extension | readonly Extension[]
  readonly state?: Record<string, unknown>
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  path: '',
  filePath: '',
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
const editorRef = ref<{ focus?: () => boolean; measure?: () => void; reveal?: (line: number) => boolean } | null>(null)

const activePath = computed(() => {
  return (
    props.filePath ||
    props.path ||
    (props.state?.path as string) ||
    (props.state?.file as string) ||
    (props.state?.at as string) ||
    ''
  )
})

const editorLanguage = computed(() => {
  if (activePath.value.endsWith('.txt')) return 'plaintext'
  if (activePath.value.endsWith('.json')) return 'json'
  return 'markdown'
})

const initialBody = props.modelValue || (props.state?.body as string) || ''
const docContent = ref(initialBody)
const initialContent = ref(initialBody)

const isDirty = computed(() => docContent.value !== initialContent.value)
const frontmatter = computed<FrontmatterResult>(() => parseFrontmatter(docContent.value))

watch(
  () => props.modelValue,
  (next) => {
    if (next !== undefined && next !== docContent.value) {
      docContent.value = next
      initialContent.value = next
    }
  },
)

/* --------------------------------- Hooks ---------------------------------- */
function syncEditorState(val: typeof editorRef.value) {
  if (props.state && typeof props.state === 'object') {
    const s = props.state as Record<string, unknown>
    s.editor = val
    s.measure = () => val?.measure?.()
  }
}

watch(editorRef, (val) => syncEditorState(val), { immediate: true })
onMounted(() => syncEditorState(editorRef.value))

/* -------------------------------- Handlers -------------------------------- */
function onUpdateModelValue(value: string) {
  docContent.value = value
  emit('update:modelValue', value)
  emit('change', value)
}

function onSave() {
  initialContent.value = docContent.value
  emit('save')
}

function onOpen(url: string) {
  emit('open', url)
}

/* --------------------------------- Helpers --------------------------------- */
defineExpose({
  focus: () => editorRef.value?.focus?.() ?? false,
  measure: () => editorRef.value?.measure?.(),
  reveal: (line: number) => editorRef.value?.reveal?.(line) ?? false,
  isDirty,
  frontmatter,
  getContent: () => docContent.value,
  setContent: (value: string) => {
    docContent.value = value
    initialContent.value = value
  },
})
</script>

<template>
  <div class="markdown-note-editor">
    <Editor
      ref="editorRef"
      :model-value="docContent"
      :language="editorLanguage"
      :is-live="true"
      v-bind="props.extensions ? { extensions: props.extensions as any } : {}"
      class="markdown-note-editor__pane"
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

.markdown-note-editor__pane {
  flex: 1;
  min-block-size: 0;
  overflow: auto;
}
</style>
