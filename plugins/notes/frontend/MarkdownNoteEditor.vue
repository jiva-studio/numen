<script setup lang="ts">
/**
 * Standalone Markdown and plain text editor tab component using CodeMirror 6.
 */
import { computed, ref, useTemplateRef, watch, type Ref } from 'vue'
import { Editor } from './editor'
import type { Extension } from '@codemirror/state'
import { createWikilinkExtension, type WikilinkOptions } from './extensions/wikilinks'
import { editorPreferences } from './preferences'
import {
  getEditorLanguage,
  readFileConflict,
  useMarkdownDocument,
  type EditorHandle,
} from './useMarkdownDocument'
import { Button } from '@numen/ui'
import { WORDS } from './words'

/* --------------------------------- Props ---------------------------------- */
interface Props {
  readonly modelValue?: string
  readonly path?: string
  readonly filePath?: string
  /** A tab of the host passes the error ref of its state. */
  readonly errorMessage?: string | Readonly<Ref<string>>
  readonly isReadOnly?: boolean
  readonly extensions?: Extension | readonly Extension[] | undefined
  readonly state?: Record<string, unknown> | undefined
  readonly softWrap?: boolean | undefined
  readonly lineNumbers?: boolean | undefined
  readonly tabSize?: number | undefined
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

/** The completion of wikilinks over the notes the host offers, unless the caller passes extensions. */
const editorExtensions = computed(() => {
  if (props.extensions) return props.extensions as Extension
  const links = props.state?.['wikilinks'] as
    Pick<WikilinkOptions, 'search' | 'headings'> | undefined
  return links ? createWikilinkExtension(links) : undefined
})

const editorLanguage = computed(() => getEditorLanguage(props))
const fileConflict = computed(() => readFileConflict(props.state?.['fileConflict']))

const softWrap = computed(() => props.softWrap ?? editorPreferences.softWrap)
const lineNumbers = computed(() => props.lineNumbers ?? editorPreferences.lineNumbers)
const tabSize = computed(() => props.tabSize ?? editorPreferences.tabSize)

/* -------------------------------- Handlers -------------------------------- */
function onKeep() {
  const keep = props.state?.['keepMine']
  if (typeof keep === 'function') keep()
}

function onTake() {
  const take = props.state?.['takeFile']
  if (typeof take === 'function') take()
}

function onDismiss() {
  isDismissed.value = true
}

function onOpen(url: string) {
  emit('open', url)
  const follow = props.state?.['followLink']
  if (typeof follow === 'function') follow(url)
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
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  margin: 0;
  padding: 0.4rem 1rem;
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
