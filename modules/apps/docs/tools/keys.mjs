/**
 * The keyboard page's table, written from the window's own table of chords.
 *
 * A key a person reads here is a key the application answers, because both are
 * read out of `keying.ts` and the words the commands are drawn with. Run with
 * `--check` it writes nothing and fails where the page has drifted, which is
 * what the build asks of it.
 */
import { readFile, writeFile } from 'node:fs/promises'

const UI = new URL('../../desktop/ui/src/', import.meta.url)
const PAGE = new URL('../src/content/docs/keyboard.md', import.meta.url)
const BEGIN = '<!-- BEGIN AUTOGEN -->'
const END = '<!-- END AUTOGEN -->'

const source = (path) => readFile(new URL(path, UI), 'utf8')

const die = (message) => {
  process.stderr.write(`keys: ${message}\n`)
  process.exit(1)
}

/** Every chord the window carries out a command for, in the order it declares them. */
const chords = (keying) => {
  const found = [
    ...keying.matchAll(
      /\{\s*command:\s*'([a-z]+)',\s*letter:\s*'([a-z])',\s*shift:\s*(true|false)\s*\}/g,
    ),
  ].map(([, command, letter, shift]) => ({ command, letter, shift: shift === 'true' }))
  if (found.length === 0) die('no chords found in keying.ts')
  return found
}

/** Which entry of the words a command is drawn with. */
const spoken = (commanding, command) => {
  const row = new RegExp(`id:\\s*'${command}',[\\s\\S]{0,120}?text:\\s*words\\.([A-Za-z]+)`)
  const found = commanding.match(row)
  if (!found) die(`no row in commanding.ts draws the command '${command}'`)
  return found[1]
}

/**
 * What one entry of the words says. An entry standing for another module's is
 * followed there, which is where the four tab kinds keep their own names.
 */
const said = async (words, name, seen = new Set()) => {
  const literal = words.match(new RegExp(`^\\s*${name}:\\s*'([^']+)',`, 'm'))
  if (literal) return literal[1]

  const elsewhere = words.match(new RegExp(`^\\s*${name}:\\s*([a-z]+)\\.([A-Za-z]+),`, 'm'))
  if (!elsewhere) die(`the words say nothing under '${name}'`)
  const [, module, key] = elsewhere
  if (seen.has(module)) die(`the words under '${name}' point at themselves`)
  return said(await source(`${module}/words.ts`), key, new Set([...seen, module]))
}

/** One row of the table: the chord as it is drawn, and what it does. */
const row = ({ letter, shift }, text) => {
  const keys = ['<kbd>Ctrl</kbd>/<kbd>⌘</kbd>']
  if (shift) keys.push('<kbd>⇧</kbd>')
  keys.push(`<kbd>${letter.toUpperCase()}</kbd>`)
  return `| ${keys.join(' ')} | ${text} |`
}

const table = async () => {
  const keying = await source('keying.ts')
  const commanding = await source('commanding.ts')
  const words = await source('words.ts')

  // The two the window keeps for itself are not in that table: they put a
  // panel up rather than carry a command out.
  const app = await source('App.vue')
  for (const letter of ['k', 'p']) {
    if (!app.includes(`key === '${letter}'`)) {
      die(`the window no longer answers '${letter}' itself`)
    }
  }

  const lines = ['| | |', '| --- | --- |']
  lines.push(row({ letter: 'k', shift: false }, await said(words, 'find')))
  lines.push(row({ letter: 'p', shift: false }, 'Commands'))
  for (const chord of chords(keying)) {
    lines.push(row(chord, await said(words, spoken(commanding, chord.command))))
  }
  return lines.join('\n')
}

const page = await readFile(PAGE, 'utf8')
const begins = page.indexOf(BEGIN)
const ends = page.indexOf(END)
if (begins < 0 || ends < begins) die('the page has no autogen block')

const written = `${page.slice(0, begins + BEGIN.length)}\n${await table()}\n${page.slice(ends)}`

if (process.argv.includes('--check')) {
  if (written !== page) die('the keyboard page is not what the window says. Run `npm run keys`.')
  process.stdout.write('keys: the keyboard page says what the window does\n')
} else if (written !== page) {
  await writeFile(PAGE, written)
  process.stdout.write('keys: the keyboard page was written again\n')
}
