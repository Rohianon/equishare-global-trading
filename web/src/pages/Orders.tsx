import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import apiClient from '../api/client';

export default function Orders() {
  const queryClient = useQueryClient();
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [cancellingId, setCancellingId] = useState<string | null>(null);

  const { data, isLoading, error } = useQuery({
    queryKey: ['orders', statusFilter],
    queryFn: () => apiClient.getOrders(statusFilter || undefined),
    refetchInterval: 10000, // Refresh every 10 seconds
  });

  const cancelMutation = useMutation({
    mutationFn: (orderId: string) => apiClient.cancelOrder(orderId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      queryClient.invalidateQueries({ queryKey: ['walletBalance'] });
      setCancellingId(null);
    },
    onError: () => {
      setCancellingId(null);
    },
  });

  const handleCancel = (orderId: string) => {
    setCancellingId(orderId);
    cancelMutation.mutate(orderId);
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

  const getStatusBadge = (status: string) => {
    const statusClasses: Record<string, string> = {
      new: 'bg-blue-100 text-blue-800',
      pending: 'bg-yellow-100 text-yellow-800',
      filled: 'bg-green-100 text-green-800',
      partial_fill: 'bg-emerald-100 text-emerald-800',
      canceled: 'bg-gray-100 text-gray-800',
      failed: 'bg-red-100 text-red-800',
    };
    return statusClasses[status.toLowerCase()] || 'bg-gray-100 text-gray-800';
  };

  const getSideBadge = (side: string) => {
    return side === 'buy'
      ? 'bg-green-100 text-green-800'
      : 'bg-red-100 text-red-800';
  };

  const canCancel = (status: string) => {
    return ['new', 'pending'].includes(status.toLowerCase());
  };

  const orders = data?.orders || [];

  return (
    <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="flex justify-between items-center mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Orders</h1>
          <p className="mt-1 text-gray-600">Your order history and pending orders</p>
        </div>
        <Link to="/trade" className="btn btn-primary">
          New Order
        </Link>
      </div>

      {/* Filters */}
      <div className="mb-6">
        <div className="flex gap-2">
          {['', 'pending', 'filled', 'canceled'].map((status) => (
            <button
              key={status}
              onClick={() => setStatusFilter(status)}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                statusFilter === status
                  ? 'bg-primary-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              {status === '' ? 'All' : status.charAt(0).toUpperCase() + status.slice(1)}
            </button>
          ))}
        </div>
      </div>

      {/* Orders List */}
      <div className="card">
        {isLoading ? (
          <div className="animate-pulse space-y-4">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="h-16 bg-gray-200 rounded"></div>
            ))}
          </div>
        ) : error ? (
          <div className="text-center py-12">
            <p className="text-red-600">Failed to load orders</p>
            <button
              onClick={() => window.location.reload()}
              className="mt-4 btn btn-primary"
            >
              Retry
            </button>
          </div>
        ) : orders.length === 0 ? (
          <div className="text-center py-12">
            <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
            <p className="mt-4 text-gray-500">No orders found</p>
            <Link to="/trade" className="mt-4 inline-block btn btn-primary">
              Place an Order
            </Link>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="text-left text-sm text-gray-500 border-b border-gray-200">
                  <th className="pb-3 font-medium">Date</th>
                  <th className="pb-3 font-medium">Symbol</th>
                  <th className="pb-3 font-medium">Side</th>
                  <th className="pb-3 font-medium text-right">Amount</th>
                  <th className="pb-3 font-medium text-right">Qty</th>
                  <th className="pb-3 font-medium text-right">Filled</th>
                  <th className="pb-3 font-medium text-right">Avg Price</th>
                  <th className="pb-3 font-medium">Status</th>
                  <th className="pb-3"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {orders.map((order) => (
                  <tr key={order.id} className="hover:bg-gray-50">
                    <td className="py-4 text-sm text-gray-600">
                      {formatDate(order.created_at)}
                    </td>
                    <td className="py-4">
                      <span className="font-semibold text-gray-900">{order.symbol}</span>
                    </td>
                    <td className="py-4">
                      <span className={`px-2 py-1 text-xs font-medium rounded-full ${getSideBadge(order.side)}`}>
                        {order.side.toUpperCase()}
                      </span>
                    </td>
                    <td className="py-4 text-right text-gray-900">
                      {formatCurrency(order.amount)}
                    </td>
                    <td className="py-4 text-right text-gray-600">
                      {order.qty > 0 ? order.qty.toFixed(6) : '-'}
                    </td>
                    <td className="py-4 text-right text-gray-600">
                      {order.filled_qty > 0 ? order.filled_qty.toFixed(6) : '-'}
                    </td>
                    <td className="py-4 text-right text-gray-600">
                      {order.filled_avg_price > 0 ? formatCurrency(order.filled_avg_price) : '-'}
                    </td>
                    <td className="py-4">
                      <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusBadge(order.status)}`}>
                        {order.status.replace('_', ' ')}
                      </span>
                    </td>
                    <td className="py-4 text-right">
                      {canCancel(order.status) && (
                        <button
                          onClick={() => handleCancel(order.id)}
                          disabled={cancellingId === order.id}
                          className="text-sm text-red-600 hover:text-red-700 disabled:text-red-300"
                        >
                          {cancellingId === order.id ? 'Cancelling...' : 'Cancel'}
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Summary */}
      {orders.length > 0 && (
        <div className="mt-4 text-sm text-gray-500 text-center">
          Showing {orders.length} order{orders.length !== 1 ? 's' : ''}
        </div>
      )}

      {/* Quick Links */}
      <div className="mt-6 flex gap-4 text-sm">
        <Link to="/portfolio" className="text-primary-600 hover:text-primary-700">
          View Portfolio →
        </Link>
        <Link to="/transactions" className="text-primary-600 hover:text-primary-700">
          Transaction History →
        </Link>
      </div>
    </div>
  );
}
