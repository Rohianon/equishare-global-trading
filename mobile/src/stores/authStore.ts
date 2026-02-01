import { create } from 'zustand';
import apiClient from '../api/client';
import authApi from '../api/auth';
import type { User } from '../types';

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  needsPhone: boolean;
  needsUsername: boolean;
  error: string | null;

  // Actions
  initialize: () => Promise<void>;
  login: (phone: string, pin: string) => Promise<void>;
  register: (phone: string) => Promise<{ expires_in: number }>;
  verify: (phone: string, otp: string, pin: string) => Promise<void>;
  logout: () => Promise<void>;
  setUser: (user: User) => void;
  setNeedsPhone: (needs: boolean) => void;
  setNeedsUsername: (needs: boolean) => void;
  clearError: () => void;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  isAuthenticated: false,
  isLoading: true,
  needsPhone: false,
  needsUsername: false,
  error: null,

  initialize: async () => {
    try {
      const hasTokens = await apiClient.hasTokens();
      if (hasTokens) {
        const user = await authApi.getMe();
        set({
          user,
          isAuthenticated: true,
          isLoading: false,
          needsPhone: !user.phone_verified,
          needsUsername: !user.username,
        });
      } else {
        set({ isLoading: false });
      }
    } catch (error) {
      await apiClient.clearTokens();
      set({ isLoading: false, isAuthenticated: false, user: null });
    }
  },

  login: async (phone: string, pin: string) => {
    set({ isLoading: true, error: null });
    try {
      const response = await authApi.login({ phone, pin });
      set({
        user: response.user,
        isAuthenticated: true,
        isLoading: false,
        needsPhone: !response.user.phone_verified,
        needsUsername: !response.user.username,
      });
    } catch (error: any) {
      const message = error.response?.data?.error || 'Login failed';
      set({ error: message, isLoading: false });
      throw error;
    }
  },

  register: async (phone: string) => {
    set({ isLoading: true, error: null });
    try {
      const response = await authApi.register({ phone });
      set({ isLoading: false });
      return response;
    } catch (error: any) {
      const message = error.response?.data?.error || 'Registration failed';
      set({ error: message, isLoading: false });
      throw error;
    }
  },

  verify: async (phone: string, otp: string, pin: string) => {
    set({ isLoading: true, error: null });
    try {
      const response = await authApi.verify({ phone, otp, pin });
      set({
        user: response.user,
        isAuthenticated: true,
        isLoading: false,
        needsPhone: false,
        needsUsername: !response.user.username,
      });
    } catch (error: any) {
      const message = error.response?.data?.error || 'Verification failed';
      set({ error: message, isLoading: false });
      throw error;
    }
  },

  logout: async () => {
    await authApi.logout();
    set({
      user: null,
      isAuthenticated: false,
      needsPhone: false,
      needsUsername: false,
      error: null,
    });
  },

  setUser: (user: User) => set({ user }),

  setNeedsPhone: (needs: boolean) => set({ needsPhone: needs }),

  setNeedsUsername: (needs: boolean) => set({ needsUsername: needs }),

  clearError: () => set({ error: null }),
}));

export default useAuthStore;
