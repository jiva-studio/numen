import { describe, expect, it } from 'vitest'
import { createApi, type JsonValue } from '@numen/plugin-sdk'
import { createTaxonomyApi, SERVICE } from './createTaxonomyApi'

const answerJson = (value: JsonValue): Response =>
  new Response(JSON.stringify(value), { headers: { 'Content-Type': 'application/json' } })

const callWithBody = (body: JsonValue) => {
  const api = createApi(SERVICE, {
    origin: 'http://window',
    fetch: async () => answerJson(body),
  })
  return createTaxonomyApi(api).getTagMap()
}

describe('the taxonomy API client', () => {
  it('accepts an empty tag map object', async () => {
    const result = await callWithBody({ tags: {} })

    expect(result).toStrictEqual({ ok: true, value: { tags: {} } })
  })

  it('accepts a map of paths to string arrays', async () => {
    const result = await callWithBody({ tags: { 'notes/Quay.md': ['Port', 'Sea'] } })

    expect(result).toStrictEqual({
      ok: true,
      value: { tags: { 'notes/Quay.md': ['Port', 'Sea'] } },
    })
  })

  it('rejects a reply whose tags field is null or not a map of string arrays', async () => {
    for (const malformed of [
      null,
      {},
      { tags: null },
      { tags: [] },
      { tags: { 'notes/Quay.md': 'Port' } },
      { tags: { 'notes/Quay.md': [123] } },
    ]) {
      const result = await callWithBody(malformed)

      expect(result).toStrictEqual({
        ok: false,
        error: { code: 'internal', message: 'Invalid GetTagMap response' },
      })
    }
  })
})
