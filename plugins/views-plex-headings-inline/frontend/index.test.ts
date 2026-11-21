/** The view of the plugin as the plex draws it under a node, read from a vault that answers headings. */
import { afterEach, describe, expect, it } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { Component } from 'vue'
import type { JsonValue, NoteHeading, VaultNotes, WorkspaceHost } from '@numen/plugin-sdk'
import { createMockHostServices, createMockPluginContext } from '@numen/plugin-sdk/testing'
import { activate } from './index'

const PATH = 'physics/Entropy.md'

const sections = (count: number): NoteHeading[] =>
  Array.from({ length: count }, (_, index) => ({
    text: `Section ${index + 1}`,
    level: 2,
    line: index * 4,
  }))

const wrappers: VueWrapper[] = []

const setup = (
  headings: readonly NoteHeading[],
  initialSettings: Record<string, JsonValue> = {},
) => {
  const requested: (readonly string[])[] = []
  const opened: [string, number | undefined][] = []
  const notes = {
    getHeadings: async (paths: readonly string[]) => {
      requested.push(paths)
      return new Map(paths.map((path) => [path, headings]))
    },
  } as unknown as VaultNotes
  const workspace = {
    files: {
      open: (path: string, options?: { line?: number }) => opened.push([path, options?.line]),
    },
  } as unknown as WorkspaceHost
  const context = createMockPluginContext({
    pluginId: 'md.numen.views.plex.headings.inline',
    initialSettings,
  })
  activate(context, createMockHostServices({ notes, workspace }))
  const view = context.getRegisteredSlotViews()[0]!.component as Component
  const show = async (path = PATH) => {
    const wrapper = mount(view, { props: { path } })
    wrappers.push(wrapper)
    await flushPromises()
    return wrapper
  }
  return { context, show, requested, opened }
}

afterEach(() => {
  while (wrappers.length > 0) wrappers.pop()?.unmount()
})

describe('the plugin', () => {
  it('registers the one view its manifest places under the nodes', () => {
    const { context } = setup([])

    expect(context.getRegisteredSlotViews().map((one) => one.id)).toStrictEqual(['headings'])
  })

  it('refuses to activate without the notes of the vault or the window', () => {
    const context = createMockPluginContext({ pluginId: 'md.numen.views.plex.headings.inline' })

    expect(() => activate(context, createMockHostServices())).toThrow(/need the notes/)
  })
})

describe('a node', () => {
  it('writes each heading of its note after a hash, in order', async () => {
    const { show } = setup(sections(3))
    const wrapper = await show()

    expect(wrapper.findAll('button').map((one) => one.text())).toStrictEqual([
      '# Section 1',
      '# Section 2',
      '# Section 3',
    ])
  })

  it('reads the headings of that one note', async () => {
    const { show, requested } = setup(sections(1))
    await show()

    expect(requested).toStrictEqual([[PATH]])
  })

  it('writes no more than the limit', async () => {
    const { show } = setup(sections(10), { headingLimit: 2 })
    const wrapper = await show()

    expect(wrapper.findAll('button')).toHaveLength(2)
  })

  it('writes nothing for a note with no heading', async () => {
    const { show } = setup([])
    const wrapper = await show()

    expect(wrapper.find('.inline-headings').exists()).toBe(false)
  })

  it('opens the note at the line of the chosen heading', async () => {
    const { show, opened } = setup(sections(3))
    const wrapper = await show()

    await wrapper.findAll('button')[1]!.trigger('click')

    expect(opened).toStrictEqual([[PATH, 4]])
  })
})
