import type { InjectionKey, Ref } from 'vue'
import type { NodeId, Tab, TabId } from '../lib/node'
import type { StripEdge } from '../lib/edges'

export { NO_STRIP_EDGE } from '../lib/edges'

/** Shared state and actions provided to workspace branches and panes. */
export interface WorkspaceContext {
  readonly getStripEdge: (pane: NodeId) => StripEdge
  readonly findTab: (id: TabId) => Tab | undefined
  readonly focus: NodeId
  readonly minimum: number
  readonly choose: (tab: TabId) => void
  readonly close: (tab: TabId) => void
  readonly startTabPress: (tab: TabId, pointer: PointerEvent) => void
  readonly menu: (tab: TabId, pos: { x: number; y: number }) => void
  readonly claim: (pane: NodeId) => void
  readonly resize: (branch: NodeId, sizes: readonly number[]) => void
  readonly show: (tab: TabId) => void
}

/** Injection key for the reactive workspace context. */
export const WORKSPACE_CONTEXT = Symbol('workspace') as InjectionKey<Ref<WorkspaceContext>>
