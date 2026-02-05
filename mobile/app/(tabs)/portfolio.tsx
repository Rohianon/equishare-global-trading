import { View, Text, StyleSheet, ScrollView, RefreshControl, TouchableOpacity } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useQuery } from '@tanstack/react-query';
import { router } from 'expo-router';
import { tradingApi } from '../../src/api/trading';
import type { PortfolioHolding } from '../../src/types';

export default function PortfolioScreen() {
  const { data: portfolio, isLoading, refetch } = useQuery({
    queryKey: ['portfolio'],
    queryFn: () => tradingApi.getPortfolio(),
    refetchInterval: 30000,
  });

  const summary = portfolio?.summary;
  const holdings = portfolio?.holdings || [];

  return (
    <SafeAreaView style={styles.container} edges={['top']}>
      <View style={styles.header}>
        <Text style={styles.title}>Portfolio</Text>
      </View>

      <ScrollView
        style={styles.scrollView}
        refreshControl={
          <RefreshControl refreshing={isLoading} onRefresh={refetch} />
        }
      >
        {/* Portfolio Summary */}
        <View style={styles.summaryCard}>
          <Text style={styles.summaryLabel}>Total Portfolio Value</Text>
          <Text style={styles.summaryValue}>
            ${(summary?.total_value ?? 0).toFixed(2)}
          </Text>
          <View style={styles.plContainer}>
            <Text
              style={[
                styles.plValue,
                (summary?.total_unrealized_pl ?? 0) >= 0 ? styles.positive : styles.negative,
              ]}
            >
              {(summary?.total_unrealized_pl ?? 0) >= 0 ? '+' : ''}${(summary?.total_unrealized_pl ?? 0).toFixed(2)}
            </Text>
            <Text
              style={[
                styles.plPercent,
                (summary?.total_unrealized_pl_pct ?? 0) >= 0 ? styles.positive : styles.negative,
              ]}
            >
              ({(summary?.total_unrealized_pl_pct ?? 0) >= 0 ? '+' : ''}{(summary?.total_unrealized_pl_pct ?? 0).toFixed(2)}%)
            </Text>
          </View>
          {summary && (
            <View style={styles.summaryDetails}>
              <View style={styles.summaryRow}>
                <Text style={styles.summaryDetailLabel}>Day Change</Text>
                <Text
                  style={[
                    styles.summaryDetailValue,
                    summary.day_change >= 0 ? styles.positive : styles.negative,
                  ]}
                >
                  {summary.day_change >= 0 ? '+' : ''}${summary.day_change.toFixed(2)} ({summary.day_change >= 0 ? '+' : ''}{summary.day_change_pct.toFixed(2)}%)
                </Text>
              </View>
              <View style={styles.summaryRow}>
                <Text style={styles.summaryDetailLabel}>Cash Balance</Text>
                <Text style={styles.summaryDetailValue}>${summary.cash_balance.toFixed(2)}</Text>
              </View>
            </View>
          )}
        </View>

        {/* Holdings List */}
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Holdings ({holdings.length})</Text>

          {holdings.length === 0 ? (
            <View style={styles.emptyState}>
              <Text style={styles.emptyIcon}>📊</Text>
              <Text style={styles.emptyTitle}>No holdings yet</Text>
              <Text style={styles.emptyText}>
                Buy your first stock to start building your portfolio
              </Text>
              <TouchableOpacity
                style={styles.emptyButton}
                onPress={() => router.push('/(tabs)/trade')}
              >
                <Text style={styles.emptyButtonText}>Start Trading</Text>
              </TouchableOpacity>
            </View>
          ) : (
            <View style={styles.holdingsList}>
              {holdings.map((holding: PortfolioHolding) => (
                <TouchableOpacity
                  key={holding.symbol}
                  style={styles.holdingCard}
                  onPress={() => router.push({ pathname: '/stock/[symbol]', params: { symbol: holding.symbol } })}
                >
                  <View style={styles.holdingHeader}>
                    <View>
                      <Text style={styles.holdingSymbol}>{holding.symbol}</Text>
                      <Text style={styles.holdingShares}>{holding.quantity.toFixed(6)} shares</Text>
                    </View>
                    <View style={styles.holdingPriceContainer}>
                      <Text style={styles.holdingPrice}>
                        ${holding.current_price.toFixed(2)}
                      </Text>
                      <Text
                        style={[
                          styles.holdingChange,
                          holding.unrealized_pl_pct >= 0 ? styles.positive : styles.negative,
                        ]}
                      >
                        {holding.unrealized_pl_pct >= 0 ? '+' : ''}
                        {holding.unrealized_pl_pct.toFixed(2)}%
                      </Text>
                    </View>
                  </View>

                  <View style={styles.holdingDetails}>
                    <View style={styles.detailRow}>
                      <Text style={styles.detailLabel}>Market Value</Text>
                      <Text style={styles.detailValue}>
                        ${holding.market_value.toFixed(2)}
                      </Text>
                    </View>
                    <View style={styles.detailRow}>
                      <Text style={styles.detailLabel}>Avg Cost</Text>
                      <Text style={styles.detailValue}>
                        ${holding.avg_cost_basis.toFixed(2)}
                      </Text>
                    </View>
                    <View style={styles.detailRow}>
                      <Text style={styles.detailLabel}>P/L</Text>
                      <Text
                        style={[
                          styles.detailValue,
                          holding.unrealized_pl >= 0 ? styles.positive : styles.negative,
                        ]}
                      >
                        {holding.unrealized_pl >= 0 ? '+' : ''}$
                        {holding.unrealized_pl.toFixed(2)}
                      </Text>
                    </View>
                    <View style={styles.detailRow}>
                      <Text style={styles.detailLabel}>Allocation</Text>
                      <Text style={styles.detailValue}>
                        {holding.allocation_pct.toFixed(1)}%
                      </Text>
                    </View>
                  </View>
                </TouchableOpacity>
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
  header: {
    padding: 20,
    backgroundColor: '#fff',
    borderBottomWidth: 1,
    borderBottomColor: '#E5E7EB',
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#111827',
  },
  scrollView: {
    flex: 1,
  },
  summaryCard: {
    margin: 20,
    padding: 24,
    backgroundColor: '#fff',
    borderRadius: 16,
    alignItems: 'center',
  },
  summaryLabel: {
    fontSize: 14,
    color: '#6B7280',
    marginBottom: 8,
  },
  summaryValue: {
    fontSize: 36,
    fontWeight: 'bold',
    color: '#111827',
    marginBottom: 8,
  },
  plContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  plValue: {
    fontSize: 16,
    fontWeight: '600',
  },
  plPercent: {
    fontSize: 14,
  },
  summaryDetails: {
    marginTop: 16,
    paddingTop: 16,
    borderTopWidth: 1,
    borderTopColor: '#F3F4F6',
    width: '100%',
  },
  summaryRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 8,
  },
  summaryDetailLabel: {
    fontSize: 14,
    color: '#6B7280',
  },
  summaryDetailValue: {
    fontSize: 14,
    fontWeight: '500',
    color: '#111827',
  },
  positive: {
    color: '#10B981',
  },
  negative: {
    color: '#EF4444',
  },
  section: {
    paddingHorizontal: 20,
  },
  sectionTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#111827',
    marginBottom: 16,
  },
  emptyState: {
    backgroundColor: '#fff',
    borderRadius: 16,
    padding: 40,
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
    textAlign: 'center',
    marginBottom: 16,
  },
  emptyButton: {
    backgroundColor: '#10B981',
    paddingHorizontal: 24,
    paddingVertical: 12,
    borderRadius: 10,
  },
  emptyButtonText: {
    color: '#fff',
    fontWeight: '600',
    fontSize: 16,
  },
  holdingsList: {
    gap: 12,
  },
  holdingCard: {
    backgroundColor: '#fff',
    borderRadius: 16,
    padding: 16,
  },
  holdingHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    marginBottom: 16,
    paddingBottom: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#F3F4F6',
  },
  holdingSymbol: {
    fontSize: 18,
    fontWeight: '700',
    color: '#111827',
  },
  holdingShares: {
    fontSize: 14,
    color: '#6B7280',
    marginTop: 4,
  },
  holdingPriceContainer: {
    alignItems: 'flex-end',
  },
  holdingPrice: {
    fontSize: 18,
    fontWeight: '700',
    color: '#111827',
  },
  holdingChange: {
    fontSize: 14,
    fontWeight: '500',
    marginTop: 4,
  },
  holdingDetails: {
    gap: 8,
  },
  detailRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
  },
  detailLabel: {
    fontSize: 14,
    color: '#6B7280',
  },
  detailValue: {
    fontSize: 14,
    fontWeight: '500',
    color: '#111827',
  },
});
