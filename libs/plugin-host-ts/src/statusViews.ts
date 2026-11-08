/**
 * The status bar views plugins register: components the window draws in its status bar, in the order
 * the plugins registered them. A plugin registers one component per id; registering an id again
 * replaces the earlier component.
 */
import { shallowRef, type ShallowRef } from 'vue'
import type { Disposable, StatusViewRegistration } from '@numen/plugin-sdk'

/** A status bar view with the plugin that registered it. */
export interface DeclaredStatusView {
  readonly pluginId: string
  readonly registration: StatusViewRegistration
}

export interface StatusViewRegistry {
  readonly views: Readonly<ShallowRef<readonly DeclaredStatusView[]>>
  registerStatusView(pluginId: string, registration: StatusViewRegistration): Disposable
  /** Removes every view the plugin registered. */
  disposePlugin(pluginId: string): void
}

export function createStatusViewRegistry(): StatusViewRegistry {
  const views = shallowRef<readonly DeclaredStatusView[]>([])

  const registerStatusView = (
    pluginId: string,
    registration: StatusViewRegistration,
  ): Disposable => {
    const declared: DeclaredStatusView = { pluginId, registration }
    views.value = [
      ...views.value.filter(
        (one) => one.pluginId !== pluginId || one.registration.id !== registration.id,
      ),
      declared,
    ]
    return {
      dispose: () => {
        views.value = views.value.filter((one) => one !== declared)
      },
    }
  }

  const disposePlugin = (pluginId: string): void => {
    views.value = views.value.filter((one) => one.pluginId !== pluginId)
  }

  return { views, registerStatusView, disposePlugin }
}
