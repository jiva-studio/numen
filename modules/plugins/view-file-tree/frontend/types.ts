import type { PluginContext } from '@numen/plugin-sdk'

export interface FileTreeEntry {
  readonly path: string
  readonly name: string
  readonly isFolder: boolean
  readonly children?: readonly FileTreeEntry[]
}

export interface FileTreeViewProps {
  readonly context?: PluginContext
  readonly initialPath?: string
  readonly entries?: readonly FileTreeEntry[]
}
