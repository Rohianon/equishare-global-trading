import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import apiClient from '../api/client';
import { useAuth } from '../contexts/AuthContext';

export default function Dashboard() {
  const { user } = useAuth();

  const { data: balance, isLoading: balanceLoading } = useQuery({
    queryKey: ['walletBalance'],
    queryFn: () => apiClient.getWalletBalance(),
  });

  const { data: transactionsData, isLoading: txLoading } = useQuery({
    queryKey: ['recentTransactions'],
    queryFn: () => apiClient.getTransactions(1, 5),
  });

  const formatCurrency = (amount: number, currency: string) => {
    return new Intl.NumberFormat('en-KE', {
      style: 'currency',
      currency: currency || 'KES',
    }).format(amount);
  };

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">
          Welcome back, {user?.full_name || user?.phone}
        </h1>
        <p className="text-gray-600">Here's an overview of your account</p>
      </div>

      {/* Balance Card */}
      <div className="card bg-gradient-to-r from-primary-600 to-primary-700 text-white">
        <p className="text-primary-100 text-sm mb-1">Available Balance</p>
        {balanceLoading ? (
          <div className="h-10 bg-primary-500 rounded animate-pulse w-32" />
        ) : (
          <p className="text-3xl font-bold">
            {formatCurrency(balance?.available || 0, balance?.currency || 'KES')}
          </p>
        )}
        {balance && balance.pending > 0 && (
          <p className="text-primary-200 text-sm mt-2">
            Pending: {formatCurrency(balance.pending, balance.currency)}
          </p>
        )}
        <div className="mt-4 flex gap-3">
          <Link
            to="/deposit"
            className="bg-white text-primary-600 px-4 py-2 rounded-lg font-medium hover:bg-primary-50 transition-colors"
          >
            Deposit
          </Link>
          <button
            className="bg-primary-500 text-white px-4 py-2 rounded-lg font-medium hover:bg-primary-400 transition-colors"
            disabled
          >
            Withdraw
          </button>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="grid md:grid-cols-3 gap-4">
        <Link to="/deposit" className="card hover:shadow-md transition-shadow">
          <h3 className="font-semibold text-gray-900 mb-1">Deposit Funds</h3>
          <p className="text-sm text-gray-600">Add money via M-Pesa</p>
        </Link>
        <div className="card opacity-50 cursor-not-allowed">
          <h3 className="font-semibold text-gray-900 mb-1">Trade Stocks</h3>
          <p className="text-sm text-gray-600">Coming soon</p>
        </div>
        <Link to="/transactions" className="card hover:shadow-md transition-shadow">
          <h3 className="font-semibold text-gray-900 mb-1">Transactions</h3>
          <p className="text-sm text-gray-600">View history</p>
        </Link>
      </div>

      {/* Recent Transactions */}
      <div className="card">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-semibold text-gray-900">Recent Transactions</h2>
          <Link to="/transactions" className="text-primary-600 hover:text-primary-700 text-sm font-medium">
            View All
          </Link>
        </div>

        {txLoading ? (
          <div className="space-y-3">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-12 bg-gray-100 rounded animate-pulse" />
            ))}
          </div>
        ) : transactionsData?.transactions?.length ? (
          <div className="divide-y divide-gray-100">
            {transactionsData.transactions.map((tx) => (
              <div key={tx.id} className="py-3 flex justify-between items-center">
                <div>
                  <p className="font-medium text-gray-900">{tx.description || tx.type}</p>
                  <p className="text-sm text-gray-500">
                    {new Date(tx.created_at).toLocaleDateString()}
                  </p>
                </div>
                <div className="text-right">
                  <p className={`font-semibold ${tx.type === 'deposit' ? 'text-success-500' : 'text-gray-900'}`}>
                    {tx.type === 'deposit' ? '+' : '-'}
                    {formatCurrency(tx.amount, tx.currency)}
                  </p>
                  <p className={`text-xs ${tx.status === 'completed' ? 'text-success-500' : 'text-yellow-500'}`}>
                    {tx.status}
                  </p>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-gray-500 text-center py-8">No transactions yet</p>
        )}
      </div>
    </div>
  );
}
