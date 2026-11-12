/**
 * A server of its own for one scenario: the debug binary of the editor or of the flashcards window
 * on a port the system gives, with its own certificate, state folders and log, and the stand-ins the
 * e2e server has, so nothing reaches the network. Only the binary and the plugins are shared with
 * the other servers.
 */
import { spawn, type ChildProcess } from 'node:child_process'
import { createHash, X509Certificate } from 'node:crypto'
import fs from 'node:fs'
import http from 'node:http'
import net from 'node:net'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { STAND_IN_CONNECTION, STAND_IN_MODEL, startStandIns } from '../../../e2e/desktop/scripts/stand-ins.mjs'
import { freeFromRemoval, holdForRemoval } from './reaper.js'
import { applyEnvironment, type EnvironmentSetting, type EnvironmentTargets } from './environment.js'
import type { Application } from './rules.js'

const here = path.dirname(fileURLToPath(import.meta.url))
const REPO_ROOT = path.resolve(here, '../../../..')
const BINARIES = {
  editor: path.join(REPO_ROOT, 'apps/desktop/numen-debug'),
  flashcards: path.join(REPO_ROOT, 'apps/desktop/numen-flashcards-debug'),
} as const
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

const getTargets = (state: string, updateUrl: string): EnvironmentTargets => ({
  origin: new URL(updateUrl).origin,
  embeddingHoldFile: path.join(state, 'embedding-hold'),
})

const escapeHtml = (text: string): string => text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

const startPageProxy = async (readPageText: (address: string) => string | undefined): Promise<http.Server> => {
  const proxy = http.createServer((request, response) => {
    const host = (request.headers.host ?? '').replace(/:\d+$/, '')
    if (host !== 'numen.dev') {
      response.writeHead(404).end()
      return
    }
    const page = fs.readFileSync(ARTICLE, 'utf8')
    const text = readPageText(request.url ?? '')
    const body = text === undefined ? page : page.replace(/<p>[\s\S]*?<\/p>/, () => `<p>${escapeHtml(text)}</p>`)
    response.writeHead(200, { 'content-type': 'text/html; charset=utf-8' }).end(body)
  })
  await new Promise<void>((resolve, reject) => {
    proxy.once('error', reject)
    proxy.listen(0, HOST, resolve)
  })
  return proxy
}

export interface ServerOptions {
  readonly application: Application
  readonly vault: string
  readonly readPageText: (address: string) => string | undefined
  readonly environment: readonly EnvironmentSetting[]
}

interface Services {
  readonly standIns: { close: () => void; chatUrl: string; updateUrl: string }
  readonly proxy: http.Server
  readonly mcpPort: number
}

export class Server {
  private child: ChildProcess | undefined
  private baseURL = ''
  private publicKeyHash = ''

  private constructor(
    private readonly options: ServerOptions,
    private readonly state: string,
    private readonly services: Services,
  ) {}

  /** Starts the binary of the application with the vault of the options as the vault it opens. */
  static async start(options: ServerOptions): Promise<Server> {
    const state = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), 'numen-state-')))
    holdForRemoval('dir', state)
    const standIns = await startStandIns()
    const proxy = await startPageProxy(options.readPageText)
    const server = new Server(options, state, { standIns, proxy, mcpPort: await getFreePort() })
    try {
      for (const name of STATE_FOLDERS) fs.mkdirSync(path.join(state, name))
      fs.mkdirSync(path.join(state, 'home', 'run'))
      server.writeSettings(standIns)
      await server.setEnvironment(options.environment)
      await server.launch(server.getStartFlags(true))
    } catch (error) {
      await server.stop()
      throw error
    }
    return server
  }

  getBaseURL(): string {
    return this.baseURL
  }

  /** The address of the release feed the updater plugin asks. */
  getUpdateUrl(): string {
    return this.services.standIns.updateUrl
  }

  /** Makes the folder dialog answer with the folder, which the command Create vault asks for. */
  writeFolderChoice(folder: string): void {
    fs.writeFileSync(path.join(this.state, 'folder-choice.txt'), folder)
  }

  /** Puts the settings of the environment in force in the stand-ins of this server. */
  async setEnvironment(settings: readonly EnvironmentSetting[]): Promise<void> {
    await applyEnvironment(settings, getTargets(this.state, this.services.standIns.updateUrl))
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
    await this.launch(this.getStartFlags(false))
  }

  /** Ends the binary and removes everything it kept. */
  async stop(): Promise<void> {
    try {
      await this.end()
    } finally {
      this.services.standIns.close()
      this.services.proxy.close()
      fs.rmSync(this.state, { recursive: true, force: true })
      freeFromRemoval('dir', this.state)
    }
  }

  private getStartFlags(isFirst: boolean): string[] {
    if (this.options.application === 'flashcards') return [`-debug-vault=${this.options.vault}`]
    const common = ['-debug-mock-ml', `-debug-folder-file=${path.join(this.state, 'folder-choice.txt')}`]
    return isFirst ? [`-vault=${this.options.vault}`, ...common] : common
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
      connections: [
        {
          id: STAND_IN_CONNECTION,
          type: 'md.numen.connections.llm/openai-compatible',
          name: 'Stand-in',
          revision: 1,
          fields: { baseUrl: standIns.chatUrl },
        },
      ],
      'md.numen.agents.numen': { connections: [{ connection: STAND_IN_CONNECTION, model: STAND_IN_MODEL }] },
      'md.numen.agents.acp': { enabled: false },
      ...(this.options.application === 'editor'
        ? {
            'md.numen.tools.updater': { channel: 'custom', endpoint: standIns.updateUrl },
            'md.numen.agents.mcp.server': { port: this.services.mcpPort },
            // The scenarios write short notes, and a piece shorter than 40 characters gets no vector by default.
            'md.numen.engines.embeddings': { minChunkLength: 0 },
          }
        : {}),
    }
    fs.writeFileSync(path.join(folder, 'numen.json'), JSON.stringify(settings, null, 2))
  }

  private async launch(flags: readonly string[]): Promise<void> {
    const certificate = path.join(this.state, 'cert.pem')
    const log = fs.openSync(path.join(this.state, 'server.log'), 'a')
    const proxyAddress = this.services.proxy.address() as net.AddressInfo
    const editorEnv =
      this.options.application === 'editor'
        ? { NUMEN_TEST_MOCK_ML: 'true', HTTP_PROXY: `http://${HOST}:${proxyAddress.port}`, NO_PROXY: HOST }
        : {}
    const child = spawn(BINARIES[this.options.application], [`-debug-http-addr=${HOST}:0`, ...flags], {
      stdio: ['ignore', 'pipe', log],
      env: {
        ...process.env,
        XDG_CONFIG_HOME: path.join(this.state, 'config'),
        XDG_CACHE_HOME: path.join(this.state, 'cache'),
        XDG_DATA_HOME: path.join(this.state, 'data'),
        XDG_STATE_HOME: path.join(this.state, 'state'),
        HOME: path.join(this.state, 'home'),
        // The run has no session bus and no Secret Service, so the keys of its connections are kept in a file.
        XDG_RUNTIME_DIR: path.join(this.state, 'home', 'run'),
        DBUS_SESSION_BUS_ADDRESS: '',
        ...editorEnv,
        NUMEN_TEST_MOCK_EMBED_HOLD_FILE: path.join(this.state, 'embedding-hold'),
        NUMEN_PLUGINS_PATH: path.join(REPO_ROOT, 'plugins'),
        NUMEN_DEBUG_TLS_CERT: certificate,
        NUMEN_DEBUG_TLS_KEY: path.join(this.state, 'key.pem'),
      },
    })
    this.child = child
    if (child.pid !== undefined) holdForRemoval('pid', child.pid)
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
    if (child.pid !== undefined) freeFromRemoval('pid', child.pid)
  }
}
