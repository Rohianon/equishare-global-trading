import { View, Text, StyleSheet, TouchableOpacity, Platform } from 'react-native';
import { Link, router } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import * as WebBrowser from 'expo-web-browser';
import * as AuthSession from 'expo-auth-session';
import * as Crypto from 'expo-crypto';
import { useAuthStore } from '../../src/stores/authStore';
import authApi from '../../src/api/auth';

WebBrowser.maybeCompleteAuthSession();

// Generate a cryptographically random string for PKCE
const generateCodeVerifier = async (length: number = 64): Promise<string> => {
  const bytes = await Crypto.getRandomBytesAsync(length);
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~';
  return Array.from(bytes).map(b => chars[b % chars.length]).join('');
};

export default function WelcomeScreen() {
  const { setUser, setNeedsPhone, setNeedsUsername } = useAuthStore();

  const handleGoogleLogin = async () => {
    try {
      const codeVerifier = await generateCodeVerifier(64);
      const redirectUri = AuthSession.makeRedirectUri({ scheme: 'equishare' });

      const { authorization_url, state } = await authApi.initiateOAuth(
        'google',
        redirectUri,
        codeVerifier
      );

      const result = await WebBrowser.openAuthSessionAsync(authorization_url, redirectUri);

      if (result.type === 'success' && result.url) {
        const url = new URL(result.url);
        const code = url.searchParams.get('code');
        const returnedState = url.searchParams.get('state');

        if (code && returnedState === state) {
          const response = await authApi.handleOAuthCallback(
            'google',
            code,
            state,
            codeVerifier
          );

          setUser(response.user);
          setNeedsPhone(response.needs_phone);
          setNeedsUsername(response.needs_username);

          if (response.needs_username) {
            router.replace('/(auth)/set-username');
          } else if (response.needs_phone) {
            router.replace('/(auth)/link-phone');
          } else {
            router.replace('/(tabs)');
          }
        }
      }
    } catch (error) {
      console.error('Google login error:', error);
    }
  };

  const handleAppleLogin = async () => {
    try {
      const codeVerifier = await generateCodeVerifier(64);
      const redirectUri = AuthSession.makeRedirectUri({ scheme: 'equishare' });

      const { authorization_url, state } = await authApi.initiateOAuth(
        'apple',
        redirectUri,
        codeVerifier
      );

      const result = await WebBrowser.openAuthSessionAsync(authorization_url, redirectUri);

      if (result.type === 'success' && result.url) {
        const url = new URL(result.url);
        const code = url.searchParams.get('code');
        const returnedState = url.searchParams.get('state');

        if (code && returnedState === state) {
          const response = await authApi.handleOAuthCallback(
            'apple',
            code,
            state,
            codeVerifier
          );

          setUser(response.user);
          setNeedsPhone(response.needs_phone);
          setNeedsUsername(response.needs_username);

          if (response.needs_username) {
            router.replace('/(auth)/set-username');
          } else if (response.needs_phone) {
            router.replace('/(auth)/link-phone');
          } else {
            router.replace('/(tabs)');
          }
        }
      }
    } catch (error) {
      console.error('Apple login error:', error);
    }
  };

  return (
    <SafeAreaView style={styles.container}>
      {/* Hero Section */}
      <View style={styles.hero}>
        <Text style={styles.title}>Welcome to EquiShare</Text>
        <Text style={styles.subtitle}>
          Trade equities globally with seamless M-Pesa integration.
          Start investing in US stocks directly from Kenya.
        </Text>
      </View>

      {/* Auth Buttons */}
      <View style={styles.authSection}>
        <Link href="/(auth)/register" asChild>
          <TouchableOpacity style={styles.primaryButton} activeOpacity={0.8}>
            <Text style={styles.primaryButtonText}>Get Started</Text>
          </TouchableOpacity>
        </Link>

        <Link href="/(auth)/login" asChild>
          <TouchableOpacity style={styles.secondaryButton} activeOpacity={0.8}>
            <Text style={styles.secondaryButtonText}>Login</Text>
          </TouchableOpacity>
        </Link>
      </View>

      {/* Feature Cards */}
      <View style={styles.features}>
        <View style={styles.card}>
          <Text style={styles.cardNumber}>1</Text>
          <Text style={styles.cardTitle}>Register</Text>
          <Text style={styles.cardText}>
            Sign up with your phone number and verify via SMS OTP
          </Text>
        </View>

        <View style={styles.card}>
          <Text style={styles.cardNumber}>2</Text>
          <Text style={styles.cardTitle}>Fund Account</Text>
          <Text style={styles.cardText}>
            Deposit funds instantly using M-Pesa STK Push
          </Text>
        </View>

        <View style={styles.card}>
          <Text style={styles.cardNumber}>3</Text>
          <Text style={styles.cardTitle}>Trade</Text>
          <Text style={styles.cardText}>
            Buy and sell fractional shares of US equities
          </Text>
        </View>
      </View>

      {/* Social Login Section */}
      <View style={styles.socialSection}>
        <View style={styles.divider}>
          <View style={styles.dividerLine} />
          <Text style={styles.dividerText}>or continue with</Text>
          <View style={styles.dividerLine} />
        </View>

        <View style={styles.socialButtons}>
          <TouchableOpacity
            style={styles.socialButton}
            onPress={handleGoogleLogin}
            activeOpacity={0.8}
          >
            <Text style={styles.socialIcon}>G</Text>
          </TouchableOpacity>

          {Platform.OS === 'ios' && (
            <TouchableOpacity
              style={[styles.socialButton, styles.appleButton]}
              onPress={handleAppleLogin}
              activeOpacity={0.8}
            >
              <Text style={styles.appleIcon}></Text>
            </TouchableOpacity>
          )}
        </View>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f9fafb',
    paddingHorizontal: 24,
  },
  hero: {
    alignItems: 'center',
    paddingTop: 48,
    paddingBottom: 32,
  },
  title: {
    fontSize: 28,
    fontWeight: '700',
    color: '#111827',
    textAlign: 'center',
    marginBottom: 16,
  },
  subtitle: {
    fontSize: 16,
    color: '#4b5563',
    textAlign: 'center',
    lineHeight: 24,
    maxWidth: 320,
  },
  authSection: {
    flexDirection: 'row',
    justifyContent: 'center',
    gap: 12,
    marginBottom: 32,
  },
  primaryButton: {
    backgroundColor: '#0284c7',
    paddingVertical: 14,
    paddingHorizontal: 28,
    borderRadius: 10,
  },
  primaryButtonText: {
    color: '#ffffff',
    fontSize: 16,
    fontWeight: '600',
  },
  secondaryButton: {
    backgroundColor: '#f3f4f6',
    paddingVertical: 14,
    paddingHorizontal: 28,
    borderRadius: 10,
  },
  secondaryButtonText: {
    color: '#374151',
    fontSize: 16,
    fontWeight: '600',
  },
  features: {
    gap: 12,
    marginBottom: 32,
  },
  card: {
    backgroundColor: '#ffffff',
    borderRadius: 12,
    padding: 20,
    borderWidth: 1,
    borderColor: '#f3f4f6',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.05,
    shadowRadius: 2,
    elevation: 1,
  },
  cardNumber: {
    fontSize: 24,
    fontWeight: '600',
    color: '#0284c7',
    marginBottom: 8,
  },
  cardTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: '#111827',
    marginBottom: 4,
  },
  cardText: {
    fontSize: 14,
    color: '#4b5563',
    lineHeight: 20,
  },
  socialSection: {
    marginTop: 'auto',
    paddingBottom: 24,
  },
  divider: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 20,
  },
  dividerLine: {
    flex: 1,
    height: 1,
    backgroundColor: '#e5e7eb',
  },
  dividerText: {
    marginHorizontal: 16,
    color: '#9ca3af',
    fontSize: 13,
  },
  socialButtons: {
    flexDirection: 'row',
    justifyContent: 'center',
    gap: 16,
  },
  socialButton: {
    width: 56,
    height: 56,
    borderRadius: 12,
    backgroundColor: '#ffffff',
    borderWidth: 1,
    borderColor: '#e5e7eb',
    justifyContent: 'center',
    alignItems: 'center',
  },
  socialIcon: {
    fontSize: 20,
    fontWeight: '700',
    color: '#ea4335',
  },
  appleButton: {
    backgroundColor: '#000000',
    borderColor: '#000000',
  },
  appleIcon: {
    fontSize: 22,
    color: '#ffffff',
  },
});
