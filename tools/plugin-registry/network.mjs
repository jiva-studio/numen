/**
 * What the source of a plugin shows about its network use, read statically: the network client
 * calls of its Go code and of its frontend code (tests, stories, fixtures and test data left out),
 * the hosts of the URL literals in that code, the origins its manifest frames, the hosts its
 * downloads come from and the `link:open` permission. Calls made inside its dependencies are not
 * traced.
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
  { name: 'fetch', pattern: /(?<![\w.$])fetch\(/ },
  { name: 'XMLHttpRequest', pattern: /\bnew XMLHttpRequest\b/ },
  { name: 'WebSocket', pattern: /\bnew WebSocket\(/ },
  { name: 'EventSource', pattern: /\bnew EventSource\(/ },
]

const URL_LITERAL = /["'`]https?:\/\/([^"'`/:?#\s]+)/g

const HOST_NAME = /^(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,}$/i

/** Hosts that name XML namespaces and reserved names, not servers a plugin reaches. */
const IGNORED_HOSTS = new Set(['www.w3.org', 'purl.org', 'www.idpf.org', 'www.daisy.org'])

const IGNORED_SUFFIXES = ['.test', '.example', '.invalid', '.localhost', '.local']

const isReachableHost = (host) =>
  HOST_NAME.test(host) &&
  !IGNORED_HOSTS.has(host) &&
  !IGNORED_SUFFIXES.some((suffix) => host.endsWith(suffix)) &&
  !/^(?:www\.)?example\.(?:com|org|net)$/.test(host)

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
 * @param {string} dir - the plugin folder
 * @param {Record<string, any>} json - the manifest in the schema's JSON form
 * @returns {{ isNetworked: boolean, calls: { language: 'go' | 'typescript', name: string, files: string[] }[], hosts: string[], frameOrigins: string[], downloadHosts: string[], canOpenLinks: boolean }}
 */
export const readNetworkUse = (dir, json) => {
  const byCall = new Map()
  const hosts = []
  for (const file of listSourceFiles(dir)) {
    const isGo = file.endsWith('.go')
    const path = toPosix(relative(dir, file))
    const isFrontend = /\.(ts|vue)$/.test(file) && path.startsWith('frontend/')
    if (!isGo && !isFrontend) continue
    const text = readFileSync(file, 'utf8')
    const language = isGo ? 'go' : 'typescript'
    for (const call of isGo ? GO_CALLS : TS_CALLS) {
      if (!call.pattern.test(text)) continue
      const key = `${language}:${call.name}`
      const found = byCall.get(key) ?? { language, name: call.name, files: [] }
      found.files.push(path)
      byCall.set(key, found)
    }
    for (const match of text.matchAll(URL_LITERAL)) {
      if (isReachableHost(match[1])) hosts.push(match[1].toLowerCase())
    }
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
