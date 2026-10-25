import { describe, expect, it } from 'vitest'
import { buildPluginCatalog } from './catalog'

const root = '../../plugins'

describe('buildPluginCatalog', () => {
  it('keys manifests and loaders by the manifest id, not the directory', async () => {
    const module = { activate: () => {} }
    const catalog = buildPluginCatalog(
      {
        [`${root}/readers.pdf/manifest.json`]: {
          id: 'md.numen.readers.pdf',
          permissions: ['vault:read'],
        },
      },
      { [`${root}/readers.pdf/frontend/index.ts`]: async () => module },
    )
    expect(catalog.manifests['md.numen.readers.pdf']).toStrictEqual({
      id: 'md.numen.readers.pdf',
      permissions: ['vault:read'],
    })
    expect(await catalog.loaders['md.numen.readers.pdf']?.()).toBe(module)
  })

  it('holds a manifest of a plugin that has no frontend and no loader for it', () => {
    const catalog = buildPluginCatalog(
      { [`${root}/engines.ocr/manifest.json`]: { id: 'md.numen.engines.ocr' } },
      {},
    )
    expect(Object.keys(catalog.manifests)).toStrictEqual(['md.numen.engines.ocr'])
    expect(catalog.loaders).toStrictEqual({})
  })

  it('throws when two manifests declare one id', () => {
    expect(() =>
      buildPluginCatalog(
        {
          [`${root}/a/manifest.json`]: { id: 'md.numen.x' },
          [`${root}/b/manifest.json`]: { id: 'md.numen.x' },
        },
        {},
      ),
    ).toThrow('md.numen.x')
  })

  it('throws for a frontend whose plugin has no manifest', () => {
    expect(() =>
      buildPluginCatalog({}, { [`${root}/orphan/frontend/index.ts`]: async () => ({}) }),
    ).toThrow('orphan')
  })
})
