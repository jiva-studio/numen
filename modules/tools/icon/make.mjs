/**
 * Every icon the product ships, cut from one drawing.
 *
 * `icon.svg` is the drawing: a rounded plate and the letter as an outline, so
 * nothing here needs the face it was set in. What comes out of it is what each
 * platform asks for, and the formats are written by hand because they are three
 * headers and a list of PNGs between them.
 */
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import sharp from 'sharp'

const HERE = dirname(fileURLToPath(import.meta.url))
const ROOT = join(HERE, '..', '..', '..')
const MASTER = join(HERE, 'icon.svg')

const LANDING = join(ROOT, 'modules', 'apps', 'landing', 'public')
const BUILD = join(ROOT, 'modules', 'apps', 'desktop', 'build')

/**
 * How much of a mac icon the plate itself is. The rest is clearance the system
 * expects around it, and a plate drawn to the edge sits larger than everything
 * beside it in the dock.
 */
const MAC_PLATE = 0.82

const drawing = readFileSync(MASTER)

/** The plate at a size, square, with nothing around it. */
const flat = (size) => sharp(drawing, { density: 600 }).resize(size, size).png()

/** The plate inside the clearance a mac icon is drawn with. */
const macos = async (size) => {
  const plate = Math.round(size * MAC_PLATE)
  const inset = Math.round((size - plate) / 2)
  return sharp({
    create: {
      width: size,
      height: size,
      channels: 4,
      background: { r: 0, g: 0, b: 0, alpha: 0 },
    },
  })
    .composite([{ input: await flat(plate).toBuffer(), left: inset, top: inset }])
    .png()
    .toBuffer()
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

  images.forEach(({ size, data }, index) => {
    const entry = index * 16
    directory.writeUInt8(size >= 256 ? 0 : size, entry)
    directory.writeUInt8(size >= 256 ? 0 : size, entry + 1)
    directory.writeUInt8(0, entry + 2)
    directory.writeUInt8(0, entry + 3)
    directory.writeUInt16LE(1, entry + 4)
    directory.writeUInt16LE(32, entry + 6)
    directory.writeUInt32LE(data.length, entry + 8)
    directory.writeUInt32LE(at, entry + 12)
    at += data.length
  })

  return Buffer.concat([header, directory, ...images.map((image) => image.data)])
}

/** A mac icon: a header, then one typed block per size. */
const icns = (blocks) => {
  const body = blocks.map(({ type, data }) => {
    const head = Buffer.alloc(8)
    head.write(type, 0, 4, 'ascii')
    head.writeUInt32BE(data.length + 8, 4)
    return Buffer.concat([head, data])
  })

  const length = body.reduce((sum, block) => sum + block.length, 8)
  const head = Buffer.alloc(8)
  head.write('icns', 0, 4, 'ascii')
  head.writeUInt32BE(length, 4)
  return Buffer.concat([head, ...body])
}

/** What each mac block is called, by the side it holds. */
const MAC_BLOCKS = [
  ['ic07', 128],
  ['ic08', 256],
  ['ic09', 512],
  ['ic10', 1024],
  ['ic11', 32],
  ['ic12', 64],
  ['ic13', 512],
  ['ic14', 1024],
]

/** The sides a Linux desktop keeps an icon at. */
const LINUX = [16, 24, 32, 48, 64, 128, 256, 512]

const write = (path, data) => {
  mkdirSync(dirname(path), { recursive: true })
  writeFileSync(path, data)
  console.log(path.replace(`${ROOT}/`, ''))
}

// The page on the web: the drawing itself, and the one raster iOS asks for.
write(join(LANDING, 'favicon.svg'), drawing)
write(join(LANDING, 'apple-touch-icon.png'), await flat(180).toBuffer())

// Linux: the drawing, and a raster at every size a desktop looks for.
write(join(BUILD, 'linux', 'numen.svg'), drawing)
for (const size of LINUX) {
  write(join(BUILD, 'linux', 'icons', `${size}.png`), await flat(size).toBuffer())
}

// Windows: one file holding every size.
write(
  join(BUILD, 'windows', 'numen.ico'),
  ico(
    await Promise.all(
      [16, 24, 32, 48, 64, 128, 256].map(async (size) => ({
        size,
        data: await flat(size).toBuffer(),
      })),
    ),
  ),
)

// macOS: one file holding every size, each drawn with its clearance.
write(
  join(BUILD, 'darwin', 'numen.icns'),
  icns(
    await Promise.all(
      MAC_BLOCKS.map(async ([type, size]) => ({ type, data: await macos(size) })),
    ),
  ),
)
