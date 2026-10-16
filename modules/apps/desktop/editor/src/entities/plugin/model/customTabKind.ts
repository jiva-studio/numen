/**
 * Tab kinds created dynamically for plugins and readers.
 */
import {
  computed,
  defineAsyncComponent,
  defineComponent,
  h,
  onErrorCaptured,
  ref,
  type Component,
} from 'vue'
import type { ReaderRegistration, TabKindRegistration } from '@numen/plugin-sdk'
import type { AnyTabKind } from '@/entities/tab/@x/plugin'
import type { PluginHostDeps } from './types'

function isAsyncComponent(comp: unknown): comp is () => Promise<Component> {
  if (typeof comp !== 'function') return false
  return !('setup' in comp || 'render' in comp || 'props' in comp || (comp as { length?: number }).length! > 0)
}

function resolvePluginComponent(rawComponent: unknown): Component | null {
  if (!rawComponent) return null
  const comp =
    typeof rawComponent === 'object' && 'default' in rawComponent
      ? (rawComponent as { default: unknown }).default
      : rawComponent
  if (!comp) return null
  if (isAsyncComponent(comp)) {
    return defineAsyncComponent(comp)
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

export function createPluginPaneWrapper(pluginIdOrComponent: string | unknown, maybeComponent?: unknown): Component {
  const pluginId = typeof pluginIdOrComponent === 'string' ? pluginIdOrComponent : 'plugin'
  const rawComponent = maybeComponent !== undefined ? maybeComponent : pluginIdOrComponent
  const component = resolvePluginComponent(rawComponent)

  return defineComponent({
    name: `PluginPaneWrapper[${pluginId}]`,
    props: {
      state: {
        type: null,
        required: true,
      },
    },
    setup(props) {
      const error = ref<Error | null>(null)
      onErrorCaptured((err) => {
        error.value = err instanceof Error ? err : new Error(String(err))
        return false
      })
      return () => {
        if (error.value) {
          return h('div', { class: 'plugin-error-fallback p-4 text-red-500' }, [
            h('h3', { class: 'font-bold' }, `Plugin "${pluginId}" crashed`),
            h('pre', { class: 'text-xs mt-2' }, error.value.message),
          ])
        }
        if (!component) return null
        const stateObj =
          typeof props.state === 'object' && props.state !== null
            ? (props.state as Record<string, unknown>)
            : {}
        const path = extractStatePath(props.state)
        return h(component as Component, {
          ...stateObj,
          ...(path ? { path, filePath: path, file: path } : {}),
          state: props.state,
        })
      }
    },
  })
}

export function createReaderTabKind(pluginId: string, registration: ReaderRegistration): AnyTabKind {
  const kindId = registration.id.startsWith('reader:') ? registration.id : `reader:${registration.id}`
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
      const source = getReaderTargetSource(path)
      return {
        ...(path !== undefined ? { file: path, path } : {}),
        title: s?.title ?? '',
        source,
      }
    },
    getOpenTab: (state: unknown) => ({ path: (state as { path?: string })?.path ?? '' }),
  }
}

function createCustomTabState(at: string, pluginId: string, deps?: PluginHostDeps) {
  const defaultName = stripMarkdownExtension(extractFilename(at))
  const noteRef = ref({ path: at, body: '' })
  const errorMessageRef = ref('')
  if (deps?.readNote) {
    void deps.readNote(at).then((result) => {
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
  }
  return {
    at,
    path: at,
    pluginId,
    title: defaultName,
    note: noteRef,
    errorMessage: errorMessageRef,
    body: computed(() => noteRef.value.body),
    updateBody: (body: string) => {
      noteRef.value = { path: at, body }
      void deps?.writeNote?.(at, body)
    },
    save: () => {
      void deps?.writeNote?.(at, noteRef.value.body)
    },
    addCard: (_stencil: string, values: readonly { field: string; text: string }[]) => {
      deps?.onCustomCardAdded?.(at, values)
    },
  }
}

export function createCustomTabKind(
  pluginId: string,
  registration: TabKindRegistration,
  deps?: PluginHostDeps,
): AnyTabKind {
  return {
    kind: registration.kind,
    open: (at: string) => createCustomTabState(at, pluginId, deps),
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
    getTarget: (state: unknown) => {
      const s = state as { path?: string; at?: string; title?: string } | undefined
      const p = s?.path ?? s?.at ?? ''
      const defaultName = stripMarkdownExtension(extractFilename(p))
      return {
        ...(p ? { file: p, path: p } : {}),
        title: s?.title ?? defaultName ?? registration.title ?? registration.kind,
        source: 'note' as const,
      }
    },
    getOpenTab: (state: unknown) => {
      const s = state as { path?: string; at?: string } | undefined
      return { path: s?.path ?? s?.at ?? '' }
    },
    ...(registration.onClose
      ? { onClose: (state: unknown, id: string) => registration.onClose!(state, id) }
      : {}),
  }
}
