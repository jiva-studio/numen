import { computed, shallowRef } from 'vue'
import type { Row } from '@numen/ui'
import { ROOT } from '../lib/entries'
import type { Vault } from './vault'

/** The folders of the file tree that are open, and the rows they show. */
export function createFilesTree(vault: Vault, initiallyOpen: readonly string[]) {
  const open = shallowRef<ReadonlySet<string>>(new Set([ROOT, ...initiallyOpen]))

  const getRows = (folder: string): Row[] =>
    vault.read.list(folder).map((one) => ({
      id: one.path,
      name: one.name,
      hasChildren: one.isFolder,
      rows: one.isFolder && open.value.has(one.path) ? getRows(one.path) : [],
    }))

  const rows = computed(() => getRows(ROOT))
  const openRows = computed(() => [...open.value])

  const openFolder = (folder: string) => {
    open.value = new Set(open.value).add(folder)
  }

  const closeFolder = (folder: string) => {
    if (folder === ROOT) return
    const rest = new Set(open.value)
    rest.delete(folder)
    open.value = rest
  }

  return { rows, openRows, openFolder, closeFolder }
}
