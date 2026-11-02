// Reads a screen recording of the reading area made while pages were turned, and reports every time
// a page that had been left was drawn again against the direction of the turns.
//
// Usage: node analyze.mjs RECORDING.mkv PRESSES.txt START_MS COUNTS.txt
//   PRESSES.txt has one `epoch-ms key` line per key press, START_MS is the epoch of the recording's
//   start, COUNTS.txt has one `epoch-ms number` line per change of the page counter of the reader.
// Exit code 1 when a page came back.
import { execFileSync, spawn } from 'node:child_process'
import fs from 'node:fs'

const [recording, pressesFile, startText, countsFile] = process.argv.slice(2)
if (!recording || !pressesFile || !startText || !countsFile) {
  console.error('usage: analyze.mjs RECORDING PRESSES START_MS COUNTS')
  process.exit(2)
}

const FPS = 60
/** A page that stands for at least this many frames is a page the reader settled on. */
const SETTLED_FRAMES = 20
/** How many pixels of a frame may differ from a page before it is another picture. */
const SAME_PIXELS = 12
const PIXEL_STEP = 5
const PIXEL_DELTA = 40

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

const countDifferent = (a, b) => {
  let count = 0
  for (let index = 0; index < frameSize; index += PIXEL_STEP) {
    if (Math.abs(a[index] - b[index]) > PIXEL_DELTA) count += 1
  }
  return count
}

/** Gives each settled picture a number in the order the pictures settled, and every frame its picture or -1. */
const labelFrames = (frames) => {
  const pictures = []
  const labels = new Array(frames.length).fill(-1)
  let from = 0
  while (from < frames.length) {
    let to = from + 1
    while (to < frames.length && countDifferent(frames[to], frames[from]) < SAME_PIXELS) to += 1
    if (to - from >= SETTLED_FRAMES) {
      let label = pictures.findIndex((picture) => countDifferent(picture, frames[from]) < SAME_PIXELS)
      if (label < 0) {
        pictures.push(frames[from])
        label = pictures.length - 1
      }
      labels.fill(label, from, to)
    }
    from = to
  }
  for (let index = 0; index < frames.length; index += 1) {
    if (labels[index] < 0) {
      labels[index] = pictures.findIndex((picture) => countDifferent(picture, frames[index]) < SAME_PIXELS)
    }
  }
  return { labels, pictureCount: pictures.length }
}

const getRuns = (labels) => {
  const runs = []
  labels.forEach((label, index) => {
    if (label < 0) return
    const last = runs[runs.length - 1]
    if (last && last.label === label && index - last.end <= 3) {
      last.frames += 1
      last.end = index
    } else {
      runs.push({ label, at: index / FPS, end: index, frames: 1 })
    }
  })
  return runs
}

const start = Number(startText)
const presses = fs.readFileSync(pressesFile, 'utf8').trim().split('\n').map((line) => {
  const [time, key] = line.split(' ')
  return { at: (Number(time) - start) / 1000, key }
})

const frames = await readFrames()
const { labels, pictureCount } = labelFrames(frames)
const runs = getRuns(labels)

// The recording lags the clock a little: the first change of the picture marks the first press.
const lag = (runs[1]?.at ?? presses[0].at) - presses[0].at
const counts = fs.readFileSync(countsFile, 'utf8').trim().split('\n').map((line) => {
  const [time, number] = line.split(' ')
  return { at: (Number(time) - start) / 1000 + lag, number: Number(number) }
})
const getNumberAt = (seconds) => counts.filter((count) => count.at <= seconds).at(-1)?.number

// The page number of a picture is what the counter of the reader said while the picture stood longest.
const longest = new Map()
for (const run of runs) if ((longest.get(run.label)?.frames ?? 0) < run.frames) longest.set(run.label, run)
const numbers = new Map([...longest].map(([label, run]) => [label, getNumberAt(run.at + (run.frames - 3) / FPS)]))

const firstBack = presses.find((press) => press.key === 'ArrowLeft' || press.key === 'PageUp')
const turnsBackFrom = firstBack ? firstBack.at + lag - 0.25 : Infinity

// Forward, the page numbers only rise, and back they only fall: a picture of a page left behind is
// a page that came back.
const returns = []
let highest = -Infinity
let lowest = Infinity
for (const run of runs) {
  const number = numbers.get(run.label)
  if (number === undefined) continue
  if (run.at < turnsBackFrom) {
    if (number < highest) returns.push({ at: run.at, direction: 'forward', page: number, after: highest, frames: run.frames })
    highest = Math.max(highest, number)
    lowest = number
  } else {
    if (number > lowest) returns.push({ at: run.at, direction: 'back', page: number, after: lowest, frames: run.frames })
    lowest = Math.min(lowest, number)
  }
}

console.log(`frames ${frames.length} (${(frames.length / FPS).toFixed(1)} s), settled pages ${pictureCount}, runs ${runs.length}, presses ${presses.length}`)
console.log(`runs: ${runs.map((run) => `${numbers.get(run.label) ?? '?'}@${run.at.toFixed(2)}x${run.frames}`).join(' ')}`)
console.log(`pages drawn again against the direction of the turns: ${returns.length}`)
for (const one of returns) console.log(`  ${JSON.stringify(one)}`)
process.exit(returns.length === 0 ? 0 : 1)
