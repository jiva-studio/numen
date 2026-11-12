/**
 * Tab kinds created dynamically for plugins and readers.
 */
import {
  computed,
  defineComponent,
  onBeforeUnmount,
  onErrorCaptured,
  onMounted,
  reactive,
  ref,
  shallowReactive,
  shallowRef,
  type Component,
} from 'vue'
import {
  isNotePath,
  type EditorHandle,
  type JsonValue,
  type PluginContext,
  type ReaderHandle,
  type ReaderRegistration,
  type Selector,
  type TabKindRegistration,
  type TabPlace,
  type TextRange,
  type Unvalidated,
} from '@numen/plugin-sdk'
import { adoptPluginStyle } from '../styles'
import {
  getReaderKindId,
  type AnyTabKind,
  type OpenTab,
  type TabKind,
  type TabKindAddress,
  type TabKindDisplay,
} from './tabKinds'
import {
  attachPluginRoot,
  extractFilename,
  parsePluginComponent,
  parseTitle,
  renderFallback,
  renderPaneContent,
  renderWrapper,
  stripMarkdownExtension,
  type PluginComponentSource,
} from './pane/pluginView'
import { createNoteWriter, startNoteRead } from './customTabNote'
import type { PluginHostDeps } from './types'
import { pickDeclaredParts } from './pluginTabKind'
import { createBufferedNoteState } from './customTabBuffer'

export function createPluginPaneWrapper(
  pluginId: string,
  rawComponent: PluginComponentSource,
  getContext?: () => PluginContext | undefined,
): Component {
  const component = parsePluginComponent(rawComponent)
  const sanitizedId = pluginId.replace(/[^a-zA-Z0-9_-]/g, '-')
  const classes = {
    wrapper: `plugin-pane-wrapper plugin-pane-${sanitizedId}`,
    container: `plugin-view-container plugin-view-${sanitizedId}`,
  }

  return defineComponent({
    // Stryker disable next-line StringLiteral: Vue component debug name
    name: `PluginPaneWrapper[${pluginId}]`,
    // Stryker disable next-line ObjectLiteral,BooleanLiteral: Vue prop descriptor
    props: { state: { type: null, required: true } },
    setup(props) {
      const error = ref<Error | null>(null)
      const hostEl = shallowRef<HTMLElement | null>(null)
      const shadowRoot = shallowRef<ShadowRoot | null>(null)
      const isReady = ref(false)
      let releaseStyle: (() => void) | undefined

      onErrorCaptured((err) => {
        error.value = err instanceof Error ? err : new Error(String(err))
        return false
      })

      onMounted(() => {
        shadowRoot.value = attachPluginRoot(hostEl.value)
        if (shadowRoot.value) releaseStyle = adoptPluginStyle(pluginId, shadowRoot.value)
        isReady.value = true
      })
      onBeforeUnmount(() => releaseStyle?.())

      return () => {
        if (error.value) return renderFallback(pluginId, error.value)
        const content = isReady.value
          ? renderPaneContent({
              pluginId,
              component,
              state: props.state,
              shadowRoot: shadowRoot.value,
              containerClass: classes.container,
              context: getContext?.(),
              hasAdoptedStyle: releaseStyle !== undefined,
            })
          : null
        return renderWrapper({ wrapperClass: classes.wrapper, pluginId, hostEl, content })
      }
    },
  })
}

const isSelector = (value: Unvalidated): value is Selector =>
  // Stryker disable next-line ConditionalExpression: null saved value maps to null place in both branches
  typeof value === 'object' && value !== null && !Array.isArray(value)

const createReaderState = (pluginId: string, targetPath: string, saved: JsonValue | undefined) => {
  const place = isSelector(saved) ? saved : null
  let reader: ReaderHandle | null = null
  const state = reactive({
    path: targetPath,
    file: targetPath,
    pluginId,
    title: extractFilename(targetPath),
    selector: place,
    ranges: null as readonly TextRange[] | null,
    place,
    savePlace: (next: Selector) => {
      state.place = next
    },
    reported: null as TabPlace | null,
    attachReader: (attached: ReaderHandle | null): void => {
      reader = attached
    },
    focusTab: (): void => reader?.focus(),
    measure: (): void => reader?.measure(),
    reportPlace: (next: TabPlace) => {
      state.reported = next
    },
  })
  return state
}

type ReaderState = ReturnType<typeof createReaderState>

const getReaderOpenTab = (state: ReaderState): OpenTab =>
  state.reported ? { path: state.path, place: { ...state.reported } } : { path: state.path }

export function createReaderTabKind(
  pluginId: string,
  registration: ReaderRegistration,
  getContext?: () => PluginContext | undefined,
): AnyTabKind {
  const kindId = getReaderKindId(registration.id)
  const display: TabKindDisplay<ReaderState> = {
    onShow: (state) => {
      state.measure()
      state.focusTab()
    },
    ...(registration.icon ? { icon: registration.icon } : {}),
  }
  const kind: TabKind<ReaderState> = {
    kind: kindId,
    pluginId,
    open: (targetPath, _id, saved) => createReaderState(pluginId, targetPath, saved),
    pane: createPluginPaneWrapper(pluginId, registration.component, getContext),
    // A reader may keep the title as a ref, which the state of a tab does not unwrap.
    getTitle: (state) =>
      parseTitle(state.title) || extractFilename(state.path) || registration.title || '',
    display,
    address: {
      getState: (state) => state.place ?? undefined,
      identity: (targetPath) => targetPath,
      getTarget: (state) => ({ file: state.path, path: state.path, title: state.title }),
      getOpenTab: getReaderOpenTab,
    },
  }
  return kind
}

interface CustomTabStateDeps {
  readonly pluginId: string
  readonly registration: TabKindRegistration
  readonly hostDeps?: PluginHostDeps | undefined
}

function createCustomTabState(targetPath: string, deps: CustomTabStateDeps) {
  const defaultName = stripMarkdownExtension(extractFilename(targetPath))
  const noteRef = ref({ path: targetPath, body: '' })
  const errorMessageRef = ref('')
  const isLoading = ref(false)
  let currentPath = targetPath
  let hasEdited = false
  let editor: EditorHandle | null = null
  let write = createNoteWriter(targetPath, deps.hostDeps?.writeNote, errorMessageRef)

  const readAt = (pathToRead: string): void => {
    const readNote = deps.hostDeps?.readNote
    if (!readNote) return
    isLoading.value = true
    startNoteRead(pathToRead, readNote, {
      note: noteRef,
      errorMessage: errorMessageRef,
      isLoading,
      hasEdited: () => hasEdited || currentPath !== pathToRead,
    })
  }

  readAt(targetPath)

  const state = shallowReactive({
    at: targetPath,
    path: targetPath,
    pluginId: deps.pluginId,
    title: defaultName,
    note: noteRef,
    errorMessage: errorMessageRef,
    isLoading,
    body: computed(() => noteRef.value.body),
    attachEditor: (attached: EditorHandle | null): void => {
      editor = attached
    },
    measure: (): void => editor?.measure?.(),
    updateBody: (body: string) => {
      hasEdited = true
      noteRef.value = { path: currentPath, body }
      write(body)
    },
    save: () => {
      if (isLoading.value && !hasEdited) return
      write(noteRef.value.body)
    },
    navigate: (nextPath: string, nextTitle?: string) => {
      if (!nextPath || nextPath === currentPath) return
      currentPath = nextPath
      hasEdited = false
      state.at = nextPath
      state.path = nextPath
      state.title = resolveTargetTitle(nextTitle, stripMarkdownExtension(extractFilename(nextPath)))
      errorMessageRef.value = ''
      noteRef.value = { path: nextPath, body: '' }
      write = createNoteWriter(nextPath, deps.hostDeps?.writeNote, errorMessageRef)
      readAt(nextPath)
    },
  })
  return state
}

function resolveTargetTitle(title?: string, defaultName?: string, fallback = ''): string {
  return title || defaultName || fallback
}

type CustomTabState =
  ReturnType<typeof createCustomTabState> | ReturnType<typeof createBufferedNoteState>

function resolveCustomTabTarget(state: CustomTabState, registration: TabKindRegistration) {
  const defaultName = stripMarkdownExtension(extractFilename(state.path))
  const title = resolveTargetTitle(
    state.title,
    defaultName,
    registration.title || registration.kind,
  )

  return {
    ...(state.path ? { file: state.path, path: state.path } : {}),
    title,
  }
}

function notifyCustomTabShown(
  state: CustomTabState,
  id: string | undefined,
  deps?: PluginHostDeps,
): void {
  state.measure()
  if (!id || !deps?.getWorkspace) return
  if (state.path) deps.getWorkspace()?.files.syncPath(id, state.path, state.title)
}

export function createCustomTabKind(
  pluginId: string,
  registration: TabKindRegistration,
  deps?: PluginHostDeps & {
    readonly getContext?: (() => PluginContext | undefined) | undefined
    /** The manifest declares the kind as the editor of the text of a note: its tabs hold the note in the store of open notes. */
    readonly isNoteEditor?: boolean
  },
): AnyTabKind {
  const declared = pickDeclaredParts(registration)
  const display: TabKindDisplay<CustomTabState> = {
    onShow: (state, id) => notifyCustomTabShown(state, id, deps),
    ...declared.display,
  }
  const address: TabKindAddress<CustomTabState> = {
    identity: (targetPath) => targetPath || registration.kind,
    navigate: (state, targetPath, title) => {
      if (!isNotePath(targetPath)) return
      state.navigate(targetPath, title)
    },
    getTarget: (state) => resolveCustomTabTarget(state, registration),
    getOpenTab: (state) => ({ path: state.path }),
    ...declared.address,
  }
  const kind: TabKind<CustomTabState> = {
    kind: registration.kind,
    pluginId,
    open: (targetPath, id) => {
      const buffer = deps?.isNoteEditor ? deps.getNoteBuffer?.() : undefined
      if (!buffer)
        return createCustomTabState(targetPath, { pluginId, registration, hostDeps: deps })
      return createBufferedNoteState(
        { targetPath, id, buffer },
        {
          pluginId,
          deps,
        },
      )
    },
    pane: createPluginPaneWrapper(pluginId, registration.component, deps?.getContext),
    getTitle: (state) =>
      state.title ||
      stripMarkdownExtension(extractFilename(state.path)) ||
      registration.title ||
      registration.kind,
    display,
    events: declared.events,
    address,
  }
  return kind
}
