import path from 'node:path'
import { expect, test } from '../tests/support/certificate.js'
import { STAND_IN_REPLY } from '../scripts/stand-ins.mjs'
import type { JsonObject } from '../tests/support/json.js'

const vault = process.env.NUMEN_FLASHCARDS_VAULT ?? ''

function readFrame(body: Buffer): JsonObject {
  return JSON.parse(body.subarray(5, 5 + body.readUInt32BE(1)).toString('utf8'))
}

const STUDIO = 'md.numen.views.flashcards.studio'

// The task runs on a connection row of the studio, numen's own agent over the stand-in chat API the
// server script starts.
test('the review mounts the conversation of the agent chat plugin in its agent panel', async ({
  page,
}) => {
  const requests: JsonObject[] = []
  page.on('request', (request) => {
    if (request.url().endsWith('/numen.v1.ConnectionCallService/Call')) {
      requests.push(readFrame(request.postDataBuffer() ?? Buffer.alloc(5)))
    }
  })
  await page.goto('/')

  await page.getByRole('button', { name: new RegExp(path.basename(vault)) }).click()
  await page.getByRole('button', { name: /Review/ }).click()
  await page.getByRole('button', { name: /Ask/ }).click()

  const panel = page.getByRole('region', { name: 'Ask about this card' })
  const field = panel.getByRole('textbox', { name: 'Ask about this card' })
  await expect(field).toBeVisible()
  await field.fill('Where does it live?')
  await panel.getByRole('button', { name: 'Send' }).click()

  await expect(panel.getByText('Where does it live?')).toBeVisible()
  await expect(panel).toContainText(STAND_IN_REPLY)
  // The task runs on the rows of the studio, with the card in the context of its tools.
  const tasks = requests.filter((one) => one['operation'] === 'acp.task')
  expect(tasks).toHaveLength(1)
  expect(tasks[0]).toMatchObject({ pluginId: STUDIO })
  const payload = Buffer.from(String(tasks[0]?.['payload'] ?? ''), 'base64').toString('utf8')
  expect(payload).toContain('Where does it live?')
  for (const key of ['deck', 'card', 'template']) expect(payload).toContain(`${STUDIO}/${key}`)
})
