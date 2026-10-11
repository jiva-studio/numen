#!/usr/bin/env node
/**
 * The matrix of the nightly workflow: one entry per shard of each target, or
 * of the one target named.
 *
 *   node shards.mjs [target|all]   prints `matrix=[...]`
 */
import { getTarget, getTargets } from './targets.mjs'

export const listShards = (name = 'all') =>
  (name === 'all' ? getTargets() : [getTarget(name)]).flatMap((target) =>
    Array.from({ length: target.shards }, (_, at) => ({
      target: target.name,
      kind: target.kind,
      at: target.at,
      shard: `${at + 1}/${target.shards}`,
      name: target.shards > 1 ? `${target.name}-${at + 1}-of-${target.shards}` : target.name,
    })),
  )

if (process.argv[1] === new URL(import.meta.url).pathname) {
  const name = process.argv[2] || 'all'
  console.log(`matrix=${JSON.stringify(listShards(name))}`)
  console.log(
    `targets=${JSON.stringify(name === 'all' ? getTargets().map((one) => one.name) : [name])}`,
  )
}
