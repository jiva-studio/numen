/**
 * A tab kind a plugin declares whole: what its tabs hold, and every hook the
 * window asks of a kind.
 */
import { defineComponent, h, onErrorCaptured, ref, type Component } from 'vue'
import type { TabKindRegistration } from '@numen/plugin-sdk'
import type { AnyTabKind } from '@/entities/tab/@x/plugin'
import { renderFallback, resolvePluginComponent } from './styles'

/** A registration that says what its tabs hold. */
export type DeclaredTabKindRegistration = TabKindRegistration & {
  open: NonNullable<TabKindRegistration['open']>
}

/** Whether the registration says what its tabs hold. */
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
  'isTool',
  'icon',
  'placement',
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
 * The pane of a declared kind: its component, given the tab's state and nothing
 * else. A component that fails is replaced by the plugin's failure.
 */
export function createDeclaredPane(pluginId: string, rawComponent: unknown): Component {
  const component = resolvePluginComponent(rawComponent)
  return defineComponent({
    name: `PluginKindPane[${pluginId}]`,
    props: { state: { type: null, required: true } },
    setup(props) {
      const error = ref<Error | null>(null)
      onErrorCaptured((err) => {
        error.value = err instanceof Error ? err : new Error(String(err))
        return false
      })
      return () => {
        if (error.value) return renderFallback(pluginId, error.value)
        return component ? h(component, { state: props.state }) : null
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
    open: (at, id) => registration.open(at, id),
    getTitle: (state) => getTitle?.(state) ?? title ?? kind,
    pane: createDeclaredPane(pluginId, registration.component),
  }
}
