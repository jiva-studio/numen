import { test, expect } from '../../support/test.js'
import {
  createNote,
  createUniqueTitle,
  destroyNote,
  openNote,
  writeNote,
  writeSetting,
} from '../../support/vault.js'
import { getGraph, openWorkspace } from '../../support/workspace.js'

const PLEX = 'md.numen.views.plex.graph'
const WITHIN = { timeout: 15_000 }

test.describe('The plex of a note that is linked in the text', () => {
  const paths: string[] = []

  test.afterEach(async ({ request }) => {
    await writeSetting(request, [PLEX, 'references'], 'both')
    for (const notePath of paths.splice(0)) await destroyNote(request, notePath)
  })

  test('shows the notes it links to and the note that links to it, as the setting says', async ({
    page,
    request,
  }) => {
    const focusTitle = createUniqueTitle('Salt marsh')
    const heronTitle = createUniqueTitle('Heron nesting')
    const tidesTitle = createUniqueTitle('Tide tables')
    const surveyTitle = createUniqueTitle('Estuary survey')

    const focus = await createNote(request, focusTitle)
    const heron = await createNote(request, heronTitle)
    const tides = await createNote(request, tidesTitle)
    const survey = await createNote(request, surveyTitle)
    paths.push(focus, heron, tides, survey)
    await writeNote(
      request,
      focus,
      `Birds are in [[${heronTitle}]], water is in [[${tidesTitle}]].`,
    )
    await writeNote(request, survey, `The marsh is described in [[${focusTitle}]].`)

    await openWorkspace(page)
    await openNote(page, focus)
    // A plex follows the note it was opened on, so the one of the note in front is opened now.
    await page.getByRole('button', { name: 'Open plex graph' }).click()
    const graph = getGraph(page).filter({ has: page.getByLabel(`${focusTitle}, focus`) })
    const references = graph.getByLabel(/, reference$/)

    await expect(references).toHaveCount(3, WITHIN)
    await expect(graph.getByLabel(`${heronTitle}, reference`)).toBeVisible()
    await expect(graph.getByLabel(`${tidesTitle}, reference`)).toBeVisible()
    await expect(graph.getByLabel(`${surveyTitle}, reference`)).toBeVisible()

    await writeSetting(request, [PLEX, 'references'], 'outgoing')
    await expect(references).toHaveCount(2, WITHIN)
    await expect(graph.getByLabel(`${surveyTitle}, reference`)).toHaveCount(0)

    await writeSetting(request, [PLEX, 'references'], 'off')
    await expect(references).toHaveCount(0, WITHIN)
  })
})
