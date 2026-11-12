import http from 'node:http'

/** The words every chat completion of the stand-in chat API answers with. */
export const STAND_IN_REPLY = 'The agent answered.'

/**
 * A prompt holding these words is answered once `DELETE /stand-in/hold` opens the gate that
 * `PUT /stand-in/hold` closed, so a test sees the stop state for as long as it needs to.
 */
export const HELD_PROMPT = 'Answer when released'

/** The model the stand-in chat API lists. */
export const STAND_IN_MODEL = 'stand-in/model'

/** A prompt holding these words is answered with a call of the vault tool that writes a note. */
export const TOOL_PROMPT = 'Write the tide table note'

/** The note the tool call of `TOOL_PROMPT` writes, and what it writes there. */
export const TOOL_NOTE_PATH = 'notes/Tide table.md'
export const TOOL_NOTE_TEXT = '# Tide table\n\nHigh water at the quay at 06:40 and at 19:05.\n'

/** The vault tool the stand-in calls, as the tool server names it. */
const NOTE_WRITE_TOOL = 'note_write'

/** The release the stand-in update manifest names first: older than any build, so no update is offered. */
const OLD_RELEASE = { version: '0.0.0', build: '0', released: '2026-01-01T00:00:00Z', files: [] }

/** The release the update manifest names now. */
let release = OLD_RELEASE

/** The held replies wait while this gate is closed; each is the function that sends one reply. */
let isHoldClosed = false
/** Whether the closed gate holds every reply, and not only the replies to a held prompt. */
let isHoldAll = false
const heldReplies = new Set()

const closeHold = (req, res) =>
  readBody(req).then((raw) => {
    isHoldClosed = true
    isHoldAll = raw.trim() !== '' && JSON.parse(raw).all === true
    sendJson(res, 200, { held: heldReplies.size })
  })

const openHold = (_req, res) => {
  isHoldClosed = false
  isHoldAll = false
  for (const send of heldReplies) send()
  heldReplies.clear()
  sendJson(res, 200, { held: 0 })
}

const sendJson = (res, status, body) => {
  res.writeHead(status, { 'Content-Type': 'application/json' })
  res.end(JSON.stringify(body))
}

const readBody = (req) =>
  new Promise((resolve) => {
    const chunks = []
    req.on('data', (chunk) => chunks.push(chunk))
    req.on('end', () => resolve(Buffer.concat(chunks).toString('utf8')))
  })

/** The text of a message, whose content is a string or a list of parts. */
const readText = (message) =>
  Array.isArray(message?.content)
    ? message.content.map((part) => part?.text ?? '').join('')
    : String(message?.content ?? '')

/** Whether a tool result answers a call of the conversation. */
const hasToolResult = (request) => (request.messages ?? []).some((one) => one.role === 'tool')

/** Whether the chat request offers the vault tool that writes a note. */
const hasNoteWriteTool = (request) =>
  (request.tools ?? []).some((one) => one.function?.name === NOTE_WRITE_TOOL)

/**
 * The tool calls the reply to a chat request makes: one note_write for a last prompt holding
 * `TOOL_PROMPT` that no tool result answers yet, and none otherwise.
 */
const createToolCalls = (request) => {
  const prompt = readText((request.messages ?? []).filter((one) => one.role === 'user').at(-1))
  if (!prompt.includes(TOOL_PROMPT) || !hasNoteWriteTool(request)) return []
  if (hasToolResult(request)) return []
  return [
    {
      id: 'call-tide-table',
      type: 'function',
      function: {
        name: NOTE_WRITE_TOOL,
        arguments: JSON.stringify({ path: TOOL_NOTE_PATH, text: TOOL_NOTE_TEXT }),
      },
    },
  ]
}

/** The text of a reply that makes no tool call, which says why where a tool call was asked for. */
const createReplyText = (raw, request) => {
  if (raw.includes('Link to Entropy')) return `${STAND_IN_REPLY} See [Entropy](notes/Entropy.md).`
  if (!raw.includes(TOOL_PROMPT)) return STAND_IN_REPLY
  const result = (request.messages ?? []).find((one) => one.role === 'tool')
  if (result)
    return `${STAND_IN_REPLY} The tide table is written. The tool answered: ${readText(result)}`
  return hasNoteWriteTool(request)
    ? `${STAND_IN_REPLY} The prompt held no ${TOOL_PROMPT} the stand-in read.`
    : `${STAND_IN_REPLY} No ${NOTE_WRITE_TOOL} tool was offered.`
}

const answerChat = (req, res) =>
  readBody(req).then((raw) => {
    const request = JSON.parse(raw)
    const calls = createToolCalls(request)
    const message =
      calls.length > 0
        ? { role: 'assistant', content: null, tool_calls: calls }
        : { role: 'assistant', content: createReplyText(raw, request) }
    const reply = () => {
      if (res.destroyed) return
      sendJson(res, 200, {
        choices: [{ message, finish_reason: calls.length > 0 ? 'tool_calls' : 'stop' }],
      })
    }
    if ((isHoldAll || raw.includes(HELD_PROMPT)) && isHoldClosed) {
      heldReplies.add(reply)
      res.on('close', () => heldReplies.delete(reply))
      return
    }
    reply()
  })

const switchRelease = (req, res) =>
  readBody(req).then((raw) => {
    release = JSON.parse(raw)
    sendJson(res, 200, release)
  })

/** The answers by method and path; each takes the request and the response. */
const routes = new Map([
  ['GET /latest/latest.json', (_req, res) => sendJson(res, 200, release)],
  ['PUT /stand-in/release', switchRelease],
  [
    'DELETE /stand-in/release',
    (_req, res) => {
      release = OLD_RELEASE
      sendJson(res, 200, release)
    },
  ],
  ['PUT /stand-in/hold', closeHold],
  ['DELETE /stand-in/hold', openHold],
  [
    'GET /v1/models',
    (_req, res) => sendJson(res, 200, { data: [{ id: STAND_IN_MODEL, name: 'Test model' }] }),
  ],
  ['POST /v1/chat/completions', answerChat],
])

const handle = (req, res) => {
  const { pathname } = new URL(req.url ?? '/', 'http://127.0.0.1')
  const answer = routes.get(`${req.method} ${pathname}`)
  if (!answer) {
    sendJson(res, 404, { error: `the stand-in answers no ${req.method} ${pathname}` })
    return
  }
  Promise.resolve(answer(req, res)).catch((error) => {
    if (!res.headersSent) sendJson(res, 500, { error: String(error) })
  })
}

/**
 * Starts the local server that is in for what the application reaches outside the
 * machine: the release manifest of dl.numen.md and an OpenAI-compatible chat API. A test switches
 * the release through `/stand-in/release`.
 *
 * @returns {Promise<{ updateUrl: string, chatUrl: string, close: () => void }>}
 */
export const startStandIns = () => {
  // A worker starts the stand-ins once per scenario: what an earlier scenario switched does not carry over.
  release = OLD_RELEASE
  isHoldClosed = false
  isHoldAll = false
  heldReplies.clear()
  return startServer()
}

const startServer = () =>
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
