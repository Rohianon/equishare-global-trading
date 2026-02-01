// User types
export interface User {
  id: string;
  phone?: string;
  email?: string;
  username?: string;
  display_name?: string;
  avatar_url?: string;
  kyc_status: 'pending' | 'submitted' | 'verified' | 'rejected';
  kyc_tier: 'tier1' | 'tier2' | 'tier3';
  phone_verified: boolean;
  email_verified: boolean;
}

// Auth types
export interface AuthTokens {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface LoginRequest {
  phone: string;
  pin: string;
}

export interface RegisterRequest {
  phone: string;
}

export interface VerifyRequest {
  phone: string;
  otp: string;
  pin: string;
}

export interface OAuthCallbackResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  expires_in: number;
  is_new_user: boolean;
  needs_phone: boolean;
  needs_username: boolean;
}

// Wallet types
export interface Wallet {
  available: number;
  pending: number;
  total: number;
  currency: string;
}

// Transaction types
export interface Transaction {
  id: string;
  type: 'deposit' | 'withdrawal' | 'buy' | 'sell';
  amount: number;
  currency: string;
  status: 'pending' | 'completed' | 'failed';
  description: string;
  created_at: string;
}

// Holdings types
export interface Holding {
  id: string;
  symbol: string;
  name: string;
  quantity: number;
  avg_cost: number;
  current_price: number;
  market_value: number;
  unrealized_pl: number;
  unrealized_pl_percent: number;
}

// Order types
export interface Order {
  id: string;
  symbol: string;
  side: 'buy' | 'sell';
  type: 'market' | 'limit';
  quantity: number;
  price?: number;
  status: 'pending' | 'filled' | 'cancelled' | 'rejected';
  filled_quantity: number;
  filled_avg_price?: number;
  created_at: string;
  filled_at?: string;
}

export interface CreateOrderRequest {
  symbol: string;
  side: 'buy' | 'sell';
  type: 'market' | 'limit';
  quantity: number;
  price?: number;
}

// Market types
export interface Quote {
  symbol: string;
  name: string;
  price: number;
  change: number;
  change_percent: number;
  volume: number;
  high: number;
  low: number;
  open: number;
  previous_close: number;
}

export interface SearchResult {
  symbol: string;
  name: string;
  exchange: string;
  type: string;
}

// API response wrapper
export interface ApiResponse<T> {
  data?: T;
  error?: {
    code: string;
    message: string;
    details?: string[];
  };
  meta?: {
    request_id: string;
    timestamp: string;
  };
}
