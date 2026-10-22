import { effectScope, type EffectScope } from 'vue'
import type { TabKindRegistration } from '../types'

// MockTabWindow opens the tabs of the kinds registered with it the way a window does: one scope to a tab,
// a tab per identity the kind gives, and a close the kind may refuse.
export interface MockTabWindow {
  register<State>(registration: TabKindRegistration<State>): void
  // The id of the tab opened on `at`, which is the tab it already has where the kind gives the same identity.
  openTab(kind: string, at?: string): Promise<string>
  // The state of one tab, and nothing where the window holds no such tab.
  getTabState<State>(id: string): State | null
  // The tabs of a kind, in the order they were opened.
  each<State>(kind: string): readonly { id: string; state: State }[]
  // The person closes a tab; true where the kind let it go at once.
  requestClose(id: string): boolean
  // A tab that took its own close goes now. This is `closeTab` of the plugin's context.
  finishClose(id: string): void
}

// createMockTabWindow builds a window that holds tabs in memory.
export function createMockTabWindow(): MockTabWindow {
  const kinds = new Map<string, TabKindRegistration<unknown>>()
  const open = new Map<string, { kind: TabKindRegistration<unknown>; state: unknown }>()
  const scopes = new Map<string, EffectScope>()

  const finishClose = (id: string): void => {
    open.delete(id)
    scopes.get(id)?.stop()
    scopes.delete(id)
  }

  return {
    register: <State>(registration: TabKindRegistration<State>) => {
      kinds.set(registration.kind, registration as TabKindRegistration<unknown>)
    },
    openTab: async (kind, targetPath = '') => {
      const registration = kinds.get(kind)
      if (!registration?.open) throw new Error(`no kind of tab declares ${kind}`)
      const id = `${kind}:${registration.identity ? registration.identity(targetPath) : crypto.randomUUID()}`
      if (open.has(id)) return id
      const scope = effectScope(true)
      const state = scope.run(() => registration.open?.(targetPath, id))
      scopes.set(id, scope)
      open.set(id, { kind: registration, state })
      return id
    },
    getTabState: <State>(id: string) => (open.get(id)?.state as State | undefined) ?? null,
    each: <State>(kind: string) =>
      [...open]
        .filter(([, one]) => one.kind.kind === kind)
        .map(([id, one]) => ({ id, state: one.state as State })),
    requestClose: (id) => {
      const one = open.get(id)
      if (!one) return true
      if (one.kind.onClose && !one.kind.onClose(one.state, id)) return false
      finishClose(id)
      return true
    },
    finishClose,
  }
}
