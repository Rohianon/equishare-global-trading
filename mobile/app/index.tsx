import { Redirect } from 'expo-router';
import { useAuthStore } from '../src/stores/authStore';

export default function Index() {
  const { isAuthenticated, needsUsername, needsPhone } = useAuthStore();

  if (!isAuthenticated) {
    return <Redirect href="/(auth)/welcome" />;
  }

  if (needsUsername) {
    return <Redirect href="/(auth)/set-username" />;
  }

  if (needsPhone) {
    return <Redirect href="/(auth)/link-phone" />;
  }

  return <Redirect href="/(tabs)" />;
}
