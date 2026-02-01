import { View, Text, StyleSheet, ScrollView, RefreshControl } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useQuery } from '@tanstack/react-query';
import apiClient from '../../src/api/client';
import type { Holding } from '../../src/types';

export default function PortfolioScreen() {
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['holdings'],
    queryFn: () => apiClient.get<{ holdings: Holding[] }>('/trading/holdings'),
  });

  const holdings = data?.holdings || [];
  const totalValue = holdings.reduce((sum, h) => sum + h.market_value, 0);
  const totalCost = holdings.reduce((sum, h) => sum + h.avg_cost * h.quantity, 0);
  const totalPL = totalValue - totalCost;
  const totalPLPercent = totalCost > 0 ? (totalPL / totalCost) * 100 : 0;

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
          <Text style={styles.summaryValue}>${totalValue.toFixed(2)}</Text>
          <View style={styles.plContainer}>
            <Text
              style={[
                styles.plValue,
                totalPL >= 0 ? styles.positive : styles.negative,
              ]}
            >
              {totalPL >= 0 ? '+' : ''}${totalPL.toFixed(2)}
            </Text>
            <Text
              style={[
                styles.plPercent,
                totalPL >= 0 ? styles.positive : styles.negative,
              ]}
            >
              ({totalPL >= 0 ? '+' : ''}{totalPLPercent.toFixed(2)}%)
            </Text>
          </View>
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
            </View>
          ) : (
            <View style={styles.holdingsList}>
              {holdings.map((holding) => (
                <View key={holding.id} style={styles.holdingCard}>
                  <View style={styles.holdingHeader}>
                    <View>
                      <Text style={styles.holdingSymbol}>{holding.symbol}</Text>
                      <Text style={styles.holdingName}>{holding.name}</Text>
                    </View>
                    <View style={styles.holdingPriceContainer}>
                      <Text style={styles.holdingPrice}>
                        ${holding.current_price.toFixed(2)}
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

                  <View style={styles.holdingDetails}>
                    <View style={styles.detailRow}>
                      <Text style={styles.detailLabel}>Shares</Text>
                      <Text style={styles.detailValue}>{holding.quantity}</Text>
                    </View>
                    <View style={styles.detailRow}>
                      <Text style={styles.detailLabel}>Avg Cost</Text>
                      <Text style={styles.detailValue}>
                        ${holding.avg_cost.toFixed(2)}
                      </Text>
                    </View>
                    <View style={styles.detailRow}>
                      <Text style={styles.detailLabel}>Market Value</Text>
                      <Text style={styles.detailValue}>
                        ${holding.market_value.toFixed(2)}
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
                  </View>
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
  holdingName: {
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
