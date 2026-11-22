/**
 * Frontend entrypoint for the Templates plugin: a command that makes a new note from a template.
 */
import { definePlugin, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import type { CreateNoteResponse } from './api/templates'
import { templatesApi, type TemplatesApi } from './api/wire'
import { getTargetFolder } from './lib/getTargetFolder'
import { parseMasks } from './lib/masks'
import { createFromTemplate } from './model/createFromTemplate'
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
      const outcome = await createFromTemplate(
        { api, picker: host.ui.picker },
        itemId,
        getTargetFolder(note),
      )
      if (outcome.kind === 'made') openAnswer(host, outcome.answer)
      else if (outcome.kind === 'failed') host.ui.workspace?.messages.showError(outcome.message)
    },
  })
}

export default definePlugin({
  activate: (context, host) => activate(context, host),
})
