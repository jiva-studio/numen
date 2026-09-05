/** Where the cruise finds its rules and its binary. The modules are the tools'. */
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

export { modules, root } from '../modules.mjs'

export const here = dirname(fileURLToPath(import.meta.url))
export const config = join(here, 'rules.cjs')
export const depcruise = join(here, 'node_modules/.bin/depcruise')

/** One box per folder of ours and one per package, which is the level a drawing reads at. */
export const COLLAPSE = '^(node_modules/@[a-z0-9-]+|node_modules|src)/[^/]+'
