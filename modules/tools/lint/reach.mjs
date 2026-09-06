/** The corpus the keyboard walk runs over: the interface library's stories. */
import { readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'
import { root } from './source.mjs'

const AT = 'modules/libs/ui/src'

function walk(at, found) {
  for (const name of readdirSync(at)) {
    if (name === 'node_modules') continue
    const path = join(at, name)
    if (statSync(path).isDirectory()) walk(path, found)
    else if (path.endsWith('.stories.ts')) found.push(path)
  }
  return found
}

/** Every story file of the interface library, by absolute path. */
export const stories = () => walk(join(root, AT), [])
