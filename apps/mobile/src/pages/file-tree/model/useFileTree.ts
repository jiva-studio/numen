import { onMounted, ref } from 'vue'
import { startCore, type Core, type FileRow } from '@/entities/core'
import { formatError } from '@/shared/api/errors'

/** What the page draws of a file: the row, and whether its download is running. */
export interface TreeRow extends FileRow {
  readonly isDownloading: boolean
}

/**
 * The files of the folder the person is in, and what tapping one does: a folder is entered, a
 * downloaded file is opened, and a file that is not downloaded is downloaded first. `open` receives the path
 * of each file to open. A failure is reported as `errorMessage`.
 */
export function useFileTree(open: (path: string) => void) {
  const core = ref<Core | null>(null)
  const folder = ref('')
  const rows = ref<TreeRow[]>([])
  const errorMessage = ref('')

  async function show(next: string): Promise<void> {
    if (!core.value) return
    try {
      const listed = await core.value.listFiles(next)
      rows.value = listed.map((row) => ({ ...row, isDownloading: false }))
      folder.value = next
    } catch (why) {
      errorMessage.value = formatError(why)
    }
  }

  function change(path: string, changes: Partial<TreeRow>): void {
    rows.value = rows.value.map((row) => (row.path === path ? { ...row, ...changes } : row))
  }

  async function download(row: TreeRow): Promise<void> {
    if (!core.value) return
    change(row.path, { isDownloading: true })
    const answer = await core.value.fetchFile(row.path)
    if (!answer.ok) {
      change(row.path, { isDownloading: false })
      errorMessage.value = answer.error
      return
    }
    change(row.path, { isDownloading: false, isAbsent: false })
    open(row.path)
  }

  async function tap(row: TreeRow): Promise<void> {
    if (row.isFolder) return show(row.path)
    if (row.isDownloading) return
    if (!row.isAbsent) return open(row.path)
    return download(row)
  }

  async function dropDownload(row: TreeRow): Promise<void> {
    if (!core.value) return
    const answer = await core.value.dropDownload(row.path)
    if (!answer.ok) {
      errorMessage.value = answer.error
      return
    }
    change(row.path, { isAbsent: true })
  }

  onMounted(async () => {
    try {
      core.value = await startCore()
      await show('')
    } catch (why) {
      errorMessage.value = formatError(why)
    }
  })

  return { folder, rows, errorMessage, tap, dropDownload, show }
}
