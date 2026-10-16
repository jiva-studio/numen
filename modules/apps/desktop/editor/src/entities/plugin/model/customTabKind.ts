/**
 * Tab kinds created dynamically for plugins and readers.
 */
import { defineComponent, h, onErrorCaptured, ref, type Component } from 'vue'
import type { ReaderRegistration, TabKindRegistration } from '@numen/plugin-sdk'
import type { AnyTabKind } from '@/entities/tab/@x/plugin'
import type { PluginHostDeps } from './types'

export function createPluginPaneWrapper(pluginIdOrComponent: string | unknown, maybeComponent?: unknown): Component {
  const pluginId = typeof pluginIdOrComponent === 'string' ? pluginIdOrComponent : 'plugin'
  const component = maybeComponent !== undefined ? maybeComponent : pluginIdOrComponent
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
        return h(component as Component, {
          ...stateObj,
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
    open: (at: string) => ({
      path: at,
      file: at,
      pluginId,
      title: registration.title ?? at.split('/').pop() ?? at,
    }),
    getTitle: (state: unknown) => {
      const s = state as { title?: string; path?: string } | undefined
      return registration.title ?? (s?.title || s?.path?.split('/').pop() || s?.path || '')
    },
    pane: createPluginPaneWrapper(pluginId, registration.component),
    identity: (at: string) => at,
    getTarget: (state: unknown) => {
      const path = (state as { path?: string } | undefined)?.path
      const isAudio = path ? /\.(mp3|m4a|wav|ogg|aac)$/i.test(path) : false
      return {
        ...(path !== undefined ? { file: path } : {}),
        source: isAudio ? ('recording' as const) : ('other' as const),
      }
    },
    getOpenTab: (state: unknown) => ({ path: (state as { path?: string })?.path ?? '' }),
  }
}

export function createCustomTabKind(
  pluginId: string,
  registration: TabKindRegistration,
  deps?: PluginHostDeps,
): AnyTabKind {
  return {
    kind: registration.kind,
    open: (at: string) => ({
      at,
      path: at,
      pluginId,
      title: registration.title ?? registration.kind,
      note: { value: { path: at } },
      addCard: (_stencil: string, values: readonly { field: string; text: string }[]) => {
        deps?.onCustomCardAdded?.(at, values)
      },
    }),
    getTitle: (state: unknown) => {
      const s = state as { title?: string; at?: string } | undefined
      return registration.title ?? (s?.title || s?.at || registration.kind)
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
      return {
        ...(p ? { file: p, path: p } : {}),
        title: s?.title ?? registration.title ?? registration.kind,
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
