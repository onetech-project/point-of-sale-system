import SseClient from '../../src/services/sse'

class MockEventSource {
  url: string
  onopen: any
  onmessage: any
  onerror: any
  readyState = 0
  constructor(url: string) {
    this.url = url
    MockEventSource.lastInstance = this
    setTimeout(() => {
      this.readyState = 1
      this.onopen && this.onopen()
    }, 0)
  }
  close() {
    this.readyState = 2
  }
  // helper to simulate a message
  static emitMessage(data: any, lastEventId?: string) {
    const inst = MockEventSource.lastInstance
    if (!inst) return
    const ev: any = { data: JSON.stringify(data), lastEventId: lastEventId }
    inst.onmessage && inst.onmessage(ev)
  }
  static emitError(err?: any) {
    const inst = MockEventSource.lastInstance
    if (!inst) return
    inst.onerror && inst.onerror(err)
  }
  static lastInstance: any
}

describe('SseClient', () => {
  let origES: any
  beforeAll(() => {
    origES = (global as any).EventSource
      ; (global as any).EventSource = MockEventSource
  })
  afterAll(() => {
    ; (global as any).EventSource = origES
  })

  test('connects and receives messages, preserves lastEventId', async () => {
    const sse = new SseClient({ url: 'http://localhost/api/v1/sse' })
    const events: any[] = []
    sse.connect({ onEvent: (e) => events.push(e) })

    // Wait for next tick when onopen fired
    await new Promise((r) => setTimeout(r, 0))

    MockEventSource.emitMessage({
      event: 'order_created',
      data: { order: { id: 'o1' } }
    }, '123')

    expect(events.length).toBe(1)
    expect(events[0].id).toBe('123')
    expect(events[0].event).toBe('order_created')

    sse.close()
  })

  test('calls snapshotFn on error when closed', async () => {
    const snapshot = jest.fn(async () => { })
    const sse = new SseClient({ url: 'http://localhost/api/v1/sse', snapshotFn: snapshot })
    sse.connect()
    await new Promise((r) => setTimeout(r, 0))

    // simulate closed readyState
    MockEventSource.lastInstance.readyState = 2
    MockEventSource.emitError(new Error('closed'))

    // allow snapshot to be called
    await new Promise((r) => setTimeout(r, 10))
    expect(snapshot).toHaveBeenCalled()

    sse.close()
  })
})
