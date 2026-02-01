import { View, Text, StyleSheet, ScrollView, TouchableOpacity, RefreshControl } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'expo-router';
import { useAuthStore } from '../../src/stores/authStore';
import apiClient from '../../src/api/client';
import type { Wallet, Holding, Transaction } from '../../src/types';

export default function HomeScreen() {
  const { user } = useAuthStore();

  const { data: wallet, isLoading: walletLoading, refetch: refetchWallet } = useQuery({
    queryKey: ['wallet'],
    queryFn: () => apiClient.get<Wallet>('/payments/wallet/balance'),
  });

  const { data: holdings, refetch: refetchHoldings } = useQuery({
    queryKey: ['holdings'],
    queryFn: () => apiClient.get<{ holdings: Holding[] }>('/trading/holdings'),
  });

  const { data: transactions, refetch: refetchTransactions } = useQuery({
    queryKey: ['transactions'],
    queryFn: () => apiClient.get<{ transactions: Transaction[] }>('/payments/transactions?page=1&per_page=5'),
  });

  const onRefresh = () => {
    refetchWallet();
    refetchHoldings();
    refetchTransactions();
  };

  const totalPortfolioValue = (holdings?.holdings || []).reduce(
    (sum, h) => sum + h.market_value,
    0
  );

  const formatCurrency = (amount: number, currency = 'KES') => {
    return new Intl.NumberFormat('en-KE', {
      style: 'currency',
      currency,
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(amount);
  };

  return (
    <SafeAreaView style={styles.container} edges={['top']}>
      <ScrollView
        style={styles.scrollView}
        refreshControl={
          <RefreshControl refreshing={walletLoading} onRefresh={onRefresh} />
        }
      >
        {/* Header */}
        <View style={styles.header}>
          <View>
            <Text style={styles.greeting}>
              Hello, {user?.display_name || user?.username || 'Trader'} 👋
            </Text>
            <Text style={styles.date}>
              {new Date().toLocaleDateString('en-KE', {
                weekday: 'long',
                month: 'long',
                day: 'numeric',
              })}
            </Text>
          </View>
          <View style={styles.avatar}>
            <Text style={styles.avatarText}>
              {(user?.display_name || user?.username || 'U')[0].toUpperCase()}
            </Text>
          </View>
        </View>

        {/* Balance Card */}
        <View style={styles.balanceCard}>
          <Text style={styles.balanceLabel}>Total Balance</Text>
          <Text style={styles.balanceAmount}>
            {formatCurrency((wallet?.total || 0) + totalPortfolioValue)}
          </Text>
          <View style={styles.balanceBreakdown}>
            <View style={styles.balanceItem}>
              <Text style={styles.balanceItemLabel}>Cash</Text>
              <Text style={styles.balanceItemValue}>
                {formatCurrency(wallet?.available || 0)}
              </Text>
            </View>
            <View style={styles.balanceItem}>
              <Text style={styles.balanceItemLabel}>Invested</Text>
              <Text style={styles.balanceItemValue}>
                {formatCurrency(totalPortfolioValue, 'USD')}
              </Text>
            </View>
          </View>
        </View>

        {/* Quick Actions */}
        <View style={styles.actions}>
          <Link href="/deposit" asChild>
            <TouchableOpacity style={styles.actionButton}>
              <Text style={styles.actionIcon}>💰</Text>
              <Text style={styles.actionLabel}>Deposit</Text>
            </TouchableOpacity>
          </Link>
          <Link href="/withdraw" asChild>
            <TouchableOpacity style={styles.actionButton}>
              <Text style={styles.actionIcon}>📤</Text>
              <Text style={styles.actionLabel}>Withdraw</Text>
            </TouchableOpacity>
          </Link>
          <Link href="/(tabs)/trade" asChild>
            <TouchableOpacity style={styles.actionButton}>
              <Text style={styles.actionIcon}>📈</Text>
              <Text style={styles.actionLabel}>Trade</Text>
            </TouchableOpacity>
          </Link>
        </View>

        {/* Holdings Preview */}
        <View style={styles.section}>
          <View style={styles.sectionHeader}>
            <Text style={styles.sectionTitle}>Your Holdings</Text>
            <Link href="/(tabs)/portfolio" asChild>
              <TouchableOpacity>
                <Text style={styles.seeAllLink}>See all</Text>
              </TouchableOpacity>
            </Link>
          </View>

          {(holdings?.holdings || []).length === 0 ? (
            <View style={styles.emptyState}>
              <Text style={styles.emptyIcon}>📊</Text>
              <Text style={styles.emptyTitle}>No holdings yet</Text>
              <Text style={styles.emptyText}>Start trading to build your portfolio</Text>
              <Link href="/(tabs)/trade" asChild>
                <TouchableOpacity style={styles.emptyButton}>
                  <Text style={styles.emptyButtonText}>Start Trading</Text>
                </TouchableOpacity>
              </Link>
            </View>
          ) : (
            <View style={styles.holdingsList}>
              {(holdings?.holdings || []).slice(0, 3).map((holding) => (
                <View key={holding.id} style={styles.holdingItem}>
                  <View style={styles.holdingInfo}>
                    <Text style={styles.holdingSymbol}>{holding.symbol}</Text>
                    <Text style={styles.holdingName}>{holding.name}</Text>
                  </View>
                  <View style={styles.holdingValue}>
                    <Text style={styles.holdingPrice}>
                      ${holding.market_value.toFixed(2)}
                    </Text>
                    <Text
                      style={[
                        styles.holdingChange,
                        holding.unrealized_pl >= 0 ? styles.positive : styles.negative,
                      ]}
                    >
                      {holding.unrealized_pl >= 0 ? '+' : ''}
                      {holding.unrealized_pl_percent.toFixed(2)}%
                    </Text>
                  </View>
                </View>
              ))}
            </View>
          )}
        </View>

        {/* Recent Transactions */}
        <View style={styles.section}>
          <View style={styles.sectionHeader}>
            <Text style={styles.sectionTitle}>Recent Activity</Text>
          </View>

          {(transactions?.transactions || []).length === 0 ? (
            <Text style={styles.noTransactions}>No recent transactions</Text>
          ) : (
            <View style={styles.transactionsList}>
              {(transactions?.transactions || []).map((tx) => (
                <View key={tx.id} style={styles.transactionItem}>
                  <View style={styles.transactionIcon}>
                    <Text>
                      {tx.type === 'deposit' && '💰'}
                      {tx.type === 'withdrawal' && '📤'}
                      {tx.type === 'buy' && '🛒'}
                      {tx.type === 'sell' && '💵'}
                    </Text>
                  </View>
                  <View style={styles.transactionInfo}>
                    <Text style={styles.transactionTitle}>{tx.description}</Text>
                    <Text style={styles.transactionDate}>
                      {new Date(tx.created_at).toLocaleDateString()}
                    </Text>
                  </View>
                  <Text
                    style={[
                      styles.transactionAmount,
                      tx.type === 'deposit' || tx.type === 'sell'
                        ? styles.positive
                        : styles.negative,
                    ]}
                  >
                    {tx.type === 'deposit' || tx.type === 'sell' ? '+' : '-'}
                    {formatCurrency(tx.amount, tx.currency)}
                  </Text>
                </View>
              ))}
            </View>
          )}
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#F9FAFB',
  },
  scrollView: {
    flex: 1,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: 20,
  },
  greeting: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#111827',
  },
  date: {
    fontSize: 14,
    color: '#6B7280',
    marginTop: 4,
  },
  avatar: {
    width: 48,
    height: 48,
    borderRadius: 24,
    backgroundColor: '#10B981',
    justifyContent: 'center',
    alignItems: 'center',
  },
  avatarText: {
    fontSize: 20,
    fontWeight: 'bold',
    color: '#fff',
  },
  balanceCard: {
    margin: 20,
    marginTop: 0,
    padding: 24,
    backgroundColor: '#10B981',
    borderRadius: 20,
  },
  balanceLabel: {
    fontSize: 14,
    color: 'rgba(255,255,255,0.8)',
    marginBottom: 8,
  },
  balanceAmount: {
    fontSize: 36,
    fontWeight: 'bold',
    color: '#fff',
    marginBottom: 20,
  },
  balanceBreakdown: {
    flexDirection: 'row',
    justifyContent: 'space-between',
  },
  balanceItem: {},
  balanceItemLabel: {
    fontSize: 12,
    color: 'rgba(255,255,255,0.7)',
    marginBottom: 4,
  },
  balanceItemValue: {
    fontSize: 16,
    fontWeight: '600',
    color: '#fff',
  },
  actions: {
    flexDirection: 'row',
    justifyContent: 'space-around',
    paddingHorizontal: 20,
    marginBottom: 24,
  },
  actionButton: {
    alignItems: 'center',
    backgroundColor: '#fff',
    paddingVertical: 16,
    paddingHorizontal: 24,
    borderRadius: 16,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.05,
    shadowRadius: 8,
    elevation: 2,
  },
  actionIcon: {
    fontSize: 24,
    marginBottom: 8,
  },
  actionLabel: {
    fontSize: 12,
    fontWeight: '600',
    color: '#374151',
  },
  section: {
    paddingHorizontal: 20,
    marginBottom: 24,
  },
  sectionHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 16,
  },
  sectionTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#111827',
  },
  seeAllLink: {
    fontSize: 14,
    color: '#10B981',
    fontWeight: '500',
  },
  emptyState: {
    backgroundColor: '#fff',
    borderRadius: 16,
    padding: 32,
    alignItems: 'center',
  },
  emptyIcon: {
    fontSize: 48,
    marginBottom: 16,
  },
  emptyTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#111827',
    marginBottom: 8,
  },
  emptyText: {
    fontSize: 14,
    color: '#6B7280',
    marginBottom: 20,
    textAlign: 'center',
  },
  emptyButton: {
    backgroundColor: '#10B981',
    paddingVertical: 12,
    paddingHorizontal: 24,
    borderRadius: 12,
  },
  emptyButtonText: {
    color: '#fff',
    fontWeight: '600',
  },
  holdingsList: {
    backgroundColor: '#fff',
    borderRadius: 16,
  },
  holdingItem: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#F3F4F6',
  },
  holdingInfo: {},
  holdingSymbol: {
    fontSize: 16,
    fontWeight: '600',
    color: '#111827',
  },
  holdingName: {
    fontSize: 13,
    color: '#6B7280',
    marginTop: 2,
  },
  holdingValue: {
    alignItems: 'flex-end',
  },
  holdingPrice: {
    fontSize: 16,
    fontWeight: '600',
    color: '#111827',
  },
  holdingChange: {
    fontSize: 13,
    fontWeight: '500',
    marginTop: 2,
  },
  positive: {
    color: '#10B981',
  },
  negative: {
    color: '#EF4444',
  },
  noTransactions: {
    textAlign: 'center',
    color: '#6B7280',
    paddingVertical: 24,
  },
  transactionsList: {
    backgroundColor: '#fff',
    borderRadius: 16,
  },
  transactionItem: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#F3F4F6',
  },
  transactionIcon: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: '#F3F4F6',
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  transactionInfo: {
    flex: 1,
  },
  transactionTitle: {
    fontSize: 14,
    fontWeight: '500',
    color: '#111827',
  },
  transactionDate: {
    fontSize: 12,
    color: '#9CA3AF',
    marginTop: 2,
  },
  transactionAmount: {
    fontSize: 14,
    fontWeight: '600',
  },
});
