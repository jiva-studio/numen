/**
 * A spec that starts a debug server of its own reads the window from it in the suite's browser.
 * The server writes its log to the stderr of the worker, which the test records.
 */
import { test, expect } from '../../support/test.js'
import { startOwnServer } from '../../support/server.js'
import { expectWorkspaceDrawn } from '../../support/workspace.js'

/** Records what the worker writes to stderr until the returned function is called. */
const recordStderr = (): (() => string) => {
  const chunks: string[] = []
  const write = process.stderr.write.bind(process.stderr)
  process.stderr.write = ((chunk: string | Uint8Array, ...rest: never[]) => {
    chunks.push(typeof chunk === 'string' ? chunk : Buffer.from(chunk).toString())
    return write(chunk, ...rest)
  }) as typeof process.stderr.write
  return () => {
    process.stderr.write = write
    return chunks.join('')
  }
}

test.describe('Debug server of a spec of its own', () => {
  test('logs no TLS handshake error while it starts and serves the window', async ({ page }) => {
    const stopRecording = recordStderr()
    let log: string
    try {
      const server = await startOwnServer([])
      try {
        await page.goto(server.url)
        await expectWorkspaceDrawn(page)
      } finally {
        await server.stop()
      }
    } finally {
      log = stopRecording()
    }

    expect(log.split('\n').filter((line) => line.includes('TLS handshake error'))).toEqual([])
  })
})
