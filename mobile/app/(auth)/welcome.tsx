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
  // Use URL-safe base64 encoding
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~';
  return Array.from(bytes).map(b => chars[b % chars.length]).join('');
};

export default function WelcomeScreen() {
  const { setUser, setNeedsPhone, setNeedsUsername } = useAuthStore();

  const handleGoogleLogin = async () => {
    try {
      // Generate PKCE verifier
      const codeVerifier = await generateCodeVerifier(64);
      const redirectUri = AuthSession.makeRedirectUri({ scheme: 'equishare' });

      // Get auth URL from backend
      const { authorization_url, state } = await authApi.initiateOAuth(
        'google',
        redirectUri,
        codeVerifier
      );

      // Open browser for OAuth
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
      <View style={styles.header}>
        <View style={styles.logoContainer}>
          <Text style={styles.logoText}>📈</Text>
        </View>
        <Text style={styles.title}>EquiShare</Text>
        <Text style={styles.subtitle}>Invest in global markets from Kenya</Text>
      </View>

      <View style={styles.content}>
        <Text style={styles.sectionTitle}>Get started</Text>

        <TouchableOpacity style={styles.socialButton} onPress={handleGoogleLogin}>
          <Text style={styles.socialIcon}>G</Text>
          <Text style={styles.socialButtonText}>Continue with Google</Text>
        </TouchableOpacity>

        {Platform.OS === 'ios' && (
          <TouchableOpacity
            style={[styles.socialButton, styles.appleButton]}
            onPress={handleAppleLogin}
          >
            <Text style={[styles.socialIcon, styles.appleIcon]}></Text>
            <Text style={[styles.socialButtonText, styles.appleButtonText]}>
              Continue with Apple
            </Text>
          </TouchableOpacity>
        )}

        <View style={styles.divider}>
          <View style={styles.dividerLine} />
          <Text style={styles.dividerText}>or</Text>
          <View style={styles.dividerLine} />
        </View>

        <Link href="/(auth)/register" asChild>
          <TouchableOpacity style={styles.phoneButton}>
            <Text style={styles.phoneIcon}>📱</Text>
            <Text style={styles.phoneButtonText}>Continue with Phone</Text>
          </TouchableOpacity>
        </Link>
      </View>

      <View style={styles.footer}>
        <Text style={styles.footerText}>Already have an account?</Text>
        <Link href="/(auth)/login" asChild>
          <TouchableOpacity>
            <Text style={styles.loginLink}>Log in</Text>
          </TouchableOpacity>
        </Link>
      </View>

      <Text style={styles.disclaimer}>
        By continuing, you agree to our Terms of Service and Privacy Policy
      </Text>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#fff',
    padding: 24,
  },
  header: {
    alignItems: 'center',
    marginTop: 40,
    marginBottom: 48,
  },
  logoContainer: {
    width: 80,
    height: 80,
    borderRadius: 20,
    backgroundColor: '#10B981',
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 16,
  },
  logoText: {
    fontSize: 40,
  },
  title: {
    fontSize: 32,
    fontWeight: 'bold',
    color: '#111827',
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 16,
    color: '#6B7280',
    textAlign: 'center',
  },
  content: {
    flex: 1,
  },
  sectionTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#111827',
    marginBottom: 16,
  },
  socialButton: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: '#E5E7EB',
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
  },
  appleButton: {
    backgroundColor: '#000',
    borderColor: '#000',
  },
  socialIcon: {
    fontSize: 20,
    fontWeight: 'bold',
    marginRight: 12,
    color: '#EA4335',
  },
  appleIcon: {
    color: '#fff',
  },
  socialButtonText: {
    fontSize: 16,
    fontWeight: '600',
    color: '#111827',
  },
  appleButtonText: {
    color: '#fff',
  },
  divider: {
    flexDirection: 'row',
    alignItems: 'center',
    marginVertical: 24,
  },
  dividerLine: {
    flex: 1,
    height: 1,
    backgroundColor: '#E5E7EB',
  },
  dividerText: {
    marginHorizontal: 16,
    color: '#9CA3AF',
    fontSize: 14,
  },
  phoneButton: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: '#10B981',
    borderRadius: 12,
    padding: 16,
  },
  phoneIcon: {
    fontSize: 20,
    marginRight: 12,
  },
  phoneButtonText: {
    fontSize: 16,
    fontWeight: '600',
    color: '#fff',
  },
  footer: {
    flexDirection: 'row',
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 16,
  },
  footerText: {
    color: '#6B7280',
    fontSize: 14,
  },
  loginLink: {
    color: '#10B981',
    fontSize: 14,
    fontWeight: '600',
    marginLeft: 4,
  },
  disclaimer: {
    textAlign: 'center',
    color: '#9CA3AF',
    fontSize: 12,
    lineHeight: 18,
  },
});
