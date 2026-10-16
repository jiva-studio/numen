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

function getInitialBody(): string {
  if (props.modelValue) return props.modelValue
  const fromBody = extractBodyValue(props.state?.body)
  if (fromBody) return fromBody
  const fromNote = extractBodyValue(props.state?.note)
  if (fromNote) return fromNote
  return ''
}

const docContent = ref(getInitialBody())
const initialContent = ref(getInitialBody())

const isDirty = computed(() => docContent.value !== initialContent.value)
const frontmatter = computed<FrontmatterResult>(() => parseFrontmatter(docContent.value))

watch(
  () => {
    if (props.modelValue) return props.modelValue
    const fromBody = extractBodyValue(props.state?.body)
    if (fromBody) return fromBody
    const fromNote = extractBodyValue(props.state?.note)
    if (fromNote) return fromNote
    return ''
  },
  (next) => {
    if (next && next !== docContent.value) {
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

async function loadContentFromPath(filePath: string): Promise<void> {
  if (!filePath) return
  try {
    const res = await fetch(`/assets/${encodeURIComponent(filePath)}`)
    if (res.ok) {
      const text = await res.text()
      if (!docContent.value) {
        docContent.value = text
        initialContent.value = text
      }
    }
  } catch {
    // Ignore fetch failures on test mocks
  }
}

watch(editorRef, (val) => syncEditorState(val), { immediate: true })
onMounted(() => {
  syncEditorState(editorRef.value)
  if (!docContent.value && activePath.value) {
    void loadContentFromPath(activePath.value)
  }
})
watch(activePath, (newPath) => {
  if (newPath && !docContent.value) {
    void loadContentFromPath(newPath)
  }
})

/* -------------------------------- Handlers -------------------------------- */
function onUpdateModelValue(value: string) {
  docContent.value = value
  emit('update:modelValue', value)
  emit('change', value)
  if (props.state && typeof props.state === 'object') {
    const update = (props.state as { updateBody?: (v: string) => void }).updateBody
    if (typeof update === 'function') update(value)
  }
}

function onSave() {
  initialContent.value = docContent.value
  emit('save')
  if (props.state && typeof props.state === 'object') {
    const save = (props.state as { save?: () => void }).save
    if (typeof save === 'function') save()
  }
}

function onOpen(url: string) {
  emit('open', url)
}

/* -------------------------------- Helpers --------------------------------- */
function extractBodyValue(source: unknown): string {
  if (typeof source === 'string') return source
  if (typeof source === 'object' && source !== null && 'value' in source) {
    const val = (source as { value: unknown }).value
    if (typeof val === 'string') return val
    if (typeof val === 'object' && val !== null && 'body' in val) {
      return String((val as { body: unknown }).body ?? '')
    }
  }
  return ''
}

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
  <div class="markdown-note-editor note">
    <Editor
      ref="editorRef"
      :model-value="docContent"
      :language="editorLanguage"
      :is-live="true"
      v-bind="props.extensions ? { extensions: props.extensions as any } : {}"
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

.markdown-note-editor__pane {
  flex: 1;
  min-block-size: 0;
  overflow: auto;
}
</style>
