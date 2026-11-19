import { expect, it } from 'vitest'
import { ref } from 'vue'
import type { ThemeMode as Mode, ThemeScales } from './themes'
import { createAppearancePreview } from './preview'

it('reports an error when applying a previewed theme rejects', async () => {
  const messages: [string, string][] = []
  const preview = createAppearancePreview({
    write: (text, kind) => messages.push([text, kind]),
    applied: ref('preset:numen'),
    mode: ref<Mode>('system'),
    settings: ref<ThemeScales<number>>({ interfaceScale: 1, textScale: 1 }),
    applyTheme: async () => {
      throw new Error('apply failed')
    },
  })

  preview.keyboard.previewTheme('mine:sea')
  await Promise.resolve()

  expect(messages).toStrictEqual([['apply failed', 'error']])
})
