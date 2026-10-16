import js from '@eslint/js'
import pluginVue from 'eslint-plugin-vue'
import tseslint from 'typescript-eslint'

export default tseslint.config(
  {
    ignores: ['**/dist/**', '**/coverage/**', '**/node_modules/**'],
  },
  js.configs.recommended,
  tseslint.configs.recommended,
  pluginVue.configs['flat/essential'],
  {
    files: ['**/*.{ts,vue}'],
    rules: {
      'no-restricted-imports': [
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
              message:
                'plugins must not import host internals (@/*); use @numen/plugin-sdk or @numen/ui instead',
            },
          ],
        },
      ],
    },
  },
)
