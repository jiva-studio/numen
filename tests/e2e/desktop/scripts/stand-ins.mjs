import http from 'node:http'

/** The words every chat completion of the stand-in chat API answers with. */
export const STAND_IN_REPLY = 'The stand-in chat API answered.'

/** The model the stand-in chat API lists. */
export const STAND_IN_MODEL = 'stand-in/model'

/** The release the stand-in update manifest names: older than any build, so no update is offered. */
const RELEASE = { version: '0.0.0', build: '0', released: '2026-01-01T00:00:00Z', files: [] }

const sendJson = (res, status, body) => {
  res.writeHead(status, { 'Content-Type': 'application/json' })
  res.end(JSON.stringify(body))
}

const handle = (req, res) => {
  const { pathname } = new URL(req.url ?? '/', 'http://127.0.0.1')
  if (req.method === 'GET' && pathname === '/latest/latest.json') {
    sendJson(res, 200, RELEASE)
    return
  }
  if (req.method === 'GET' && pathname === '/v1/models') {
    sendJson(res, 200, { data: [{ id: STAND_IN_MODEL, name: 'Stand-in model' }] })
    return
  }
  if (req.method === 'POST' && pathname === '/v1/chat/completions') {
    req.resume()
    req.on('end', () =>
      sendJson(res, 200, {
        choices: [
          { message: { role: 'assistant', content: STAND_IN_REPLY }, finish_reason: 'stop' },
        ],
      }),
    )
    return
  }
  sendJson(res, 404, { error: `the stand-in answers no ${req.method} ${pathname}` })
}

/**
 * Starts the local server that stands in for what the application reaches outside the
 * machine: the release manifest of dl.numen.md and an OpenAI-compatible chat API.
 *
 * @returns {Promise<{ updateUrl: string, chatUrl: string, close: () => void }>}
 */
export const startStandIns = () =>
  new Promise((resolve, reject) => {
    const server = http.createServer(handle)
    server.once('error', reject)
    server.listen(0, '127.0.0.1', () => {
      const { port } = /** @type {import('node:net').AddressInfo} */ (server.address())
      const origin = `http://127.0.0.1:${port}`
      resolve({
        updateUrl: `${origin}/latest/latest.json`,
        chatUrl: `${origin}/v1`,
        close: () => server.close(),
      })
    })
  })
