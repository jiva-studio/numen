import type { PluginContext } from '@numen/plugin-sdk'
import type { PlexTabState } from '@/pages/plex-graph/types'

export type {
  MenuRequest,
  QuickLinkRequest,
  LinkDirectionMode,
  DirectionalLinkItem,
  LinkInspectorNode,
  LinkInspectorRequest,
  PlexEditor,
  OpenNoteOptions,
  PlexTabDeps,
  PlexViewState,
  PlexMenuState,
  PlexHeadingsState,
  PlexNodeActions,
  PlexQuickLinkState,
  PlexLinkInspectorState,
  PlexTabState,
} from '@/pages/plex-graph/types'

export interface PlexGraphViewProps {
  readonly state: PlexTabState
  readonly context?: PluginContext
}
