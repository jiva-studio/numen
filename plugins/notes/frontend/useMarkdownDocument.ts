import { computed, onMounted, ref, watch, type Ref } from 'vue'
import { getAssetUrl } from '@numen/plugin-sdk'
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
  readonly state?: DocumentState | undefined
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

function extractErrorString(source: unknown): string {
  if (typeof source === 'string') return source.trim()
  if (typeof source === 'object' && source !== null && 'value' in source) {
    const val = (source as { value: unknown }).value
    if (typeof val === 'string') return val.trim()
  }
  return ''
}

interface DelimiterState {
  inSingleQuote: boolean
  inDoubleQuote: boolean
  bracketDepth: number
  braceDepth: number
}

function scanQuoteChar(
  char: string,
  nextChar: string | undefined,
  state: DelimiterState,
): { skipNext: boolean; isHandled: boolean } {
  if (state.inSingleQuote) {
    if (char === "'") {
      if (nextChar === "'") return { skipNext: true, isHandled: true }
      state.inSingleQuote = false
    }
    return { skipNext: false, isHandled: true }
  }
  if (state.inDoubleQuote) {
    if (char === '\\') return { skipNext: true, isHandled: true }
    if (char === '"') state.inDoubleQuote = false
    return { skipNext: false, isHandled: true }
  }
  return { skipNext: false, isHandled: false }
}

function updateBracketDepth(char: string, state: DelimiterState): void {
  if (char === '[') state.bracketDepth++
  else if (char === ']') state.bracketDepth--
  else if (char === '{') state.braceDepth++
  else if (char === '}') state.braceDepth--
}

function scanCharDelimiters(
  char: string,
  nextChar: string | undefined,
  state: DelimiterState,
): { skipNext: boolean; isDone: boolean } {
  const quote = scanQuoteChar(char, nextChar, state)
  if (quote.isHandled) return { skipNext: quote.skipNext, isDone: false }

  if (char === '#') return { skipNext: false, isDone: true }
  if (char === "'") state.inSingleQuote = true
  else if (char === '"') state.inDoubleQuote = true
  else updateBracketDepth(char, state)

  return { skipNext: false, isDone: false }
}

function scanLineDelimiters(line: string, state: DelimiterState): boolean {
  for (let i = 0; i < line.length; i++) {
    const res = scanCharDelimiters(line[i]!, line[i + 1], state)
    if (res.skipNext) i++
    if (res.isDone) break
    if (state.bracketDepth < 0 || state.braceDepth < 0) return true
  }
  return false
}

function isTopLevelKeyLine(line: string, lineContent: string): boolean {
  return (
    !line.startsWith(' ') &&
    !line.startsWith('\t') &&
    !lineContent.startsWith('#') &&
    !lineContent.startsWith('-')
  )
}

function checkTopLevelMappingLine(lineContent: string, topKeys: Set<string>): boolean {
  const colonIdx = lineContent.indexOf(':')
  if (colonIdx <= 0) return true
  const key = lineContent.slice(0, colonIdx).trim()
  if (topKeys.has(key)) return true
  topKeys.add(key)
  const after = lineContent.slice(colonIdx + 1).trim()
  return after.endsWith(':') && !after.startsWith('{') && !after.startsWith('[')
}

function isTopLevel(state: DelimiterState): boolean {
  return (
    state.bracketDepth === 0 &&
    state.braceDepth === 0 &&
    !state.inSingleQuote &&
    !state.inDoubleQuote
  )
}

function isMalformedLine(
  line: string,
  lineContent: string,
  state: DelimiterState,
  topKeys: Set<string>,
): boolean {
  if (
    isTopLevel(state) &&
    isTopLevelKeyLine(line, lineContent) &&
    checkTopLevelMappingLine(lineContent, topKeys)
  ) {
    return true
  }
  return scanLineDelimiters(line, state)
}

function hasMalformedFrontmatter(rawYaml: string): boolean {
  const state: DelimiterState = {
    inSingleQuote: false,
    inDoubleQuote: false,
    bracketDepth: 0,
    braceDepth: 0,
  }
  const topKeys = new Set<string>()

  for (const line of rawYaml.split(/\r?\n/)) {
    const lineContent = line.trim()
    if (lineContent && isMalformedLine(line, lineContent, state, topKeys)) return true
  }

  return (
    state.inSingleQuote || state.inDoubleQuote || state.bracketDepth !== 0 || state.braceDepth !== 0
  )
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
  if (hasMalformedFrontmatter(rawYaml)) {
    return 'frontmatter unreadable'
  }
  return ''
}

export function getEditorLanguage(props: DocumentOptions): 'markdown' | 'json' | 'plaintext' {
  const path =
    props.filePath ||
    props.path ||
    (props.state?.path as string) ||
    (props.state?.file as string) ||
    (props.state?.at as string) ||
    ''
  if (path.endsWith('.txt')) return 'plaintext'
  if (path.endsWith('.json')) return 'json'
  return 'markdown'
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

  const displayErrorMessage = computed<string>(() => {
    const propErr = extractErrorString(props.errorMessage)
    if (propErr) return propErr
    const stateErr = extractErrorString(props.state?.errorMessage)
    if (stateErr) return stateErr
    const fmErr = frontmatterError.value
    if (typeof fmErr === 'string' && fmErr.trim()) return fmErr.trim()
    return ''
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
      const res = await fetch(getAssetUrl(filePath))
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
