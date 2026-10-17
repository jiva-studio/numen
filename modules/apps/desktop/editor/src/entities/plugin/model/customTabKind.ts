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
import type { PluginHostDeps } from './types'

function getReaderTargetSource(path: string | undefined): 'recording' | 'book' | 'other' {
  if (!path) return 'other'
  if (/\.(mp3|m4a|wav|ogg|aac|flac)$/i.test(path)) return 'recording'
  if (/\.(epub)$/i.test(path)) return 'book'
  return 'other'
}

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

function hasCapability(
  capabilities: readonly string[] | undefined,
  targets: readonly string[],
): boolean {
  return capabilities ? targets.some((target) => capabilities.includes(target)) : false
}

function resolveReaderSource(
  registration: ReaderRegistration,
  path?: string,
): 'note' | 'book' | 'recording' | 'url' | 'other' {
  if (registration.source) {
    return registration.source as 'note' | 'book' | 'recording' | 'url' | 'other'
  }
  if (hasCapability(registration.capabilities, ['recording', 'media', 'audio'])) {
    return 'recording'
  }
  if (hasCapability(registration.capabilities, ['book', 'document'])) {
    return 'book'
  }
  if (hasCapability(registration.capabilities, ['note'])) {
    return 'note'
  }
  const metaSource = registration.metadata?.source
  if (typeof metaSource === 'string') {
    return metaSource as 'note' | 'book' | 'recording' | 'url' | 'other'
  }
  return getReaderTargetSource(path)
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
    open: (at: string) => {
      const filename = extractFilename(at)
      return reactive({
        path: at,
        file: at,
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
    identity: (at: string) => at,
    getTarget: (state: unknown) => {
      const s = state as { path?: string; title?: string } | undefined
      const path = s?.path
      return {
        ...(path !== undefined ? { file: path, path } : {}),
        title: s?.title ?? '',
        source: resolveReaderSource(registration, path),
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
  'body',
  'updateBody',
  'save',
])

function bindCustomTabHandlers(
  at: string,
  actions: Record<string, unknown>,
  handlers: Record<string, (at: string, ...args: unknown[]) => void> | undefined,
): void {
  if (!handlers) return
  for (const [name, handler] of Object.entries(handlers)) {
    if (FORBIDDEN_ACTION_NAMES.has(name) || actions[name]) continue
    actions[name] = (...args: unknown[]) => handler(at, ...args)
  }
}

function bindLegacyAddCard(
  at: string,
  actions: Record<string, unknown>,
  handler: ((at: string, values: readonly { field: string; text: string }[]) => void) | undefined,
): void {
  if (actions.addCard || !handler) return
  actions.addCard = (_stencil: string, values: readonly { field: string; text: string }[]) => {
    handler(at, values)
  }
}

function appendHostActions(
  at: string,
  actions: Record<string, unknown>,
  deps?: PluginHostDeps,
): void {
  if (!deps) return
  bindCustomTabHandlers(at, actions, deps.tabActionHandlers)
  if (deps.onTabAction && !actions.dispatch) {
    actions.dispatch = (action: string, ...args: unknown[]) =>
      deps.onTabAction!(action, at, ...args)
  }
  bindLegacyAddCard(at, actions, deps.onCustomCardAdded)
}

function createTabActions(
  at: string,
  registration: TabKindRegistration,
  deps?: PluginHostDeps,
): Record<string, unknown> {
  const actions: Record<string, unknown> = {}

  for (const [actionName, handler] of Object.entries(registration.actions ?? {})) {
    if (!FORBIDDEN_ACTION_NAMES.has(actionName)) {
      actions[actionName] = (...args: unknown[]) => handler(at, ...args)
    }
  }

  appendHostActions(at, actions, deps)
  return actions
}

interface CustomTabStateDeps {
  readonly pluginId: string
  readonly registration: TabKindRegistration
  readonly hostDeps?: PluginHostDeps | undefined
}

function createCustomTabState(at: string, deps: CustomTabStateDeps) {
  const defaultName = stripMarkdownExtension(extractFilename(at))
  const noteRef = ref({ path: at, body: '' })
  const errorMessageRef = ref('')
  let hasEdited = false
  let isInitialReadComplete = false

  if (deps.hostDeps?.readNote) {
    void deps.hostDeps
      .readNote(at)
      .then((result) => {
        isInitialReadComplete = true
        if (hasEdited) return
        if (typeof result === 'string') {
          noteRef.value = { path: at, body: result }
        } else if (result && typeof result === 'object') {
          if (result.body !== undefined) {
            noteRef.value = { path: at, body: result.body }
          }
          if (result.error) {
            errorMessageRef.value = result.error
          }
        }
      })
      .catch((err) => {
        isInitialReadComplete = true
        errorMessageRef.value = err instanceof Error ? err.message : String(err)
      })
  } else {
    isInitialReadComplete = true
  }
  return {
    at,
    path: at,
    pluginId: deps.pluginId,
    title: defaultName,
    note: noteRef,
    errorMessage: errorMessageRef,
    body: computed(() => noteRef.value.body),
    updateBody: (body: string) => {
      hasEdited = true
      noteRef.value = { path: at, body }
      void deps.hostDeps?.writeNote?.(at, body)
    },
    save: () => {
      if (!isInitialReadComplete && !hasEdited) return
      void deps.hostDeps?.writeNote?.(at, noteRef.value.body)
    },
    ...createTabActions(at, deps.registration, deps.hostDeps),
  }
}

function resolveTargetTitle(title?: string, defaultName?: string, fallback = ''): string {
  return title || defaultName || fallback
}

function resolveCustomTabTarget(state: unknown, registration: TabKindRegistration) {
  const s = state as { path?: string; at?: string; title?: string } | undefined
  const p = s?.path ?? s?.at
  const defaultName = stripMarkdownExtension(extractFilename(p))
  const title = resolveTargetTitle(s?.title, defaultName, registration.title || registration.kind)
  const source =
    (registration.source as 'note' | 'book' | 'recording' | 'url' | 'other' | undefined) ?? 'note'

  return {
    ...(p ? { file: p, path: p } : {}),
    title,
    source,
  }
}

export function createCustomTabKind(
  pluginId: string,
  registration: TabKindRegistration,
  deps?: PluginHostDeps & { readonly getContext?: () => PluginContext | undefined },
): AnyTabKind {
  return {
    kind: registration.kind,
    open: (at: string) => createCustomTabState(at, { pluginId, registration, hostDeps: deps }),
    getTitle: (state: unknown) => {
      const s = state as { title?: unknown; at?: string; path?: string } | undefined
      const titleVal = extractTitleValue(s?.title)
      if (titleVal) return titleVal
      const p = s?.path ?? s?.at
      const defaultName = stripMarkdownExtension(extractFilename(p))
      if (defaultName) return defaultName
      return registration.title || registration.kind
    },
    pane: createPluginPaneWrapper(pluginId, registration.component, deps?.getContext),
    identity: (at: string) => at || registration.kind,
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
    ...(registration.onClose
      ? { onClose: (state: unknown, id: string) => registration.onClose!(state, id) }
      : {}),
  }
}
