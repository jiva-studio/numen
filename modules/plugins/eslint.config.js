import js from '@eslint/js'
import globals from 'globals'
import pluginVue from 'eslint-plugin-vue'
import tseslint from 'typescript-eslint'

/** The packages a plugin frontend names besides its own directory and third-party code. */
const contract = ['plugin-sdk', 'ui', 'wire', 'protocol']

/** The shared fixtures of the contract that a test reads by path. */
const fixtureReaders = [
  'reader-epub/frontend/lib/markup.test.ts',
  'flashcards-studio/frontend/cards/lib/presets.test.ts',
  'flashcards-studio/frontend/preset/lib/sketch.test.ts',
]

/** The harness the stories of every plugin are drawn in, which belongs to no plugin. */
const storyHarness = ['.storybook/**']

/** The build configuration of each plugin, which reads the build every plugin shares from `../vite.shared`. */
const buildConfigs = ['*/vite.config.ts', '*/vitest.config.ts']

const hostMessage =
  'plugins must not import host internals (@/*); use @numen/plugin-sdk or @numen/ui instead'

/** Imports a plugin frontend may not write, given how far out of its directory a relative path reaches. */
function createRestrictions(escapes, shared) {
  return [
    'error',
    {
      patterns: [
        {
          group: [
            '@/**',
            '@/pages/**',
            '@/shared/**',
            '@/entities/**',
            '@/features/**',
            '@/widgets/**',
            '@/app/**',
          ],
          message: hostMessage,
        },
        {
          group: ['@plugins', '@plugins/**'],
          message: 'a plugin names no other plugin; share through @numen/plugin-sdk',
        },
        {
          group: [
            '@numen/*',
            ...contract.flatMap((name) => [`!@numen/${name}`, `!@numen/${name}/**`]),
          ],
          message: `a plugin names only ${contract.map((name) => '@numen/' + name).join(', ')} and its own directory`,
        },
        ...(escapes === undefined
          ? []
          : [
              {
                regex: `^(\\.\\./){${escapes},}${shared === undefined ? '' : `(?!${shared}$)`}`,
                message: 'a plugin names only files of its own directory',
              },
            ]),
      ],
    },
  ]
}

/** One block per depth below frontend/: a file n directories down leaves its directory with n+1 or more steps up. */
function pluginImportBlocks() {
  const blocks = [
    { files: ['**/*.{ts,vue}'], rules: { 'no-restricted-imports': createRestrictions() } },
  ]
  for (let depth = 0; depth <= 8; depth++) {
    const subdirs = depth === 0 ? '' : '*/'.repeat(depth)
    blocks.push({
      files: [`*/frontend/${subdirs}*.{ts,vue}`],
      ignores: [...fixtureReaders, ...storyHarness],
      rules: { 'no-restricted-imports': createRestrictions(depth + 1) },
    })
  }
  blocks.push({
    files: buildConfigs,
    rules: { 'no-restricted-imports': createRestrictions(1, 'vite\\.shared') },
  })
  return blocks
}

export default tseslint.config(
  {
    ignores: ['**/dist/**', '**/coverage/**', '**/node_modules/**'],
  },
  js.configs.recommended,
  tseslint.configs.recommended,
  pluginVue.configs['flat/essential'],
  {
    languageOptions: {
      globals: {
        ...globals.browser,
      },
    },
  },
  {
    files: ['**/*.vue'],
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser,
      },
    },
  },
  {
    rules: {
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
      ],
    },
  },
  {
    files: ['**/*.test.ts'],
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
    },
  },
  ...pluginImportBlocks(),
)
