import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import type { ApiError, Result } from '@numen/plugin-sdk'
import { createTagMap } from '../../model/createTagMap'
import type { TagMapResponse, TagsByPath } from '../../types'
import { createPlexTags } from './createPlexTags'

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

const getNames = (wrapper: ReturnType<typeof mount>) =>
  wrapper.findAll('[data-plex-tag]').map((chip) => chip.text())

describe('the tag chips view of a plex node', () => {
  it('draws the tags the map holds for the path it is given', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await map.waitForLoad()

    const wrapper = mount(createPlexTags(map), { props: { path: 'notes/Dock.md' } })

    expect(getNames(wrapper)).toEqual(['Port', 'Sea', 'Quay'])
    map.stop()
  })

  it('draws nothing for a path without tags', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await map.waitForLoad()

    const wrapper = mount(createPlexTags(map), { props: { path: 'notes/Pier.md' } })

    expect(wrapper.find('[data-plex-tags]').exists()).toBe(false)
    map.stop()
  })

  it('draws the tags again when the tags of its path change, without another request', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await map.waitForLoad()
    const wrapper = mount(createPlexTags(map), { props: { path: 'notes/Pier.md' } })

    map.setTags({ 'notes/Pier.md': ['Sea'] })
    await wrapper.vm.$nextTick()

    expect(getNames(wrapper)).toEqual(['Sea'])
    expect(feed.getTagMap).toHaveBeenCalledTimes(1)
    map.stop()
  })

  it('draws nothing again when its path is left with no tags', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await map.waitForLoad()
    const wrapper = mount(createPlexTags(map), { props: { path: 'notes/Quay.md' } })

    map.setTags({ 'notes/Quay.md': [] })
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-plex-tags]').exists()).toBe(false)
    map.stop()
  })

  it('makes no request of its own for each node it is drawn under', async () => {
    const feed = createFeed()
    const map = createTagMap({ api: feed.api })
    await map.waitForLoad()
    const view = createPlexTags(map)

    for (let i = 0; i < 30; i++) mount(view, { props: { path: `notes/Note ${i}.md` } })

    expect(feed.getTagMap).toHaveBeenCalledTimes(1)
    map.stop()
  })
})
