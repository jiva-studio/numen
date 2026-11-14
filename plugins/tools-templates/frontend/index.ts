/**
 * Frontend entrypoint for the Templates plugin: a command that makes a new note from a template.
 */
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { templatesApi, type TemplatesApi } from './api/wire'

export const activate = (_context: PluginContext, _api: TemplatesApi = templatesApi): void => {
  // The command is registered when the feature is implemented.
}

export default definePlugin({
  activate,
})
