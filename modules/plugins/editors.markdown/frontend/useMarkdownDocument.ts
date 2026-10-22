import { computed, onMounted, ref, watch, type Ref } from 'vue'
import { parseFrontmatter, type FrontmatterResult } from './frontmatter'

export interface EditorHandle {
  focus?: () => boolean
  measure?: () => void
  reveal?: (line: number) => boolean
}

export interface DocumentState {
  readonly [key: string]: unknown
  body?: unknown
  note?: unknown
  path?: string
  file?: string
  at?: string
  isLoading?: unknown
  errorMessage?: unknown
  editor?: EditorHandle | null
  measure?: () => void
  updateBody?: (val: string) => void
  save?: () => void
}

export interface DocumentOptions {
  readonly modelValue?: string
  readonly path?: string
  readonly filePath?: string
  readonly errorMessage?: string
  readonly state?: DocumentState
}

export interface DocumentEmit {
  (e: 'update:modelValue', value: string): void
  (e: 'save'): void
  (e: 'change', value: string): void
}

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

function hasInvalidBracket(rawYaml: string): boolean {
  for (const line of rawYaml.split(/\r?\n/)) {
    const cleanLine = line.trim()
    if (cleanLine.includes('[') && !cleanLine.includes(']')) {
      return true
    }
  }
  return false
}

function checkFrontmatterError(content: string): string {
  if (!content.startsWith('---\n') && !content.startsWith('---\r\n')) {
    return ''
  }
  const newlineOffset = content.indexOf('\n')
  const rest = content.slice(newlineOffset + 1)
  const delimiterMatch = rest.match(/\n---\s*(\r?\n|$)/)
  if (!delimiterMatch || delimiterMatch.index === undefined) {
    return ''
  }
  const rawYaml = rest.slice(0, delimiterMatch.index)
  if (hasInvalidBracket(rawYaml)) {
    return 'the frontmatter of this note cannot be read'
  }
  return ''
}

export function useMarkdownDocument(
  props: DocumentOptions,
  emit: DocumentEmit,
  editorRef: Ref<EditorHandle | null>,
) {
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
  const frontmatterError = computed(() => checkFrontmatterError(docContent.value))

  const isLoading = computed(() => {
    const loading = props.state?.isLoading
    if (typeof loading === 'object' && loading !== null && 'value' in loading) {
      return (loading as { value?: unknown }).value === true
    }
    return loading === true
  })

  const displayErrorMessage = computed(() => {
    if (props.errorMessage) return props.errorMessage
    const stateErr = props.state?.errorMessage
    if (typeof stateErr === 'string' && stateErr) return stateErr
    if (typeof stateErr === 'object' && stateErr !== null && 'value' in stateErr) {
      const val = (stateErr as { value?: unknown }).value
      if (typeof val === 'string' && val) return val
    }
    return frontmatterError.value
  })

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

  function syncEditorState(val: EditorHandle | null) {
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
