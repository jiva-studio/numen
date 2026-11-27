/**
 * The screenshots of a plugin as the pages show them: those of this repository optimized by the
 * build from `plugins/<folder>/screenshots/`, those of an external plugin from the downloads zone.
 */
import { getImage } from 'astro:assets'
import type { ImageMetadata } from 'astro'
import { type CatalogPlugin, getPublishedUrl } from './catalog'

/** One appearance of a screenshot; an external one has no known size. */
type ShotImage = { src: string; width?: number; height?: number }

/** A screenshot in light and in dark; either may be missing. */
export type Shot = { name: string; light: ShotImage | null; dark: ShotImage | null }

const files = import.meta.glob<{ default: ImageMetadata }>(
  '../../../../../plugins/*/screenshots/*.webp',
  { eager: true },
)
const repositoryImages = new Map(
  Object.entries(files).map(([key, module]) => [
    key.slice(key.indexOf('plugins/')),
    module.default,
  ]),
)

const SHOWN = { format: 'webp' as const, quality: 84 }

const getShotImage = async (
  plugin: CatalogPlugin,
  path: string | null,
): Promise<ShotImage | null> => {
  if (path === null) return null
  if (plugin.source.kind === 'external') return { src: getPublishedUrl(path) }
  const image = repositoryImages.get(path)
  if (!image) return null
  const { src } = await getImage({ src: image, ...SHOWN })
  return { src, width: image.width / 2, height: image.height / 2 }
}

/** The first `count` screenshots of a plugin that have an image, in the registry's order. */
export async function listShots(plugin: CatalogPlugin, count: number): Promise<Shot[]> {
  const shots = await Promise.all(
    plugin.screenshots.map(async (one) => ({
      name: one.name,
      light: await getShotImage(plugin, one.light),
      dark: await getShotImage(plugin, one.dark),
    })),
  )
  return shots.filter((one) => one.light !== null || one.dark !== null).slice(0, count)
}

/** A screenshot of this repository by its path from the repository root; an unknown path fails the build. */
export async function getRepositoryShot(path: string): Promise<ShotImage> {
  const image = repositoryImages.get(path)
  if (!image) throw new Error(`no screenshot ${path}`)
  const { src } = await getImage({ src: image, ...SHOWN })
  return { src, width: image.width / 2, height: image.height / 2 }
}
