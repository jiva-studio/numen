import tseslint from 'typescript-eslint'
import {
  createNoUnknownContractConfig,
  createPlainLibraryConfig,
} from '../../../tools/lint-layers/base-eslint.mjs'

export default tseslint.config(
  ...createPlainLibraryConfig(import.meta.dirname),
  createNoUnknownContractConfig(['src/hosting/types.ts']),
)
