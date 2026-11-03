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
/** The seconds of the spec files of `paths`, or of every spec file when `paths` is undefined. */
export const readSelectedDurations: (paths?: string[]) => Record<string, number>
/** The paths NUMEN_E2E_PATHS names, or undefined when it names none. */
export const readPathsFromEnv: () => string[] | undefined
/** The spec files of each of `count` shards, out of the files of `paths`. */
export const planShards: (count: number, paths?: string[]) => string[][]
