import apiClient from './client';
import type {
  DetailedQuote,
  Asset,
  Portfolio,
  PlaceOrderRequest,
  PlaceOrderResponse,
  Order,
  OrdersResponse,
  AssetSearchResponse,
  Wallet,
} from '../types';

// Asset search
export async function searchAssets(query: string, limit = 20): Promise<AssetSearchResponse> {
  return apiClient.get<AssetSearchResponse>('/trading/assets/search', { q: query, limit });
}

// Get quote for a symbol
export async function getQuote(symbol: string): Promise<DetailedQuote> {
  return apiClient.get<DetailedQuote>(`/trading/quotes/${symbol}`);
}

// Get asset details
export async function getAsset(symbol: string): Promise<Asset> {
  return apiClient.get<Asset>(`/trading/assets/${symbol}`);
}

// Get portfolio summary and holdings
export async function getPortfolio(): Promise<Portfolio> {
  return apiClient.get<Portfolio>('/portfolio');
}

// Get wallet balance
export async function getWalletBalance(): Promise<Wallet> {
  return apiClient.get<Wallet>('/wallet/balance');
}

// Place an order
export async function placeOrder(request: PlaceOrderRequest): Promise<PlaceOrderResponse> {
  return apiClient.post<PlaceOrderResponse>('/trading/orders', request);
}

// Get orders with optional status filter
export async function getOrders(status?: string, limit = 50): Promise<OrdersResponse> {
  const params: Record<string, any> = { limit };
  if (status) {
    params.status = status;
  }
  return apiClient.get<OrdersResponse>('/trading/orders', params);
}

// Get a single order by ID
export async function getOrder(orderId: string): Promise<Order> {
  return apiClient.get<Order>(`/trading/orders/${orderId}`);
}

// Cancel an order
export async function cancelOrder(orderId: string): Promise<{ message: string }> {
  return apiClient.delete<{ message: string }>(`/trading/orders/${orderId}`);
}

// Export all functions as a trading API object
export const tradingApi = {
  searchAssets,
  getQuote,
  getAsset,
  getPortfolio,
  getWalletBalance,
  placeOrder,
  getOrders,
  getOrder,
  cancelOrder,
};

export default tradingApi;
