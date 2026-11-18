import { defineComponent, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Tooltip from './Tooltip.vue'
import { useTooltip } from './useTooltip'

const mountTooltip = (slot: string) => {
  return mount(Tooltip, {
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

  it('opens for the target element, updates reactive text, and closes on empty text', async () => {
    const dynamic = ref('First')
    const Harness = defineComponent({
      components: { Tooltip },
      setup() {
        const tip = useTooltip()
        return { tip, dynamic }
      },
      template: `
        <div>
          <button class="one" :aria-describedby="tip.describedBy.value" @mouseenter="tip.open($event, () => dynamic)" @mouseleave="tip.close" />
          <button class="empty" @mouseenter="tip.open($event, '')" />
          <Tooltip v-if="tip.at.value" :id="tip.id" :at="tip.at.value">{{ tip.text.value }}</Tooltip>
        </div>
      `,
    })
    const wrapper = mount(Harness)
    expect(wrapper.find('[role="tooltip"]').exists()).toBe(false)
    expect(wrapper.find('.one').attributes('aria-describedby')).toBeUndefined()

    await wrapper.find('.one').trigger('mouseenter')
    expect(wrapper.find('[role="tooltip"]').text()).toBe('First')
    expect(wrapper.find('.one').attributes('aria-describedby')).toBe(
      wrapper.find('[role="tooltip"]').attributes('id'),
    )

    dynamic.value = 'Second'
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[role="tooltip"]').text()).toBe('Second')

    await wrapper.find('.empty').trigger('mouseenter')
    expect(wrapper.find('[role="tooltip"]').exists()).toBe(false)
  })
})
