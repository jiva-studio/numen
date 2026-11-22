/**
 * Frontend entrypoint for the Templates plugin: a command that makes a new note from a template.
 */
import { definePlugin, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import { templatesApi, type TemplatesApi } from './api/wire'
import { getTargetFolder } from './lib/getTargetFolder'
import { normalizeFolder } from './lib/normalizeFolder'
import { createTemplateList } from './model/createTemplateList'

export const activate = (
  context: PluginContext,
  host: HostServices,
  api: TemplatesApi = templatesApi,
): void => {
  const files = host.vault.files
  if (!files) throw new Error('the Templates plugin needs the files of the vault')
  const list = createTemplateList({
    files,
    getFolder: () => normalizeFolder(context.settings.get<string>('folder') ?? ''),
  })
  context.register.command({
    id: 'note.createFromTemplate',
    listItems: () => list.listItems(),
    handler: async (note, itemId) => {
      if (!itemId) return
      const answer = await api.createNote({ template: itemId, folder: getTargetFolder(note) })
      if (answer.ok) host.ui.workspace?.files.open(answer.value.path)
      else host.ui.workspace?.messages.showError(answer.error.message)
    },
  })
}

export default definePlugin({
  activate: (context, host) => activate(context, host),
})
