import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import apiClient from '../api/client';

export default function Portfolio() {
  const { data: portfolio, isLoading, error } = useQuery({
    queryKey: ['portfolio'],
    queryFn: () => apiClient.getPortfolio(),
    refetchInterval: 30000, // Refresh every 30 seconds
  });

  const formatCurrency = (value: number, currency = 'USD') => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency,
    }).format(value);
  };

  const formatPercent = (value: number) => {
    const sign = value >= 0 ? '+' : '';
    return `${sign}${value.toFixed(2)}%`;
  };

  const getPLColor = (value: number) => {
    if (value > 0) return 'text-green-600';
    if (value < 0) return 'text-red-600';
    return 'text-gray-600';
  };

  if (isLoading) {
    return (
      <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="animate-pulse space-y-6">
          <div className="h-8 bg-gray-200 rounded w-1/4"></div>
          <div className="grid md:grid-cols-4 gap-4">
            {[...Array(4)].map((_, i) => (
              <div key={i} className="h-24 bg-gray-200 rounded-xl"></div>
            ))}
          </div>
          <div className="h-64 bg-gray-200 rounded-xl"></div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="bg-red-50 border border-red-200 rounded-lg p-6 text-center">
          <p className="text-red-700">Failed to load portfolio. Please try again.</p>
          <button
            onClick={() => window.location.reload()}
            className="mt-4 btn btn-primary"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  const summary = portfolio?.summary;
  const holdings = portfolio?.holdings || [];

  return (
    <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="flex justify-between items-center mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Portfolio</h1>
          <p className="mt-1 text-gray-600">Your investment holdings and performance</p>
        </div>
        <Link to="/trade" className="btn btn-primary">
          Trade
        </Link>
      </div>

      {/* Summary Cards */}
      <div className="grid md:grid-cols-4 gap-4 mb-8">
        <div className="card">
          <p className="text-sm text-gray-500 mb-1">Total Value</p>
          <p className="text-2xl font-bold text-gray-900">
            {summary ? formatCurrency(summary.total_value) : '$0.00'}
          </p>
        </div>

        <div className="card">
          <p className="text-sm text-gray-500 mb-1">Total P&L</p>
          <p className={`text-2xl font-bold ${getPLColor(summary?.total_unrealized_pl || 0)}`}>
            {summary ? formatCurrency(summary.total_unrealized_pl) : '$0.00'}
          </p>
          <p className={`text-sm ${getPLColor(summary?.total_unrealized_pl_pct || 0)}`}>
            {summary ? formatPercent(summary.total_unrealized_pl_pct) : '+0.00%'}
          </p>
        </div>

        <div className="card">
          <p className="text-sm text-gray-500 mb-1">Day Change</p>
          <p className={`text-2xl font-bold ${getPLColor(summary?.day_change || 0)}`}>
            {summary ? formatCurrency(summary.day_change) : '$0.00'}
          </p>
          <p className={`text-sm ${getPLColor(summary?.day_change_pct || 0)}`}>
            {summary ? formatPercent(summary.day_change_pct) : '+0.00%'}
          </p>
        </div>

        <div className="card">
          <p className="text-sm text-gray-500 mb-1">Cash Balance</p>
          <p className="text-2xl font-bold text-gray-900">
            {summary ? formatCurrency(summary.cash_balance) : '$0.00'}
          </p>
          <Link to="/deposit" className="text-sm text-primary-600 hover:text-primary-700">
            Add funds →
          </Link>
        </div>
      </div>

      {/* Holdings Table */}
      <div className="card">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-semibold text-gray-900">Holdings</h2>
          <span className="text-sm text-gray-500">
            {holdings.length} position{holdings.length !== 1 ? 's' : ''}
          </span>
        </div>

        {holdings.length === 0 ? (
          <div className="text-center py-12">
            <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
            </svg>
            <p className="mt-4 text-gray-500">No holdings yet</p>
            <Link to="/trade" className="mt-4 inline-block btn btn-primary">
              Start Trading
            </Link>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="text-left text-sm text-gray-500 border-b border-gray-200">
                  <th className="pb-3 font-medium">Symbol</th>
                  <th className="pb-3 font-medium text-right">Shares</th>
                  <th className="pb-3 font-medium text-right">Price</th>
                  <th className="pb-3 font-medium text-right">Market Value</th>
                  <th className="pb-3 font-medium text-right">Avg Cost</th>
                  <th className="pb-3 font-medium text-right">P&L</th>
                  <th className="pb-3 font-medium text-right">Allocation</th>
                  <th className="pb-3"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {holdings.map((holding) => (
                  <tr key={holding.symbol} className="hover:bg-gray-50">
                    <td className="py-4">
                      <span className="font-semibold text-gray-900">{holding.symbol}</span>
                    </td>
                    <td className="py-4 text-right text-gray-900">
                      {holding.quantity.toFixed(6)}
                    </td>
                    <td className="py-4 text-right text-gray-900">
                      {formatCurrency(holding.current_price)}
                    </td>
                    <td className="py-4 text-right font-medium text-gray-900">
                      {formatCurrency(holding.market_value)}
                    </td>
                    <td className="py-4 text-right text-gray-600">
                      {formatCurrency(holding.avg_cost_basis)}
                    </td>
                    <td className="py-4 text-right">
                      <div className={`font-medium ${getPLColor(holding.unrealized_pl)}`}>
                        {formatCurrency(holding.unrealized_pl)}
                      </div>
                      <div className={`text-sm ${getPLColor(holding.unrealized_pl_pct)}`}>
                        {formatPercent(holding.unrealized_pl_pct)}
                      </div>
                    </td>
                    <td className="py-4 text-right text-gray-600">
                      {holding.allocation_pct.toFixed(1)}%
                    </td>
                    <td className="py-4 text-right">
                      <Link
                        to={`/trade?symbol=${holding.symbol}`}
                        className="text-sm text-primary-600 hover:text-primary-700"
                      >
                        Trade
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Quick Links */}
      <div className="mt-6 flex gap-4 text-sm">
        <Link to="/orders" className="text-primary-600 hover:text-primary-700">
          Order History →
        </Link>
        <Link to="/transactions" className="text-primary-600 hover:text-primary-700">
          Transaction History →
        </Link>
      </div>
    </div>
  );
}
