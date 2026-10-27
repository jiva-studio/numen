import { spawn } from 'node:child_process'
import fs from 'node:fs'
import http from 'node:http'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { STAND_IN_MODEL, startStandIns } from './stand-ins.mjs'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const sourceVault = path.resolve(__dirname, '../../../vault')
const fixturesDir = path.resolve(__dirname, '../fixtures')
const binaryPath = path.resolve(__dirname, '../../../../apps/desktop/numen-debug')
const repoRoot = path.resolve(__dirname, '../../../..')
const pluginsPath = path.resolve(repoRoot, 'plugins')

const port = process.env.NUMEN_TEST_PORT ?? '3456'
const host = '127.0.0.1'

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

// Create isolated temporary vault from template.
const tmpVault = fs.mkdtempSync(path.join(os.tmpdir(), 'numen-e2e-vault-'))
fs.cpSync(sourceVault, tmpVault, { recursive: true })
createSampleWav(path.join(tmpVault, 'assets', 'sample.wav'))

// Copy dynamic fixtures into isolated vault to prevent template fixture pollution.
fs.copyFileSync(path.join(fixturesDir, 'sample.pdf'), path.join(tmpVault, 'assets', 'sample.pdf'))
fs.copyFileSync(path.join(fixturesDir, 'sample.epub'), path.join(tmpVault, 'assets', 'sample.epub'))
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

// The agent panel answers from numen's own agent over the stand-in chat API. The agents that
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
      agent: { use: 'numen' },
      plugins: {
        'md.numen.agents.numen': { baseUrl: standIns.chatUrl, model: STAND_IN_MODEL },
        'md.numen.agents.claude': { enabled: false },
        'md.numen.agents.codex': { enabled: false },
        'md.numen.agents.antigravity': { enabled: false },
        'md.numen.agents.acp': { enabled: false },
      },
    },
    null,
    2,
  ),
)

function cleanup() {
  standIns.close()
  try {
    fs.rmSync(tmpVault, { recursive: true, force: true })
  } catch {
    // Ignore cleanup errors on busy files.
  }
}

// The proxy every plain-http fetch of the binary and its plugins goes through. It serves PAGES by
// host and answers every other host 404, so a page a test imports is never fetched from the network.
const pagesPort = Number(port) + 1
const PAGES = new Map([['tidal-mills.numen.test', path.join(fixturesDir, 'article.html')]])
const pages = http.createServer((req, res) => {
  const page = PAGES.get(new URL(req.url ?? '/', 'http://' + req.headers.host).hostname)
  if (!page) {
    res.writeHead(404).end()
    return
  }
  res.writeHead(200, { 'content-type': 'text/html; charset=utf-8' }).end(fs.readFileSync(page))
})
pages.listen(pagesPort, host)

const child = spawn(
  binaryPath,
  [
    `-debug-http-addr=${host}:${port}`,
    '-debug-mock-ml',
    `-debug-update-url=${standIns.updateUrl}`,
    '-no-mcp',
    `-vault=${tmpVault}`,
  ],
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
      HTTP_PROXY: `http://${host}:${pagesPort}`,
      NO_PROXY: host,
    },
  },
)

child.on('exit', (code) => {
  pages.close()
  cleanup()
  process.exit(code ?? 0)
})

for (const sig of ['SIGINT', 'SIGTERM']) {
  process.on(sig, () => {
    cleanup()
    child.kill(sig)
  })
}
