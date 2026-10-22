/**
 * The link inspector of a plex tab: the links the core reports between two notes, and the call that replaces them.
 */
import { ref } from 'vue'
import type { NodeIdMap } from '../lib/nodeIdMap'
import type { DirectionalLinkItem, LinkInspectorRequest, PlexEditor, PlexView } from '../types'
import { areLinksEqual, createInspectorLinks } from './linkInspector'

export function useLinkInspector(
  view: PlexView,
  editor: PlexEditor,
  ...[map, getName]: [map: NodeIdMap, getName: (path: string) => string]
) {
  const linkInspector = ref<LinkInspectorRequest | null>(null)

  const openLinkInspector = (pair: string, position: { x: number; y: number }) => {
    const [idA, idB] = pair.split(' ')
    const pathA = map.getNodePath(idA ?? '')
    const pathB = map.getNodePath(idB ?? '')
    if (!idA || !idB || !pathA || !pathB) return

    linkInspector.value = {
      nodeA: { id: idA, title: getName(pathA), path: pathA },
      nodeB: { id: idB, title: getName(pathB), path: pathB },
      links: createInspectorLinks(view.neighbourhood.value, pathA, pathB),
      at,
    }
  }

  const dismissLinkInspector = () => {
    linkInspector.value = null
  }

  const saveLinkInspector = async (rows: readonly DirectionalLinkItem[]) => {
    const open = linkInspector.value
    if (!open) return

    if (areLinksEqual(rows, open.links)) {
      dismissLinkInspector()
      return
    }

    const isReplaced = await editor.replaceLinks(
      open.nodeA.path,
      open.nodeB.path,
      rows.map(({ from, to, role, label }) => ({ from, to, relation: role, label })),
    )
    const here = view.here.value
    if (here) await view.go(here)
    if (isReplaced) dismissLinkInspector()
  }

  return { linkInspector, openLinkInspector, dismissLinkInspector, saveLinkInspector }
}
