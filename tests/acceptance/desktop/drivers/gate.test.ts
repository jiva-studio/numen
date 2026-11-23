import http from 'node:http'
import type { AddressInfo } from 'node:net'
import { afterEach, describe, expect, it } from 'vitest'
import { Gate } from './gate.js'

const stops: (() => void)[] = []

afterEach(() => {
  for (const stop of stops.splice(0)) stop()
})

const listenServer = async (): Promise<{ readonly host: string; readonly port: number }> => {
  const server = http.createServer((_request, response) => {
    response.setHeader('Connection', 'close')
    response.end('answer')
  })
  await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve))
  stops.push(() => server.close())
  return { host: '127.0.0.1', port: (server.address() as AddressInfo).port }
}

const ask = (address: string): Promise<string> =>
  new Promise((resolve, reject) => {
    http
      .get(address, { agent: false }, (response) => {
        let text = ''
        response.on('data', (chunk: Buffer) => (text += chunk.toString()))
        response.on('end', () => resolve(text))
      })
      .on('error', reject)
  })

describe('Gate', () => {
  it('forwards a request to the server while it is open', async () => {
    const target = await listenServer()
    const gate = await Gate.listen(() => target)
    stops.push(() => gate.stop())

    expect(await ask(gate.getAddress())).toBe('answer')
  })

  it('refuses a request while it is closed and forwards again once it is reopened', async () => {
    const target = await listenServer()
    const gate = await Gate.listen(() => target)
    stops.push(() => gate.stop())

    gate.close()
    await expect(ask(gate.getAddress())).rejects.toThrow()
    gate.reopen()

    expect(await ask(gate.getAddress())).toBe('answer')
  })
})
