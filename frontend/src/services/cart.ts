import axios from 'axios';
import { Cart, CartItem } from '../types/cart';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Re-export types for backward compatibility
export type { Cart, CartItem };

class CartService {
  private sessionId: string | null = null;
  private readonly SESSION_KEY = 'guest_session_id';
  private readonly CART_KEY = 'guest_cart';

  constructor() {
    if (typeof window !== 'undefined') {
      this.initializeSession();
    }
  }

  private initializeSession(): void {
    let sessionId = localStorage.getItem(this.SESSION_KEY);
    if (!sessionId) {
      sessionId = this.generateSessionId();
      localStorage.setItem(this.SESSION_KEY, sessionId);
    }
    this.sessionId = sessionId;
  }

  private generateSessionId(): string {
    return `sess_${Date.now()}_${Math.random().toString(36).substring(2, 15)}`;
  }

  getSessionId(): string {
    if (!this.sessionId) {
      this.initializeSession();
    }
    return this.sessionId!;
  }

  private getHeaders() {
    return {
      'X-Session-Id': this.getSessionId(),
      'Content-Type': 'application/json',
    };
  }

  private getLocalCart(tenantId: string): Cart | null {
    const cartData = localStorage.getItem(`${this.CART_KEY}_${tenantId}`);
    if (cartData) {
      try {
        return JSON.parse(cartData);
      } catch {
        return null;
      }
    }
    return null;
  }

  private saveLocalCart(cart: Cart): void {
    localStorage.setItem(`${this.CART_KEY}_${cart.tenant_id}`, JSON.stringify(cart));
  }

  async getCart(tenantId: string): Promise<Cart> {
    try {
      const response = await axios.get(
        `${API_BASE_URL}/api/v1/public/${tenantId}/cart`,
        { headers: this.getHeaders() }
      );
      const cart = response.data;
      this.saveLocalCart(cart);
      return cart;
    } catch (error) {
      // Fallback to localStorage if server is unavailable
      const localCart = this.getLocalCart(tenantId);
      if (localCart) {
        return localCart;
      }
      // Return empty cart
      return {
        tenant_id: tenantId,
        session_id: this.getSessionId(),
        items: [],
        updated_at: new Date().toISOString(),
        total: 0,
      };
    }
  }

  async addItem(
    tenantId: string,
    productId: string,
    productName: string,
    quantity: number,
    unitPrice: number
  ): Promise<Cart> {
    const response = await axios.post(
      `${API_BASE_URL}/api/v1/public/${tenantId}/cart/items`,
      {
        product_id: productId,
        product_name: productName,
        quantity,
        unit_price: unitPrice,
      },
      { headers: this.getHeaders() }
    );
    const cart = response.data;
    this.saveLocalCart(cart);
    return cart;
  }

  async updateItem(tenantId: string, productId: string, quantity: number): Promise<Cart> {
    const response = await axios.patch(
      `${API_BASE_URL}/api/v1/public/${tenantId}/cart/items/${productId}`,
      { quantity },
      { headers: this.getHeaders() }
    );
    const cart = response.data;
    this.saveLocalCart(cart);
    return cart;
  }

  async removeItem(tenantId: string, productId: string): Promise<Cart> {
    const response = await axios.delete(
      `${API_BASE_URL}/api/v1/public/${tenantId}/cart/items/${productId}`,
      { headers: this.getHeaders() }
    );
    const cart = response.data;
    this.saveLocalCart(cart);
    return cart;
  }

  async clearCart(tenantId: string): Promise<void> {
    await axios.delete(
      `${API_BASE_URL}/api/v1/public/${tenantId}/cart`,
      { headers: this.getHeaders() }
    );
    localStorage.removeItem(`${this.CART_KEY}_${tenantId}`);
  }

  getCartTotal(cart: Cart): number {
    return cart.items.reduce((sum, item) => sum + item.total_price, 0);
  }

  getCartItemCount(cart: Cart): number {
    return cart.items.reduce((sum, item) => sum + item.quantity, 0);
  }
}

export const cart = new CartService();
