import fs from 'node:fs'
import path from 'node:path'
import { test, expect, type APIRequestContext } from '../../support/test.js'
import { getVaultPath, readNote } from '../../support/vault.js'
import type { JsonObject } from '../../support/json.js'

const askRPC = async (request: APIRequestContext, method: string, data: object) => {
  const answer = await request.post(`/numen.v1.${method}`, {
    data,
    headers: { 'Content-Type': 'application/json' },
  })
  const text = await answer.text()
  return {
    status: answer.status(),
    body: text ? (JSON.parse(text) as JsonObject) : {},
  }
}

const isRefused = (answer: { status: number; body: JsonObject }): boolean =>
  answer.status !== 200 || answer.body['error'] !== undefined

test.describe('Paths outside the vault', () => {
  test('refuses to read, write or move a note by a path that leaves the vault', async ({
    request,
  }) => {
    const vault = getVaultPath()
    const name = `outside-${Math.random().toString(36).slice(2, 7)}`
    const outside = path.join(path.dirname(vault), `${name}.md`)
    fs.writeFileSync(outside, 'A file beside the vault.\n')
    try {
      for (const target of [`../${name}.md`, outside, `notes/../../${name}.md`]) {
        const read = await askRPC(request, 'NoteService/ReadNote', { path: target })
        expect(isRefused(read), `ReadNote ${target}`).toBe(true)
        expect(JSON.stringify(read.body)).not.toContain('A file beside the vault.')

        const write = await askRPC(request, 'NoteService/WriteNote', {
          path: target,
          body: 'Written from the window.\n',
        })
        expect(isRefused(write), `WriteNote ${target}`).toBe(true)
        expect(fs.readFileSync(outside, 'utf8')).toBe('A file beside the vault.\n')
      }

      const escaped = `../${name}-moved.md`
      const move = await askRPC(request, 'FileService/MoveFile', {
        from: 'notes/Entropy.md',
        to: escaped,
      })
      expect(isRefused(move)).toBe(true)
      expect(fs.existsSync(path.join(path.dirname(vault), `${name}-moved.md`))).toBe(false)
      expect((await readNote(request, 'notes/Entropy.md')).body).toContain(
        'A measure of uncertainty',
      )

      const created = await askRPC(request, 'NoteService/CreateNote', {
        title: `${name} made`,
        path: '..',
      })
      expect(isRefused(created)).toBe(true)
      expect(fs.readdirSync(path.dirname(vault)).filter((one) => one.startsWith(name))).toEqual([
        `${name}.md`,
      ])
    } finally {
      for (const one of fs
        .readdirSync(path.dirname(vault))
        .filter((file) => file.startsWith(name))) {
        fs.rmSync(path.join(path.dirname(vault), one), { force: true })
      }
    }
  })

  test('serves no file outside the vault or a plugin folder by an escaping address', async ({
    request,
  }) => {
    const vault = getVaultPath()
    // A folder whose name starts with the vault's own: a check by prefix alone takes it as inside.
    const sibling = `${vault}-beside`
    fs.mkdirSync(sibling, { recursive: true })
    fs.writeFileSync(path.join(sibling, 'secret.txt'), 'Kept beside the vault.\n')
    try {
      const inside = await request.get('/assets/notes/Entropy.md')
      expect(inside.status()).toBe(200)

      const escapes = [
        `/assets/%2E%2E%2F${encodeURIComponent(path.basename(sibling))}%2Fsecret.txt`,
        `/assets/%2E%2E%2F%2E%2E%2F${encodeURIComponent(path.basename(path.dirname(vault)))}%2F${encodeURIComponent(path.basename(sibling))}%2Fsecret.txt`,
        '/plugin/md.numen.notes/%2E%2E%2F%2E%2E%2Fpackage.json',
        '/plugin/md.numen.notes/..%2F..%2Fpackage.json',
      ]
      for (const address of escapes) {
        const answer = await request.get(address)
        expect(answer.status(), address).toBeGreaterThanOrEqual(400)
        expect(await answer.text(), address).not.toContain('Kept beside the vault.')
      }
    } finally {
      fs.rmSync(sibling, { recursive: true, force: true })
    }
  })
})
