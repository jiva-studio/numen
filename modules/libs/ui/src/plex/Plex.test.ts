/**
 * What the component does, not where it puts things. The negatives matter
 * most here: they are what fails silently and still looks right.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Plex from './Plex.vue'
import { neighbourhoods } from './fixtures/neighbourhoods'

/** No movement unless a test is about movement. */
const mountPlex = (props: Partial<InstanceType<typeof Plex>['$props']> = {}) =>
  mount(Plex, {
    props: { neighbourhood: neighbourhoods.typical, duration: 0, ...props },
    attachTo: document.body,
  })

describe('choosing a node', () => {
  it('hands back the identifier it was given, untouched', () => {
    const plex = mountPlex()
    plex.get('[aria-label^="Domain"]').trigger('click')
    expect(plex.emitted('activate')).toStrictEqual([['child-0']])
  })

  it('says nothing when the focus itself is clicked', () => {
    const plex = mountPlex()
    plex.get('[aria-label^="Hexagonal architecture"]').trigger('click')
    expect(plex.emitted('activate')).toBeUndefined()
  })

  it('answers Enter and Space, and nothing else', async () => {
    const plex = mountPlex()
    const node = plex.get('[aria-label^="Domain"]')

    await node.trigger('keydown', { key: 'Enter' })
    await node.trigger('keydown', { key: ' ' })
    await node.trigger('keydown', { key: 'a' })
    await node.trigger('keydown', { key: 'Escape' })

    expect(plex.emitted('activate')).toStrictEqual([['child-0'], ['child-0']])
  })
})

describe('what a screen reader and a keyboard are given', () => {
  it('makes every node but the focus reachable by tab', () => {
    const plex = mountPlex()
    const reachable = plex.findAll('[tabindex="0"]')
    const focus = plex.findAll('[tabindex="-1"]')
    expect(focus).toHaveLength(1)
    expect(reachable).toHaveLength(neighbourhoods.typical.nodes.length - 1)
  })

  it('names a node by its title and its seat', () => {
    const plex = mountPlex()
    expect(plex.find('[aria-label="Domain, child"]').exists()).toBe(true)
    expect(plex.find('[aria-label="Software architecture, parent"]').exists()).toBe(true)
    expect(plex.find('[aria-label="Onion architecture, jump"]').exists()).toBe(true)
  })

  it('still names a node whose title is empty', () => {
    const plex = mountPlex({ neighbourhood: neighbourhoods.awkwardLabels })
    expect(plex.find('[aria-label="Untitled, child"]').exists()).toBe(true)
  })

  it('keeps the full title in the accessible name when the box truncates it', () => {
    const long = 'Supercalifragilisticexpialidociousandthensomemore'
    const plex = mountPlex({ neighbourhood: neighbourhoods.awkwardLabels })
    expect(plex.find(`[aria-label="${long}, child"]`).exists()).toBe(true)
  })
})

describe('what did not fit', () => {
  it('says so rather than hiding it', () => {
    const plex = mountPlex({ neighbourhood: neighbourhoods.overcrowded })
    const status = plex.get('[role="status"]').text()

    // How much fits depends on the window, so the sentence is read against
    // what was actually drawn rather than against a number written here.
    const drawn = (seat: string) =>
      plex.findAll('.plex__node').filter((n) => n.attributes('aria-label')?.endsWith(`, ${seat}`))
        .length

    expect(status).toContain(`${200 - drawn('child')} children`)
    expect(status).toContain(`${40 - drawn('jump')} jumps`)
    expect(status).toContain('not shown')
  })

  it('says nothing when everything fits', () => {
    expect(mountPlex().find('[role="status"]').exists()).toBe(false)
  })

  it('hands the counts over, so the words are not the plex to choose', () => {
    // Knowing English is the same mistake as knowing the domain, one step down.
    const plex = mount(Plex, {
      props: { neighbourhood: neighbourhoods.overcrowded, duration: 0 },
      slots: {
        overflow: `<template #default="{ overflow }">спрятано: {{ overflow.length }}</template>`,
      },
    })
    expect(plex.get('[role="status"]').text()).toBe('спрятано: 2')
  })

  it('can be kept quiet by a slot that draws nothing', () => {
    const plex = mount(Plex, {
      props: { neighbourhood: neighbourhoods.overcrowded, duration: 0 },
      slots: { overflow: '<span class="hushed" />' },
    })
    expect(plex.get('[role="status"]').text()).toBe('')
  })
})

describe('being given the next neighbourhood', () => {
  it('arrives at once when there is no movement to make', async () => {
    const plex = mountPlex()
    expect(plex.find('[aria-label^="Domain"]').exists()).toBe(true)

    await plex.setProps({ neighbourhood: neighbourhoods.leaf })
    expect(plex.find('[aria-label^="Domain"]').exists()).toBe(false)
    expect(plex.find('[aria-label="Inbox, parent"]').exists()).toBe(true)
  })

  it('sets off rather than jumping when it has time to move', async () => {
    const plex = mountPlex({ duration: 400 })
    await plex.setProps({ neighbourhood: neighbourhoods.leaf })

    // Still showing the old picture: on its way, not replaced.
    expect(plex.find('[aria-label^="Domain"]').exists()).toBe(true)
    expect(plex.vm.moving).toBe(true)
  })

  it('stands still when it has nowhere left to go', async () => {
    const plex = mountPlex()
    await plex.setProps({ neighbourhood: neighbourhoods.leaf })
    expect(plex.vm.moving).toBe(false)
  })

  it('re-aims at the newest neighbourhood instead of queueing them', async () => {
    const plex = mountPlex({ duration: 400 })
    await plex.setProps({ neighbourhood: neighbourhoods.leaf })
    await plex.setProps({ neighbourhood: neighbourhoods.root })
    await plex.setProps({ neighbourhood: neighbourhoods.diamond, duration: 0 })

    // Four clicks in a second end at the fourth, not at a queue of three.
    expect(plex.find('[aria-label^="Recursive CTE"]').exists()).toBe(true)
    expect(plex.vm.moving).toBe(false)
  })
})
