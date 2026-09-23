/**
 * Every icon the product ships, cut from the drawings.
 *
 * A drawing is one letter as an outline on a rounded plate, so nothing here
 * needs the face it was set in. `icon.svg` carries the product's n and
 * `flashcards.svg` a c, and the two are held to differing in the letter and the
 * plate's colour alone. What comes out is what each platform asks for, and the
 * formats are written by hand because they are three headers and a list of PNGs
 * between them.
 */
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import sharp from 'sharp'

const HERE = dirname(fileURLToPath(import.meta.url))
const ROOT = join(HERE, '..', '..', '..')
const MASTER = join(HERE, 'icon.svg')
const FLASHCARDS = join(HERE, 'flashcards.svg')

const LANDING = join(ROOT, 'modules', 'apps', 'landing', 'public')
const BUILD = join(ROOT, 'modules', 'apps', 'desktop', 'build')

/**
 * How much of a mac icon the plate itself is. The rest is clearance the system
 * expects around it, and a plate drawn to the edge sits larger than everything
 * beside it in the dock.
 */
const MAC_PLATE = 0.82

const drawing = readFileSync(MASTER)
const flashcards = readFileSync(FLASHCARDS)

/** A drawing with the plate's colour and the letter taken out of it. */
const shape = (art) =>
  art
    .toString('utf8')
    .replace(/(<rect[^>]*fill=")#[0-9a-f]{6}(")/, '$1$2')
    .replace(/ transform="[^"]*"/, '')
    .replace(/ d="[^"]*"/, '')

if (shape(drawing) !== shape(flashcards)) {
  throw new Error('the drawings differ in more than the plate and the letter')
}

/**
 * A PNG carrying no physical resolution. An icon is measured in points, and a
 * `pHYs` chunk makes the system read a side of 1024 pixels as 2903 points.
 */
const bare = (png) => {
  const keep = [png.subarray(0, 8)]
  let at = 8
  while (at + 8 <= png.length) {
    const length = png.readUInt32BE(at)
    const type = png.toString('ascii', at + 4, at + 8)
    if (type !== 'pHYs') keep.push(png.subarray(at, at + 12 + length))
    at += 12 + length
  }
  return Buffer.concat(keep)
}

/** A drawing at a size, square, with nothing around it. */
const flat = async (art, size) =>
  bare(await sharp(art, { density: 600 }).resize(size, size).png().toBuffer())

/** A drawing inside the clearance a mac icon is drawn with. */
const macos = async (art, size) => {
  const plate = Math.round(size * MAC_PLATE)
  const inset = Math.round((size - plate) / 2)
  return bare(
    await sharp({
      create: {
        width: size,
        height: size,
        channels: 4,
        background: { r: 0, g: 0, b: 0, alpha: 0 },
      },
    })
      .composite([{ input: await flat(art, plate), left: inset, top: inset }])
      .png()
      .toBuffer(),
  )
}

/**
 * A Windows icon: a header, one entry per size, and the PNGs after them.
 *
 * A side of 256 is written as zero, which is what the format says a side of
 * 256 is.
 */
const ico = (images) => {
  const header = Buffer.alloc(6)
  header.writeUInt16LE(0, 0)
  header.writeUInt16LE(1, 2)
  header.writeUInt16LE(images.length, 4)

  const directory = Buffer.alloc(images.length * 16)
  let at = 6 + directory.length

  images.forEach(({ size, png }, index) => {
    const entry = index * 16
    directory.writeUInt8(size >= 256 ? 0 : size, entry)
    directory.writeUInt8(size >= 256 ? 0 : size, entry + 1)
    directory.writeUInt8(0, entry + 2)
    directory.writeUInt8(0, entry + 3)
    directory.writeUInt16LE(1, entry + 4)
    directory.writeUInt16LE(32, entry + 6)
    directory.writeUInt32LE(png.length, entry + 8)
    directory.writeUInt32LE(at, entry + 12)
    at += png.length
  })

  return Buffer.concat([header, directory, ...images.map((image) => image.png)])
}

/** A mac icon: a header, then one typed block per size. */
const icns = (blocks) => {
  const body = blocks.map(({ type, png }) => {
    const head = Buffer.alloc(8)
    head.write(type, 0, 4, 'ascii')
    head.writeUInt32BE(png.length + 8, 4)
    return Buffer.concat([head, png])
  })

  const length = body.reduce((sum, block) => sum + block.length, 8)
  const head = Buffer.alloc(8)
  head.write('icns', 0, 4, 'ascii')
  head.writeUInt32BE(length, 4)
  return Buffer.concat([head, ...body])
}

/**
 * What each mac block is called, by the side it holds. A block names a size in
 * points, and the retina half of a pair holds twice that in pixels.
 */
const MAC_BLOCKS = [
  ['icp4', 16],
  ['icp5', 32],
  ['ic07', 128],
  ['ic08', 256],
  ['ic09', 512],
  ['ic10', 1024],
  ['ic11', 32],
  ['ic12', 64],
  ['ic13', 256],
  ['ic14', 512],
]

/** The sides a Linux desktop keeps an icon at. */
const LINUX = [16, 24, 32, 48, 64, 128, 256, 512]

/** The sides a Windows icon holds. */
const WINDOWS = [16, 24, 32, 48, 64, 128, 256]

const write = (path, bytes) => {
  mkdirSync(dirname(path), { recursive: true })
  writeFileSync(path, bytes)
  console.log(path.replace(`${ROOT}/`, ''))
}

/** Every desktop icon one drawing is cut into, under the name it ships as. */
const cut = async (art, name, rasters) => {
  // Linux: the drawing, and a raster at every size a desktop looks for.
  write(join(BUILD, 'linux', `${name}.svg`), art)
  for (const size of LINUX) {
    write(join(rasters, `${size}.png`), await flat(art, size))
  }

  // Windows: one file holding every size.
  write(
    join(BUILD, 'windows', `${name}.ico`),
    ico(
      await Promise.all(
        WINDOWS.map(async (size) => ({ size, png: await flat(art, size) })),
      ),
    ),
  )

  // macOS: one file holding every size, each drawn with its clearance.
  write(
    join(BUILD, 'darwin', `${name}.icns`),
    icns(
      await Promise.all(
        MAC_BLOCKS.map(async ([type, size]) => ({ type, png: await macos(art, size) })),
      ),
    ),
  )
}

// The page on the web: the drawing itself, and the one raster iOS asks for.
write(join(LANDING, 'favicon.svg'), drawing)
write(join(LANDING, 'apple-touch-icon.png'), await flat(drawing, 180))

await cut(drawing, 'numen', join(BUILD, 'linux', 'icons'))
await cut(flashcards, 'numen-flashcards', join(BUILD, 'linux', 'icons', 'flashcards'))
