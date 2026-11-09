import { describe, expect, it } from 'vitest'
import { parsePluginReadme } from './readme'

describe('parsePluginReadme', () => {
  it('falls back to the trimmed manifest description when README is empty or only a title', () => {
    expect(parsePluginReadme('', 'md.numen.notes', '  Fallback description  ')).toStrictEqual({
      summary: 'Fallback description',
      more: '',
    })
    expect(
      parsePluginReadme('  # Only Title  \n\n', 'md.numen.notes', '  Fallback  '),
    ).toStrictEqual({
      summary: 'Fallback',
      more: '',
    })
  })

  it('returns the trimmed body as summary when there is no <!-- more --> marker', () => {
    const raw = `  # Plugin Title

  Single paragraph summary.

# Inner heading stays  `

    expect(parsePluginReadme(raw, 'md.numen.notes', 'Fallback')).toStrictEqual({
      summary: 'Single paragraph summary.\n\n# Inner heading stays',
      more: '',
    })
  })

  it('strips an H1 whose title follows several spaces or a line break', () => {
    expect(parsePluginReadme('#   Title  \n\n\nBody', 'md.numen.notes', 'Fallback').summary).toBe(
      'Body',
    )
    expect(parsePluginReadme('#\nBody\nMore', 'md.numen.notes', 'Fallback').summary).toBe('More')
  })

  it('keeps a leading # with no space after it', () => {
    expect(parsePluginReadme('#Tag\n\nBody', 'md.numen.notes', 'Fallback').summary).toBe(
      '#Tag\n\nBody',
    )
  })

  it('keeps a leading subheading that is not an H1', () => {
    expect(parsePluginReadme('## Subheading\n\nText', 'md.numen.notes', 'Fallback')).toStrictEqual({
      summary: '## Subheading\n\nText',
      more: '',
    })
  })

  it('falls back to the manifest description when nothing precedes <!-- more -->', () => {
    const raw = `# Title

<!--  MORE  -->

Expanded details.`

    expect(parsePluginReadme(raw, 'md.numen.notes', '  Manifest summary  ')).toStrictEqual({
      summary: 'Manifest summary',
      more: 'Expanded details.',
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
