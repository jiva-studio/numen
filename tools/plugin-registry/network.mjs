/**
 * What the source of a plugin shows about its network use, read statically from the syntax trees
 * of its code: the network client calls of its Go code and of its frontend code (tests, stories,
 * fixtures and test data left out), the hosts of the URL literals in that code, the origins its
 * manifest frames, the hosts its downloads come from and the `link:open` permission. Only a call
 * that can reach the internet counts: one on a relative path, on a URL the host builds for the
 * plugin or on a loopback address does not. Calls made inside its dependencies are not traced.
 */
import { readFileSync, readdirSync } from 'node:fs'
import { join, relative, sep } from 'node:path'
import { readGoFiles } from './gosource.mjs'
import { listNetworkCalls, listStringTexts, parseSourceFile } from './tssource.mjs'

const SKIPPED_FOLDERS = new Set(['node_modules', 'dist', 'testdata', 'fixtures', '.git'])

const TEST_SUFFIXES = ['_test.go', '.test.ts', '.stories.ts', '.d.ts']

const FRONTEND_SUFFIXES = ['.ts', '.vue']

const WEB_SCHEMES = ['http:', 'https:']

const SOCKET_SCHEMES = ['ws:', 'wss:']

const HOST_ENDS = new Set(['/', ':', '?', '#', '"', "'", '`'])

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

const isHostEnd = (char) => HOST_ENDS.has(char) || char.trim() === ''

/**
 * The scheme and host of a URL text: `scheme` is `http:`, `https:`, `ws:` or `wss:` in lower case,
 * or empty for a protocol-relative `//host`. Null when the text is no such URL or names no host.
 *
 * @param {string} text
 * @returns {{ scheme: string, host: string } | null}
 */
const readUrlHost = (text) => {
  const lower = text.toLowerCase()
  const scheme = [...WEB_SCHEMES, ...SOCKET_SCHEMES, ''].find((one) => lower.startsWith(`${one}//`))
  if (scheme === undefined) return null
  const rest = text.slice(scheme.length + 2)
  if (rest.startsWith('[')) {
    const close = rest.indexOf(']')
    return close === -1 ? null : { scheme, host: rest.slice(0, close + 1) }
  }
  let end = 0
  while (end < rest.length && !isHostEnd(rest[end])) end += 1
  return end === 0 ? null : { scheme, host: rest.slice(0, end) }
}

/** The reachable hosts of the http and https URLs among `texts`, in lower case. */
const listReachableHosts = (texts) =>
  texts
    .map(readUrlHost)
    .filter((url) => url !== null && WEB_SCHEMES.includes(url.scheme) && isReachableHost(url.host))
    .map((url) => url.host.toLowerCase())

/**
 * The target of a frontend call: `local` for a relative path, a URL the host builds or a loopback
 * URL, `external` with its host for an absolute URL of another host, `unknown` when the start of
 * the target is not a literal.
 */
const readFrontendTarget = (target) => {
  if (target.kind === 'unknown') return target
  if (target.kind === 'helper') return { kind: 'local' }
  const url = readUrlHost(target.text)
  if (url === null || isLoopbackHost(url.host)) return { kind: 'local' }
  return { kind: 'external', host: url.host.toLowerCase() }
}

const getAddressHost = (address) => address.slice(0, address.lastIndexOf(':'))

/** Whether a Go file has literal targets and every one of them is a loopback host. */
const isLoopbackOnly = (facts) => {
  const hosts = [
    ...facts.urls.map(readUrlHost).filter((url) => url !== null && url.scheme !== '').map((url) => url.host),
    ...facts.addresses.map(getAddressHost).filter(isLoopbackHost),
  ]
  return hosts.length > 0 && hosts.every(isLoopbackHost)
}

const isTestFile = (name) => TEST_SUFFIXES.some((suffix) => name.endsWith(suffix))

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
    return entry.isFile() && !isTestFile(entry.name) ? [path] : []
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
  const toPath = (file) => toPosix(relative(dir, file))
  const files = listSourceFiles(dir)
  for (const [file, facts] of readGoFiles(files.filter((one) => one.endsWith('.go')))) {
    hosts.push(...listReachableHosts(facts.urls))
    if (isLoopbackOnly(facts)) continue
    for (const call of facts.calls) addCall('go', call.name, toPath(file))
  }
  for (const file of files) {
    const path = toPath(file)
    if (!path.startsWith('frontend/') || !FRONTEND_SUFFIXES.some((suffix) => file.endsWith(suffix))) continue
    for (const sourceFile of parseSourceFile(file, readFileSync(file, 'utf8'))) {
      const literalHosts = listReachableHosts(listStringTexts(sourceFile))
      hosts.push(...literalHosts)
      if (literalHosts.length > 0) isNamingFrontendHosts = true
      for (const call of listNetworkCalls(sourceFile)) {
        const target = readFrontendTarget(call.target)
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
