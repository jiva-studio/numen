// Prints the step definitions of the acceptance suite by file: the list to read before a new step is written.
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const stepsDir = path.join(root, 'steps')

for (const file of fs.readdirSync(stepsDir).filter((name) => name.endsWith('.ts')).sort()) {
  console.log(`\n${file}`)
  const source = fs.readFileSync(path.join(stepsDir, file), 'utf8')
  for (const hit of source.matchAll(/\b(Given|When|Then)\(\s*(['"`])((?:\\.|(?!\2).)*)\2/g)) {
    console.log(`  ${hit[1].padEnd(5)} ${hit[3]}`)
  }
}
