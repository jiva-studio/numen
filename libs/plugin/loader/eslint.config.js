import tseslint from 'typescript-eslint'
import { createPlainLibraryConfig } from '../../../tools/lint-layers/base-eslint.mjs'

export default tseslint.config(...createPlainLibraryConfig(import.meta.dirname))
