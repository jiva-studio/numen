import { describe, expect, it } from 'vitest'
import { defineComponent } from 'vue'
import { createDialogStack } from './createDialogStack'

const component = defineComponent({ render: () => null })

describe('createDialogStack', () => {
  it('lists the dialogs in the order they opened', () => {
    const stack = createDialogStack()

    stack.host.open({ component, props: { name: 'first' } })
    stack.host.open({ component })

    expect(stack.entries.value.map((one) => one.options.props)).toStrictEqual([
      { name: 'first' },
      undefined,
    ])
  })

  it('removes the dialog its handle closes and no other', () => {
    const stack = createDialogStack()
    const first = stack.host.open({ component })
    stack.host.open({ component })

    first.close()
    first.close()

    expect(stack.entries.value).toHaveLength(1)
  })
})
