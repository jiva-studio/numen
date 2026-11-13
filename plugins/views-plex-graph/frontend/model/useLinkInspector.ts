/**
 * The link inspector of a plex tab: the connection between two drawn notes as the core reports it,
 * and the calls that save or remove it. The plugin holds the window state; the core decides the roles.
 */
import { ref } from 'vue'
import { getRelationErrorWords, type Connection } from '@numen/plugin-sdk'
import type { NodeIdMap } from '../lib/nodeIdMap'
import type { LinkInspectorRequest, PlexEditor, PlexView } from '../types'
import { areConnectionsEqual, isConnectionEmpty } from './connectionDraft'

const isConnected = (view: PlexView, first: string, second: string): boolean =>
  view.noteGraph.value?.connections.some(
    (one) =>
      (one.first === first && one.second === second) ||
      (one.first === second && one.second === first),
  ) ?? false

export function useLinkInspector(
  view: PlexView,
  options: {
    readonly editor: PlexEditor
    readonly showMessage: (text: string) => void
  },
  ...[map, getName]: [map: NodeIdMap, getName: (path: string) => string]
) {
  const { editor, showMessage } = options
  const linkInspector = ref<LinkInspectorRequest | null>(null)

  const openLinkInspector = async (pair: string, position: { x: number; y: number }) => {
    const [idA, idB] = pair.split(' ')
    const pathA = map.getNodePath(idA ?? '')
    const pathB = map.getNodePath(idB ?? '')
    if (!idA || !idB || !pathA || !pathB) return
    // The line to a reference is no connection: no record joins its two notes, so there is nothing to inspect.
    if (!isConnected(view, pathA, pathB)) return

    const read = await editor.getConnection(pathA, pathB)
    if (!read.ok) {
      showMessage(getRelationErrorWords(read.error))
      return
    }
    linkInspector.value = {
      nodeA: { id: idA, title: getName(pathA), path: pathA },
      nodeB: { id: idB, title: getName(pathB), path: pathB },
      connection: read.value,
      at: position,
    }
  }

  const dismissLinkInspector = () => {
    linkInspector.value = null
  }

  /** Shows the notes again once the vault changed, and closes the inspector where the change landed. */
  const finish = async (isLanded: boolean) => {
    const here = view.here.value
    if (here) await view.go(here)
    if (isLanded) dismissLinkInspector()
  }

  const saveLinkInspector = async (connection: Connection) => {
    const open = linkInspector.value
    if (!open) return

    if (areConnectionsEqual(connection, open.connection)) {
      dismissLinkInspector()
      return
    }

    const saved = await editor.saveConnection(connection)
    if (!saved.ok) showMessage(getRelationErrorWords(saved.error))
    await finish(saved.ok)
  }

  const removeLinkInspector = async () => {
    const open = linkInspector.value
    if (!open) return
    if (isConnectionEmpty(open.connection)) {
      dismissLinkInspector()
      return
    }

    const removed = await editor.removeConnection(
      open.nodeA.path,
      open.nodeB.path,
      open.connection.type,
    )
    if (!removed.ok) showMessage(getRelationErrorWords(removed.error))
    await finish(removed.ok)
  }

  return {
    linkInspector,
    openLinkInspector,
    dismissLinkInspector,
    saveLinkInspector,
    removeLinkInspector,
  }
}
