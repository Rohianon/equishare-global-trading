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
import { router } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useAuthStore } from '../../src/stores/authStore';
import authApi from '../../src/api/auth';

type Step = 'phone' | 'otp' | 'pin';

export default function LinkPhoneScreen() {
  const [step, setStep] = useState<Step>('phone');
  const [phone, setPhone] = useState('+254');
  const [otp, setOtp] = useState(['', '', '', '', '', '']);
  const [pin, setPin] = useState(['', '', '', '']);
  const [confirmPin, setConfirmPin] = useState(['', '', '', '']);
  const [pinStep, setPinStep] = useState<'create' | 'confirm'>('create');
  const [isLoading, setIsLoading] = useState(false);
  const [timeLeft, setTimeLeft] = useState(300);
  const otpRefs = useRef<TextInput[]>([]);
  const pinRefs = useRef<TextInput[]>([]);
  const { setNeedsPhone } = useAuthStore();

  const handleSendOtp = async () => {
    if (phone.length < 13) {
      Alert.alert('Error', 'Please enter a valid Kenyan phone number');
      return;
    }

    setIsLoading(true);
    try {
      await authApi.linkPhone(phone);
      setStep('otp');
      setTimeLeft(300);
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.error || 'Failed to send OTP');
    } finally {
      setIsLoading(false);
    }
  };

  const handleOtpChange = (text: string, index: number) => {
    const newOtp = [...otp];
    newOtp[index] = text;
    setOtp(newOtp);

    if (text && index < 5) {
      otpRefs.current[index + 1]?.focus();
    }

    if (newOtp.every((d) => d) && newOtp.join('').length === 6) {
      setStep('pin');
    }
  };

  const handlePinChange = (text: string, index: number) => {
    const currentPin = pinStep === 'create' ? pin : confirmPin;
    const setCurrentPin = pinStep === 'create' ? setPin : setConfirmPin;

    const newPin = [...currentPin];
    newPin[index] = text;
    setCurrentPin(newPin);

    if (text && index < 3) {
      pinRefs.current[index + 1]?.focus();
    }

    if (newPin.every((d) => d) && newPin.join('').length === 4) {
      if (pinStep === 'create') {
        setPinStep('confirm');
        setTimeout(() => pinRefs.current[0]?.focus(), 200);
      } else {
        handleVerify(newPin.join(''));
      }
    }
  };

  const handleVerify = async (confirmedPin: string) => {
    if (pin.join('') !== confirmedPin) {
      Alert.alert('PINs don\'t match', 'Please try again');
      setPin(['', '', '', '']);
      setConfirmPin(['', '', '', '']);
      setPinStep('create');
      pinRefs.current[0]?.focus();
      return;
    }

    setIsLoading(true);
    try {
      await authApi.verifyPhoneLink(phone, otp.join(''), confirmedPin);
      setNeedsPhone(false);
      Alert.alert('Success', 'Phone linked successfully! M-Pesa is now available.', [
        { text: 'Continue', onPress: () => router.replace('/(tabs)') },
      ]);
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.error || 'Verification failed');
    } finally {
      setIsLoading(false);
    }
  };

  const handleSkip = () => {
    Alert.alert(
      'Skip phone linking?',
      'You won\'t be able to deposit or withdraw via M-Pesa until you link a phone number.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Skip for now',
          onPress: () => router.replace('/(tabs)'),
        },
      ]
    );
  };

  return (
    <SafeAreaView style={styles.container}>
      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        style={styles.keyboardView}
      >
        <TouchableOpacity style={styles.skipButton} onPress={handleSkip}>
          <Text style={styles.skipButtonText}>Skip for now</Text>
        </TouchableOpacity>

        <View style={styles.header}>
          <Text style={styles.title}>
            {step === 'phone' && 'Link your phone'}
            {step === 'otp' && 'Verify your number'}
            {step === 'pin' && (pinStep === 'create' ? 'Create PIN' : 'Confirm PIN')}
          </Text>
          <Text style={styles.subtitle}>
            {step === 'phone' && 'Add your phone number to enable M-Pesa deposits and withdrawals'}
            {step === 'otp' && `Enter the 6-digit code sent to ${phone}`}
            {step === 'pin' && 'Set a 4-digit PIN for secure transactions'}
          </Text>
        </View>

        {step === 'phone' && (
          <View style={styles.form}>
            <Text style={styles.label}>Phone Number</Text>
            <TextInput
              style={styles.input}
              value={phone}
              onChangeText={setPhone}
              placeholder="+254712345678"
              keyboardType="phone-pad"
              maxLength={13}
              autoFocus
            />

            <TouchableOpacity
              style={[styles.button, isLoading && styles.buttonDisabled]}
              onPress={handleSendOtp}
              disabled={isLoading}
            >
              <Text style={styles.buttonText}>
                {isLoading ? 'Sending...' : 'Send verification code'}
              </Text>
            </TouchableOpacity>
          </View>
        )}

        {step === 'otp' && (
          <View style={styles.otpContainer}>
            {otp.map((digit, index) => (
              <TextInput
                key={index}
                ref={(ref) => { if (ref) otpRefs.current[index] = ref; }}
                style={[styles.otpInput, digit && styles.otpInputFilled]}
                value={digit}
                onChangeText={(text) => handleOtpChange(text.slice(-1), index)}
                keyboardType="number-pad"
                maxLength={1}
                autoFocus={index === 0}
              />
            ))}
          </View>
        )}

        {step === 'pin' && (
          <View style={styles.pinContainer}>
            {(pinStep === 'create' ? pin : confirmPin).map((digit, index) => (
              <TextInput
                key={`${pinStep}-${index}`}
                ref={(ref) => { if (ref) pinRefs.current[index] = ref; }}
                style={[styles.pinInput, digit && styles.pinInputFilled]}
                value={digit ? '•' : ''}
                onChangeText={(text) => handlePinChange(text.slice(-1), index)}
                keyboardType="number-pad"
                maxLength={1}
                secureTextEntry
                autoFocus={index === 0}
              />
            ))}
          </View>
        )}

        {isLoading && step === 'pin' && (
          <Text style={styles.loadingText}>Linking phone number...</Text>
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
  skipButton: {
    alignSelf: 'flex-end',
    marginBottom: 24,
  },
  skipButtonText: {
    fontSize: 14,
    color: '#6B7280',
    fontWeight: '500',
  },
  header: {
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
  input: {
    backgroundColor: '#F9FAFB',
    borderWidth: 1,
    borderColor: '#E5E7EB',
    borderRadius: 12,
    padding: 16,
    fontSize: 18,
    marginBottom: 24,
  },
  button: {
    backgroundColor: '#10B981',
    borderRadius: 12,
    padding: 16,
    alignItems: 'center',
  },
  buttonDisabled: {
    opacity: 0.6,
  },
  buttonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  otpContainer: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 32,
  },
  otpInput: {
    width: 48,
    height: 56,
    borderWidth: 2,
    borderColor: '#E5E7EB',
    borderRadius: 12,
    fontSize: 24,
    fontWeight: '600',
    textAlign: 'center',
  },
  otpInputFilled: {
    borderColor: '#10B981',
    backgroundColor: '#ECFDF5',
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
  },
  pinInputFilled: {
    borderColor: '#10B981',
    backgroundColor: '#ECFDF5',
  },
  loadingText: {
    textAlign: 'center',
    color: '#6B7280',
    marginTop: 24,
  },
});
