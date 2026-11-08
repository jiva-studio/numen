import type { Api } from '@numen/plugin-sdk'
import { describe, expect, it, vi } from 'vitest'
import { createState } from '../model/fixtures'
import { createFloorClient } from './wire'

function createFakeApi(answer: { value: unknown } | { error: { code: string; message: string } }) {
  const call = vi.fn(async () =>
    'value' in answer
      ? ({ ok: true, value: answer.value } as const)
      : ({ ok: false, error: answer.error } as const),
  )
  const api = { call, stream: vi.fn() } as unknown as Api
  return { api, call }
}

describe('the client of the relevance service', () => {
  it('calls each method by name with the request of the panel', async () => {
    const state = createState()
    const { api, call } = createFakeApi({ value: state })
    const client = createFloorClient(api)

    await client.getFloor({})
    await client.calibrateFloor({})
    await client.setFloor({ isManual: true, floor: 0.5 })
    await client.probeFloor({ query: 'entropy' })

    expect(call.mock.calls).toEqual([
      ['GetFloor', {}],
      ['CalibrateFloor', {}],
      ['SetFloor', { isManual: true, floor: 0.5 }],
      ['ProbeFloor', { query: 'entropy' }],
    ])
  })

  it('returns the response of the method', async () => {
    const state = createState()
    const { api } = createFakeApi({ value: state })
    await expect(createFloorClient(api).getFloor({})).resolves.toBe(state)
  })

  it('fails with the code and the text the backend gave', async () => {
    const { api } = createFakeApi({ error: { code: 'unavailable', message: 'no embedder' } })
    await expect(createFloorClient(api).probeFloor({ query: 'x' })).rejects.toThrow(
      '[unavailable] no embedder',
    )
  })
})
