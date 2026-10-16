/**
 * Tab kinds created dynamically for plugins and readers.
 */
import {
  computed,
  defineAsyncComponent,
  defineComponent,
  h,
  onErrorCaptured,
  onMounted,
  ref,
  Teleport,
  type Component,
} from 'vue'
import type { ReaderRegistration, TabKindRegistration } from '@numen/plugin-sdk'
import type { AnyTabKind } from '@/entities/tab/@x/plugin'
import { getPluginStyle } from './styles'
import type { PluginHostDeps } from './types'

type AsyncComponentRegistration = {
  type: 'async'
  loader: () => Promise<Component>
}

function isAsyncComponent(
  comp: unknown,
): comp is (() => Promise<Component>) | AsyncComponentRegistration {
  if (typeof comp === 'object' && comp !== null && (comp as { type?: unknown }).type === 'async') {
    return typeof (comp as { loader?: unknown }).loader === 'function'
  }
  if (typeof comp !== 'function') return false
  return !(
    'setup' in comp ||
    'render' in comp ||
    'props' in comp ||
    '__name' in comp ||
    (comp as { length?: number }).length! > 0
  )
}

function resolvePluginComponent(rawComponent: unknown): Component | null {
  if (!rawComponent) return null
  const comp =
    typeof rawComponent === 'object' && 'default' in rawComponent
      ? (rawComponent as { default: unknown }).default
      : rawComponent
  if (!comp) return null
  if (isAsyncComponent(comp)) {
    const loader = typeof comp === 'function' ? comp : comp.loader
    return defineAsyncComponent(loader)
  }
  return comp as Component
}

function extractTitleValue(title: unknown): string {
  if (typeof title === 'string') return title
  if (typeof title === 'object' && title !== null && 'value' in title) {
    return String((title as { value: unknown }).value)
  }
  return ''
}

function extractFilename(path: string | undefined): string {
  if (!path) return ''
  return path.split('/').pop() ?? path
}

function stripMarkdownExtension(filename: string): string {
  return filename.replace(/\.md$/, '')
}

function extractStatePath(state: unknown): string {
  if (typeof state !== 'object' || state === null) return ''
  const s = state as Record<string, unknown>
  const val = s.path ?? s.file ?? s.at
  return typeof val === 'string' ? val : ''
}

function getReaderTargetSource(path: string | undefined): 'recording' | 'book' | 'other' {
  if (!path) return 'other'
  if (/\.(mp3|m4a|wav|ogg|aac|flac)$/i.test(path)) return 'recording'
  if (/\.(epub)$/i.test(path)) return 'book'
  return 'other'
}

function attachPluginShadow(el: HTMLElement | null): ShadowRoot | null {
  if (!el) return null
  if (el.shadowRoot) return el.shadowRoot
  if (typeof el.attachShadow === 'function') {
    try {
      return el.attachShadow({ mode: 'open' })
    } catch {
      return el.shadowRoot
    }
  }
  return null
}

function renderFallback(pluginId: string, error: Error) {
  return h('div', { class: 'plugin-error-fallback p-4 text-red-500' }, [
    h('h3', { class: 'font-bold' }, `Plugin "${pluginId}" crashed`),
    h('pre', { class: 'text-xs mt-2' }, error.message),
  ])
}

interface PaneContentDeps {
  readonly pluginId: string
  readonly component: Component | null
  readonly state: unknown
  readonly shadowRoot: ShadowRoot | null
  readonly containerClass: string
}

function renderPaneContent(deps: PaneContentDeps) {
  if (!deps.component) return null
  const stateObj =
    typeof deps.state === 'object' && deps.state !== null
      ? (deps.state as Record<string, unknown>)
      : {}
  const path = extractStatePath(deps.state)
  const child = h(deps.component, {
    ...stateObj,
    ...(path ? { path, filePath: path, file: path } : {}),
    state: deps.state,
  })
  const styleUrl = getPluginStyle(deps.pluginId)
  const styleNode = styleUrl
    ? h('link', { rel: 'stylesheet', href: styleUrl, 'data-plugin-id': deps.pluginId })
    : null
  const inner = h('div', { class: deps.containerClass }, [child])

  if (deps.shadowRoot) {
    return h(Teleport, { to: deps.shadowRoot }, [styleNode, inner])
  }
  return [styleNode, inner]
}

export function createPluginPaneWrapper(
  pluginIdOrComponent: string | unknown,
  maybeComponent?: unknown,
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
        return h('div', { class: classes.wrapper, 'data-plugin-id': pluginId, ref: hostEl }, [
          isReady.value
            ? renderPaneContent({
                pluginId,
                component,
                state: props.state,
                shadowRoot: shadowRoot.value,
                containerClass: classes.container,
              })
            : null,
        ])
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
): AnyTabKind {
  const kindId = registration.id.startsWith('reader:')
    ? registration.id
    : `reader:${registration.id}`
  return {
    kind: kindId,
    open: (at: string) => {
      const filename = extractFilename(at)
      return {
        path: at,
        file: at,
        pluginId,
        title: filename,
      }
    },
    getTitle: (state: unknown) => {
      const s = state as { title?: unknown; path?: string } | undefined
      const titleVal = extractTitleValue(s?.title)
      if (titleVal) return titleVal
      const filename = extractFilename(s?.path)
      if (filename) return filename
      return registration.title || ''
    },
    pane: createPluginPaneWrapper(pluginId, registration.component),
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
  deps?: PluginHostDeps,
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
    pane: createPluginPaneWrapper(pluginId, registration.component),
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
