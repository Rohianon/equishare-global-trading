import { useState, useEffect } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  TextInput,
  ActivityIndicator,
  Alert,
  KeyboardAvoidingView,
  Platform,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useLocalSearchParams, router } from 'expo-router';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { tradingApi } from '../../src/api/trading';

export default function StockDetailScreen() {
  const { symbol } = useLocalSearchParams<{ symbol: string }>();
  const queryClient = useQueryClient();
  const [side, setSide] = useState<'buy' | 'sell'>('buy');
  const [amount, setAmount] = useState('');
  const [showOrderForm, setShowOrderForm] = useState(false);

  // Fetch quote
  const { data: quote, isLoading: quoteLoading, refetch: refetchQuote } = useQuery({
    queryKey: ['quote', symbol],
    queryFn: () => tradingApi.getQuote(symbol!),
    enabled: !!symbol,
    refetchInterval: 10000,
  });

  // Fetch asset details
  const { data: asset } = useQuery({
    queryKey: ['asset', symbol],
    queryFn: () => tradingApi.getAsset(symbol!),
    enabled: !!symbol,
  });

  // Fetch wallet balance
  const { data: balance } = useQuery({
    queryKey: ['walletBalance'],
    queryFn: () => tradingApi.getWalletBalance(),
  });

  // Place order mutation
  const placeOrderMutation = useMutation({
    mutationFn: tradingApi.placeOrder,
    onSuccess: (data) => {
      Alert.alert(
        'Order Placed',
        `Your ${side} order for ${symbol} has been submitted.\n\nOrder ID: ${data.order_id}`,
        [
          {
            text: 'View Orders',
            onPress: () => router.push('/(tabs)/orders'),
          },
          {
            text: 'OK',
            style: 'cancel',
          },
        ]
      );
      setAmount('');
      setShowOrderForm(false);
      queryClient.invalidateQueries({ queryKey: ['walletBalance'] });
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      queryClient.invalidateQueries({ queryKey: ['portfolio'] });
    },
    onError: (error: any) => {
      Alert.alert(
        'Order Failed',
        error.response?.data?.message || error.message || 'Failed to place order'
      );
    },
  });

  const handlePlaceOrder = () => {
    const amountNum = parseFloat(amount);
    if (isNaN(amountNum) || amountNum <= 0) {
      Alert.alert('Invalid Amount', 'Please enter a valid amount');
      return;
    }

    if (side === 'buy' && balance && amountNum > balance.available) {
      Alert.alert('Insufficient Funds', 'You do not have enough balance for this order');
      return;
    }

    Alert.alert(
      'Confirm Order',
      `Are you sure you want to ${side} $${amountNum.toFixed(2)} of ${symbol}?`,
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Confirm',
          onPress: () => {
            placeOrderMutation.mutate({
              symbol: symbol!,
              side,
              amount: amountNum,
            });
          },
        },
      ]
    );
  };

  const quickAmounts = [50, 100, 250, 500, 1000];

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
    }).format(value);
  };

  const estimatedShares = amount && quote ? parseFloat(amount) / quote.mid_price : 0;

  if (!symbol) {
    return (
      <SafeAreaView style={styles.container}>
        <Text>Invalid symbol</Text>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.container} edges={['top']}>
      <KeyboardAvoidingView
        style={{ flex: 1 }}
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      >
        {/* Header */}
        <View style={styles.header}>
          <TouchableOpacity onPress={() => router.back()} style={styles.backButton}>
            <Text style={styles.backText}>Back</Text>
          </TouchableOpacity>
          <View style={styles.headerTitle}>
            <Text style={styles.symbol}>{symbol}</Text>
            {asset && <Text style={styles.name}>{asset.name}</Text>}
          </View>
          <TouchableOpacity onPress={() => refetchQuote()} style={styles.refreshButton}>
            <Text style={styles.refreshText}>Refresh</Text>
          </TouchableOpacity>
        </View>

        <ScrollView style={styles.content}>
          {/* Quote Card */}
          <View style={styles.quoteCard}>
            {quoteLoading ? (
              <ActivityIndicator size="large" color="#10B981" />
            ) : quote ? (
              <>
                <Text style={styles.price}>{formatCurrency(quote.mid_price)}</Text>
                <View style={styles.quoteGrid}>
                  <View style={styles.quoteItem}>
                    <Text style={styles.quoteLabel}>Bid</Text>
                    <Text style={styles.quoteValue}>{formatCurrency(quote.bid_price)}</Text>
                  </View>
                  <View style={styles.quoteItem}>
                    <Text style={styles.quoteLabel}>Ask</Text>
                    <Text style={styles.quoteValue}>{formatCurrency(quote.ask_price)}</Text>
                  </View>
                  <View style={styles.quoteItem}>
                    <Text style={styles.quoteLabel}>Spread</Text>
                    <Text style={styles.quoteValue}>{formatCurrency(quote.spread)}</Text>
                  </View>
                  <View style={styles.quoteItem}>
                    <Text style={styles.quoteLabel}>Updated</Text>
                    <Text style={styles.quoteValue}>
                      {new Date(quote.timestamp).toLocaleTimeString()}
                    </Text>
                  </View>
                </View>
                {asset?.fractionable && (
                  <View style={styles.fractionableTag}>
                    <Text style={styles.fractionableText}>Fractional shares available</Text>
                  </View>
                )}
              </>
            ) : (
              <Text style={styles.errorText}>Unable to load quote</Text>
            )}
          </View>

          {/* Balance Card */}
          <View style={styles.balanceCard}>
            <Text style={styles.balanceLabel}>Available Balance</Text>
            <Text style={styles.balanceValue}>
              {balance ? formatCurrency(balance.available) : '---'}
            </Text>
          </View>

          {/* Order Form */}
          {showOrderForm ? (
            <View style={styles.orderForm}>
              <Text style={styles.orderTitle}>Place Order</Text>

              {/* Buy/Sell Toggle */}
              <View style={styles.sideToggle}>
                <TouchableOpacity
                  style={[styles.sideButton, side === 'buy' && styles.buyActive]}
                  onPress={() => setSide('buy')}
                >
                  <Text style={[styles.sideText, side === 'buy' && styles.sideTextActive]}>
                    Buy
                  </Text>
                </TouchableOpacity>
                <TouchableOpacity
                  style={[styles.sideButton, side === 'sell' && styles.sellActive]}
                  onPress={() => setSide('sell')}
                >
                  <Text style={[styles.sideText, side === 'sell' && styles.sideTextActive]}>
                    Sell
                  </Text>
                </TouchableOpacity>
              </View>

              {/* Amount Input */}
              <View style={styles.inputContainer}>
                <Text style={styles.inputLabel}>Amount (USD)</Text>
                <View style={styles.inputWrapper}>
                  <Text style={styles.currencySymbol}>$</Text>
                  <TextInput
                    style={styles.input}
                    value={amount}
                    onChangeText={setAmount}
                    placeholder="0.00"
                    placeholderTextColor="#9CA3AF"
                    keyboardType="decimal-pad"
                  />
                </View>
              </View>

              {/* Quick Amounts */}
              <View style={styles.quickAmounts}>
                {quickAmounts.map((amt) => (
                  <TouchableOpacity
                    key={amt}
                    style={styles.quickAmountButton}
                    onPress={() => setAmount(amt.toString())}
                  >
                    <Text style={styles.quickAmountText}>${amt}</Text>
                  </TouchableOpacity>
                ))}
              </View>

              {/* Order Preview */}
              {amount && parseFloat(amount) > 0 && quote && (
                <View style={styles.orderPreview}>
                  <View style={styles.previewRow}>
                    <Text style={styles.previewLabel}>Estimated shares</Text>
                    <Text style={styles.previewValue}>
                      ~{estimatedShares.toFixed(6)} {symbol}
                    </Text>
                  </View>
                  <View style={styles.previewRow}>
                    <Text style={styles.previewLabel}>Price per share</Text>
                    <Text style={styles.previewValue}>{formatCurrency(quote.mid_price)}</Text>
                  </View>
                  <View style={[styles.previewRow, styles.previewTotal]}>
                    <Text style={styles.previewTotalLabel}>Total</Text>
                    <Text style={styles.previewTotalValue}>
                      {formatCurrency(parseFloat(amount))}
                    </Text>
                  </View>
                </View>
              )}

              {/* Action Buttons */}
              <View style={styles.actionButtons}>
                <TouchableOpacity
                  style={styles.cancelButton}
                  onPress={() => {
                    setShowOrderForm(false);
                    setAmount('');
                  }}
                >
                  <Text style={styles.cancelButtonText}>Cancel</Text>
                </TouchableOpacity>
                <TouchableOpacity
                  style={[
                    styles.submitButton,
                    side === 'buy' ? styles.buyButton : styles.sellButton,
                    (!amount || placeOrderMutation.isPending) && styles.submitButtonDisabled,
                  ]}
                  onPress={handlePlaceOrder}
                  disabled={!amount || placeOrderMutation.isPending}
                >
                  {placeOrderMutation.isPending ? (
                    <ActivityIndicator color="#fff" size="small" />
                  ) : (
                    <Text style={styles.submitButtonText}>
                      {side === 'buy' ? 'Buy' : 'Sell'} {symbol}
                    </Text>
                  )}
                </TouchableOpacity>
              </View>
            </View>
          ) : (
            /* Trade Button */
            <TouchableOpacity
              style={styles.tradeButton}
              onPress={() => setShowOrderForm(true)}
            >
              <Text style={styles.tradeButtonText}>Trade {symbol}</Text>
            </TouchableOpacity>
          )}
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#F9FAFB',
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: 16,
    backgroundColor: '#fff',
    borderBottomWidth: 1,
    borderBottomColor: '#E5E7EB',
  },
  backButton: {
    padding: 8,
  },
  backText: {
    color: '#10B981',
    fontSize: 16,
    fontWeight: '500',
  },
  headerTitle: {
    alignItems: 'center',
  },
  symbol: {
    fontSize: 20,
    fontWeight: '700',
    color: '#111827',
  },
  name: {
    fontSize: 14,
    color: '#6B7280',
    marginTop: 2,
  },
  refreshButton: {
    padding: 8,
  },
  refreshText: {
    color: '#10B981',
    fontSize: 14,
  },
  content: {
    flex: 1,
    padding: 16,
  },
  quoteCard: {
    backgroundColor: '#fff',
    borderRadius: 16,
    padding: 24,
    alignItems: 'center',
    marginBottom: 16,
  },
  price: {
    fontSize: 40,
    fontWeight: 'bold',
    color: '#111827',
    marginBottom: 20,
  },
  quoteGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    justifyContent: 'space-between',
    width: '100%',
  },
  quoteItem: {
    width: '48%',
    marginBottom: 12,
  },
  quoteLabel: {
    fontSize: 13,
    color: '#6B7280',
    marginBottom: 4,
  },
  quoteValue: {
    fontSize: 16,
    fontWeight: '600',
    color: '#111827',
  },
  fractionableTag: {
    backgroundColor: '#ECFDF5',
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: 8,
    marginTop: 16,
  },
  fractionableText: {
    color: '#10B981',
    fontSize: 13,
    fontWeight: '500',
  },
  errorText: {
    color: '#EF4444',
    fontSize: 14,
  },
  balanceCard: {
    backgroundColor: '#F3F4F6',
    borderRadius: 12,
    padding: 16,
    marginBottom: 16,
  },
  balanceLabel: {
    fontSize: 13,
    color: '#6B7280',
    marginBottom: 4,
  },
  balanceValue: {
    fontSize: 24,
    fontWeight: '700',
    color: '#111827',
  },
  tradeButton: {
    backgroundColor: '#10B981',
    borderRadius: 12,
    paddingVertical: 16,
    alignItems: 'center',
  },
  tradeButtonText: {
    color: '#fff',
    fontSize: 18,
    fontWeight: '600',
  },
  orderForm: {
    backgroundColor: '#fff',
    borderRadius: 16,
    padding: 20,
  },
  orderTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#111827',
    marginBottom: 16,
  },
  sideToggle: {
    flexDirection: 'row',
    backgroundColor: '#F3F4F6',
    borderRadius: 10,
    padding: 4,
    marginBottom: 20,
  },
  sideButton: {
    flex: 1,
    paddingVertical: 12,
    borderRadius: 8,
    alignItems: 'center',
  },
  buyActive: {
    backgroundColor: '#10B981',
  },
  sellActive: {
    backgroundColor: '#EF4444',
  },
  sideText: {
    fontSize: 16,
    fontWeight: '600',
    color: '#6B7280',
  },
  sideTextActive: {
    color: '#fff',
  },
  inputContainer: {
    marginBottom: 16,
  },
  inputLabel: {
    fontSize: 14,
    fontWeight: '500',
    color: '#374151',
    marginBottom: 8,
  },
  inputWrapper: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#F9FAFB',
    borderRadius: 10,
    borderWidth: 1,
    borderColor: '#E5E7EB',
    paddingHorizontal: 16,
  },
  currencySymbol: {
    fontSize: 18,
    color: '#6B7280',
    marginRight: 4,
  },
  input: {
    flex: 1,
    fontSize: 18,
    color: '#111827',
    paddingVertical: 14,
  },
  quickAmounts: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 8,
    marginBottom: 20,
  },
  quickAmountButton: {
    paddingHorizontal: 16,
    paddingVertical: 10,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: '#E5E7EB',
    backgroundColor: '#fff',
  },
  quickAmountText: {
    fontSize: 14,
    color: '#374151',
    fontWeight: '500',
  },
  orderPreview: {
    backgroundColor: '#F9FAFB',
    borderRadius: 12,
    padding: 16,
    marginBottom: 20,
  },
  previewRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 8,
  },
  previewLabel: {
    fontSize: 14,
    color: '#6B7280',
  },
  previewValue: {
    fontSize: 14,
    fontWeight: '500',
    color: '#111827',
  },
  previewTotal: {
    borderTopWidth: 1,
    borderTopColor: '#E5E7EB',
    paddingTop: 8,
    marginTop: 4,
  },
  previewTotalLabel: {
    fontSize: 15,
    fontWeight: '600',
    color: '#374151',
  },
  previewTotalValue: {
    fontSize: 15,
    fontWeight: '700',
    color: '#111827',
  },
  actionButtons: {
    flexDirection: 'row',
    gap: 12,
  },
  cancelButton: {
    flex: 1,
    paddingVertical: 14,
    borderRadius: 10,
    alignItems: 'center',
    borderWidth: 1,
    borderColor: '#E5E7EB',
  },
  cancelButtonText: {
    fontSize: 16,
    fontWeight: '600',
    color: '#6B7280',
  },
  submitButton: {
    flex: 2,
    paddingVertical: 14,
    borderRadius: 10,
    alignItems: 'center',
  },
  buyButton: {
    backgroundColor: '#10B981',
  },
  sellButton: {
    backgroundColor: '#EF4444',
  },
  submitButtonDisabled: {
    opacity: 0.5,
  },
  submitButtonText: {
    fontSize: 16,
    fontWeight: '600',
    color: '#fff',
  },
});
