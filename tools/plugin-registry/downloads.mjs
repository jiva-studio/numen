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

/** The size a server reports for `url` in Content-Length, or null. */
export const fetchSize = async (url, fetcher = fetch) => {
  try {
    const response = await fetcher(url, { method: 'HEAD', redirect: 'follow' })
    const length = Number(response.headers.get('content-length'))
    return response.ok && Number.isFinite(length) && length > 0 ? length : null
  } catch {
    return null
  }
}

const listSources = (dependency) => {
  if ((dependency.sources ?? []).length > 0) {
    return dependency.sources.map((one) => ({
      platform: one.platform,
      url: one.sourceUrls?.[0] ?? one.sourceUrl ?? null,
      sha256: one.checksumSha256 ?? '',
    }))
  }
  return [
    {
      platform: ANY_PLATFORM,
      url: dependency.sourceUrls?.[0] ?? dependency.sourceUrl ?? null,
      sha256: dependency.checksumSha256 ?? '',
    },
  ]
}

/**
 * The downloads of a manifest. A file whose SHA-256 the previous registry has a size for is not
 * asked again.
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
      const size = known ?? (source.url ? await getSize(source.url) : null)
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
