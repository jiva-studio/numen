/**
 * What the person sees and does in the files of a vault on the phone: which files are not downloaded, what
 * a tap does to them, and what removing a download does.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { Core, FileRow } from '@/entities/core'
import FileTreePage from './FileTreePage.vue'

const NOT_DOWNLOADED = 'Not downloaded'

const start = vi.hoisted(() => ({ startCore: vi.fn() }))

vi.mock('@/entities/core', () => start)

const guide: FileRow = { path: 'Books/Guide.pdf', name: 'Guide.pdf', isFolder: false, isAbsent: true }
const list: FileRow = {
  path: 'Shopping list.md',
  name: 'Shopping list.md',
  isFolder: false,
  isAbsent: false,
}

/** A core whose files are `rows`; a download or a removal answers as the test says. */
function coreWith(rows: FileRow[], answers: Partial<Pick<Core, 'fetchFile' | 'dropDownload'>> = {}) {
  const core = {
    listFiles: vi.fn(async () => rows),
    fetchFile: vi.fn(async () => ({ ok: true as const, value: null })),
    dropDownload: vi.fn(async () => ({ ok: true as const, value: null })),
    ...answers,
  }
  start.startCore.mockResolvedValue(core)
  return core
}

async function showFiles() {
  const page = mount(FileTreePage)
  await flushPromises()
  return page
}

const rowOf = (page: Awaited<ReturnType<typeof showFiles>>, name: string) =>
  page.findAll('[data-testid="file-row"]').find((row) => row.text().includes(name))

beforeEach(() => {
  start.startCore.mockReset()
})

describe('the files of the vault', () => {
  it('lists every file and marks the one that is not downloaded in words', async () => {
    coreWith([list, guide])
    const page = await showFiles()

    expect(page.findAll('[data-testid="file-row"]')).toHaveLength(2)
    expect(rowOf(page, 'Guide.pdf')?.text()).toContain(NOT_DOWNLOADED)
    expect(rowOf(page, 'Shopping list.md')?.text()).not.toContain(NOT_DOWNLOADED)
  })
})

describe('tapping a file', () => {
  it('opens a downloaded file without downloading anything', async () => {
    const core = coreWith([list, guide])
    const page = await showFiles()

    await rowOf(page, 'Shopping list.md')?.trigger('click')
    await flushPromises()

    expect(core.fetchFile).not.toHaveBeenCalled()
    expect(page.emitted('open')).toStrictEqual([['Shopping list.md']])
  })

  it('downloads a file that is not downloaded and opens it once it is there', async () => {
    const core = coreWith([guide])
    const page = await showFiles()

    await rowOf(page, 'Guide.pdf')?.trigger('click')
    await flushPromises()

    expect(core.fetchFile).toHaveBeenCalledExactlyOnceWith('Books/Guide.pdf')
    expect(page.emitted('open')).toStrictEqual([['Books/Guide.pdf']])
  })

  it('shows that the file is downloading until the download ends and opens nothing before', async () => {
    let finish: () => void = () => undefined
    const core = coreWith([guide], {
      fetchFile: vi.fn(
        () =>
          new Promise((resolve) => {
            finish = () => resolve({ ok: true, value: null })
          }),
      ),
    })
    const page = await showFiles()

    await rowOf(page, 'Guide.pdf')?.trigger('click')

    expect(rowOf(page, 'Guide.pdf')?.text()).toContain('Downloading')
    expect(page.emitted('open')).toBeUndefined()
    finish()
    await flushPromises()
    expect(core.fetchFile).toHaveBeenCalledTimes(1)
    expect(page.emitted('open')).toHaveLength(1)
  })

  it('downloads once for two taps on the same file', async () => {
    const core = coreWith([guide], {
      fetchFile: vi.fn(() => new Promise(() => undefined)),
    })
    const page = await showFiles()

    await rowOf(page, 'Guide.pdf')?.trigger('click')
    await rowOf(page, 'Guide.pdf')?.trigger('click')

    expect(core.fetchFile).toHaveBeenCalledTimes(1)
  })

  it('warns in words and opens nothing when the download fails', async () => {
    coreWith([guide], {
      fetchFile: vi.fn(async () => ({ ok: false as const, error: 'Could not reach the sync server' })),
    })
    const page = await showFiles()

    await rowOf(page, 'Guide.pdf')?.trigger('click')
    await flushPromises()

    expect(page.find('[data-testid="error"]').text()).toContain('Could not reach the sync server')
    expect(page.emitted('open')).toBeUndefined()
    expect(rowOf(page, 'Guide.pdf')?.text()).toContain(NOT_DOWNLOADED)
  })
})

describe('removing the download of a file', () => {
  it('is offered for a downloaded file and not for one that is not downloaded', async () => {
    coreWith([list, guide])
    const page = await showFiles()

    expect(rowOf(page, 'Shopping list.md')?.find('[data-testid="drop-download"]').exists()).toBe(true)
    expect(rowOf(page, 'Guide.pdf')?.find('[data-testid="drop-download"]').exists()).toBe(false)
  })

  it('turns the row to not downloaded', async () => {
    const core = coreWith([list])
    const page = await showFiles()

    await rowOf(page, 'Shopping list.md')?.find('[data-testid="drop-download"]').trigger('click')
    await flushPromises()

    expect(core.dropDownload).toHaveBeenCalledExactlyOnceWith('Shopping list.md')
    expect(rowOf(page, 'Shopping list.md')?.text()).toContain(NOT_DOWNLOADED)
    expect(page.emitted('open')).toBeUndefined()
  })

  it('warns in words and keeps the row downloaded when the file has changes that are not sent', async () => {
    coreWith([list], {
      dropDownload: vi.fn(async () => ({
        ok: false as const,
        error: 'The file has changes that are not sent',
      })),
    })
    const page = await showFiles()

    await rowOf(page, 'Shopping list.md')?.find('[data-testid="drop-download"]').trigger('click')
    await flushPromises()

    expect(page.find('[data-testid="error"]').text()).toContain(
      'The file has changes that are not sent',
    )
    expect(rowOf(page, 'Shopping list.md')?.text()).not.toContain(NOT_DOWNLOADED)
  })
})
