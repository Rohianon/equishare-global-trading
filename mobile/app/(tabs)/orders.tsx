import { useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  FlatList,
  TouchableOpacity,
  RefreshControl,
  Alert,
  ActivityIndicator,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { router } from 'expo-router';
import { tradingApi } from '../../src/api/trading';
import type { Order } from '../../src/types';

type StatusFilter = '' | 'pending' | 'filled' | 'canceled';

export default function OrdersScreen() {
  const queryClient = useQueryClient();
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('');
  const [cancellingId, setCancellingId] = useState<string | null>(null);

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['orders', statusFilter],
    queryFn: () => tradingApi.getOrders(statusFilter || undefined),
    refetchInterval: 10000,
  });

  const cancelMutation = useMutation({
    mutationFn: tradingApi.cancelOrder,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      queryClient.invalidateQueries({ queryKey: ['walletBalance'] });
      setCancellingId(null);
      Alert.alert('Order Cancelled', 'Your order has been cancelled successfully.');
    },
    onError: (error: any) => {
      setCancellingId(null);
      Alert.alert(
        'Cancel Failed',
        error.response?.data?.message || 'Failed to cancel order'
      );
    },
  });

  const handleCancel = (orderId: string) => {
    Alert.alert('Cancel Order', 'Are you sure you want to cancel this order?', [
      { text: 'No', style: 'cancel' },
      {
        text: 'Yes, Cancel',
        style: 'destructive',
        onPress: () => {
          setCancellingId(orderId);
          cancelMutation.mutate(orderId);
        },
      },
    ]);
  };

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
    }).format(value);
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const getStatusStyle = (status: string) => {
    switch (status.toLowerCase()) {
      case 'new':
      case 'pending':
        return styles.statusPending;
      case 'filled':
        return styles.statusFilled;
      case 'partial_fill':
        return styles.statusPartial;
      case 'canceled':
        return styles.statusCanceled;
      case 'failed':
        return styles.statusFailed;
      default:
        return styles.statusDefault;
    }
  };

  const canCancel = (status: string) => {
    return ['new', 'pending'].includes(status.toLowerCase());
  };

  const orders = data?.orders || [];

  const filters: { label: string; value: StatusFilter }[] = [
    { label: 'All', value: '' },
    { label: 'Pending', value: 'pending' },
    { label: 'Filled', value: 'filled' },
    { label: 'Canceled', value: 'canceled' },
  ];

  const renderOrder = ({ item }: { item: Order }) => (
    <TouchableOpacity
      style={styles.orderCard}
      onPress={() => router.push({ pathname: '/stock/[symbol]', params: { symbol: item.symbol } })}
    >
      <View style={styles.orderHeader}>
        <View>
          <View style={styles.symbolRow}>
            <Text style={styles.orderSymbol}>{item.symbol}</Text>
            <View style={[styles.sideBadge, item.side === 'buy' ? styles.buyBadge : styles.sellBadge]}>
              <Text style={styles.sideText}>{item.side.toUpperCase()}</Text>
            </View>
          </View>
          <Text style={styles.orderDate}>{formatDate(item.created_at)}</Text>
        </View>
        <View style={[styles.statusBadge, getStatusStyle(item.status)]}>
          <Text style={styles.statusText}>{item.status.replace('_', ' ')}</Text>
        </View>
      </View>

      <View style={styles.orderDetails}>
        <View style={styles.detailRow}>
          <Text style={styles.detailLabel}>Amount</Text>
          <Text style={styles.detailValue}>
            {item.quantity > 0 ? `${item.quantity.toFixed(6)} shares` : '---'}
          </Text>
        </View>
        {item.filled_quantity > 0 && (
          <View style={styles.detailRow}>
            <Text style={styles.detailLabel}>Filled</Text>
            <Text style={styles.detailValue}>{item.filled_quantity.toFixed(6)} shares</Text>
          </View>
        )}
        {item.filled_avg_price && item.filled_avg_price > 0 && (
          <View style={styles.detailRow}>
            <Text style={styles.detailLabel}>Avg Price</Text>
            <Text style={styles.detailValue}>{formatCurrency(item.filled_avg_price)}</Text>
          </View>
        )}
      </View>

      {canCancel(item.status) && (
        <TouchableOpacity
          style={styles.cancelButton}
          onPress={() => handleCancel(item.id)}
          disabled={cancellingId === item.id}
        >
          {cancellingId === item.id ? (
            <ActivityIndicator size="small" color="#EF4444" />
          ) : (
            <Text style={styles.cancelText}>Cancel Order</Text>
          )}
        </TouchableOpacity>
      )}
    </TouchableOpacity>
  );

  return (
    <SafeAreaView style={styles.container} edges={['top']}>
      <View style={styles.header}>
        <Text style={styles.title}>Orders</Text>
        <TouchableOpacity
          style={styles.newOrderButton}
          onPress={() => router.push('/(tabs)/trade')}
        >
          <Text style={styles.newOrderText}>+ New Order</Text>
        </TouchableOpacity>
      </View>

      {/* Filters */}
      <View style={styles.filterContainer}>
        {filters.map((filter) => (
          <TouchableOpacity
            key={filter.value}
            style={[
              styles.filterButton,
              statusFilter === filter.value && styles.filterButtonActive,
            ]}
            onPress={() => setStatusFilter(filter.value)}
          >
            <Text
              style={[
                styles.filterText,
                statusFilter === filter.value && styles.filterTextActive,
              ]}
            >
              {filter.label}
            </Text>
          </TouchableOpacity>
        ))}
      </View>

      <FlatList
        data={orders}
        keyExtractor={(item) => item.id}
        renderItem={renderOrder}
        contentContainerStyle={styles.listContent}
        refreshControl={
          <RefreshControl refreshing={isLoading} onRefresh={refetch} />
        }
        ListEmptyComponent={
          !isLoading ? (
            <View style={styles.emptyState}>
              <Text style={styles.emptyIcon}>📋</Text>
              <Text style={styles.emptyTitle}>No orders found</Text>
              <Text style={styles.emptyText}>
                {statusFilter
                  ? `No ${statusFilter} orders`
                  : 'Place your first order to get started'}
              </Text>
              <TouchableOpacity
                style={styles.emptyButton}
                onPress={() => router.push('/(tabs)/trade')}
              >
                <Text style={styles.emptyButtonText}>Start Trading</Text>
              </TouchableOpacity>
            </View>
          ) : null
        }
      />

      {orders.length > 0 && (
        <View style={styles.footer}>
          <Text style={styles.footerText}>
            Showing {orders.length} order{orders.length !== 1 ? 's' : ''}
          </Text>
        </View>
      )}
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
    justifyContent: 'space-between',
    alignItems: 'center',
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
  newOrderButton: {
    backgroundColor: '#10B981',
    paddingHorizontal: 16,
    paddingVertical: 10,
    borderRadius: 10,
  },
  newOrderText: {
    color: '#fff',
    fontWeight: '600',
    fontSize: 14,
  },
  filterContainer: {
    flexDirection: 'row',
    padding: 16,
    gap: 8,
    backgroundColor: '#fff',
  },
  filterButton: {
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderRadius: 20,
    backgroundColor: '#F3F4F6',
  },
  filterButtonActive: {
    backgroundColor: '#10B981',
  },
  filterText: {
    fontSize: 14,
    fontWeight: '500',
    color: '#6B7280',
  },
  filterTextActive: {
    color: '#fff',
  },
  listContent: {
    padding: 16,
    gap: 12,
  },
  orderCard: {
    backgroundColor: '#fff',
    borderRadius: 16,
    padding: 16,
  },
  orderHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    marginBottom: 12,
  },
  symbolRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  orderSymbol: {
    fontSize: 18,
    fontWeight: '700',
    color: '#111827',
  },
  sideBadge: {
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 6,
  },
  buyBadge: {
    backgroundColor: '#ECFDF5',
  },
  sellBadge: {
    backgroundColor: '#FEF2F2',
  },
  sideText: {
    fontSize: 11,
    fontWeight: '600',
  },
  orderDate: {
    fontSize: 13,
    color: '#6B7280',
    marginTop: 4,
  },
  statusBadge: {
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 12,
  },
  statusText: {
    fontSize: 12,
    fontWeight: '600',
    textTransform: 'capitalize',
  },
  statusPending: {
    backgroundColor: '#FEF3C7',
  },
  statusFilled: {
    backgroundColor: '#ECFDF5',
  },
  statusPartial: {
    backgroundColor: '#D1FAE5',
  },
  statusCanceled: {
    backgroundColor: '#F3F4F6',
  },
  statusFailed: {
    backgroundColor: '#FEE2E2',
  },
  statusDefault: {
    backgroundColor: '#F3F4F6',
  },
  orderDetails: {
    borderTopWidth: 1,
    borderTopColor: '#F3F4F6',
    paddingTop: 12,
    gap: 6,
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
  cancelButton: {
    marginTop: 12,
    paddingVertical: 10,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: '#FEE2E2',
    backgroundColor: '#FEF2F2',
    alignItems: 'center',
  },
  cancelText: {
    color: '#EF4444',
    fontWeight: '600',
    fontSize: 14,
  },
  emptyState: {
    alignItems: 'center',
    padding: 40,
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
    marginBottom: 20,
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
  footer: {
    padding: 16,
    alignItems: 'center',
    borderTopWidth: 1,
    borderTopColor: '#E5E7EB',
    backgroundColor: '#fff',
  },
  footerText: {
    fontSize: 13,
    color: '#6B7280',
  },
});
