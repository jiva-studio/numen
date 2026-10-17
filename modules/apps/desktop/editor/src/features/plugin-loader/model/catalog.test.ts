import { describe, expect, it } from 'vitest'
import { buildPluginCatalog } from './catalog'

const root = '../../plugins'

describe('buildPluginCatalog', () => {
  it('keys manifests and loaders by the manifest id, not the directory', async () => {
    const module = { activate: () => {} }
    const catalog = buildPluginCatalog(
      { [`${root}/reader-pdf/manifest.json`]: { id: 'md.numen.pdf', permissions: ['vault:read'] } },
      { [`${root}/reader-pdf/frontend/index.ts`]: async () => module },
    )
    expect(catalog.manifests['md.numen.pdf']).toStrictEqual({
      id: 'md.numen.pdf',
      permissions: ['vault:read'],
    })
    expect(await catalog.loaders['md.numen.pdf']?.()).toBe(module)
  })

  it('holds a manifest of a plugin that has no frontend and no loader for it', () => {
    const catalog = buildPluginCatalog(
      { [`${root}/ocr/manifest.json`]: { id: 'md.numen.ocr' } },
      {},
    )
    expect(Object.keys(catalog.manifests)).toStrictEqual(['md.numen.ocr'])
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
