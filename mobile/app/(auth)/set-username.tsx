import { useState, useCallback } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TextInput,
  TouchableOpacity,
  KeyboardAvoidingView,
  Platform,
  Alert,
} from 'react-native';
import { router } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useAuthStore } from '../../src/stores/authStore';
import authApi from '../../src/api/auth';
import { useDebouncedCallback } from '../../src/hooks/useDebounce';

export default function SetUsernameScreen() {
  const [username, setUsername] = useState('');
  const [isChecking, setIsChecking] = useState(false);
  const [isAvailable, setIsAvailable] = useState<boolean | null>(null);
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { setNeedsUsername, needsPhone } = useAuthStore();

  const checkUsername = useDebouncedCallback(async (value: string) => {
    if (value.length < 3) {
      setIsAvailable(null);
      setSuggestions([]);
      return;
    }

    setIsChecking(true);
    try {
      const result = await authApi.checkUsername(value);
      setIsAvailable(result.available);
      setSuggestions(result.suggestions || []);
    } catch (error) {
      console.error('Username check error:', error);
    } finally {
      setIsChecking(false);
    }
  }, 500);

  const handleUsernameChange = (value: string) => {
    const cleaned = value.toLowerCase().replace(/[^a-z0-9_]/g, '');
    setUsername(cleaned);
    setIsAvailable(null);
    checkUsername(cleaned);
  };

  const handleSubmit = async () => {
    if (!username || username.length < 3) {
      Alert.alert('Error', 'Username must be at least 3 characters');
      return;
    }

    if (isAvailable === false) {
      Alert.alert('Error', 'This username is not available');
      return;
    }

    setIsSubmitting(true);
    try {
      await authApi.setUsername(username);
      setNeedsUsername(false);

      if (needsPhone) {
        router.replace('/(auth)/link-phone');
      } else {
        router.replace('/(tabs)');
      }
    } catch (error) {
      Alert.alert('Error', 'Failed to set username. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  };

  const selectSuggestion = (suggestion: string) => {
    setUsername(suggestion);
    setIsAvailable(true);
    setSuggestions([]);
  };

  return (
    <SafeAreaView style={styles.container}>
      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        style={styles.keyboardView}
      >
        <View style={styles.header}>
          <Text style={styles.title}>Choose a username</Text>
          <Text style={styles.subtitle}>
            This will be your unique identifier on EquiShare
          </Text>
        </View>

        <View style={styles.form}>
          <Text style={styles.label}>Username</Text>
          <View style={styles.inputContainer}>
            <Text style={styles.inputPrefix}>@</Text>
            <TextInput
              style={styles.input}
              value={username}
              onChangeText={handleUsernameChange}
              placeholder="username"
              autoCapitalize="none"
              autoCorrect={false}
              maxLength={30}
              autoFocus
            />
            {isChecking && <Text style={styles.checkingText}>...</Text>}
            {!isChecking && isAvailable === true && (
              <Text style={styles.availableIcon}>✓</Text>
            )}
            {!isChecking && isAvailable === false && (
              <Text style={styles.unavailableIcon}>✗</Text>
            )}
          </View>

          {isAvailable === false && suggestions.length > 0 && (
            <View style={styles.suggestionsContainer}>
              <Text style={styles.suggestionsLabel}>Try these instead:</Text>
              <View style={styles.suggestionsList}>
                {suggestions.map((suggestion) => (
                  <TouchableOpacity
                    key={suggestion}
                    style={styles.suggestionChip}
                    onPress={() => selectSuggestion(suggestion)}
                  >
                    <Text style={styles.suggestionText}>@{suggestion}</Text>
                  </TouchableOpacity>
                ))}
              </View>
            </View>
          )}

          <Text style={styles.hint}>
            3-30 characters. Letters, numbers, and underscores only.
          </Text>

          <TouchableOpacity
            style={[
              styles.button,
              (!isAvailable || isSubmitting) && styles.buttonDisabled,
            ]}
            onPress={handleSubmit}
            disabled={!isAvailable || isSubmitting}
          >
            <Text style={styles.buttonText}>
              {isSubmitting ? 'Setting username...' : 'Continue'}
            </Text>
          </TouchableOpacity>
        </View>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#fff',
  },
  keyboardView: {
    flex: 1,
    padding: 24,
  },
  header: {
    marginTop: 40,
    marginBottom: 40,
  },
  title: {
    fontSize: 28,
    fontWeight: 'bold',
    color: '#111827',
    marginBottom: 12,
  },
  subtitle: {
    fontSize: 16,
    color: '#6B7280',
    lineHeight: 24,
  },
  form: {
    flex: 1,
  },
  label: {
    fontSize: 14,
    fontWeight: '500',
    color: '#374151',
    marginBottom: 8,
  },
  inputContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#F9FAFB',
    borderWidth: 1,
    borderColor: '#E5E7EB',
    borderRadius: 12,
    paddingHorizontal: 16,
  },
  inputPrefix: {
    fontSize: 18,
    color: '#9CA3AF',
    marginRight: 4,
  },
  input: {
    flex: 1,
    fontSize: 18,
    paddingVertical: 16,
    color: '#111827',
  },
  checkingText: {
    fontSize: 16,
    color: '#9CA3AF',
  },
  availableIcon: {
    fontSize: 18,
    color: '#10B981',
    fontWeight: 'bold',
  },
  unavailableIcon: {
    fontSize: 18,
    color: '#EF4444',
    fontWeight: 'bold',
  },
  suggestionsContainer: {
    marginTop: 16,
  },
  suggestionsLabel: {
    fontSize: 14,
    color: '#6B7280',
    marginBottom: 8,
  },
  suggestionsList: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 8,
  },
  suggestionChip: {
    backgroundColor: '#F3F4F6',
    paddingHorizontal: 12,
    paddingVertical: 8,
    borderRadius: 20,
  },
  suggestionText: {
    fontSize: 14,
    color: '#374151',
  },
  hint: {
    fontSize: 14,
    color: '#9CA3AF',
    marginTop: 12,
    marginBottom: 24,
  },
  button: {
    backgroundColor: '#10B981',
    borderRadius: 12,
    padding: 16,
    alignItems: 'center',
  },
  buttonDisabled: {
    opacity: 0.5,
  },
  buttonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
});
