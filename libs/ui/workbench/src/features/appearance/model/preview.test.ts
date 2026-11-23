import { expect, it } from 'vitest'
import { ref } from 'vue'
import type { Mode, Sizes } from '../lib/appearance'
import { createAppearancePreview } from './preview'

it('reports an error when applying a previewed theme rejects', async () => {
  const messages: [string, string][] = []
  const preview = createAppearancePreview({
    write: (text, kind) => messages.push([text, kind]),
    applied: ref('preset:numen'),
    mode: ref<Mode>('system'),
    settings: ref<Sizes>({ interfaceScale: 1, textScale: 1 }),
    applyTheme: async () => {
      throw new Error('apply failed')
    },
  })

  preview.keyboard.showThemePreview('mine:sea')
  await Promise.resolve()

  expect(messages).toStrictEqual([['apply failed', 'error']])
})
