import { afterEach, expect, it, vi } from 'vitest'

afterEach(() => {
  vi.unstubAllGlobals()
  vi.resetModules()
})

it('asks the window for a model file and returns the shared local path', async () => {
  vi.stubGlobal('window', { location: { origin: 'https://numen.test' } })
  const send = vi.fn(() => Promise.resolve(Response.json({ path: '/models/weights' })))
  vi.stubGlobal('fetch', send)
  const { fetchModelFile } = await import('./dependencies')
  const url = 'https://models.test/weights.onnx'
  const sha256 = 'a'.repeat(64)
  expect(await fetchModelFile(url, sha256)).toEqual({ ok: true, value: '/models/weights' })
  expect(send).toHaveBeenCalledWith(
    'https://numen.test/numen.v1.HostDependencyService/FetchModelFile',
    expect.objectContaining({ body: new TextEncoder().encode(JSON.stringify({ url, sha256 })) }),
  )
})

it('returns the provisioning failure reported by the host', async () => {
  vi.stubGlobal('window', { location: { origin: 'https://numen.test' } })
  vi.stubGlobal('fetch', () => Promise.resolve(Response.json({ error: 'checksum mismatch' })))
  const { fetchModelFile } = await import('./dependencies')
  expect(await fetchModelFile('https://models.test/bad', 'a'.repeat(64))).toEqual({
    ok: false,
    error: 'checksum mismatch',
  })
})
