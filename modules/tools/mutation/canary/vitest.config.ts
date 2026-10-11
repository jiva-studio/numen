/**
 * The canary's tests: a module whose every mutant a working runner kills, run
 * before each TypeScript shard.
 */
import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: { environment: 'node', include: ['src/**/*.test.ts'] },
})
