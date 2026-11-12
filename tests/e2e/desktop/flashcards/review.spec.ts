import path from 'node:path'
import { expect, test } from '../tests/support/certificate.js'
import { STAND_IN_REPLY } from '../scripts/stand-ins.mjs'
import type { JsonObject } from '../tests/support/json.js'

const vault = process.env.NUMEN_FLASHCARDS_VAULT ?? ''

function readFrame(body: Buffer): JsonObject {
  return JSON.parse(body.subarray(5, 5 + body.readUInt32BE(1)).toString('utf8'))
}

// The window's agent is numen's own, over the stand-in chat API the server script starts.
test('the review mounts the conversation of the agent chat plugin in its agent panel', async ({
  page,
}) => {
  const requests: JsonObject[] = []
  page.on('request', (request) => {
    if (request.url().endsWith('/numen.v1.AgentService/AskAgent')) {
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
  // The card goes in the context of the question, and the question is only what was written.
  expect(requests).toHaveLength(1)
  expect(requests[0]).toMatchObject({
    prompt: 'Where does it live?',
    focus: expect.stringMatching(/\.md$/),
  })
  expect(requests[0]?.['context']).toEqual({
    'md.numen.views.flashcards.studio/deck': expect.stringMatching(/\.md$/),
    'md.numen.views.flashcards.studio/card': expect.any(String),
    'md.numen.views.flashcards.studio/template': expect.any(String),
  })
})
