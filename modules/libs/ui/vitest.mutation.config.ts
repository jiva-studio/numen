/** The unit project alone, rooted here: the tests mutation testing runs against each mutant. */
import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'
import base from './vitest.config'

const projects = (base.test?.projects ?? []).filter(
  (project) => typeof project === 'object' && 'test' in project && project.test?.name === 'unit',
)

export default defineConfig({
  ...base,
  root: fileURLToPath(new URL('.', import.meta.url)),
  test: { ...base.test, projects },
})
