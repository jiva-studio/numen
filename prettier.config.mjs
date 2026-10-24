import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)
let prettierPluginTailwindcss
try {
  prettierPluginTailwindcss = require('prettier-plugin-tailwindcss')
} catch {
  try {
    prettierPluginTailwindcss = require('./modules/node_modules/prettier-plugin-tailwindcss')
  } catch {
    prettierPluginTailwindcss = 'prettier-plugin-tailwindcss'
  }
}

/** @type {import("prettier").Config} */
export default {
  semi: false,
  singleQuote: true,
  tabWidth: 2,
  trailingComma: 'all',
  bracketSameLine: false,
  htmlWhitespaceSensitivity: 'ignore',
  vueIndentScriptAndStyle: false,
  printWidth: 100,
  plugins: [prettierPluginTailwindcss],
  tailwindFunctions: ['cn', 'cva', 'clsx'],
}
