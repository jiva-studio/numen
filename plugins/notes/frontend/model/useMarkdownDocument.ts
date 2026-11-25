import { computed, onMounted, ref, watch, type Ref } from 'vue'
import {
  editorNotePath,
  type EditorHandle,
  type NoteTabState,
  formatErrorMessage,
} from '@numen/plugin-sdk'
import { readTextAsset } from '../api/readTextAsset'
import { parseFrontmatter, type FrontmatterResult } from '../lib/frontmatter'
import { getFrontmatterError } from '../lib/frontmatterError'
import { createWikilinkExtension } from '../lib/wikilinks'
import type { Extension } from '@codemirror/state'
import { getPluginExtensions, pluginExtensionEntries } from './pluginExtensions'

export interface DocumentOptions {
  readonly modelValue?: string
  readonly path?: string
  readonly filePath?: string
  readonly errorMessage?: string
  readonly state?: NoteTabState | undefined
}

export interface DocumentEmit {
  (e: 'update:modelValue', value: string): void
  (e: 'save'): void
  (e: 'change', value: string): void
}

function getNoteState(props: DocumentOptions): NoteTabState | undefined {
  return props.state && 'body' in props.state ? props.state : undefined
}

function getDocumentPath(props: DocumentOptions): string {
  return props.filePath || props.path || props.state?.path || ''
}

export function getEditorLanguage(props: DocumentOptions): 'markdown' | 'json' | 'plaintext' {
  const path = getDocumentPath(props)
  if (path.endsWith('.txt')) return 'plaintext'
  if (path.endsWith('.json')) return 'json'
  return 'markdown'
}

/** The extensions the caller passes, else the wikilink completion over the notes the host offers, then those other plugins add. */
export function getEditorExtensions(
  props: DocumentOptions & { readonly extensions?: Extension | readonly Extension[] | undefined },
): Extension | undefined {
  const links = props.state?.wikilinks
  const own = props.extensions
    ? (props.extensions as Extension)
    : links
      ? createWikilinkExtension(links)
      : undefined
  const added = getPluginExtensions(pluginExtensionEntries.value)
  if (added.length === 0) return own
  const withPath = [editorNotePath.of(getDocumentPath(props)), ...added]
  return own ? [own, ...withPath] : withPath
}

export function useMarkdownDocument(
  props: DocumentOptions,
  emit: DocumentEmit,
  editorRef: Ref<EditorHandle | null>,
) {
  const activePath = computed(() => getDocumentPath(props))

  const getInitialBody = (): string => props.modelValue || getNoteState(props)?.body.value || ''

  /** A host tab supplies the body from its store, so the editor does not read the file itself. */
  const isHostOwned = (): boolean => getNoteState(props) !== undefined

  const docContent = ref(getInitialBody())
  const initialContent = ref(getInitialBody())

  const isDirty = computed(() => docContent.value !== initialContent.value)
  const frontmatter = computed<FrontmatterResult>(() => parseFrontmatter(docContent.value))
  const loadError = ref('')
  const frontmatterError = computed(() => getFrontmatterError(docContent.value))

  const isLoading = computed(() => getNoteState(props)?.isLoading.value === true)

  const displayErrorMessage = computed<string>(
    () =>
      props.errorMessage?.trim() ||
      getNoteState(props)?.errorMessage.value.trim() ||
      frontmatterError.value ||
      loadError.value,
  )

  watch(getInitialBody, (next) => {
    if ((next || isHostOwned()) && next !== docContent.value) {
      docContent.value = next
      initialContent.value = next
    }
  })

  function loadContentFromPath(filePath: string): void {
    readTextAsset(filePath).then(
      (read) => {
        if (!read.ok) {
          loadError.value = read.error
        } else if (!docContent.value) {
          docContent.value = read.value
          initialContent.value = read.value
        }
      },
      (error: unknown) => {
        loadError.value = formatErrorMessage(error)
      },
    )
  }

  watch(editorRef, (editor) => getNoteState(props)?.attachEditor?.(editor), { immediate: true })

  onMounted(() => {
    getNoteState(props)?.attachEditor?.(editorRef.value)
    if (!docContent.value && activePath.value && !isHostOwned()) {
      loadContentFromPath(activePath.value)
    }
  })

  watch(activePath, (newPath, oldPath) => {
    if (oldPath !== undefined && newPath !== oldPath) {
      const nextBody = getInitialBody()
      docContent.value = nextBody
      initialContent.value = nextBody
    }
    if (newPath && !docContent.value && !isHostOwned()) {
      loadContentFromPath(newPath)
    }
  })

  function onUpdateModelValue(value: string) {
    docContent.value = value
    emit('update:modelValue', value)
    emit('change', value)
    getNoteState(props)?.updateBody(value)
  }

  function onSave() {
    initialContent.value = docContent.value
    emit('save')
    getNoteState(props)?.save()
  }

  function setContent(value: string) {
    docContent.value = value
    initialContent.value = value
  }

  return {
    docContent,
    isDirty,
    frontmatter,
    isLoading,
    displayErrorMessage,
    onUpdateModelValue,
    onSave,
    setContent,
  }
}
