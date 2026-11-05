import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import NamedIcon from './NamedIcon.vue'

describe('NamedIcon with a name the set does not have', () => {
  it('draws a placeholder icon, so a mistyped manifest shows', async () => {
    const icon = mount(NamedIcon, { props: { name: 'no-such-icon' }, attrs: { class: 'mark' } })

    await vi.waitFor(() => expect(icon.find('svg.mark').exists()).toBe(true))
  })

  it('draws nothing for an empty name, which a command without an icon has', async () => {
    const icon = mount(NamedIcon, { props: { name: '' } })
    await flushPromises()

    expect(icon.find('svg').exists()).toBe(false)
  })
})
