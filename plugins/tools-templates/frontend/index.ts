/**
 * Frontend entrypoint for the Templates plugin: a command that makes a new note from a template.
 */
import './styles.css'
import { definePlugin, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import { templatesApi, type TemplatesApi } from './api/wire'
import { askFields, type AskFields } from './ui/askFields'
import { getTargetFolder } from './lib/getTargetFolder'
import { getTitle } from './lib/groupTemplates'
import { parseMasks } from './lib/masks'
import { createTemplateList, TemplateListError } from './model/createTemplateList'

export const activate = (
  context: PluginContext,
  host: HostServices,
  api: TemplatesApi = templatesApi,
  ask: AskFields = askFields,
): void => {
  const files = host.vault.files
  if (!files) throw new Error('the Templates plugin needs the files of the vault')
  const list = createTemplateList({
    files,
    getMasks: () => parseMasks(context.settings.get<string>('templates') ?? ''),
  })
  context.register.command({
    id: 'note.createFromTemplate',
    listItems: async () => {
      try {
        return await list.listItems()
      } catch (error) {
        if (!(error instanceof TemplateListError)) throw error
        host.ui.workspace?.messages.showError(error.message)
        return []
      }
    },
    handler: async (note, itemId) => {
      if (!itemId) return
      const folder = getTargetFolder(note)
      const asked = await api.listFields({ template: itemId, folder })
      if (!asked.ok) {
        host.ui.workspace?.messages.showError(asked.error.message)
        return
      }
      const values =
        asked.value.fields.length > 0 ? await ask(getTitle(itemId), asked.value.fields) : {}
      if (!values) return
      const answer = await api.createNote({ template: itemId, folder, values })
      if (answer.ok) host.ui.workspace?.files.open(answer.value.path)
      else host.ui.workspace?.messages.showError(answer.error.message)
    },
  })
}

export default definePlugin({
  activate: (context, host) => activate(context, host),
})
