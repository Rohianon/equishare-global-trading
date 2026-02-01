import axios, { type AxiosInstance, type AxiosError } from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export interface ApiError {
  error: string;
  message?: string;
  code?: string;
}

export interface User {
  id: string;
  phone: string;
  full_name: string;
  email?: string;
  kyc_status?: string;
}

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: User;
}

export interface RegisterResponse {
  message: string;
  expires_in: number;
}

export interface WalletBalance {
  currency: string;
  available: number;
  pending: number;
  total: number;
}

export interface Transaction {
  id: string;
  type: string;
  amount: number;
  currency: string;
  status: string;
  description: string;
  created_at: string;
}

export interface TransactionsResponse {
  transactions: Transaction[];
  total: number;
  page: number;
  per_page: number;
}

export interface DepositResponse {
  transaction_id: string;
  checkout_request_id: string;
  status: string;
  message: string;
}

class ApiClient {
  private client: AxiosInstance;
  private accessToken: string | null = null;
  private refreshToken: string | null = null;

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Load tokens from localStorage
    this.accessToken = localStorage.getItem('access_token');
    this.refreshToken = localStorage.getItem('refresh_token');

    // Request interceptor to add auth header
    this.client.interceptors.request.use((config) => {
      if (this.accessToken) {
        config.headers.Authorization = `Bearer ${this.accessToken}`;
      }
      return config;
    });

    // Response interceptor to handle token refresh
    this.client.interceptors.response.use(
      (response) => response,
      async (error: AxiosError<ApiError>) => {
        const originalRequest = error.config;

        if (error.response?.status === 401 && this.refreshToken && originalRequest) {
          try {
            const response = await this.refresh();
            this.setTokens(response.access_token, response.refresh_token);
            originalRequest.headers.Authorization = `Bearer ${response.access_token}`;
            return this.client(originalRequest);
          } catch {
            this.clearTokens();
            window.location.href = '/login';
          }
        }

        return Promise.reject(error);
      }
    );
  }

  setTokens(accessToken: string, refreshToken: string) {
    this.accessToken = accessToken;
    this.refreshToken = refreshToken;
    localStorage.setItem('access_token', accessToken);
    localStorage.setItem('refresh_token', refreshToken);
  }

  clearTokens() {
    this.accessToken = null;
    this.refreshToken = null;
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('user');
  }

  isAuthenticated(): boolean {
    return !!this.accessToken;
  }

  // Auth endpoints
  async register(phone: string, fullName?: string): Promise<RegisterResponse> {
    const { data } = await this.client.post<RegisterResponse>('/api/v1/auth/register', {
      phone,
      full_name: fullName,
    });
    return data;
  }

  async verify(phone: string, otp: string, pin: string): Promise<AuthResponse> {
    const { data } = await this.client.post<AuthResponse>('/api/v1/auth/verify', {
      phone,
      otp,
      pin,
    });
    return data;
  }

  async login(phone: string, pin: string): Promise<AuthResponse> {
    const { data } = await this.client.post<AuthResponse>('/api/v1/auth/login', {
      phone,
      pin,
    });
    return data;
  }

  async refresh(): Promise<AuthResponse> {
    const { data } = await this.client.post<AuthResponse>('/api/v1/auth/refresh', {
      refresh_token: this.refreshToken,
    });
    return data;
  }

  async getMe(): Promise<User> {
    const { data } = await this.client.get<User>('/api/v1/auth/me');
    return data;
  }

  // Wallet endpoints
  async getWalletBalance(): Promise<WalletBalance> {
    const { data } = await this.client.get<WalletBalance>('/api/v1/payments/wallet/balance');
    return data;
  }

  async initiateDeposit(amount: number, phoneNumber: string): Promise<DepositResponse> {
    const { data } = await this.client.post<DepositResponse>('/api/v1/payments/deposit', {
      amount,
      phone_number: phoneNumber,
    });
    return data;
  }

  async getTransactions(page = 1, perPage = 10): Promise<TransactionsResponse> {
    const { data } = await this.client.get<TransactionsResponse>(
      `/api/v1/payments/transactions?page=${page}&per_page=${perPage}`
    );
    return data;
  }
}

export const apiClient = new ApiClient();
export default apiClient;
