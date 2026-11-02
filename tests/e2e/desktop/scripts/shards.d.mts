/** The spec files under tests/, relative to the e2e folder. */
export const listSpecs: () => string[]
/** The seconds of every spec file in shards.json. */
export const readDurations: () => Record<string, number>
/** Writes the seconds of the spec files to shards.json. */
export const writeDurations: (durations: Record<string, number>) => void
/** The problems of shards.json against the spec files. */
export const checkShards: () => string[]
/** The shard count when none is named. */
export const getDefaultShardCount: () => number
/** The spec files of each of `count` shards. */
export const planShards: (count: number) => string[][]
