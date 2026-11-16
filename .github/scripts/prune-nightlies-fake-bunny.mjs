import { createServer } from 'node:http'
import { appendFileSync, writeFileSync } from 'node:fs'

const [log, portFile, ...names] = process.argv.slice(2)
const entries = new Set(names)
const key = 'test-key'

createServer((req, res) => {
  appendFileSync(log, `${req.method} ${req.url} ${req.headers.accesskey === key ? 'keyed' : 'unkeyed'}\n`)
  if (req.headers.accesskey !== key) {
    res.writeHead(401).end()
    return
  }
  const url = new URL(req.url, 'http://x')
  if (req.method === 'GET' && url.pathname === '/zone/builds/') {
    const body = [...entries].map((name) => ({ ObjectName: name, IsDirectory: !name.endsWith('.txt'), Path: '/zone/builds/' }))
    res.writeHead(200, { 'Content-Type': 'application/json' }).end(JSON.stringify(body))
    return
  }
  const match = /^\/zone\/builds\/([^/]+)\/$/.exec(url.pathname)
  if (req.method === 'DELETE' && match && entries.delete(match[1])) {
    res.writeHead(200).end('{}')
    return
  }
  res.writeHead(400).end()
}).listen(0, '127.0.0.1', function () {
  writeFileSync(portFile, String(this.address().port))
})
