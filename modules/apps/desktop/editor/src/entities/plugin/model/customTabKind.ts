/**
 * Tab kinds created dynamically for plugins and readers.
 */
import {
  computed,
  defineComponent,
  onErrorCaptured,
  onMounted,
  reactive,
  ref,
  type Component,
} from 'vue'
import type { PluginContext, ReaderRegistration, TabKindRegistration } from '@numen/plugin-sdk'
import type { AnyTabKind } from '@/entities/tab/@x/plugin'
import {
  attachPluginShadow,
  extractFilename,
  extractTitleValue,
  renderFallback,
  renderPaneContent,
  renderWrapper,
  resolvePluginComponent,
  stripMarkdownExtension,
} from './styles'
import { createNoteWriter, startNoteRead } from './customTabNote'
import type { PluginHostDeps } from './types'
import { pickDeclaredHooks } from './pluginTabKind'

export function createPluginPaneWrapper(
  pluginIdOrComponent: string | unknown,
  maybeComponent?: unknown,
  getContext?: () => PluginContext | undefined,
): Component {
  const pluginId = typeof pluginIdOrComponent === 'string' ? pluginIdOrComponent : 'plugin'
  const rawComponent = maybeComponent !== undefined ? maybeComponent : pluginIdOrComponent
  const component = resolvePluginComponent(rawComponent)
  const sanitizedId = pluginId.replace(/[^a-zA-Z0-9_-]/g, '-')
  const classes = {
    wrapper: `plugin-pane-wrapper plugin-pane-${sanitizedId}`,
    container: `plugin-view-container plugin-view-${sanitizedId}`,
  }

  return defineComponent({
    name: `PluginPaneWrapper[${pluginId}]`,
    props: { state: { type: null, required: true } },
    setup(props) {
      const error = ref<Error | null>(null)
      const hostEl = ref<HTMLElement | null>(null)
      const shadowRoot = ref<ShadowRoot | null>(null)
      const isReady = ref(false)

      onErrorCaptured((err) => {
        error.value = err instanceof Error ? err : new Error(String(err))
        return false
      })

      onMounted(() => {
        shadowRoot.value = attachPluginShadow(hostEl.value)
        isReady.value = true
      })

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
            })
          : null
        return renderWrapper({ wrapperClass: classes.wrapper, pluginId, hostEl, content })
      }
    },
  })
}

export function createReaderTabKind(
  pluginId: string,
  registration: ReaderRegistration,
  getContext?: () => PluginContext | undefined,
): AnyTabKind {
  const kindId = registration.id.startsWith('reader:')
    ? registration.id
    : `reader:${registration.id}`
  return {
    kind: kindId,
    pluginId,
    open: (targetPath: string) => {
      const filename = extractFilename(targetPath)
      return reactive({
        path: targetPath,
        file: targetPath,
        pluginId,
        title: filename,
        spans: [] as readonly unknown[],
      })
    },
    onShow: (state: unknown) => {
      const s = state as { measure?: () => void; focusTab?: () => void } | undefined
      s?.measure?.()
      s?.focusTab?.()
    },
    onKeyPress: (state: unknown, event: KeyboardEvent) => {
      const s = state as { handleKeyPress?: (e: KeyboardEvent) => boolean } | undefined
      return s?.handleKeyPress?.(event) ?? false
    },
    getTitle: (state: unknown) => {
      const s = state as { title?: unknown; path?: string } | undefined
      const titleVal = extractTitleValue(s?.title)
      if (titleVal) return titleVal
      const filename = extractFilename(s?.path)
      if (filename) return filename
      return registration.title || ''
    },
    pane: createPluginPaneWrapper(pluginId, registration.component, getContext),
    identity: (targetPath: string) => targetPath,
    getTarget: (state: unknown) => {
      const s = state as { path?: string; title?: string } | undefined
      const path = s?.path
      return {
        ...(path !== undefined ? { file: path, path } : {}),
        title: s?.title ?? '',
        source: registration.source,
      }
    },
    getOpenTab: (state: unknown) => ({ path: (state as { path?: string })?.path ?? '' }),
  }
}

const FORBIDDEN_ACTION_NAMES = new Set([
  'at',
  'path',
  'file',
  'pluginId',
  'title',
  'note',
  'errorMessage',
  'isLoading',
  'body',
  'updateBody',
  'save',
])

function bindCustomTabHandlers(
  tabId: string,
  actions: Record<string, unknown>,
  handlers: Record<string, (targetPath: string, ...args: unknown[]) => void> | undefined,
): void {
  if (!handlers) return
  for (const [name, handler] of Object.entries(handlers)) {
    if (FORBIDDEN_ACTION_NAMES.has(name) || actions[name]) continue
    actions[name] = (...args: unknown[]) => handler(tabId, ...args)
  }
}

function appendHostActions(
  tabId: string,
  actions: Record<string, unknown>,
  deps?: PluginHostDeps,
): void {
  if (!deps) return
  bindCustomTabHandlers(tabId, actions, deps.tabActionHandlers)
  if (deps.onTabAction && !actions.dispatch) {
    actions.dispatch = (action: string, ...args: unknown[]) =>
      deps.onTabAction!(action, tabId, ...args)
  }
}

function createTabActions(
  tabId: string,
  registration: TabKindRegistration,
  deps?: PluginHostDeps,
): Record<string, unknown> {
  const actions: Record<string, unknown> = {}

  for (const [actionName, handler] of Object.entries(registration.actions ?? {})) {
    if (!FORBIDDEN_ACTION_NAMES.has(actionName)) {
      actions[actionName] = (...args: unknown[]) => handler(tabId, ...args)
    }
  }

  appendHostActions(tabId, actions, deps)
  return actions
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
  const isLoading = ref(deps.hostDeps?.readNote !== undefined)
  let hasEdited = false
  const write = createNoteWriter(targetPath, deps.hostDeps?.writeNote, errorMessageRef)

  if (deps.hostDeps?.readNote) {
    startNoteRead(targetPath, deps.hostDeps.readNote, {
      note: noteRef,
      errorMessage: errorMessageRef,
      isLoading,
      hasEdited: () => hasEdited,
    })
  }
  return {
    at: targetPath,
    path: targetPath,
    pluginId: deps.pluginId,
    title: defaultName,
    note: noteRef,
    errorMessage: errorMessageRef,
    isLoading,
    body: computed(() => noteRef.value.body),
    updateBody: (body: string) => {
      hasEdited = true
      noteRef.value = { path: targetPath, body }
      write(body)
    },
    save: () => {
      if (isLoading.value && !hasEdited) return
      write(noteRef.value.body)
    },
    ...createTabActions(targetPath, deps.registration, deps.hostDeps),
  }
}

function resolveTargetTitle(title?: string, defaultName?: string, fallback = ''): string {
  return title || defaultName || fallback
}

function resolveCustomTabTarget(state: unknown, registration: TabKindRegistration) {
  const s = state as { path?: string; at?: string; title?: string } | undefined
  const stateFile = s?.path ?? s?.at
  const defaultName = stripMarkdownExtension(extractFilename(stateFile))
  const title = resolveTargetTitle(s?.title, defaultName, registration.title || registration.kind)
  const source =
    (registration.source as 'note' | 'book' | 'recording' | 'url' | 'other' | undefined) ?? 'note'

  return {
    ...(stateFile ? { file: stateFile, path: stateFile } : {}),
    title,
    source,
  }
}

export function createCustomTabKind(
  pluginId: string,
  registration: TabKindRegistration,
  deps?: PluginHostDeps & {
    readonly getContext?: (() => PluginContext | undefined) | undefined
  },
): AnyTabKind {
  return {
    kind: registration.kind,
    pluginId,
    open: (targetPath: string) =>
      createCustomTabState(targetPath, { pluginId, registration, hostDeps: deps }),
    getTitle: (state: unknown) => {
      const s = state as { title?: unknown; at?: string; path?: string } | undefined
      const titleVal = extractTitleValue(s?.title)
      if (titleVal) return titleVal
      const stateFile = s?.path ?? s?.at
      const defaultName = stripMarkdownExtension(extractFilename(stateFile))
      if (defaultName) return defaultName
      return registration.title || registration.kind
    },
    pane: createPluginPaneWrapper(pluginId, registration.component, deps?.getContext),
    identity: (targetPath: string) => targetPath || registration.kind,
    onShow: (state: unknown) => {
      const s = state as { measure?: () => void; editor?: { measure?: () => void } } | undefined
      s?.measure?.()
      s?.editor?.measure?.()
    },
    getTarget: (state: unknown) => resolveCustomTabTarget(state, registration),
    getOpenTab: (state: unknown) => {
      const s = state as { path?: string; at?: string } | undefined
      return { path: s?.path ?? s?.at ?? '' }
    },
    ...pickDeclaredHooks(registration),
  }
}
