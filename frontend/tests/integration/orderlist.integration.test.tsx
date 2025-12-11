/** @jest-environment jsdom */
import React from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import { OrderManagement } from '../../src/components/admin/OrderManagement'

// Mock order service
jest.mock('../../src/services/order', () => ({
  order: {
    listOrders: jest.fn(async (filters?: any) => ({
      orders: [
        { order: { id: 'o1', order_reference: 'R-1', status: 'created', created_at: new Date().toISOString(), total_amount: 100 }, items: [] }
      ],
      pagination: { limit: 50, offset: 0, count: 1 }
    }))
  }
}))

// Mock EventSource like unit test
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
      // emit a sample order_paid event after open
      setTimeout(() => {
        const ev: any = {
          data: JSON.stringify({
            event: 'order_paid',
            data: { order: { id: 'o1', order_reference: 'R-1', status: 'paid', total_amount: 100, created_at: new Date().toISOString() } }
          }), lastEventId: 'evt-1'
        }
        this.onmessage && this.onmessage(ev)
      }, 10)
    }, 0)
  }
  close() { this.readyState = 2 }
  static lastInstance: any
}

describe('OrderList integration', () => {
  let origES: any
  beforeAll(() => { origES = (global as any).EventSource; (global as any).EventSource = MockEventSource })
  afterAll(() => { (global as any).EventSource = origES })

  test('renders initial orders and updates on SSE event', async () => {
    render(<OrderManagement />)

    // initial list loaded (container exists)
    const list = await waitFor(() => screen.getByTestId('order-list'))
    expect(list).toBeInTheDocument()

    // after SSE event, the status should be updated to paid
    await waitFor(() => expect(list).toHaveTextContent('paid'))
  })
})
