/**
 * What every branch and pane of one workspace shares, at any depth. A branch
 * holds branches, so nothing is threaded through by depth.
 */
import type { InjectionKey, Ref } from 'vue'
import type { NodeId, Tab, TabId } from '../lib/node'

export interface WorkspaceContext {
  /** The tab an identity refers to, or undefined for an identity absent from the workspace. */
  readonly findTab: (id: TabId) => Tab | undefined
  /** The pane a tab opens into. */
  readonly focus: NodeId
  /** The minimum size at which a pane is worth drawing. */
  readonly minimum: number
  readonly choose: (tab: TabId) => void
  readonly close: (tab: TabId) => void
  readonly lift: (tab: TabId, pointer: PointerEvent) => void
  readonly menu: (tab: TabId, pos: { x: number; y: number }) => void
  readonly claim: (pane: NodeId) => void
  readonly resize: (branch: NodeId, sizes: readonly number[]) => void
  readonly show: (tab: TabId) => void
}

export const WORKSPACE_CONTEXT = Symbol('workspace') as InjectionKey<Ref<WorkspaceContext>>
