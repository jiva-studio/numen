/**
 * A file that is not on this device opens in a tab at once; the tab shows the download, then the
 * file replaces it.
 */
import { reactive } from 'vue'
import { describe, expect, it } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { FileKind } from '@/shared/model/fileTypes'
import { createDeferred } from '@/testing/deferred'
import { createTestTabs } from '@/testing/tabs'
import DownloadPane from '../ui/DownloadPane.vue'
import { createDownloadTabs, type DownloadBytes } from './createDownloadTabs'
import { createFileOpeners } from './openers'

const OTHER: FileKind = { isNote: false, type: 'note' }

const setup = (
  fetchFile: (path: string) => Promise<void>,
  getProgress: (path: string) => DownloadBytes | undefined = () => undefined,
) => {
  const windowTabs = createTestTabs()
  const opened: string[] = []
  const fetched: string[] = []
  const retried: string[] = []
  const downloads = createDownloadTabs(
    DownloadPane,
    windowTabs,
    (path) => retried.push(path),
    getProgress,
  )
  windowTabs.kinds.registerKind(downloads.kind)
  const tabOpeners = createFileOpeners({
    readFileKinds: async (paths) => new Map(paths.map((path) => [path, OTHER])),
    isFileAbsent: async () => true,
    fetchFile: async (path) => {
      fetched.push(path)
      await fetchFile(path)
    },
    downloadTabs: downloads.tabs,
  })
  tabOpeners.registerExtensionReader('mp4', (path) => opened.push(path))
  const getTitles = () => windowTabs.tabs.tabs.value.map((one) => one.title)
  return { windowTabs, tabOpeners, opened, fetched, retried, getTitles }
}

describe('a file that is not on this device', () => {
  it('opens a tab at once that stays until the file opens, and the file opens once downloaded', async () => {
    const download = createDeferred<void>()
    const { tabOpeners, windowTabs, opened, getTitles } = setup(() => download.promise)

    const opening = tabOpeners.openFile('Videos/Lecture.mp4')
    await flushPromises()

    expect(getTitles()).toStrictEqual(['Lecture.mp4'])
    expect(windowTabs.handle.getFrontTab()?.kind).toBe('download')
    expect(opened).toStrictEqual([])

    download.answer()
    await opening

    expect(opened).toStrictEqual(['Videos/Lecture.mp4'])
    expect(windowTabs.handle.listTabs('download')).toStrictEqual([])
  })

  it('is fetched once when it is opened again while it downloads, and its tab is shown', async () => {
    const download = createDeferred<void>()
    const { tabOpeners, windowTabs, fetched, opened } = setup(() => download.promise)

    const first = tabOpeners.openFile('Lecture.mp4')
    await flushPromises()
    await tabOpeners.openFile('Lecture.mp4')
    download.answer()
    await first

    expect(fetched).toStrictEqual(['Lecture.mp4'])
    expect(opened).toStrictEqual(['Lecture.mp4'])
    expect(windowTabs.handle.listTabs('download')).toStrictEqual([])
  })

  it('is not opened after its tab is closed, and the download goes on', async () => {
    const download = createDeferred<void>()
    const { tabOpeners, windowTabs, opened } = setup(() => download.promise)

    const opening = tabOpeners.openFile('Lecture.mp4')
    await flushPromises()
    const [tab] = windowTabs.handle.listTabs('download')
    windowTabs.panes.closeTab(tab!.id)
    download.answer()
    await opening

    expect(opened).toStrictEqual([])
    expect(windowTabs.tabs.tabs.value).toStrictEqual([])
  })

  it('shows what went wrong in its tab and opens again when the person tries again', async () => {
    let attempt = 0
    const { tabOpeners, windowTabs, opened, fetched } = setup(async () => {
      attempt += 1
      if (attempt === 1)
        throw Object.assign(new Error('lost'), { name: 'ConnectError', code: 2, rawMessage: '' })
    })

    await tabOpeners.openFile('Lecture.mp4')
    const [tab] = windowTabs.handle.listTabs<{ view: { value: { failure: string } } }>('download')
    const pane = mount(DownloadPane, { props: { state: tab!.state as never } })

    expect(pane.text()).toContain('The file could not be downloaded.')
    expect(pane.text()).not.toContain('not responding')
    expect(pane.text()).toContain('Try again')

    await tabOpeners.openFile('Lecture.mp4')

    expect(fetched).toStrictEqual(['Lecture.mp4', 'Lecture.mp4'])
    expect(opened).toStrictEqual(['Lecture.mp4'])
    expect(windowTabs.handle.listTabs('download')).toStrictEqual([])
  })

  it('asks to open the file again from the retry button', async () => {
    const { tabOpeners, windowTabs, retried } = setup(async () => {
      throw new Error('offline')
    })

    await tabOpeners.openFile('Lecture.mp4')
    const [tab] = windowTabs.handle.listTabs('download')
    const pane = mount(DownloadPane, { props: { state: tab!.state as never } })
    await pane.get('button').trigger('click')

    expect(retried).toStrictEqual(['Lecture.mp4'])
  })

  it('draws one ring that turns until the progress arrives, then the percent and the bytes', async () => {
    const download = createDeferred<void>()
    const progress = reactive(new Map<string, DownloadBytes>())
    const { tabOpeners, windowTabs } = setup(
      () => download.promise,
      (path) => progress.get(path),
    )

    const opening = tabOpeners.openFile('Lecture.mp4')
    await flushPromises()
    const [tab] = windowTabs.handle.listTabs('download')
    const pane = mount(DownloadPane, { props: { state: tab!.state as never } })

    expect(pane.findAll('.progress-ring')).toHaveLength(1)
    expect(pane.find('svg.lucide').exists()).toBe(false)
    expect(pane.get('.progress-ring').attributes('data-known')).toBe('false')
    expect(pane.text()).not.toContain('%')
    expect(pane.text()).not.toContain(' of ')

    progress.set('Lecture.mp4', { bytesDone: 12 * 1024 ** 2, bytesTotal: 41 * 1024 ** 2 })
    await flushPromises()

    expect(pane.get('.progress-ring').attributes('data-known')).toBe('true')
    expect(pane.get('.progress-ring__text').text()).toBe('29%')
    expect(pane.text()).toContain('Lecture.mp4 · 12 of 41 MB')
    download.answer()
    await opening
  })

  it('shows the downloading state while it runs', async () => {
    const download = createDeferred<void>()
    const { tabOpeners, windowTabs } = setup(() => download.promise)

    const opening = tabOpeners.openFile('Lecture.mp4')
    await flushPromises()
    const [tab] = windowTabs.handle.listTabs('download')
    const pane = mount(DownloadPane, { props: { state: tab!.state as never } })

    expect(pane.text()).toContain('Downloading…')
    expect(pane.text()).toContain('Lecture.mp4')
    expect(pane.find('button').exists()).toBe(false)
    download.answer()
    await opening
  })
})
