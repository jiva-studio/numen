import * as prettierPluginTailwindcss from 'prettier-plugin-tailwindcss'

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
