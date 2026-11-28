import { onBeforeUnmount, onMounted, ref } from 'vue'
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

  let latestShow = 0
  let isDisposed = false

  async function show(next: string): Promise<void> {
    if (!core.value) return
    latestShow += 1
    const mine = latestShow
    errorMessage.value = ''
    try {
      const listed = await core.value.listFiles(next)
      if (isDisposed || mine !== latestShow) return
      rows.value = listed.map((row) => ({ ...row, isDownloading: false }))
      folder.value = next
    } catch (why) {
      if (!isDisposed && mine === latestShow) errorMessage.value = formatError(why)
    }
  }

  function change(path: string, changes: Partial<TreeRow>): void {
    rows.value = rows.value.map((row) => (row.path === path ? { ...row, ...changes } : row))
  }

  async function fetchRow(path: string): Promise<boolean> {
    if (!core.value) return false
    try {
      const answer = await core.value.fetchFile(path)
      if (!answer.ok) errorMessage.value = answer.error
      return answer.ok
    } catch (why) {
      errorMessage.value = formatError(why)
      return false
    } finally {
      change(path, { isDownloading: false })
    }
  }

  async function download(row: TreeRow): Promise<void> {
    errorMessage.value = ''
    change(row.path, { isDownloading: true })
    if (!(await fetchRow(row.path)) || isDisposed) return
    change(row.path, { isAbsent: false })
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
    errorMessage.value = ''
    try {
      const answer = await core.value.dropDownload(row.path)
      if (!answer.ok) errorMessage.value = answer.error
      else change(row.path, { isAbsent: true })
    } catch (why) {
      errorMessage.value = formatError(why)
    }
  }

  onMounted(async () => {
    try {
      core.value = await startCore()
      await show('')
    } catch (why) {
      errorMessage.value = formatError(why)
    }
  })

  onBeforeUnmount(() => {
    isDisposed = true
  })

  return { folder, rows, errorMessage, tap, dropDownload, show }
}
