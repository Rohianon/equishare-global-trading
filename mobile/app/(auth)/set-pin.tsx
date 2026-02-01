import { useState, useRef } from 'react';
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
import { router, useLocalSearchParams } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useAuthStore } from '../../src/stores/authStore';

export default function SetPinScreen() {
  const { phone, otp } = useLocalSearchParams<{ phone: string; otp: string }>();
  const [step, setStep] = useState<'create' | 'confirm'>('create');
  const [pin, setPin] = useState(['', '', '', '']);
  const [confirmPin, setConfirmPin] = useState(['', '', '', '']);
  const inputRefs = useRef<TextInput[]>([]);
  const { verify, isLoading, error, clearError } = useAuthStore();

  const currentPin = step === 'create' ? pin : confirmPin;
  const setCurrentPin = step === 'create' ? setPin : setConfirmPin;

  const handlePinChange = (text: string, index: number) => {
    const newPin = [...currentPin];
    newPin[index] = text;
    setCurrentPin(newPin);

    if (text && index < 3) {
      inputRefs.current[index + 1]?.focus();
    }

    if (newPin.every((digit) => digit) && newPin.join('').length === 4) {
      if (step === 'create') {
        setTimeout(() => {
          setStep('confirm');
          inputRefs.current[0]?.focus();
        }, 200);
      } else {
        handleConfirm(newPin.join(''));
      }
    }
  };

  const handleKeyPress = (e: any, index: number) => {
    if (e.nativeEvent.key === 'Backspace' && !currentPin[index] && index > 0) {
      inputRefs.current[index - 1]?.focus();
    }
  };

  const handleConfirm = async (confirmedPin: string) => {
    if (pin.join('') !== confirmedPin) {
      Alert.alert('PINs don\'t match', 'Please try again');
      setPin(['', '', '', '']);
      setConfirmPin(['', '', '', '']);
      setStep('create');
      inputRefs.current[0]?.focus();
      return;
    }

    try {
      await verify(phone, otp, confirmedPin);
      router.replace('/(tabs)');
    } catch (err) {
      Alert.alert('Verification Failed', error || 'Please try again');
      clearError();
    }
  };

  return (
    <SafeAreaView style={styles.container}>
      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        style={styles.keyboardView}
      >
        <TouchableOpacity
          style={styles.backButton}
          onPress={() => {
            if (step === 'confirm') {
              setStep('create');
              setConfirmPin(['', '', '', '']);
            } else {
              router.back();
            }
          }}
        >
          <Text style={styles.backButtonText}>← Back</Text>
        </TouchableOpacity>

        <View style={styles.header}>
          <Text style={styles.title}>
            {step === 'create' ? 'Create your PIN' : 'Confirm your PIN'}
          </Text>
          <Text style={styles.subtitle}>
            {step === 'create'
              ? 'Choose a 4-digit PIN to secure your account'
              : 'Enter your PIN again to confirm'}
          </Text>
        </View>

        <View style={styles.pinContainer}>
          {currentPin.map((digit, index) => (
            <TextInput
              key={`${step}-${index}`}
              ref={(ref) => { if (ref) inputRefs.current[index] = ref; }}
              style={[styles.pinInput, digit && styles.pinInputFilled]}
              value={digit ? '•' : ''}
              onChangeText={(text) => handlePinChange(text.slice(-1), index)}
              onKeyPress={(e) => handleKeyPress(e, index)}
              keyboardType="number-pad"
              maxLength={1}
              secureTextEntry
              selectTextOnFocus
              autoFocus={index === 0}
            />
          ))}
        </View>

        <View style={styles.stepIndicator}>
          <View style={[styles.stepDot, step === 'create' && styles.stepDotActive]} />
          <View style={[styles.stepDot, step === 'confirm' && styles.stepDotActive]} />
        </View>

        {isLoading && (
          <Text style={styles.loadingText}>Setting up your account...</Text>
        )}
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
  backButton: {
    marginBottom: 24,
  },
  backButtonText: {
    fontSize: 16,
    color: '#10B981',
    fontWeight: '500',
  },
  header: {
    marginBottom: 48,
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
  pinContainer: {
    flexDirection: 'row',
    justifyContent: 'center',
    gap: 16,
    marginBottom: 32,
  },
  pinInput: {
    width: 56,
    height: 64,
    borderWidth: 2,
    borderColor: '#E5E7EB',
    borderRadius: 12,
    fontSize: 32,
    fontWeight: '600',
    textAlign: 'center',
    color: '#111827',
  },
  pinInputFilled: {
    borderColor: '#10B981',
    backgroundColor: '#ECFDF5',
  },
  stepIndicator: {
    flexDirection: 'row',
    justifyContent: 'center',
    gap: 8,
  },
  stepDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    backgroundColor: '#E5E7EB',
  },
  stepDotActive: {
    backgroundColor: '#10B981',
  },
  loadingText: {
    textAlign: 'center',
    color: '#6B7280',
    marginTop: 24,
  },
});
