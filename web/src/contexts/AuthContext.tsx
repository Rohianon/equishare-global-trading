import { createContext, useContext, useState, useEffect, type ReactNode } from 'react';
import apiClient, { type User, type AuthResponse } from '../api/client';

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (phone: string, pin: string) => Promise<void>;
  register: (phone: string, fullName?: string) => Promise<{ expiresIn: number }>;
  verify: (phone: string, otp: string, pin: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // Check for existing auth on mount
    const storedUser = localStorage.getItem('user');
    if (storedUser && apiClient.isAuthenticated()) {
      setUser(JSON.parse(storedUser));
    }
    setIsLoading(false);
  }, []);

  const handleAuthResponse = (response: AuthResponse) => {
    apiClient.setTokens(response.access_token, response.refresh_token);
    setUser(response.user);
    localStorage.setItem('user', JSON.stringify(response.user));
  };

  const login = async (phone: string, pin: string) => {
    const response = await apiClient.login(phone, pin);
    handleAuthResponse(response);
  };

  const register = async (phone: string, fullName?: string) => {
    const response = await apiClient.register(phone, fullName);
    return { expiresIn: response.expires_in };
  };

  const verify = async (phone: string, otp: string, pin: string) => {
    const response = await apiClient.verify(phone, otp, pin);
    handleAuthResponse(response);
  };

  const logout = () => {
    apiClient.clearTokens();
    setUser(null);
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated: !!user,
        isLoading,
        login,
        register,
        verify,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
