/**
 * Frontend entrypoint for the Templates plugin: a command that makes a new note from a template.
 */
import { definePlugin, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import type { CreateNoteResponse } from './api/templates'
import { templatesApi, type TemplatesApi } from './api/wire'
import { askFields } from './lib/askFields'
import { getTargetFolder } from './lib/getTargetFolder'
import { getTitle } from './lib/groupTemplates'
import { parseMasks } from './lib/masks'
import { createTemplateList, TemplateListError } from './model/createTemplateList'

const openAnswer = (host: HostServices, answer: CreateNoteResponse): void => {
  host.ui.workspace?.files.open(answer.path)
  if (answer.warning) host.ui.workspace?.messages.getWriter('templates')(answer.warning, 'warning')
}

export const activate = (
  context: PluginContext,
  host: HostServices,
  api: TemplatesApi = templatesApi,
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
      const { fields } = asked.value
      const picker = host.ui.picker
      if (fields.length > 0 && !picker) {
        host.ui.workspace?.messages.showError('the window cannot ask the fields of the template')
        return
      }
      const values =
        fields.length > 0 && picker ? await askFields(picker, getTitle(itemId), fields) : {}
      if (!values) return
      const answer = await api.createNote({
        template: itemId,
        folder,
        values,
        etag: asked.value.etag,
      })
      if (answer.ok) openAnswer(host, answer.value)
      else host.ui.workspace?.messages.showError(answer.error.message)
    },
  })
}

export default definePlugin({
  activate: (context, host) => activate(context, host),
})
