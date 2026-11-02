// Fails when a spec file is not listed in shards.json, or shards.json lists a file that is gone.
import { checkShards } from './shards.mjs'

const problems = checkShards()
if (problems.length > 0) {
  console.error(problems.join('\n'))
  process.exit(1)
}
