import { describe, expect, it } from 'vitest'
import { parsePluginReadme } from './readme'

describe('parsePluginReadme', () => {
  it('falls back to the manifest description when README is empty', () => {
    expect(parsePluginReadme('', 'md.numen.notes', 'Fallback description')).toStrictEqual({
      summary: 'Fallback description',
      more: '',
    })
  })

  it('strips a leading H1 title and splits on <!-- more -->', () => {
    const raw = `# Vector Embeddings

Generates vector embeddings for semantic search.

<!-- more -->

## Providers

Supports local ONNX and remote APIs.`

    expect(parsePluginReadme(raw, 'md.numen.engines.embeddings', 'Fallback')).toStrictEqual({
      summary: 'Generates vector embeddings for semantic search.',
      more: '## Providers\n\nSupports local ONNX and remote APIs.',
    })
  })

  it('rewrites relative assets/ links to the plugin route', () => {
    const raw = `# Sample

Summary text.

<!--more-->

![Diagram](assets/diagram.png) and [Guide](./assets/guide.pdf)`

    expect(parsePluginReadme(raw, 'md.numen.sample', 'Fallback')).toStrictEqual({
      summary: 'Summary text.',
      more: '![Diagram](/plugin/md.numen.sample/assets/diagram.png) and [Guide](/plugin/md.numen.sample/assets/guide.pdf)',
    })
  })
})
