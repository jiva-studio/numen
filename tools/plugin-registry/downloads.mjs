/**
 * What a plugin downloads when it is installed: the `dependencies` of its manifest, each with the
 * size of the file for every platform.
 */

const ANY_PLATFORM = 'any'

/** The sizes the previous registry recorded, by SHA-256 of the file. */
export const listKnownSizes = (previous) => {
  const sizes = new Map()
  for (const plugin of previous?.plugins ?? []) {
    for (const download of plugin.downloads ?? []) {
      for (const one of download.platforms) {
        if (one.sha256 && one.size !== null) sizes.set(one.sha256, one.size)
      }
    }
  }
  return sizes
}

const readLength = (response) => {
  const length = Number(response.headers.get('content-length'))
  return Number.isFinite(length) && length > 0 ? length : null
}

const readRangeTotal = (response) => {
  const total = Number(/\/(\d+)\s*$/.exec(response.headers.get('content-range') ?? '')?.[1])
  return Number.isFinite(total) && total > 0 ? total : null
}

const readRangedSize = (response) =>
  response.status === 206 ? readRangeTotal(response) : readLength(response)

const requestSize = async (url, init, readSize, fetcher) => {
  try {
    const response = await fetcher(url, { ...init, redirect: 'follow' })
    const size = response.ok ? readSize(response) : null
    await response.body?.cancel().catch(() => undefined)
    return size
  } catch {
    return null
  }
}

/**
 * The size of the file at `url`, or null: the Content-Length of a HEAD request, else the total of
 * the Content-Range a GET of its first byte is answered with. A server that ignores the range
 * answers with the whole file, whose Content-Length is the size.
 */
export const fetchSize = async (url, fetcher = fetch) =>
  (await requestSize(url, { method: 'HEAD' }, readLength, fetcher)) ??
  (await requestSize(
    url,
    { method: 'GET', headers: { Range: 'bytes=0-0' } },
    readRangedSize,
    fetcher,
  ))

/** The size of the first of `urls` that answers with one, or null. */
export const fetchFirstSize = async (urls, getSize) => {
  for (const url of urls) {
    const size = await getSize(url)
    if (size !== null) return size
  }
  return null
}

const listUrls = (one) => {
  if ((one.sourceUrls ?? []).length > 0) return one.sourceUrls
  return one.sourceUrl ? [one.sourceUrl] : []
}

const listSources = (dependency) => {
  if ((dependency.sources ?? []).length > 0) {
    return dependency.sources.map((one) => ({
      platform: one.platform,
      urls: listUrls(one),
      sha256: one.checksumSha256 ?? '',
    }))
  }
  return [
    {
      platform: ANY_PLATFORM,
      urls: listUrls(dependency),
      sha256: dependency.checksumSha256 ?? '',
    },
  ]
}

/**
 * The downloads of a manifest. A file whose SHA-256 the previous registry has a size for is not
 * asked again; another is asked at each of its URLs in turn, mirrors included, until one answers.
 *
 * @param {object[]} dependencies - `dependencies` of the manifest in its JSON form
 * @param {Map<string, number>} knownSizes - sizes by SHA-256
 * @param {(url: string) => Promise<number | null>} getSize - the size of a file at a URL
 * @returns {Promise<{ id: string, kind: string, version: string, platforms: { platform: string, size: number | null, sha256: string }[] }[]>}
 */
export const describeDownloads = async (dependencies, knownSizes, getSize) => {
  const downloads = []
  for (const dependency of dependencies ?? []) {
    const platforms = []
    for (const source of listSources(dependency)) {
      const known = source.sha256 ? knownSizes.get(source.sha256) : undefined
      const size = known ?? (await fetchFirstSize(source.urls, getSize))
      platforms.push({ platform: source.platform, size, sha256: source.sha256 })
    }
    downloads.push({
      id: dependency.id,
      kind: dependency.kind ?? '',
      version: dependency.version ?? '',
      platforms,
    })
  }
  return downloads
}
