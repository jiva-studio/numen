/**
 * A tab kind that a plugin declares in full: the content of its tabs, and every
 * hook the window calls on a kind.
 */
import { defineComponent, h, onBeforeUnmount, onErrorCaptured, ref, type Component } from 'vue'
import type { TabKindRegistration } from '@numen/plugin-sdk'
import type { AnyTabKind } from '@/shared/model/tabKinds'
import { adoptPluginStyle, getPluginStyle } from '@numen/plugin-host'
import { renderFallback, resolvePluginComponent } from './styles'

/** A registration that declares the content of its tabs. */
export type DeclaredTabKindRegistration = TabKindRegistration & {
  open: NonNullable<TabKindRegistration['open']>
}

/** Whether the registration declares the content of its tabs. */
export const isDeclaredTabKind = (
  registration: TabKindRegistration,
): registration is DeclaredTabKindRegistration => typeof registration.open === 'function'

/** The hooks a registration hands the window as they are, each only where it declares one. */
const OPTIONAL_HOOKS = [
  'identity',
  'onShow',
  'receiveText',
  'onKeyPress',
  'getTarget',
  'getOpenTab',
  'onClose',
  'onDestroy',
  'navigate',
  'isAsking',
  'flushPending',
  'createFile',
  'getMark',
  'isTool',
  'icon',
  'placement',
  'getState',
] as const satisfies readonly (keyof TabKindRegistration & keyof AnyTabKind)[]

/** The hooks of the registration the window takes unchanged. */
export function pickDeclaredHooks(registration: TabKindRegistration): Partial<AnyTabKind> {
  const hooks: Record<string, unknown> = {}
  for (const name of OPTIONAL_HOOKS) {
    if (registration[name] !== undefined) hooks[name] = registration[name]
  }
  return hooks as Partial<AnyTabKind>
}

/**
 * The pane of a declared kind: its component, given the tab's state, beside the
 * stylesheet the plugin's manifest names. A component that fails is replaced by
 * the plugin's failure.
 */
export function createDeclaredPane(pluginId: string, rawComponent: unknown): Component {
  const component = resolvePluginComponent(rawComponent)
  return defineComponent({
    name: `PluginKindPane[${pluginId}]`,
    props: { state: { type: null, required: true } },
    setup(props) {
      // The sheet is adopted before the first render, so the pane is never drawn without it.
      const release =
        typeof document === 'undefined' ? undefined : adoptPluginStyle(pluginId, document)
      onBeforeUnmount(() => release?.())
      const error = ref<Error | null>(null)
      onErrorCaptured((err) => {
        error.value = err instanceof Error ? err : new Error(String(err))
        return false
      })
      return () => {
        if (error.value) return renderFallback(pluginId, error.value)
        if (!component) return null
        const node = h(component, { state: props.state })
        const styleUrl = release ? undefined : getPluginStyle(pluginId)
        if (!styleUrl) return node
        return [h('link', { rel: 'stylesheet', href: styleUrl, 'data-plugin-id': pluginId }), node]
      }
    },
  })
}

/** The kind a plugin declared whole, as the window keeps it. */
export function createDeclaredTabKind(
  pluginId: string,
  registration: DeclaredTabKindRegistration,
): AnyTabKind {
  const { kind, title, getTitle } = registration
  return {
    ...pickDeclaredHooks(registration),
    kind,
    pluginId,
    open: (targetPath, id, saved) => registration.open(targetPath, id, saved),
    getTitle: (state) => getTitle?.(state) ?? title ?? kind,
    pane: createDeclaredPane(pluginId, registration.component),
  }
}
