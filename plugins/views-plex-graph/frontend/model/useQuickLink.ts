/**
 * The quick link of a plex tab: a note joined to, or made for, the one a gesture started from.
 */
import { ref, type Ref } from 'vue'
import { getRelationErrorWords } from '@numen/plugin-sdk'
import type { NodeIdMap } from '../lib/nodeIdMap'
import { findRole } from '../lib/roles'
import type { PlexTabDeps, QuickLinkRequest, PlexView } from '../types'

export function useQuickLink(view: PlexView, deps: PlexTabDeps, map: NodeIdMap) {
  const quickLink = ref<QuickLinkRequest | null>(null)
  const isNavigatingOnCreate = (deps.isNavigatingOnCreate ?? ref(true)) as Ref<boolean>

  const openQuickLink = (request: QuickLinkRequest) => {
    quickLink.value = request
  }

  const dismissQuickLink = () => {
    quickLink.value = null
  }

  const searchNotes = async (
    query: string,
  ): Promise<readonly { path: string; title: string }[]> => {
    if (!deps.searchNames) return []
    const results = await deps.searchNames(query)
    return results.map((item) => ({ path: item.path, title: item.title }))
  }

  const confirmQuickLink = async (titleOrPath: string, isExisting: boolean) => {
    const active = quickLink.value
    quickLink.value = null
    if (!active) return

    const fromPath = map.getNodePath(active.from)
    if (!fromPath) return

    const role = findRole(deps.creatable, active.relation)
    if (!role) return

    if (isExisting) {
      const joined = await deps.editor.join(fromPath, titleOrPath, { role })
      if (!joined.ok) return deps.showMessage(getRelationErrorWords(joined.error))
      await view.go(isNavigatingOnCreate.value ? titleOrPath : fromPath)
    } else {
      const title = titleOrPath.trim()
      if (!title) return
      const created = await deps.editor.createRelated({ from: fromPath, role, title })
      if (!created.ok) return deps.showMessage(getRelationErrorWords(created.error))
      await view.go(isNavigatingOnCreate.value ? created.value : fromPath)
    }
  }

  return {
    quickLink,
    isNavigatingOnCreate,
    openQuickLink,
    dismissQuickLink,
    searchNotes,
    confirmQuickLink,
  }
}
