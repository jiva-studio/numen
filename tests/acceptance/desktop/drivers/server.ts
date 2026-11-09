/**
 * A server of its own for one scenario: the debug binary on a port the system gives, with its own
 * certificate, state folders and log, and the stand-ins the e2e server has, so nothing reaches the
 * network. Only the binary and the plugins are shared with the other servers.
 */
import { spawn, type ChildProcess } from 'node:child_process'
import { createHash, X509Certificate } from 'node:crypto'
import fs from 'node:fs'
import http from 'node:http'
import net from 'node:net'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { STAND_IN_MODEL, startStandIns } from '../../../e2e/desktop/scripts/stand-ins.mjs'

const here = path.dirname(fileURLToPath(import.meta.url))
const REPO_ROOT = path.resolve(here, '../../../..')
const BINARY = path.join(REPO_ROOT, 'apps/desktop/numen-debug')
const ARTICLE = path.join(REPO_ROOT, 'tests/e2e/desktop/fixtures/article.html')
const HOST = '127.0.0.1'
const STATE_FOLDERS = ['config', 'cache', 'data', 'state', 'home'] as const

const getFreePort = (): Promise<number> =>
  new Promise((resolve, reject) => {
    const probe = net.createServer()
    probe.once('error', reject)
    probe.listen(0, HOST, () => {
      const { port } = probe.address() as net.AddressInfo
      probe.close(() => resolve(port))
    })
  })

const startPageProxy = async (): Promise<http.Server> => {
  const proxy = http.createServer((request, response) => {
    const host = (request.headers.host ?? '').replace(/:\d+$/, '')
    if (host !== 'numen.dev') {
      response.writeHead(404).end()
      return
    }
    response.writeHead(200, { 'content-type': 'text/html; charset=utf-8' }).end(fs.readFileSync(ARTICLE))
  })
  await new Promise<void>((resolve, reject) => {
    proxy.once('error', reject)
    proxy.listen(0, HOST, resolve)
  })
  return proxy
}

export class Server {
  private child: ChildProcess | undefined
  private baseURL = ''
  private publicKeyHash = ''

  private constructor(
    private readonly state: string,
    private readonly standIns: { close: () => void },
    private readonly proxy: http.Server,
    private readonly mcpPort: number,
  ) {}

  /** Starts the binary with `vault` as the vault it opens. */
  static async start(vault: string): Promise<Server> {
    const state = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), 'numen-state-')))
    const standIns = await startStandIns()
    const server = new Server(state, standIns, await startPageProxy(), await getFreePort())
    try {
      for (const name of STATE_FOLDERS) fs.mkdirSync(path.join(state, name))
      server.writeSettings(standIns)
      await server.launch([`-vault=${vault}`])
    } catch (error) {
      await server.stop()
      throw error
    }
    return server
  }

  getBaseURL(): string {
    return this.baseURL
  }

  /** The hash that makes Chromium trust the certificate of this server alone. */
  getPublicKeyHash(): string {
    return this.publicKeyHash
  }

  readLog(): string {
    return fs.readFileSync(path.join(this.state, 'server.log'), 'utf8')
  }

  /** Ends the binary and starts it again over the same state, which opens the vault it had open. */
  async restart(): Promise<void> {
    await this.end()
    await this.launch([])
  }

  /** Ends the binary and removes everything it kept. */
  async stop(): Promise<void> {
    try {
      await this.end()
    } finally {
      this.standIns.close()
      this.proxy.close()
      fs.rmSync(this.state, { recursive: true, force: true })
    }
  }

  private writeSettings(standIns: { chatUrl: string; updateUrl: string }): void {
    const home = path.join(this.state, 'home')
    const folder =
      process.platform === 'darwin'
        ? path.join(home, 'Library', 'Application Support', 'numen')
        : path.join(this.state, 'config', 'numen')
    fs.mkdirSync(folder, { recursive: true })
    const settings = {
      v: 1,
      app: { agent: { use: 'numen' } },
      'md.numen.agents.numen': { baseUrl: standIns.chatUrl, model: STAND_IN_MODEL },
      'md.numen.tools.updater': { channel: 'custom', endpoint: standIns.updateUrl },
      'md.numen.agents.claude': { enabled: false },
      'md.numen.agents.codex': { enabled: false },
      'md.numen.agents.antigravity': { enabled: false },
      'md.numen.agents.acp': { enabled: false },
      'md.numen.agents.mcp.server': { port: this.mcpPort },
    }
    fs.writeFileSync(path.join(folder, 'numen.json'), JSON.stringify(settings, null, 2))
  }

  private async launch(flags: readonly string[]): Promise<void> {
    const certificate = path.join(this.state, 'cert.pem')
    const log = fs.openSync(path.join(this.state, 'server.log'), 'a')
    const proxyAddress = this.proxy.address() as net.AddressInfo
    const child = spawn(BINARY, [`-debug-http-addr=${HOST}:0`, '-debug-mock-ml', ...flags], {
      stdio: ['ignore', 'pipe', log],
      env: {
        ...process.env,
        XDG_CONFIG_HOME: path.join(this.state, 'config'),
        XDG_CACHE_HOME: path.join(this.state, 'cache'),
        XDG_DATA_HOME: path.join(this.state, 'data'),
        XDG_STATE_HOME: path.join(this.state, 'state'),
        HOME: path.join(this.state, 'home'),
        NUMEN_MOCK_ML: 'true',
        NUMEN_PLUGINS_PATH: path.join(REPO_ROOT, 'plugins'),
        NUMEN_DEBUG_TLS_CERT: certificate,
        NUMEN_DEBUG_TLS_KEY: path.join(this.state, 'key.pem'),
        HTTP_PROXY: `http://${HOST}:${proxyAddress.port}`,
        NO_PROXY: HOST,
      },
    })
    this.child = child
    child.once('close', () => fs.closeSync(log))
    const port = await new Promise<string>((resolve, reject) => {
      let text = ''
      child.stdout?.on('data', (chunk: Buffer) => {
        text += chunk.toString()
        fs.writeSync(log, chunk)
        const found = /listening on https:\/\/127\.0\.0\.1:(\d+)/.exec(text)
        if (found?.[1]) resolve(found[1])
      })
      child.once('error', reject)
      child.once('exit', (code, signal) => reject(new Error(`the server ended on its own (code ${code}, signal ${signal})`)))
    })
    this.baseURL = `https://${HOST}:${port}`
    const publicKey = new X509Certificate(fs.readFileSync(certificate)).publicKey
    this.publicKeyHash = createHash('sha256').update(publicKey.export({ type: 'spki', format: 'der' })).digest('base64')
  }

  private async end(): Promise<void> {
    const child = this.child
    this.child = undefined
    if (child === undefined || child.exitCode !== null) return
    child.removeAllListeners('exit')
    const ended = new Promise((resolve) => child.once('exit', resolve))
    child.kill('SIGTERM')
    await ended
  }
}
