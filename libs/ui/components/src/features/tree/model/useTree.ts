import { createRefRegistry, type RefRegistry, type RefTarget } from '@/shared/model/refs'
import type { RowId } from '../lib/row'
import {
  createTreeGestures,
  createTreeRenaming,
  type TreeGesturesOptions,
  type TreeGesturesState,
} from './gestures'

/** Function refs for the drawn rows, one per row id. */
export function useRowRefs(
  onChange: (key: RowId, element: RefTarget) => void,
): RefRegistry<RowId> {
  return createRefRegistry<RowId>(onChange)
}

/** The pointer and keyboard handlers of a tree. */
export function useTreeGestures(options: TreeGesturesOptions): TreeGesturesState {
  return createTreeGestures(options)
}

/** The rename handlers of a tree. */
export function useTreeRenaming(
  ...args: Parameters<typeof createTreeRenaming>
): ReturnType<typeof createTreeRenaming> {
  return createTreeRenaming(...args)
}
