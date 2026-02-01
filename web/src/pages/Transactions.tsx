import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import apiClient from '../api/client';

export default function Transactions() {
  const [page, setPage] = useState(1);
  const perPage = 20;

  const { data, isLoading, error } = useQuery({
    queryKey: ['transactions', page],
    queryFn: () => apiClient.getTransactions(page, perPage),
  });

  const formatCurrency = (amount: number, currency: string) => {
    return new Intl.NumberFormat('en-KE', {
      style: 'currency',
      currency: currency || 'KES',
    }).format(amount);
  };

  const getStatusBadgeClass = (status: string) => {
    switch (status.toLowerCase()) {
      case 'completed':
        return 'bg-green-100 text-green-800';
      case 'pending':
        return 'bg-yellow-100 text-yellow-800';
      case 'failed':
        return 'bg-red-100 text-red-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const getTypeBadgeClass = (type: string) => {
    switch (type.toLowerCase()) {
      case 'deposit':
        return 'bg-blue-100 text-blue-800';
      case 'withdrawal':
        return 'bg-purple-100 text-purple-800';
      case 'trade':
        return 'bg-indigo-100 text-indigo-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const totalPages = data ? Math.ceil(data.total / perPage) : 0;

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Transactions</h1>
          <p className="text-gray-600">Your complete transaction history</p>
        </div>
        <Link to="/deposit" className="btn btn-primary">
          Deposit
        </Link>
      </div>

      <div className="card">
        {isLoading ? (
          <div className="space-y-4">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="h-16 bg-gray-100 rounded animate-pulse" />
            ))}
          </div>
        ) : error ? (
          <div className="text-center py-8">
            <p className="text-red-600">Failed to load transactions</p>
            <button
              onClick={() => window.location.reload()}
              className="btn btn-secondary mt-4"
            >
              Retry
            </button>
          </div>
        ) : data?.transactions?.length ? (
          <>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-gray-200">
                    <th className="text-left py-3 px-2 text-sm font-medium text-gray-500">Date</th>
                    <th className="text-left py-3 px-2 text-sm font-medium text-gray-500">Type</th>
                    <th className="text-left py-3 px-2 text-sm font-medium text-gray-500">Description</th>
                    <th className="text-right py-3 px-2 text-sm font-medium text-gray-500">Amount</th>
                    <th className="text-center py-3 px-2 text-sm font-medium text-gray-500">Status</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {data.transactions.map((tx) => (
                    <tr key={tx.id} className="hover:bg-gray-50">
                      <td className="py-3 px-2 text-sm text-gray-600">
                        {new Date(tx.created_at).toLocaleDateString('en-KE', {
                          year: 'numeric',
                          month: 'short',
                          day: 'numeric',
                          hour: '2-digit',
                          minute: '2-digit',
                        })}
                      </td>
                      <td className="py-3 px-2">
                        <span className={`text-xs px-2 py-1 rounded-full font-medium ${getTypeBadgeClass(tx.type)}`}>
                          {tx.type}
                        </span>
                      </td>
                      <td className="py-3 px-2 text-sm text-gray-900">
                        {tx.description || '-'}
                      </td>
                      <td className={`py-3 px-2 text-sm font-semibold text-right ${
                        tx.type === 'deposit' ? 'text-success-500' : 'text-gray-900'
                      }`}>
                        {tx.type === 'deposit' ? '+' : '-'}
                        {formatCurrency(tx.amount, tx.currency)}
                      </td>
                      <td className="py-3 px-2 text-center">
                        <span className={`text-xs px-2 py-1 rounded-full font-medium ${getStatusBadgeClass(tx.status)}`}>
                          {tx.status}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Pagination */}
            {totalPages > 1 && (
              <div className="flex justify-between items-center mt-4 pt-4 border-t border-gray-100">
                <p className="text-sm text-gray-600">
                  Showing {((page - 1) * perPage) + 1} to {Math.min(page * perPage, data.total)} of {data.total}
                </p>
                <div className="flex gap-2">
                  <button
                    onClick={() => setPage(p => Math.max(1, p - 1))}
                    disabled={page === 1}
                    className="btn btn-secondary text-sm disabled:opacity-50"
                  >
                    Previous
                  </button>
                  <button
                    onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                    disabled={page >= totalPages}
                    className="btn btn-secondary text-sm disabled:opacity-50"
                  >
                    Next
                  </button>
                </div>
              </div>
            )}
          </>
        ) : (
          <div className="text-center py-12">
            <p className="text-gray-500 mb-4">No transactions yet</p>
            <Link to="/deposit" className="btn btn-primary">
              Make your first deposit
            </Link>
          </div>
        )}
      </div>
    </div>
  );
}
