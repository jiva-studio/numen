import { computed, onMounted, onUnmounted, type Component, type ComputedRef, type WritableComputedRef } from 'vue'
import type { RowMarker } from '@numen/ui'
import type { FilesTabState } from '../types'

type MenuItems = ReturnType<FilesTabState['menu']['getItems']>

export interface FilesTabView {
  readonly dropTarget: ComputedRef<RowMarker>
  readonly renamingPath: WritableComputedRef<string | null>
  readonly items: ComputedRef<MenuItems>
  readonly getRowIcon: (id: string, isOpen: boolean) => Component
  readonly getItemIcon: (id: string) => Component | null
}

/**
 * What the files tab shows of its state: the mark the window's drag and drop reads, the row being renamed, the
 * items of the open menu and the icons. The tree is refreshed whenever the window regains focus.
 */
export function useFilesTabView(getState: () => FilesTabState): FilesTabView {
  const dropTarget = computed<RowMarker>(() => ({
    attribute: 'data-file-drop-target',
    getAttributeValue: (row: string | null) => getState().transfer.getFolderFor(row),
  }))

  const renamingPath = computed({
    get: () => getState().renaming.path.value,
    set: (row: string | null) => getState().renaming.setPath(row),
  })

  const items = computed(() =>
    getState().menu.getItems(getState().menu.request.value?.path ?? null),
  )

  const getRowIcon = (id: string, isOpen: boolean): Component => {
    const state = getState()
    return state.entries.getEntryIcon(state.list.queries.getEntryAt(id), isOpen)
  }

  const getItemIcon = (id: string): Component | null => getState().menu.getIcon(id)

  const refreshTree = () => {
    getState().list.sync.refresh().catch(getState().reportFailure)
  }
  onMounted(() => globalThis.addEventListener('focus', refreshTree))
  onUnmounted(() => globalThis.removeEventListener('focus', refreshTree))

  return { dropTarget, renamingPath, items, getRowIcon, getItemIcon }
}
