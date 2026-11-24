/** The way from one computer to the sync server: a person sees a server that answers or one that is away. */
import net from 'node:net'

export class Gate {
  private readonly server: net.Server
  private readonly open = new Set<net.Socket>()
  private isClosed = false

  private constructor(private readonly getTarget: () => { readonly host: string; readonly port: number }) {
    this.server = net.createServer((client) => this.accept(client))
  }

  /** Listens on a free port of this machine and forwards each connection to the target as it is when the connection comes. */
  static async listen(getTarget: () => { readonly host: string; readonly port: number }): Promise<Gate> {
    const gate = new Gate(getTarget)
    await new Promise<void>((resolve) => gate.server.listen(0, '127.0.0.1', resolve))
    return gate
  }

  getAddress(): string {
    const address = this.server.address()
    if (address === null || typeof address === 'string') throw new Error('the gate does not listen')
    return `http://127.0.0.1:${address.port}`
  }

  /** Makes the server unreachable from this computer: the connections it has end and the new ones are refused. */
  close(): void {
    this.isClosed = true
    for (const socket of this.open) socket.destroy()
    this.open.clear()
  }

  reopen(): void {
    this.isClosed = false
  }

  stop(): void {
    this.close()
    this.server.close()
  }

  private accept(client: net.Socket): void {
    if (this.isClosed) {
      client.destroy()
      return
    }
    const target = this.getTarget()
    const upstream = net.connect(target.port, target.host)
    for (const socket of [client, upstream]) {
      this.open.add(socket)
      socket.once('close', () => {
        this.open.delete(socket)
        client.destroy()
        upstream.destroy()
      })
      socket.on('error', () => undefined)
    }
    client.pipe(upstream)
    upstream.pipe(client)
  }
}
