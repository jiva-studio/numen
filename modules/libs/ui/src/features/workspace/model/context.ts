/**
 * What every branch and pane of one workspace shares, however deep it stands.
 * A branch holds branches, so there is no depth to thread anything through.
 */
import type { InjectionKey, Ref } from 'vue'
import type { NodeId, Tab, TabId } from '../lib/node'

export interface WorkspaceContext {
  /** The tab an identity stands for, and nothing for one the workspace has lost. */
  readonly tabOf: (id: TabId) => Tab | undefined
  /** The pane a tab opens into. */
  readonly focus: NodeId
  /** The least room a pane is worth drawing in. */
  readonly minimum: number
  readonly choose: (tab: TabId) => void
  readonly close: (tab: TabId) => void
  readonly lift: (tab: TabId, at: PointerEvent) => void
  readonly claim: (pane: NodeId) => void
  readonly resize: (branch: NodeId, sizes: readonly number[]) => void
  readonly show: (tab: TabId) => void
}

export const WORKSPACE_CONTEXT = Symbol('workspace') as InjectionKey<Ref<WorkspaceContext>>
