/**
 * The tooltip: announced as one, and showing what it was given to say. Where it
 * stands is computed by getAdjacentPlace, which has its own tests.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { Component } from 'vue'
import * as ui from '@/index'

const Tooltip = (ui as unknown as Record<string, Component | undefined>)['Tooltip']

const mountTooltip = (slot: string) => {
  expect(Tooltip).toBeDefined()
  return mount(Tooltip as Component, {
    props: { at: { x: 100, y: 200, width: 20, height: 20 } },
    slots: { default: slot },
  })
}

describe('the tooltip of the library', () => {
  it('is announced as one, and says what it was given to say', () => {
    const tooltip = mountTooltip('What this is')

    expect(tooltip.findAll('[role="tooltip"]')).toHaveLength(1)
    expect(tooltip.text()).toBe('What this is')
  })

  it('says the tags it was given, in the order it was given them', () => {
    const tooltip = mountTooltip('Port, Sea, Quay')

    expect(tooltip.text()).toBe('Port, Sea, Quay')
  })
})
