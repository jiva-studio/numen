/**
 * That every picture this page shows is still a story somebody can take again.
 *
 * The camera names its stories as strings, and a string is not a reference:
 * renaming a story leaves the name here pointing at nothing, and nothing says
 * so until somebody runs the camera — which for this page happens at a
 * release, months later, and shows up as a picture that never arrived. So the
 * names are held against the tree here, by a check that runs with the rest.
 *
 * What a story is called and what is wrong with a shot are `@numen/stories`,
 * which the manual holds its own pictures to as well.
 */
import { fileURLToPath } from 'node:url'

import { faults, proves, stories } from '@numen/stories'

import { SHOTS } from './shots.mjs'

const die = (message) => {
  process.stderr.write(`stories: ${message}\n`)
  process.exit(1)
}

export const checked = async () => {
  const unproved = proves()
  if (unproved) die(unproved)

  const wrong = faults(SHOTS, await stories())
  if (wrong.length > 0) die(wrong.join('; '))
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  await checked()
  process.stdout.write(`stories: every picture on the page is still a story\n`)
}
