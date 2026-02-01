import apiClient from './client';
import type {
  User,
  LoginRequest,
  RegisterRequest,
  VerifyRequest,
  OAuthCallbackResponse,
} from '../types';

interface RegisterResponse {
  message: string;
  expires_in: number;
}

interface AuthResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

interface OAuthInitResponse {
  authorization_url: string;
  state: string;
}

export const authApi = {
  // Phone + PIN authentication
  async register(data: RegisterRequest): Promise<RegisterResponse> {
    return apiClient.post('/auth/register', data);
  },

  async verify(data: VerifyRequest): Promise<AuthResponse> {
    const response = await apiClient.post<AuthResponse>('/auth/verify', data);
    await apiClient.setTokens(response.access_token, response.refresh_token);
    return response;
  },

  async login(data: LoginRequest): Promise<AuthResponse> {
    const response = await apiClient.post<AuthResponse>('/auth/login', data);
    await apiClient.setTokens(response.access_token, response.refresh_token);
    return response;
  },

  async logout(): Promise<void> {
    await apiClient.clearTokens();
  },

  // OAuth authentication
  async initiateOAuth(
    provider: 'google' | 'apple',
    redirectUri: string,
    codeVerifier?: string
  ): Promise<OAuthInitResponse> {
    return apiClient.post(`/auth/oauth/${provider}`, {
      redirect_uri: redirectUri,
      code_verifier: codeVerifier,
    });
  },

  async handleOAuthCallback(
    provider: 'google' | 'apple',
    code: string,
    state: string,
    codeVerifier?: string
  ): Promise<OAuthCallbackResponse> {
    const response = await apiClient.post<OAuthCallbackResponse>(
      `/auth/oauth/${provider}/callback`,
      {
        code,
        state,
        code_verifier: codeVerifier,
      }
    );
    await apiClient.setTokens(response.access_token, response.refresh_token);
    return response;
  },

  // Magic link
  async sendMagicLink(email: string): Promise<{ message: string; expires_in: number }> {
    return apiClient.post('/auth/magic-link', { email });
  },

  async verifyMagicLink(token: string): Promise<OAuthCallbackResponse> {
    const response = await apiClient.post<OAuthCallbackResponse>('/auth/magic-link/verify', {
      token,
    });
    await apiClient.setTokens(response.access_token, response.refresh_token);
    return response;
  },

  // Account linking
  async linkPhone(phone: string): Promise<{ message: string; expires_in: number }> {
    return apiClient.post('/auth/link/phone', { phone });
  },

  async verifyPhoneLink(phone: string, otp: string, pin: string): Promise<{ message: string }> {
    return apiClient.post('/auth/link/phone/verify', { phone, otp, pin });
  },

  // Username
  async checkUsername(username: string): Promise<{ available: boolean; suggestions?: string[] }> {
    return apiClient.get('/auth/username/check', { username });
  },

  async setUsername(username: string): Promise<{ message: string }> {
    return apiClient.post('/auth/username', { username });
  },

  // Current user
  async getMe(): Promise<User> {
    return apiClient.get('/users/me');
  },
};

export default authApi;
