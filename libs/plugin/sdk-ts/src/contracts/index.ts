/**
 * The contracts between plugins, one namespace per contract. A plugin imports them from this entry,
 * the main entry of the SDK does not carry them. acptest and llmtest serve their contracts in the
 * tests of a plugin.
 */
export * as acp from './acp'
export * as acptest from './acptest'
export * as llm from './llm'
export * as llmtest from './llmtest'
