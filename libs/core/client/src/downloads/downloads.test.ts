import { Code, ConnectError } from '@connectrpc/connect'
import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { GetDownloadProgressResponseSchema } from '@numen/protocol'
import { createDownloadClient } from './downloads'

describe('getDownloadProgress', () => {
  it('returns the bytes done and the size of the file for the requested path', async () => {
    const asked: string[] = []
    const client = createDownloadClient({
      getDownloadProgress: (request) => {
        asked.push(request.path ?? '')
        return Promise.resolve(
          create(GetDownloadProgressResponseSchema, { isKnown: true, bytesDone: 3n, bytesTotal: 9n }),
        )
      },
    })

    const result = await client.getDownloadProgress('Books/Guide.pdf')

    expect(asked).toStrictEqual(['Books/Guide.pdf'])
    expect(result).toStrictEqual({
      ok: true,
      value: { isKnown: true, bytesDone: 3, bytesTotal: 9 },
    })
  })

  it('returns not known when the core says no download runs', async () => {
    const client = createDownloadClient({
      getDownloadProgress: () => Promise.resolve(create(GetDownloadProgressResponseSchema)),
    })

    const result = await client.getDownloadProgress('a.pdf')

    expect(result).toStrictEqual({
      ok: true,
      value: { isKnown: false, bytesDone: 0, bytesTotal: 0 },
    })
  })

  it('returns the message of a failed call', async () => {
    const client = createDownloadClient({
      getDownloadProgress: () => Promise.reject(new ConnectError('No vault is open', Code.FailedPrecondition)),
    })

    const result = await client.getDownloadProgress('a.pdf')

    expect(result).toStrictEqual({ ok: false, error: 'No vault is open' })
  })
})
