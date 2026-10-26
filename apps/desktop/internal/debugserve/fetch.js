// Sends the page's own POST requests as PUT, which the debug server answers as POST (see page.go).
const send = globalThis.fetch
globalThis.fetch = (input, init) => {
  const isOwn =
    (typeof input === 'string' || input instanceof URL) &&
    new URL(input, location.href).origin === location.origin
  if (isOwn && init?.method?.toUpperCase() === 'POST') {
    return send(input, { ...init, method: 'PUT' })
  }
  return send(input, init)
}
