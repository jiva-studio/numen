/**
 * What the source of a plugin shows about its network use, read statically: the network client
 * calls of its Go code and of its frontend code (tests, stories, fixtures and test data left out),
 * the hosts of the URL literals in that code, the origins its manifest frames, the hosts its
 * downloads come from and the `link:open` permission. Only a call that can reach the internet
 * counts: one on a relative path, on a URL the host builds for the plugin or on a loopback address
 * does not. Calls made inside its dependencies are not traced.
 */
import { readFileSync, readdirSync } from 'node:fs'
import { join, relative, sep } from 'node:path'

const SKIPPED_FOLDERS = new Set(['node_modules', 'dist', 'testdata', 'fixtures', '.git'])

const TEST_FILE = /(_test\.go|\.test\.ts|\.stories\.ts|\.d\.ts)$/

const GO_CALLS = [
  { name: 'http.Client', pattern: /\bhttp\.(?:Client|Transport)\s*\{|\bhttp\.Default(?:Client|Transport)\b/ },
  { name: 'http.Get', pattern: /\bhttp\.(?:Get|Head|Post|PostForm)\(/ },
  { name: 'http.NewRequest', pattern: /\bhttp\.NewRequest(?:WithContext)?\(/ },
  { name: 'net.Dial', pattern: /\bnet\.Dial(?:Timeout)?\(|\bnet\.Dialer\s*\{/ },
  { name: 'tls.Dial', pattern: /\btls\.Dial(?:WithDialer)?\(/ },
  { name: 'websocket.Dial', pattern: /\bwebsocket\.(?:Dial|DefaultDialer)\b/ },
  { name: 'grpc.Dial', pattern: /\bgrpc\.(?:Dial|DialContext|NewClient)\(/ },
]

const TS_CALLS = [
  { name: 'fetch', pattern: /(?<![\w.$])fetch\(/g },
  { name: 'XMLHttpRequest', pattern: /\bnew XMLHttpRequest\b/g },
  { name: 'WebSocket', pattern: /\bnew WebSocket\(/g },
  { name: 'EventSource', pattern: /\bnew EventSource\(/g },
]

const URL_LITERAL = /["'`]https?:\/\/([^"'`/:?#\s]+)/g

/** The http(s) and ws(s) literals of Go code, and its bare `host:port` literals of a loopback host. */
const GO_TARGET_LITERAL =
  /"(?:(?:https?|wss?):\/\/(\[[^\]"]*\]|[^"/:?#\s]+)|(localhost|127(?:\.\d{1,3}){3}|\[::1\]):\d+")/g

/** Helpers of `@numen/plugin-sdk` that build a URL of the host's own server for the plugin. */
const HOST_URL_HELPER = /^getAssetUrl\s*\(/

const ABSOLUTE_URL = /^(?:(?:https?|wss?):)?\/\/(\[[^\]]*\]|[^/:?#\s]+)/i

const HOST_NAME = /^(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,}$/i

/** Hosts that name XML namespaces and reserved names, not servers a plugin reaches. */
const IGNORED_HOSTS = new Set(['www.w3.org', 'purl.org', 'www.idpf.org', 'www.daisy.org'])

const IGNORED_SUFFIXES = ['.test', '.example', '.invalid', '.localhost', '.local']

const isReachableHost = (host) =>
  HOST_NAME.test(host) &&
  !IGNORED_HOSTS.has(host) &&
  !IGNORED_SUFFIXES.some((suffix) => host.endsWith(suffix)) &&
  !/^(?:www\.)?example\.(?:com|org|net)$/.test(host)

const isLoopbackHost = (host) => {
  const name = host.toLowerCase().replace(/^\[|\]$/g, '')
  return (
    name === 'localhost' ||
    name.endsWith('.localhost') ||
    /^127(?:\.\d{1,3}){3}$/.test(name) ||
    name === '::1' ||
    name === '0.0.0.0'
  )
}

/**
 * The target of a frontend call whose argument starts at `text`: `local` for a relative path, a
 * URL the host builds or a loopback URL, `external` with its host for an absolute URL of another
 * host, `unknown` when the start of the target is not a literal.
 */
const readFrontendTarget = (text) => {
  const argument = text.trimStart()
  if (HOST_URL_HELPER.test(argument)) return { kind: 'local' }
  const quote = argument[0]
  if (!["'", '"', '`'].includes(quote)) return { kind: 'unknown' }
  const rest = argument.slice(1)
  const end = rest.search(quote === '`' ? /`|\$\{/ : new RegExp(quote))
  if (end === 0 && rest.startsWith('${')) return { kind: 'unknown' }
  const literal = end === -1 ? rest : rest.slice(0, end)
  const host = ABSOLUTE_URL.exec(literal)?.[1]
  if (host === undefined || isLoopbackHost(host)) return { kind: 'local' }
  return { kind: 'external', host: host.toLowerCase() }
}

/** Whether Go code has literal targets and every one of them is a loopback host. */
const isLoopbackOnly = (text) => {
  const hosts = [...text.matchAll(GO_TARGET_LITERAL)].map((match) => match[1] ?? match[2])
  return hosts.length > 0 && hosts.every(isLoopbackHost)
}

const listSourceFiles = (dir) => {
  let entries
  try {
    entries = readdirSync(dir, { withFileTypes: true })
  } catch {
    return []
  }
  return entries.flatMap((entry) => {
    const path = join(dir, entry.name)
    if (entry.isDirectory()) return SKIPPED_FOLDERS.has(entry.name) ? [] : listSourceFiles(path)
    return entry.isFile() && !TEST_FILE.test(entry.name) ? [path] : []
  })
}

const getHost = (url) => {
  try {
    return new URL(url).hostname
  } catch {
    return null
  }
}

const listDownloadUrls = (dependencies = []) =>
  dependencies.flatMap((one) => [
    ...(one.sourceUrls ?? []),
    ...(one.sourceUrl ? [one.sourceUrl] : []),
    ...(one.sources ?? []).flatMap((source) => [
      ...(source.sourceUrls ?? []),
      ...(source.sourceUrl ? [source.sourceUrl] : []),
    ]),
  ])

const toPosix = (path) => path.split(sep).join('/')

const sortUnique = (values) => [...new Set(values.filter(Boolean))].sort()

/**
 * The network use the plugin folder `dir` and its manifest show.
 *
 * A Go client call counts unless every URL or `host:port` literal of its file is a loopback host;
 * a file without such a literal calls an address it is given, so its calls count. A frontend call
 * counts when its target is an absolute URL of a host that is not loopback; a relative path, a
 * `getAssetUrl(…)` of the SDK and a loopback URL do not. A frontend call whose target is not a
 * literal counts when the frontend code names a reachable host in a URL literal.
 *
 * @param {string} dir - the plugin folder
 * @param {Record<string, any>} json - the manifest in the schema's JSON form
 * @returns {{ isNetworked: boolean, calls: { language: 'go' | 'typescript', name: string, files: string[] }[], hosts: string[], frameOrigins: string[], downloadHosts: string[], canOpenLinks: boolean }}
 */
export const readNetworkUse = (dir, json) => {
  const byCall = new Map()
  const hosts = []
  const unknownTargets = []
  let isNamingFrontendHosts = false
  const addCall = (language, name, path) => {
    const key = `${language}:${name}`
    const found = byCall.get(key) ?? { language, name, files: [] }
    if (!found.files.includes(path)) found.files.push(path)
    byCall.set(key, found)
  }
  for (const file of listSourceFiles(dir)) {
    const isGo = file.endsWith('.go')
    const path = toPosix(relative(dir, file))
    const isFrontend = /\.(ts|vue)$/.test(file) && path.startsWith('frontend/')
    if (!isGo && !isFrontend) continue
    const text = readFileSync(file, 'utf8')
    const literalHosts = [...text.matchAll(URL_LITERAL)]
      .map((match) => match[1])
      .filter(isReachableHost)
      .map((host) => host.toLowerCase())
    hosts.push(...literalHosts)
    if (isGo) {
      if (isLoopbackOnly(text)) continue
      for (const call of GO_CALLS) {
        if (call.pattern.test(text)) addCall('go', call.name, path)
      }
      continue
    }
    if (literalHosts.length > 0) isNamingFrontendHosts = true
    for (const call of TS_CALLS) {
      for (const match of text.matchAll(call.pattern)) {
        const target = readFrontendTarget(text.slice(match.index + match[0].length))
        if (target.kind === 'external') {
          addCall('typescript', call.name, path)
          hosts.push(target.host)
        } else if (target.kind === 'unknown') {
          unknownTargets.push({ name: call.name, path })
        }
      }
    }
  }
  if (isNamingFrontendHosts) {
    for (const one of unknownTargets) addCall('typescript', one.name, one.path)
  }
  const calls = [...byCall.values()]
    .map((one) => ({ ...one, files: one.files.sort() }))
    .sort((a, b) => a.language.localeCompare(b.language) || a.name.localeCompare(b.name))
  const frameOrigins = sortUnique(json.contributes?.frameOrigins ?? [])
  const downloadHosts = sortUnique(listDownloadUrls(json.dependencies).map(getHost))
  return {
    isNetworked: calls.length > 0 || frameOrigins.length > 0 || downloadHosts.length > 0,
    calls,
    hosts: sortUnique(hosts),
    frameOrigins,
    downloadHosts,
    canOpenLinks: (json.permissions ?? []).includes('link:open'),
  }
}
