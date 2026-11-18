/**
 * The views of the nodes of a plex that opens ask the vault for the notes of all the nodes with one
 * call, not with a call for each node.
 */
import { afterEach, describe, expect, it } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { Component } from 'vue'
import type { NoteGraph, VaultNotes } from '@numen/plugin-sdk'
import {
  createMockHostServices,
  createMockNoticeHost,
  createMockPluginContext,
} from '@numen/plugin-sdk/testing'
import { activate } from './index'

const NODE_COUNT = 15
const NODES = Array.from({ length: NODE_COUNT }, (_, index) => `notes/Leaf ${index + 1}.md`)

const getTitle = (path: string) => path.replace(/^.*\//, '').replace(/\.md$/, '')

const setup = () => {
  const calls: { paths: readonly string[]; depth: number | undefined }[] = []
  const notes = {
    listRelatedNotes: async (paths: readonly string[], depth?: 1 | 2): Promise<NoteGraph[]> => {
      calls.push({ paths, depth })
      return paths.map((path) => ({
        focus: { path, title: getTitle(path) },
        focusType: 'note',
        related: [
          {
            path: 'notes/Orchard plan.md',
            title: 'Orchard plan',
            relation: 'parent' as const,
            type: 'note',
          },
        ],
        connections: [],
      }))
    },
  } as unknown as VaultNotes
  const context = createMockPluginContext({ pluginId: 'md.numen.views.plex.related.names' })
  activate(context, createMockHostServices({ notes, notices: createMockNoticeHost() }))
  const wrappers: VueWrapper[] = []
  const show = (id: string, path: string) => {
    const found = context.getRegisteredSlotViews().find((one) => one.id === id)
    const wrapper = mount(found!.component as Component, {
      props: { path },
      attachTo: document.body,
    })
    wrappers.push(wrapper)
    return wrapper
  }
  return { calls, show, wrappers }
}

const cleanups: (() => void)[] = []

afterEach(() => {
  while (cleanups.length > 0) cleanups.pop()?.()
  document.body.replaceChildren()
})

describe('the views of the nodes of a plex that opens', () => {
  it('mounts the parents and related views of 15 nodes and calls the vault once for 15 notes', async () => {
    const { calls, show, wrappers } = setup()
    cleanups.push(() => wrappers.forEach((one) => one.unmount()))

    NODES.forEach((one) => {
      show('parents', one)
      show('related', one)
    })
    await flushPromises()

    expect(calls).toHaveLength(1)
    expect(calls[0]?.paths).toStrictEqual(NODES)
    expect(calls[0]?.depth).toBe(1)
  })
})
