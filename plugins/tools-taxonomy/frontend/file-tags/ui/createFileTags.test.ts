import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import type { ApiError, Result } from '@numen/plugin-sdk'
import { createTagMap } from '../../model/createTagMap'
import type { TagMapResponse, TagsByPath } from '../../types'
import { createFileTags } from './createFileTags'

enableAutoUnmount(afterEach)

const WHOLE: TagsByPath = {
  'notes/Quay.md': ['Port', 'Sea'],
  'notes/Dock.md': ['Port', 'Sea', 'Quay'],
}

const createFeed = () => {
  const getTagMap = vi.fn(async (): Promise<Result<TagMapResponse, ApiError>> => ({
    ok: true,
    value: { tags: WHOLE },
  }))
  const api = { getTagMap }
  return {
    api,
    getTagMap,
  }
}

describe('the tag dots view of a file tree row', () => {
  it('draws the tags the map holds for the path it is given', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await map.waitForLoad()

    const wrapper = mount(createFileTags(map), { props: { path: 'notes/Dock.md' } })

    expect(wrapper.get('[data-tag-dots]').attributes('aria-label')).toBe('Port, Sea, Quay')
    map.stop()
  })

  it('draws nothing for a path without tags', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await map.waitForLoad()

    const wrapper = mount(createFileTags(map), { props: { path: 'notes/Pier.md' } })

    expect(wrapper.find('[data-tag-dots]').exists()).toBe(false)
    map.stop()
  })

  it('draws nothing for a folder', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await map.waitForLoad()

    const wrapper = mount(createFileTags(map), { props: { path: 'notes' } })

    expect(wrapper.find('[data-tag-dots]').exists()).toBe(false)
    map.stop()
  })

  it('draws the tags again when the tags of its path change, without another request', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await map.waitForLoad()
    const wrapper = mount(createFileTags(map), { props: { path: 'notes/Pier.md' } })

    map.setTags({ 'notes/Pier.md': ['Sea'] })
    await wrapper.vm.$nextTick()

    expect(wrapper.get('[data-tag-dots]').attributes('aria-label')).toBe('Sea')
    expect(feed.getTagMap).toHaveBeenCalledTimes(1)
    map.stop()
  })

  it('draws nothing again when its path is left with no tags', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await map.waitForLoad()
    const wrapper = mount(createFileTags(map), { props: { path: 'notes/Quay.md' } })

    map.setTags({ 'notes/Quay.md': [] })
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-tag-dots]').exists()).toBe(false)
    map.stop()
  })

  it('makes no request of its own for each row it is drawn in', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await map.waitForLoad()
    const view = createFileTags(map)

    for (let i = 0; i < 30; i++) mount(view, { props: { path: `notes/Note ${i}.md` } })

    expect(feed.getTagMap).toHaveBeenCalledTimes(1)
    map.stop()
  })
})
