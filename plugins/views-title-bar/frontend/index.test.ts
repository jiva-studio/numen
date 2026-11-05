import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { TITLE_BAR_SLOT } from '@numen/plugin-sdk'
import { activate, TITLE_BAR_VIEW, TitleBar } from './index'

const manifest = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as {
  targets: readonly string[]
  contributes: { views: readonly { id: string; slot: string }[] }
}

describe('the plugin', () => {
  it('registers the bar under the id the manifest declares in the title bar slot', () => {
    const context = createMockPluginContext({ pluginId: 'md.numen.views.title.bar' })

    activate(context)

    expect(context.getRegisteredSlotViews()).toStrictEqual([
      { id: TITLE_BAR_VIEW, component: TitleBar },
    ])
    expect(manifest.contributes.views).toStrictEqual([{ id: TITLE_BAR_VIEW, slot: TITLE_BAR_SLOT }])
  })

  it('runs in the editor and in the flashcards window', () => {
    expect(manifest.targets).toStrictEqual(['desktop.editor', 'desktop.flashcards'])
  })
})
