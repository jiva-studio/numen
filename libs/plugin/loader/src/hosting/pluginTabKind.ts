/**
 * A tab kind that a plugin declares in full: the content of its tabs, and every
 * hook the window calls on a kind.
 */
import { defineComponent, h, onBeforeUnmount, onErrorCaptured, ref, type Component } from 'vue'
import type { TabKindRegistration } from '@numen/plugin-sdk'
import type { AnyTabKind, TabKind } from './tabKinds'
import { adoptPluginStyle, getPluginStyle } from '../styles'
import { parsePluginComponent, renderFallback, type PluginComponentSource } from './pane/pluginView'

/** A registration that declares the content of its tabs. */
export type DeclaredTabKindRegistration = TabKindRegistration & {
  open: NonNullable<TabKindRegistration['open']>
}

/** Whether the registration declares the content of its tabs. */
export const isDeclaredTabKind = (
  registration: TabKindRegistration,
): registration is DeclaredTabKindRegistration => typeof registration.open === 'function'

/** The hooks of a registration that the window takes unchanged. */
type DeclaredHook =
  | 'identity'
  | 'onShow'
  | 'receiveText'
  | 'onKeyPress'
  | 'getTarget'
  | 'getOpenTab'
  | 'onClose'
  | 'onDestroy'
  | 'navigate'
  | 'isAsking'
  | 'flushPending'
  | 'createFile'
  | 'getMark'
  | 'isTool'
  | 'icon'
  | 'placement'
  | 'getState'

/** The fields of an object that are set. */
type DefinedParts<T> = { [K in keyof T]?: Exclude<T[K], undefined> }

function pickDefined<T extends object>(parts: T): DefinedParts<T> {
  const defined: Partial<T> = {}
  for (const key of Object.keys(parts) as (keyof T)[]) {
    if (parts[key] !== undefined) defined[key] = parts[key]
  }
  return defined as DefinedParts<T>
}

/** The hooks of the registration the window takes unchanged, each only where the registration declares one. */
export function pickDeclaredHooks<State extends object>(
  registration: TabKindRegistration<State>,
): Partial<Pick<TabKind<State>, DeclaredHook>> {
  return pickDefined({
    identity: registration.identity,
    onShow: registration.onShow,
    receiveText: registration.receiveText,
    onKeyPress: registration.onKeyPress,
    getTarget: registration.getTarget,
    getOpenTab: registration.getOpenTab,
    onClose: registration.onClose,
    onDestroy: registration.onDestroy,
    navigate: registration.navigate,
    isAsking: registration.isAsking,
    flushPending: registration.flushPending,
    createFile: registration.createFile,
    getMark: registration.getMark,
    isTool: registration.isTool,
    icon: registration.icon,
    placement: registration.placement,
    getState: registration.getState,
  })
}

/**
 * The pane of a declared kind: its component, given the tab's state, beside the
 * stylesheet the plugin's manifest names. A component that fails is replaced by
 * the plugin's failure.
 */
function createDeclaredPane(
  pluginId: string,
  rawComponent: PluginComponentSource | null,
): Component {
  const component = parsePluginComponent(rawComponent)
  return defineComponent({
    // Stryker disable next-line StringLiteral: Vue component debug name
    name: `PluginKindPane[${pluginId}]`,
    // Stryker disable next-line ObjectLiteral,BooleanLiteral: Vue prop descriptor
    props: { state: { type: null, required: true } },
    setup(props) {
      // The sheet is adopted before the first render, so the pane is never drawn without it.
      const release =
        // Stryker disable next-line ConditionalExpression,EqualityOperator,StringLiteral: JSDOM always defines document
        typeof document === 'undefined' ? undefined : adoptPluginStyle(pluginId, document)
      onBeforeUnmount(() => release?.())
      const error = ref<Error | null>(null)
      onErrorCaptured((err) => {
        error.value = err instanceof Error ? err : new Error(String(err))
        return false
      })
      return () => {
        if (error.value) return renderFallback(pluginId, error.value)
        // Stryker disable next-line ConditionalExpression: h(null) also renders an empty comment vnode in Vue
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
export function getDeclaredTabKind(
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
