import { spawn } from 'node:child_process'
import fs from 'node:fs'
import http from 'node:http'
import net from 'node:net'
import https from 'node:https'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { STAND_IN_CONNECTION, STAND_IN_MODEL, startStandIns } from './stand-ins.mjs'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const sourceVault = path.resolve(__dirname, '../../../vault')
const fixturesDir = path.resolve(__dirname, '../fixtures')
const binaryPath = path.resolve(__dirname, '../../../../apps/desktop/numen-debug')
const repoRoot = path.resolve(__dirname, '../../../..')
const pluginsPath = path.resolve(repoRoot, 'plugins')

const port = process.env.NUMEN_TEST_PORT ?? '3456'
const host = '127.0.0.1'

// A server this run did not start belongs to another run, which stops it and its vault while this
// run still uses them, and every test then fails at once: a port that answers fails the start.
const isPortTaken = await new Promise((resolve) => {
  const probe = net.connect({ host, port: Number(port) })
  probe.once('connect', () => (probe.destroy(), resolve(true)))
  probe.once('error', () => resolve(false))
})
if (isPortTaken) {
  console.error(`start-server: ${host}:${port} is already used; give NUMEN_TEST_PORT another port`)
  process.exit(1)
}

function createSampleWav(targetPath) {
  const sampleRate = 16000
  const numChannels = 1
  const bytesPerSample = 2
  const durationSec = 1
  const numSamples = sampleRate * durationSec
  const dataSize = numSamples * numChannels * bytesPerSample
  const buffer = Buffer.alloc(44 + dataSize)

  buffer.write('RIFF', 0)
  buffer.writeUInt32LE(36 + dataSize, 4)
  buffer.write('WAVE', 8)
  buffer.write('fmt ', 12)
  buffer.writeUInt32LE(16, 16)
  buffer.writeUInt16LE(1, 20)
  buffer.writeUInt16LE(numChannels, 22)
  buffer.writeUInt32LE(sampleRate, 24)
  buffer.writeUInt32LE(sampleRate * numChannels * bytesPerSample, 28)
  buffer.writeUInt16LE(numChannels * bytesPerSample, 32)
  buffer.writeUInt16LE(bytesPerSample * 8, 34)
  buffer.write('data', 36)
  buffer.writeUInt32LE(dataSize, 40)

  fs.writeFileSync(targetPath, buffer)
}

// A US Letter PDF of pageCount pages, each with its number printed on it.
function createPagedPdf(targetPath, pageCount) {
  const objects = [
    '<< /Type /Catalog /Pages 2 0 R >>',
    '',
    '<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>',
  ]
  const kids = []
  for (let page = 1; page <= pageCount; page++) {
    const content = `BT /F1 96 Tf 220 380 Td (${page}) Tj ET`
    objects.push(`<< /Length ${content.length} >>\nstream\n${content}\nendstream`)
    objects.push(
      `<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 3 0 R >> >> /Contents ${objects.length} 0 R >>`,
    )
    kids.push(`${objects.length} 0 R`)
  }
  objects[1] = `<< /Type /Pages /Kids [${kids.join(' ')}] /Count ${pageCount} >>`
  let body = '%PDF-1.4\n'
  const offsets = []
  for (const [index, object] of objects.entries()) {
    offsets.push(body.length)
    body += `${index + 1} 0 obj\n${object}\nendobj\n`
  }
  const xref = body.length
  body += `xref\n0 ${objects.length + 1}\n0000000000 65535 f \n`
  body += offsets.map((offset) => `${String(offset).padStart(10, '0')} 00000 n \n`).join('')
  body += `trailer\n<< /Size ${objects.length + 1} /Root 1 0 R >>\nstartxref\n${xref}\n%%EOF\n`
  fs.writeFileSync(targetPath, body, 'latin1')
}

// Create isolated temporary vault from template.
const tmpVault = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), 'numen-e2e-vault-')))
fs.cpSync(sourceVault, tmpVault, { recursive: true })
createSampleWav(path.join(tmpVault, 'assets', 'sample.wav'))
createPagedPdf(path.join(tmpVault, 'assets', 'pages.pdf'), 40)

// Copy dynamic fixtures into isolated vault to prevent template fixture pollution.
fs.copyFileSync(path.join(fixturesDir, 'sample.pdf'), path.join(tmpVault, 'assets', 'sample.pdf'))
fs.copyFileSync(path.join(fixturesDir, 'sample.epub'), path.join(tmpVault, 'assets', 'sample.epub'))
fs.copyFileSync(path.join(fixturesDir, 'ledger.epub'), path.join(tmpVault, 'assets', 'ledger.epub'))
for (const name of [
  'regenerative-braking.pdf',
  'flywheel.pdf',
  'shore.pdf',
  'pumped-storage.pdf',
  'lecture.wav',
  'talk.mp3',
]) {
  fs.copyFileSync(path.join(fixturesDir, name), path.join(tmpVault, 'assets', name))
}
fs.writeFileSync(path.join(tmpVault, 'assets', 'corrupted.pdf'), 'INVALID_PDF_HEADER')
fs.writeFileSync(path.join(tmpVault, 'assets', 'corrupted.epub'), 'INVALID_EPUB_ZIP')
fs.writeFileSync(path.join(tmpVault, 'assets', 'sample.mp3'), Buffer.alloc(512))
fs.writeFileSync(path.join(tmpVault, 'assets', 'sample.flac'), Buffer.alloc(512))
fs.writeFileSync(
  path.join(tmpVault, 'notes', 'plain.md'),
  'Plain text file opened via markdown editor plugin.',
)
fs.writeFileSync(
  path.join(tmpVault, 'notes', 'extra.md'),
  '# Markdown heading\nContent from extra markdown file.',
)
fs.writeFileSync(
  path.join(tmpVault, 'notes', 'crossing.md'),
  '# Crossing\n\nThe ferry leaves the harbour at dawn with the morning post.',
)
fs.writeFileSync(
  path.join(tmpVault, 'notes', 'trash-target.md'),
  '# Disposable Note\nThis note is created for deletion testing.',
)
fs.writeFileSync(
  path.join(tmpVault, 'notes', 'malformed.md'),
  '---\ntitle: [unclosed bracket\nkey: value:\n---\n# Malformed Note\nContent after malformed frontmatter.',
)
fs.writeFileSync(
  path.join(tmpVault, 'notes', 'large.md'),
  '# Large Document\n' +
    Array.from(
      { length: 800 },
      (_, i) => `Paragraph line ${i + 1} with sample text for stress testing.`,
    ).join('\n\n'),
)

// Ensure Thermodynamics.md is the most recently modified note so it is always the initial opening note.
const thermoFile = path.join(tmpVault, 'Thermodynamics.md')
const futureTime = new Date(Date.now() + 100_000)
fs.utimesSync(thermoFile, futureTime, futureTime)

const testConfigDir = path.join(tmpVault, '.config')
fs.mkdirSync(testConfigDir, { recursive: true })
// Each server keeps its caches in a folder of its own.
const testCacheDir = path.join(tmpVault, '.cache')
fs.mkdirSync(testCacheDir, { recursive: true })
// The index, the trash, the dependencies and whatever a plugin keeps under the home folder stay in
// the run too.
const testDataDir = path.join(tmpVault, '.data')
fs.mkdirSync(testDataDir, { recursive: true })
const testStateDir = path.join(tmpVault, '.state')
fs.mkdirSync(testStateDir, { recursive: true })
const testHomeDir = path.join(tmpVault, '.home')
fs.mkdirSync(testHomeDir, { recursive: true })

const standIns = await startStandIns()

// The agent panel answers from numen's own agent through a connection to the stand-in chat API. The agents that
// run a program the machine has installed stay off: those programs reach their own services.
const settingsDir =
  process.platform === 'darwin'
    ? path.join(testHomeDir, 'Library', 'Application Support', 'numen')
    : path.join(testConfigDir, 'numen')
fs.mkdirSync(settingsDir, { recursive: true })
fs.writeFileSync(
  path.join(settingsDir, 'numen.json'),
  JSON.stringify(
    {
      v: 1,
      app: { agent: { use: 'numen' } },
      connections: [
        {
          id: STAND_IN_CONNECTION,
          type: 'md.numen.connections.llm/openai-compatible',
          name: 'Stand-in',
          revision: 1,
          fields: { baseUrl: standIns.chatUrl },
        },
      ],
      'md.numen.agents.numen': {
        connections: [{ connection: STAND_IN_CONNECTION, model: STAND_IN_MODEL }],
      },
      'md.numen.tools.updater': { channel: 'custom', endpoint: standIns.updateUrl },
      'md.numen.agents.claude': { enabled: false },
      'md.numen.agents.codex': { enabled: false },
      'md.numen.agents.antigravity': { enabled: false },
      'md.numen.agents.acp': { enabled: false },
      // The host binds the MCP server's listener port 5000 above the server's port, outside the ports
      // of the other servers of a parallel run (10 apart) and of the flashcards server (3457).
      'md.numen.agents.mcp.server': { port: Number(port) + 5000 },
    },
    null,
    2,
  ),
)

function cleanup() {
  fs.rmSync(vaultPathFile, { force: true })
  fs.rmSync(certificateFile, { force: true })
  fs.rmSync(keyFile, { force: true })
  standIns.close()
  try {
    fs.rmSync(tmpVault, { recursive: true, force: true })
  } catch (error) {
    console.warn(`could not remove ${tmpVault}: ${error.message}`)
  }
}

// The proxy every plain-http fetch of the binary and its plugins goes through. It serves PAGES by
// host and answers every other host 404, so a page a test imports is never fetched from the network.
const PAGES = new Map([['tidal-mills.numen.test', path.join(fixturesDir, 'article.html')]])
const pages = http.createServer((req, res) => {
  const page = PAGES.get((req.headers.host ?? '').replace(/:\d+$/, ''))
  if (!page) {
    res.writeHead(404).end()
    return
  }
  res.writeHead(200, { 'content-type': 'text/html; charset=utf-8' }).end(fs.readFileSync(page))
})
// The port is the system's to give: the next port after the server's is another run's server.
await new Promise((resolve, reject) => {
  pages.once('error', reject)
  pages.listen(0, host, resolve)
})
const pagesPort = pages.address().port

// The tests read the folder of the run's vault from here: the file is named by the port the server listens on.
const vaultPathFile = path.join(os.tmpdir(), `numen-e2e-vault-path-${port}`)
fs.writeFileSync(vaultPathFile, tmpVault)

// The server writes its certificate here; every https call to it trusts that certificate alone.
// tests/support/certificate.ts names the same file.
const certificateFile = path.join(os.tmpdir(), `numen-e2e-certificate-${port}.pem`)
fs.rmSync(certificateFile, { force: true })
// Its key goes beside it, for the servers a spec starts itself to serve the same pair.
const keyFile = path.join(os.tmpdir(), `numen-e2e-key-${port}.pem`)
fs.rmSync(keyFile, { force: true })

/** How long the first pass over the vault may take before the server counts as failed. */
const INDEX_LIMIT_MS = 180_000

let stopping = false

/** Calls the core of the server over https and gives back what it answers, or nothing while it does not. */
const callCore = (method, body, isStream) =>
  new Promise((resolve) => {
    if (!fs.existsSync(certificateFile)) {
      resolve(undefined)
      return
    }
    const json = Buffer.from(JSON.stringify(body))
    // A Connect stream is a message in a five-byte envelope: the first message is all that is read.
    const payload = isStream ? Buffer.concat([Buffer.from([0, 0, 0, 0, 0]), json]) : json
    if (isStream) payload.writeUInt32BE(json.length, 1)
    const call = https.request(
      {
        host,
        port: Number(port),
        path: `/numen.v1.${method}`,
        method: 'POST',
        ca: fs.readFileSync(certificateFile),
        headers: { 'Content-Type': isStream ? 'application/connect+json' : 'application/json' },
      },
      (answer) => {
        let received = Buffer.alloc(0)
        const read = (message) => {
          try {
            resolve(JSON.parse(message.toString()))
          } catch {
            resolve(undefined)
          }
        }
        answer.on('data', (chunk) => {
          received = Buffer.concat([received, chunk])
          if (!isStream || received.length < 5 || received.length < 5 + received.readUInt32BE(1))
            return
          call.destroy()
          read(received.subarray(5, 5 + received.readUInt32BE(1)))
        })
        answer.on('end', () => (isStream ? resolve(undefined) : read(received)))
        answer.on('error', () => resolve(undefined))
      },
    )
    call.on('error', () => resolve(undefined))
    call.end(payload)
  })

/** Whether the vault is read, every chunk has its vector, and no task of the window is running. */
const readIdleState = async () => {
  const state = await callCore('VaultService/GetVaultState', {}, false)
  const window = await callCore('WindowService/WatchTasks', { window: 'editor' }, true)
  return (
    state?.scan?.isReady === true &&
    state.coverage?.chunkCount === state.coverage?.embeddedCount &&
    window !== undefined &&
    (window.tasks ?? []).every((task) => task.error)
  )
}

const child = spawn(
  binaryPath,
  [`-debug-http-addr=${host}:${port}`, '-debug-mock-ml', `-vault=${tmpVault}`],
  {
    stdio: 'inherit',
    env: {
      ...process.env,
      XDG_CONFIG_HOME: testConfigDir,
      XDG_CACHE_HOME: testCacheDir,
      XDG_DATA_HOME: testDataDir,
      XDG_STATE_HOME: testStateDir,
      HOME: testHomeDir,
      NUMEN_MOCK_ML: 'true',
      NUMEN_PLUGINS_PATH: pluginsPath,
      NUMEN_DEBUG_TLS_CERT: certificateFile,
      NUMEN_DEBUG_TLS_KEY: keyFile,
      HTTP_PROXY: `http://${host}:${pagesPort}`,
      NO_PROXY: host,
    },
  },
)

child.on('error', (error) => {
  console.error(`start-server: cannot start ${binaryPath}: ${error.message}`)
  pages.close()
  cleanup()
  process.exit(1)
})

child.on('exit', (code, signal) => {
  // A server that ends before the run stops it leaves every test with a refused connection.
  if (!stopping)
    console.error(`start-server: the server ended on its own (code ${code}, signal ${signal})`)
  pages.close()
  cleanup()
  process.exit(code ?? (signal ? 1 : 0))
})

for (const sig of ['SIGINT', 'SIGTERM']) {
  process.on(sig, () => {
    stopping = true
    cleanup()
    child.kill(sig)
  })
}

// The server counts as started once the vault is read, every chunk has its vector and no task runs:
// a test that begins while the first passes still run finds "1 active" where it waits for "Ready",
// and searches and completes the notes it made before the passes end. Servers that start together
// take longer for those passes than one that starts alone. The idle state must hold for four reads
// in a row, since the next pass is reported a moment after the last ends.
const startedAt = Date.now()
let idleReads = 0
while (idleReads < 4) {
  if (Date.now() - startedAt > INDEX_LIMIT_MS) {
    console.error(`start-server: the server was not idle in ${INDEX_LIMIT_MS / 1000} s`)
    child.kill('SIGTERM')
    break
  }
  idleReads = (await readIdleState()) ? idleReads + 1 : 0
  await new Promise((resolve) => setTimeout(resolve, 250))
}
console.log(`numen e2e: the vault is indexed after ${(Date.now() - startedAt) / 1000} s`)
