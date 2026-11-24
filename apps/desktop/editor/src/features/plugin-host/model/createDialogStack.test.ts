import { describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { createDialogStack } from './createDialogStack'

const component = defineComponent({ render: () => null })
const noSave = () => Promise.resolve(undefined)

describe('createDialogStack', () => {
  it('lists the dialogs in the order they opened', () => {
    const stack = createDialogStack(noSave)

    stack.host.open({ component, props: { name: 'first' } })
    stack.host.open({ component })

    expect(stack.entries.value.map((one) => one.options.props)).toStrictEqual([
      { name: 'first' },
      undefined,
    ])
  })

  it('removes the dialog its handle closes and no other', () => {
    const stack = createDialogStack(noSave)
    const first = stack.host.open({ component })
    stack.host.open({ component })

    first.close()
    first.close()

    expect(stack.entries.value).toHaveLength(1)
  })

  it('passes the text to the save dialog of the machine and answers the chosen file', async () => {
    const saveText = vi.fn(() => Promise.resolve({ path: '/home/me/a.json' }))
    const stack = createDialogStack(saveText)
    const text = { title: 'Save', fileName: 'a.json', content: '{}' }

    await expect(stack.host.saveText(text)).resolves.toStrictEqual({ path: '/home/me/a.json' })

    expect(saveText).toHaveBeenCalledWith(text)
  })
})
