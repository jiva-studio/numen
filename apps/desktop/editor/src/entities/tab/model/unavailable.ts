import type { Component } from 'vue'
import UnavailablePane from './UnavailablePane.vue'
import type { AnyTabKind } from './kinds'
import { formatUnavailableKindTitle } from './unavailableText'

/** What a tab of an unavailable kind holds: the kind it was made for, its path, and the state its plugin saved. */
export interface UnavailableState {
  readonly kind: string
  readonly targetPath: string
  readonly saved: unknown
  readonly title?: string | undefined
}

export interface UnavailableKindOptions {
  readonly title?: string | undefined
  readonly icon?: Component | undefined
}

/** The placeholder kinds made here. */
const placeholders = new WeakSet<AnyTabKind>()

/** Whether a kind is the placeholder of a kind that is not registered. */
export const isUnavailableKind = (kind: AnyTabKind): boolean => placeholders.has(kind)

/** The last segment of a path, which names the tab of a file. */
const getFileName = (path: string): string => path.split('/').pop() ?? path

/**
 * The tab kind substituted for an unregistered kind, because its plugin is disabled or missing.
 * It keeps the kind, the path and the saved state, so the open tabs keep the tab as it was and
 * the plugin opens it again once it is enabled.
 */
export const createUnavailableKind = (
  kind: string,
  options?: UnavailableKindOptions,
): AnyTabKind => {
  const placeholder: AnyTabKind = {
    kind,
    ...(options?.icon ? { icon: options.icon } : {}),
    open: (targetPath: string, _id: string, saved?: unknown): UnavailableState => ({
      kind,
      targetPath,
      saved,
      ...(options?.title ? { title: options.title } : {}),
    }),
    getTitle: (state: unknown) => {
      const unavailable = state as UnavailableState
      return (
        unavailable.title ||
        getFileName(unavailable.targetPath) ||
        options?.title ||
        formatUnavailableKindTitle(kind)
      )
    },
    getOpenTab: (state: unknown) => ({ path: (state as UnavailableState).targetPath }),
    getState: (state: unknown) => (state as UnavailableState).saved,
    pane: UnavailablePane,
    identity: (targetPath: string) => targetPath,
  }
  placeholders.add(placeholder)
  return placeholder
}
