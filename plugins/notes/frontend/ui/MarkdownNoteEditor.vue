<script setup lang="ts">
/**
 * Standalone Markdown and plain text editor tab component using CodeMirror 6.
 */
import { computed, ref, useTemplateRef, watch } from 'vue'
import { Editor } from '../editor'
import type { Extension } from '@codemirror/state'
import type { EditorHandle, NoteTabState } from '@numen/plugin-sdk'
import { getFontFamilyValue } from '../model/fontFamily'
import { editorPreferences } from '../model/preferences'
import {
  getEditorExtensions,
  getEditorLanguage,
  useMarkdownDocument,
} from '../model/useMarkdownDocument'
import { Button } from '@numen/ui'
import { WORDS } from '../words'

/* --------------------------------- Props ---------------------------------- */
interface Props {
  readonly modelValue?: string
  readonly path?: string
  readonly filePath?: string
  readonly errorMessage?: string
  readonly isReadOnly?: boolean
  readonly extensions?: Extension | readonly Extension[] | undefined
  readonly state?: NoteTabState | undefined
  readonly softWrap?: boolean | undefined
  readonly lineNumbers?: boolean | undefined
  readonly tabSize?: number | undefined
  readonly fontFamily?: string | undefined
  readonly lineHeight?: number | undefined
  readonly highlightActiveLine?: boolean | undefined
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  path: '',
  filePath: '',
  errorMessage: '',
  isReadOnly: false,
  softWrap: undefined,
  lineNumbers: undefined,
  tabSize: undefined,
  fontFamily: undefined,
  lineHeight: undefined,
  highlightActiveLine: undefined,
  extensions: undefined,
  state: undefined,
})

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  'update:modelValue': [value: string]
  save: []
  open: [url: string]
  change: [value: string]
}>()

defineExpose({
  focus: () => editorRef.value?.focus?.() ?? false,
  measure: () => editorRef.value?.measure?.(),
  reveal: (line: number) => editorRef.value?.reveal?.(line) ?? false,
  get isDirty() {
    return isDirty.value
  },
  get frontmatter() {
    return frontmatter.value
  },
  getContent: () => docContent.value,
  setContent: (value: string) => setContent(value),
})

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

const editorExtensions = computed(() => getEditorExtensions(props))

const editorLanguage = computed(() => getEditorLanguage(props))
const fileConflict = computed(() => props.state?.fileConflict?.value ?? null)

const prefer = <K extends keyof typeof editorPreferences>(key: K) =>
  computed(() => props[key] ?? editorPreferences[key])
const softWrap = prefer('softWrap')
const lineNumbers = prefer('lineNumbers')
const tabSize = prefer('tabSize')
const lineHeight = prefer('lineHeight')
const highlightActiveLine = prefer('highlightActiveLine')
const fontFamily = computed(() => getFontFamilyValue(prefer('fontFamily').value) ?? '')

/* --------------------------------- Hooks ---------------------------------- */
watch(displayErrorMessage, () => {
  isDismissed.value = false
})

/* -------------------------------- Handlers -------------------------------- */
function onKeep() {
  props.state?.keepMine?.()
}

function onTake() {
  props.state?.takeFile?.()
}

function onDismiss() {
  isDismissed.value = true
}

function onOpen(url: string) {
  emit('open', url)
  props.state?.followLink?.(url)
}
</script>

<template>
  <div class="markdown-note-editor note">
    <div v-if="fileConflict" class="caution" role="alert">
      <span class="caution__text">{{ WORDS[fileConflict] }}</span>
      <Button size="small" @click="onKeep">
        {{ fileConflict === 'gone' ? WORDS.makeAgain : WORDS.keep }}
      </Button>
      <Button v-if="fileConflict === 'stale'" variant="outline" size="small" @click="onTake">
        {{ WORDS.take }}
      </Button>
    </div>
    <div v-else-if="displayErrorMessage && !isDismissed" class="caution" role="alert">
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
      :soft-wrap="softWrap"
      :line-numbers="lineNumbers"
      :tab-size="tabSize"
      :font-family="fontFamily"
      :line-height="lineHeight"
      :highlight-active-line="highlightActiveLine"
      v-bind="editorExtensions ? { extensions: editorExtensions } : {}"
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
  --caution-gap: 0.5rem;
  --caution-pad-block: 0.4rem;
  --caution-pad-inline: 1rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--caution-gap);
  margin: 0;
  padding: var(--caution-pad-block) var(--caution-pad-inline);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  background-color: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  overflow-wrap: break-word;
}

.caution__text {
  flex: 1;
}

.caution__dismiss {
  --dismiss-pad-block: 0.125rem;
  --dismiss-pad-inline: 0.375rem;
  --dismiss-text: 1.125rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: var(--dismiss-pad-block) var(--dismiss-pad-inline);
  background: transparent;
  border: none;
  color: inherit;
  font-size: var(--dismiss-text);
  line-height: 1;
  cursor: pointer;
  border-radius: var(--numen-radius-pill);
  opacity: 0.8;
}

.caution__dismiss:hover {
  opacity: 1;
  background-color: var(--numen-highlight);
}

.markdown-note-editor__pane {
  flex: 1;
  min-block-size: 0;
  overflow: auto;
}
</style>
