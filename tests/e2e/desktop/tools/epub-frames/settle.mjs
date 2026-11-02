// Reads a screen recording of the reading area made while single pages were turned, and reports every
// frame that shows another picture than the page the turn arrived at after the frames have arrived
// at it. A turn that ends in a stray frame, where the text is drawn for one frame as it was drawn
// earlier or in two pieces, is seen here and in no state of the page.
//
// Usage: node settle.mjs RECORDING.mkv PRESSES.txt START_MS [TURNS]
//   PRESSES.txt has one `epoch-ms key` line per key press, START_MS is the epoch of the recording's
//   start, and the first TURNS presses (6) are the turns one second apart that are looked at.
// Exit code 1 when a stray frame was found.
import { execFileSync, spawn } from 'node:child_process'
import fs from 'node:fs'

const [recording, pressesFile, startText, turnsText = '6'] = process.argv.slice(2)
if (!recording || !pressesFile || !startText) {
  console.error('usage: settle.mjs RECORDING PRESSES START_MS [TURNS]')
  process.exit(2)
}

const FPS = 60
/** How many frames after the first moving one the page is taken to stand still. */
const SETTLED_AFTER = 45
/** The correlation with the settled picture above which a frame has arrived, and below which it is another picture. */
const ARRIVED = 0.8
const STRAY = 0.5
/** Every nth pixel is looked at. */
const PIXEL_STEP = 7

const [width, height] = execFileSync('ffprobe', [
  '-v', 'error', '-show_entries', 'stream=width,height', '-of', 'csv=p=0', recording,
]).toString().trim().split(',').map(Number)
const frameSize = width * height

const readFrames = () => new Promise((resolve) => {
  const ffmpeg = spawn('ffmpeg', ['-v', 'error', '-i', recording, '-pix_fmt', 'gray', '-f', 'rawvideo', '-'])
  const frames = []
  let rest = Buffer.alloc(0)
  ffmpeg.stdout.on('data', (chunk) => {
    rest = Buffer.concat([rest, chunk])
    while (rest.length >= frameSize) {
      frames.push(Buffer.from(rest.subarray(0, frameSize)))
      rest = rest.subarray(frameSize)
    }
  })
  ffmpeg.on('close', () => resolve(frames))
})

/** How alike two pictures are, from -1 to 1. */
const getCorrelation = (a, b) => {
  let count = 0
  let meanA = 0
  let meanB = 0
  for (let index = 0; index < frameSize; index += PIXEL_STEP) {
    meanA += a[index]
    meanB += b[index]
    count += 1
  }
  meanA /= count
  meanB /= count
  let both = 0
  let onlyA = 0
  let onlyB = 0
  for (let index = 0; index < frameSize; index += PIXEL_STEP) {
    const x = a[index] - meanA
    const y = b[index] - meanB
    both += x * y
    onlyA += x * x
    onlyB += y * y
  }
  return both / Math.sqrt(onlyA * onlyB + 1e-9)
}

const start = Number(startText)
const presses = fs.readFileSync(pressesFile, 'utf8').trim().split('\n').map((line) => (Number(line.split(' ')[0]) - start) / 1000)
const frames = await readFrames()

let strays = 0
presses.slice(0, Number(turnsText)).forEach((press, turn) => {
  const first = Math.floor(press * FPS)
  let moving = first
  while (moving < first + 20 && Buffer.compare(frames[moving], frames[moving + 1] ?? frames[moving]) === 0) moving += 1
  const settled = frames[moving + SETTLED_AFTER]
  if (!settled) return
  const similarities = []
  for (let index = moving; index < moving + SETTLED_AFTER; index += 1) similarities.push(getCorrelation(frames[index], settled))
  const arrived = similarities.findIndex((one) => one > ARRIVED)
  const found = similarities.flatMap((one, index) => (index > arrived && one < STRAY ? [`+${index} (${one.toFixed(2)})`] : []))
  strays += found.length
  console.log(`turn ${turn}: arrived at frame +${arrived}, stray frames after it: ${found.join(' ') || 'none'}`)
})
console.log(`stray frames after a turn arrived: ${strays}`)
process.exit(strays === 0 ? 0 : 1)
