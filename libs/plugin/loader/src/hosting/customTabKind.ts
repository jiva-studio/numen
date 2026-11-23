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
  type TextRange,
} from '@numen/plugin-sdk'
import type { PluginStyles } from '../styles'
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
import { createLineReveal } from './customTabReveal'
import { createNoteWriter, startNoteRead } from './customTabNote'
import type { PluginHostDeps } from './types'
import { pickDeclaredParts, type DeclaredParts } from './pluginTabKind'
import { createBufferedNoteState } from './customTabBuffer'

/** What a pane takes of its host: the stylesheets it adopts, and the context of its plugin. */
interface PaneHost {
  readonly styles: PluginStyles
  readonly getContext?: (() => PluginContext | undefined) | undefined
}

function createPluginPaneWrapper(
  pluginId: string,
  rawComponent: PluginComponentSource,
  host: PaneHost,
): Component {
  const { styles, getContext } = host
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
        if (shadowRoot.value) releaseStyle = styles.adopt(pluginId, shadowRoot.value)
        isReady.value = true
      })
      onBeforeUnmount(() => releaseStyle?.())

      return () => {
        if (error.value) return renderFallback(pluginId, error.value)
        const content = isReady.value
          ? renderPaneContent({
              pluginId,
              component,
              styles,
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

const isSelector = (value: unknown): value is Selector =>
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
    reported: null as string | null,
    attachReader: (attached: ReaderHandle | null): void => {
      reader = attached
    },
    focusTab: (): void => reader?.focus(),
    measure: (): void => reader?.measure(),
    reportPlace: (next: string) => {
      state.reported = next
    },
  })
  return state
}

type ReaderState = ReturnType<typeof createReaderState>

const getReaderOpenTab = (state: ReaderState): OpenTab =>
  state.reported ? { path: state.path, place: state.reported } : { path: state.path }

export function createReaderTabKind(
  pluginId: string,
  registration: ReaderRegistration,
  host: PaneHost,
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
    pane: createPluginPaneWrapper(pluginId, registration.component, host),
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
  readonly hostDeps?: Pick<PluginHostDeps, 'kinds'> | undefined
}

function createCustomTabState(targetPath: string, deps: CustomTabStateDeps) {
  const defaultName = stripMarkdownExtension(extractFilename(targetPath))
  const noteRef = ref({ path: targetPath, body: '' })
  const errorMessageRef = ref('')
  const isLoading = ref(false)
  let currentPath = targetPath
  let hasEdited = false
  const lines = createLineReveal(() => !isLoading.value)
  let write = createNoteWriter(targetPath, deps.hostDeps?.kinds.writeNote, errorMessageRef)

  const readAt = (pathToRead: string): void => {
    const readNote = deps.hostDeps?.kinds.readNote
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
      lines.attach(attached)
    },
    revealLine: lines.reveal,
    measure: lines.measure,
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
      write = createNoteWriter(nextPath, deps.hostDeps?.kinds.writeNote, errorMessageRef)
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

function resolveCustomTabTarget(state: CustomTabState, registration: CustomTabRegistration) {
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
  deps: Pick<PluginHostDeps, 'window'>,
): void {
  state.measure()
  if (!id) return
  if (state.path) deps.window.getWorkspace()?.files.syncPath(id, state.path, state.title)
}

/** What a custom tab kind reads of the window, the context of its plugin, and whether it edits notes. */
type CustomTabKindDeps = Pick<PluginHostDeps, 'window' | 'kinds' | 'vault'> & {
  readonly styles: PluginStyles
  readonly getContext?: (() => PluginContext | undefined) | undefined
  readonly isNoteEditor?: boolean
}

/** What a custom tab kind takes of its registration: the kind, title and pane, and the hooks the window takes unchanged. */
export interface CustomTabRegistration extends Pick<
  TabKindRegistration,
  'kind' | 'title' | 'component'
> {
  readonly declared: DeclaredParts<object>
}

export function pickCustomTabParts(registration: TabKindRegistration): CustomTabRegistration {
  const { kind, title, component } = registration
  return {
    kind,
    ...(title === undefined ? {} : { title }),
    component,
    declared: pickDeclaredParts(registration),
  }
}

/** isNoteEditor marks a kind the manifest declares as the editor of the text of a note: its tabs hold the note in the store of open notes. */
export function createCustomTabKind(
  pluginId: string,
  registration: CustomTabRegistration,
  deps: CustomTabKindDeps,
): AnyTabKind {
  const { declared } = registration
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
      const buffer = deps.isNoteEditor ? deps.kinds.getNoteBuffer() : undefined
      if (!buffer) return createCustomTabState(targetPath, { pluginId, hostDeps: deps })
      return createBufferedNoteState(
        { targetPath, id, buffer },
        {
          pluginId,
          deps,
        },
      )
    },
    pane: createPluginPaneWrapper(pluginId, registration.component, deps),
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
