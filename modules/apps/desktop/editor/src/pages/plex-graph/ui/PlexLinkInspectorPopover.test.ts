import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { ArrowLeft, ArrowRight, Minus } from '@lucide/vue'
import PlexLinkInspectorPopover from './PlexLinkInspectorPopover.vue'
import type { LinkInspectorRequest } from '../types'

describe('PlexLinkInspectorPopover component', () => {
  const defaultRequest: LinkInspectorRequest = {
    pairKey: 'n1 n2',
    nodeA: { id: 'n1', title: 'Note Alpha', path: 'notes/alpha.md' },
    nodeB: { id: 'n2', title: 'Note Beta', path: 'notes/beta.md' },
    links: [
      {
        id: 'notes/alpha.md->notes/beta.md',
        from: 'notes/alpha.md',
        to: 'notes/beta.md',
        role: 'jump',
        direction: 'forward',
        description: 'Relates to',
      },
    ],
    at: { x: 300, y: 200 },
  }

  const mountPopover = (request: LinkInspectorRequest = defaultRequest) => {
    return mount(PlexLinkInspectorPopover, {
      props: { request },
      attachTo: document.body,
    })
  }

  it('renders positioned at specified coordinates and shows node titles', () => {
    const wrapper = mountPopover()
    const popover = wrapper.get('.plex-link-popover')

    expect(popover.attributes('style')).toContain('left: 300px')
    expect(popover.attributes('style')).toContain('top: 200px')
    expect(wrapper.text()).toContain('Note Alpha ⟷ Note Beta')
    wrapper.unmount()
  })

  it('toggles direction locally when direction button is clicked and saves on done', async () => {
    const wrapper = mountPopover()
    const dirBtn = wrapper.get('.plex-link-popover__dir-btn')
    expect(dirBtn.findComponent(ArrowRight).exists()).toBe(true)

    await dirBtn.trigger('click')
    expect(dirBtn.findComponent(ArrowLeft).exists()).toBe(true)

    const doneBtn = wrapper.get('.plex-link-popover__btn--done')
    await doneBtn.trigger('click')

    expect(wrapper.emitted('save')).toBeTruthy()
    const payload = wrapper.emitted('save')?.[0]?.[0] as { rows: { direction: string }[] }
    expect(payload.rows[0]?.direction).toBe('reverse')
    wrapper.unmount()
  })

  it('updates description locally and saves on Enter', async () => {
    const wrapper = mountPopover()
    const input = wrapper.get('input')
    await input.setValue('Causes')
    await input.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('save')).toBeTruthy()
    const payload = wrapper.emitted('save')?.[0]?.[0] as { rows: { description: string }[] }
    expect(payload.rows[0]?.description).toBe('Causes')
    wrapper.unmount()
  })

  it('adds reverse direction row when add reverse button is clicked', async () => {
    const wrapper = mountPopover()
    const addBtn = wrapper.get('.plex-link-popover__btn--add')
    await addBtn.trigger('click')

    const inputs = wrapper.findAll('input')
    expect(inputs.length).toBe(2)
    wrapper.unmount()
  })

  it('filters out empty second reverse row on done if left blank', async () => {
    const wrapper = mountPopover()
    const addBtn = wrapper.get('.plex-link-popover__btn--add')
    await addBtn.trigger('click')

    const doneBtn = wrapper.get('.plex-link-popover__btn--done')
    await doneBtn.trigger('click')

    expect(wrapper.emitted('save')).toBeTruthy()
    const payload = wrapper.emitted('save')?.[0]?.[0] as { rows: { description: string }[] }
    expect(payload.rows.length).toBe(1)
    expect(payload.rows[0]?.description).toBe('Relates to')
    wrapper.unmount()
  })

  it('saves both rows when second reverse row has a description', async () => {
    const wrapper = mountPopover()
    const addBtn = wrapper.get('.plex-link-popover__btn--add')
    await addBtn.trigger('click')

    const inputs = wrapper.findAll('input')
    await inputs[1]?.setValue('Reverse description')

    const doneBtn = wrapper.get('.plex-link-popover__btn--done')
    await doneBtn.trigger('click')

    expect(wrapper.emitted('save')).toBeTruthy()
    const payload = wrapper.emitted('save')?.[0]?.[0] as { rows: { description: string }[] }
    expect(payload.rows.length).toBe(2)
    expect(payload.rows[0]?.description).toBe('Relates to')
    expect(payload.rows[1]?.description).toBe('Reverse description')
    wrapper.unmount()
  })

  it('saves removal when trash button is clicked and Done is clicked', async () => {
    const wrapper = mountPopover()
    const deleteBtn = wrapper.get('.plex-link-popover__delete-btn')
    await deleteBtn.trigger('click')

    expect(wrapper.findAll('input').length).toBe(0)

    const doneBtn = wrapper.get('.plex-link-popover__btn--done')
    await doneBtn.trigger('click')

    expect(wrapper.emitted('save')).toBeTruthy()
    const payload = wrapper.emitted('save')?.[0]?.[0] as {
      rows: unknown[]
      removedLinks: { from: string; to: string }[]
    }
    expect(payload.rows.length).toBe(0)
    expect(payload.removedLinks).toContainEqual({
      from: 'notes/alpha.md',
      to: 'notes/beta.md',
      role: 'jump',
    })
    wrapper.unmount()
  })

  it('cycles through all direction modes: forward -> reverse -> undirected -> forward', async () => {
    const wrapper = mountPopover()
    const dirBtn = wrapper.get('.plex-link-popover__dir-btn')

    // Initial: forward
    expect(dirBtn.findComponent(ArrowRight).exists()).toBe(true)

    // 1st click: reverse
    await dirBtn.trigger('click')
    expect(dirBtn.findComponent(ArrowLeft).exists()).toBe(true)

    // 2nd click: undirected
    await dirBtn.trigger('click')
    expect(dirBtn.findComponent(Minus).exists()).toBe(true)

    // 3rd click: back to forward
    await dirBtn.trigger('click')
    expect(dirBtn.findComponent(ArrowRight).exists()).toBe(true)

    wrapper.unmount()
  })

  it('inverts child role to parent when toggled to reverse', async () => {
    const childRequest: LinkInspectorRequest = {
      pairKey: 'parent child',
      nodeA: { id: 'parent', title: 'Parent Node', path: 'notes/parent.md' },
      nodeB: { id: 'child', title: 'Child Node', path: 'notes/child.md' },
      links: [
        {
          id: 'notes/parent.md->notes/child.md',
          from: 'notes/parent.md',
          to: 'notes/child.md',
          role: 'child',
          direction: 'forward',
          description: 'Is child of',
        },
      ],
      at: { x: 100, y: 100 },
    }

    const wrapper = mountPopover(childRequest)
    const dirBtn = wrapper.get('.plex-link-popover__dir-btn')
    await dirBtn.trigger('click') // to reverse

    const doneBtn = wrapper.get('.plex-link-popover__btn--done')
    await doneBtn.trigger('click')

    const payload = wrapper.emitted('save')?.[0]?.[0] as {
      rows: { from: string; to: string; role: string; direction: string }[]
      removedLinks: { from: string; to: string; role: string }[]
    }
    expect(payload.rows[0]?.from).toBe('notes/child.md')
    expect(payload.rows[0]?.to).toBe('notes/parent.md')
    expect(payload.rows[0]?.role).toBe('parent')
    expect(payload.rows[0]?.direction).toBe('reverse')
    expect(payload.removedLinks).toContainEqual({
      from: 'notes/parent.md',
      to: 'notes/child.md',
      role: 'child',
    })

    wrapper.unmount()
  })

  it('adds reverse row on Ctrl+Enter from input', async () => {
    const wrapper = mountPopover()
    const input = wrapper.get('input')
    await input.trigger('keydown', { key: 'Enter', ctrlKey: true })

    expect(wrapper.findAll('input').length).toBe(2)
    wrapper.unmount()
  })

  it('disables direction toggle when two bidirectional rows exist', async () => {
    const bidirectionalRequest: LinkInspectorRequest = {
      pairKey: 'n1 n2',
      nodeA: { id: 'n1', title: 'Note Alpha', path: 'notes/alpha.md' },
      nodeB: { id: 'n2', title: 'Note Beta', path: 'notes/beta.md' },
      links: [
        {
          id: 'notes/alpha.md->notes/beta.md',
          from: 'notes/alpha.md',
          to: 'notes/beta.md',
          role: 'jump',
          direction: 'forward',
          description: 'Forward',
        },
        {
          id: 'notes/beta.md->notes/alpha.md',
          from: 'notes/beta.md',
          to: 'notes/alpha.md',
          role: 'jump',
          direction: 'reverse',
          description: 'Reverse',
        },
      ],
      at: { x: 300, y: 200 },
    }

    const wrapper = mountPopover(bidirectionalRequest)
    const dirButtons = wrapper.findAll('.plex-link-popover__dir-btn')
    expect(dirButtons.length).toBe(2)
    expect(dirButtons[0]?.attributes('disabled')).toBeDefined()
    expect(dirButtons[1]?.attributes('disabled')).toBeDefined()

    // Clicking does not change direction
    await dirButtons[0]?.trigger('click')
    await dirButtons[1]?.trigger('click')

    expect(dirButtons[0]?.findComponent(ArrowRight).exists()).toBe(true)
    expect(dirButtons[1]?.findComponent(ArrowLeft).exists()).toBe(true)

    wrapper.unmount()
  })

  it('emits dismiss on escape key', async () => {
    const wrapper = mountPopover()
    const popover = wrapper.get('.plex-link-popover')
    await popover.trigger('keydown', { key: 'Escape' })

    expect(wrapper.emitted('dismiss')).toBeTruthy()
    wrapper.unmount()
  })

  it('starts in undirected mode when initial request has undirected direction', () => {
    const undirectedRequest: LinkInspectorRequest = {
      pairKey: 'n1 n2',
      nodeA: { id: 'n1', title: 'Note Alpha', path: 'notes/alpha.md' },
      nodeB: { id: 'n2', title: 'Note Beta', path: 'notes/beta.md' },
      links: [
        {
          id: 'notes/alpha.md->notes/beta.md',
          from: 'notes/alpha.md',
          to: 'notes/beta.md',
          role: 'jump',
          direction: 'undirected',
          description: '',
        },
      ],
      at: { x: 300, y: 200 },
    }

    const wrapper = mountPopover(undirectedRequest)
    const dirBtn = wrapper.get('.plex-link-popover__dir-btn')
    expect(dirBtn.findComponent(Minus).exists()).toBe(true)
    wrapper.unmount()
  })

  it('re-enables direction toggle when a row is removed from a 2-row inspector', async () => {
    const bidirectionalRequest: LinkInspectorRequest = {
      pairKey: 'n1 n2',
      nodeA: { id: 'n1', title: 'Note Alpha', path: 'notes/alpha.md' },
      nodeB: { id: 'n2', title: 'Note Beta', path: 'notes/beta.md' },
      links: [
        {
          id: 'notes/alpha.md->notes/beta.md',
          from: 'notes/alpha.md',
          to: 'notes/beta.md',
          role: 'jump',
          direction: 'forward',
          description: 'Forward text',
        },
        {
          id: 'notes/beta.md->notes/alpha.md',
          from: 'notes/beta.md',
          to: 'notes/alpha.md',
          role: 'jump',
          direction: 'reverse',
          description: 'Reverse text',
        },
      ],
      at: { x: 300, y: 200 },
    }

    const wrapper = mountPopover(bidirectionalRequest)
    const deleteButtons = wrapper.findAll('.plex-link-popover__delete-btn')
    expect(deleteButtons.length).toBe(2)

    // Delete row 2
    await deleteButtons[1]?.trigger('click')

    // Now only 1 row remains
    const dirButtons = wrapper.findAll('.plex-link-popover__dir-btn')
    expect(dirButtons.length).toBe(1)
    expect(dirButtons[0]?.attributes('disabled')).toBeUndefined()

    // Can toggle direction now
    await dirButtons[0]?.trigger('click')
    expect(dirButtons[0]?.findComponent(ArrowLeft).exists()).toBe(true)

    wrapper.unmount()
  })

  it('treats whitespace-only descriptions as empty and prunes reverse row', async () => {
    const wrapper = mountPopover()
    const addBtn = wrapper.get('.plex-link-popover__btn--add')
    await addBtn.trigger('click')

    const inputs = wrapper.findAll('input')
    await inputs[1]?.setValue('    ') // Whitespace only

    const doneBtn = wrapper.get('.plex-link-popover__btn--done')
    await doneBtn.trigger('click')

    expect(wrapper.emitted('save')).toBeTruthy()
    const payload = wrapper.emitted('save')?.[0]?.[0] as { rows: { description: string }[] }
    expect(payload.rows.length).toBe(1)
    expect(payload.rows[0]?.description).toBe('Relates to')
    wrapper.unmount()
  })
})
