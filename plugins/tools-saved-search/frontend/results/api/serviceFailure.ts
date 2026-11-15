/** A call the backend answered with a failure, or that did not reach it; the message is the backend's. */
export class ServiceFailure extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'ServiceFailure'
  }
}
