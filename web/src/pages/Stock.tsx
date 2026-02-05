import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import apiClient, { type PlaceOrderRequest } from '../api/client';
import StockChart from '../components/StockChart';
import { useWebSocket } from '../hooks/useWebSocket';

export default function Stock() {
  const { symbol } = useParams<{ symbol: string }>();
  const queryClient = useQueryClient();
  const { subscribe, unsubscribe, quotes, isConnected } = useWebSocket();

  const [side, setSide] = useState<'buy' | 'sell'>('buy');
  const [amount, setAmount] = useState('');
  const [orderSuccess, setOrderSuccess] = useState<string | null>(null);
  const [orderError, setOrderError] = useState<string | null>(null);

  // Subscribe to real-time updates
  useEffect(() => {
    if (symbol) {
      subscribe([symbol]);
      return () => unsubscribe([symbol]);
    }
  }, [symbol, subscribe, unsubscribe]);

  // Get asset details
  const { data: asset } = useQuery({
    queryKey: ['asset', symbol],
    queryFn: () => apiClient.getAsset(symbol!),
    enabled: !!symbol,
  });

  // Get quote (fallback if websocket not connected)
  const { data: quote, refetch: refetchQuote } = useQuery({
    queryKey: ['quote', symbol],
    queryFn: () => apiClient.getQuote(symbol!),
    enabled: !!symbol,
    refetchInterval: isConnected ? false : 10000, // Only poll if websocket disconnected
  });

  // Get wallet balance
  const { data: balance } = useQuery({
    queryKey: ['walletBalance'],
    queryFn: () => apiClient.getWalletBalance(),
  });

  // Place order mutation
  const placeOrderMutation = useMutation({
    mutationFn: (request: PlaceOrderRequest) => apiClient.placeOrder(request),
    onSuccess: (data) => {
      setOrderSuccess(`Order placed successfully! Order ID: ${data.order_id}`);
      setOrderError(null);
      setAmount('');
      queryClient.invalidateQueries({ queryKey: ['walletBalance'] });
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      queryClient.invalidateQueries({ queryKey: ['portfolio'] });
    },
    onError: (error: Error & { response?: { data?: { error?: string; message?: string } } }) => {
      setOrderError(error.response?.data?.message || error.response?.data?.error || 'Failed to place order');
      setOrderSuccess(null);
    },
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!symbol || !amount) return;

    const amountNum = parseFloat(amount);
    if (isNaN(amountNum) || amountNum <= 0) {
      setOrderError('Please enter a valid amount');
      return;
    }

    placeOrderMutation.mutate({
      symbol,
      side,
      amount: amountNum,
    });
  };

  // Use real-time quote if available, otherwise fall back to REST quote
  const liveQuote = symbol ? quotes.get(symbol.toUpperCase()) : null;
  const currentQuote = liveQuote || quote;

  const formatCurrency = (value: number, currency = 'USD') => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency,
    }).format(value);
  };

  const quickAmounts = [50, 100, 250, 500, 1000];

  if (!symbol) {
    return <div className="p-8 text-center text-gray-500">Invalid symbol</div>;
  }

  const estimatedShares = amount && currentQuote
    ? parseFloat(amount) / currentQuote.mid_price
    : 0;

  return (
    <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <Link to="/trade" className="text-sm text-primary-600 hover:text-primary-700 mb-2 inline-block">
            ← Back to Trade
          </Link>
          <h1 className="text-3xl font-bold text-gray-900">{symbol}</h1>
          {asset && <p className="text-gray-500 mt-1">{asset.name}</p>}
        </div>
        <div className="flex items-center gap-2">
          <span className={`inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium ${
            isConnected ? 'bg-green-100 text-green-700' : 'bg-yellow-100 text-yellow-700'
          }`}>
            <span className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-yellow-500'}`} />
            {isConnected ? 'Live' : 'Delayed'}
          </span>
          <button
            onClick={() => refetchQuote()}
            className="text-sm text-primary-600 hover:text-primary-700"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Price display */}
      {currentQuote && (
        <div className="bg-white rounded-xl p-6 mb-6">
          <div className="flex items-baseline gap-4">
            <span className="text-4xl font-bold text-gray-900">
              {formatCurrency(currentQuote.mid_price)}
            </span>
            {liveQuote && (
              <span className="text-xs text-gray-400">
                Updated {new Date(liveQuote.timestamp).toLocaleTimeString()}
              </span>
            )}
          </div>
          <div className="grid grid-cols-4 gap-4 mt-4 text-sm">
            <div>
              <span className="text-gray-500">Bid</span>
              <p className="font-medium">{formatCurrency(currentQuote.bid_price)}</p>
            </div>
            <div>
              <span className="text-gray-500">Ask</span>
              <p className="font-medium">{formatCurrency(currentQuote.ask_price)}</p>
            </div>
            <div>
              <span className="text-gray-500">Spread</span>
              <p className="font-medium">{formatCurrency(currentQuote.spread)}</p>
            </div>
            <div>
              <span className="text-gray-500">Type</span>
              <p className="font-medium">
                {asset?.fractionable ? (
                  <span className="text-green-600">Fractional</span>
                ) : (
                  <span>Whole shares</span>
                )}
              </p>
            </div>
          </div>
        </div>
      )}

      <div className="grid lg:grid-cols-3 gap-6">
        {/* Chart */}
        <div className="lg:col-span-2">
          <StockChart symbol={symbol} height={400} />
        </div>

        {/* Order Form */}
        <div className="card">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Place Order</h3>

          <form onSubmit={handleSubmit} className="space-y-4">
            {/* Buy/Sell Toggle */}
            <div className="flex rounded-lg border border-gray-200 p-1">
              <button
                type="button"
                className={`flex-1 py-2 px-4 rounded-md font-medium transition-colors ${
                  side === 'buy'
                    ? 'bg-green-500 text-white'
                    : 'text-gray-600 hover:bg-gray-100'
                }`}
                onClick={() => setSide('buy')}
              >
                Buy
              </button>
              <button
                type="button"
                className={`flex-1 py-2 px-4 rounded-md font-medium transition-colors ${
                  side === 'sell'
                    ? 'bg-red-500 text-white'
                    : 'text-gray-600 hover:bg-gray-100'
                }`}
                onClick={() => setSide('sell')}
              >
                Sell
              </button>
            </div>

            {/* Amount Input */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Amount (USD)
              </label>
              <div className="relative">
                <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">$</span>
                <input
                  type="number"
                  className="input pl-8"
                  placeholder="0.00"
                  value={amount}
                  onChange={(e) => setAmount(e.target.value)}
                  min="1"
                  step="0.01"
                />
              </div>
            </div>

            {/* Quick Amount Buttons */}
            <div className="flex flex-wrap gap-2">
              {quickAmounts.map((amt) => (
                <button
                  key={amt}
                  type="button"
                  className="px-3 py-1.5 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                  onClick={() => setAmount(amt.toString())}
                >
                  ${amt}
                </button>
              ))}
            </div>

            {/* Order Preview */}
            {amount && currentQuote && parseFloat(amount) > 0 && (
              <div className="bg-gray-50 rounded-lg p-4 space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-gray-500">Estimated shares</span>
                  <span className="font-medium">
                    ~{estimatedShares.toFixed(6)} {symbol}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">Price per share</span>
                  <span className="font-medium">{formatCurrency(currentQuote.mid_price)}</span>
                </div>
                <div className="flex justify-between border-t border-gray-200 pt-2 mt-2">
                  <span className="text-gray-700 font-medium">Total</span>
                  <span className="font-bold">{formatCurrency(parseFloat(amount))}</span>
                </div>
              </div>
            )}

            {/* Balance */}
            <div className="bg-gray-50 rounded-lg p-3">
              <p className="text-xs text-gray-500">Available Balance</p>
              <p className="text-lg font-semibold">
                {balance ? formatCurrency(balance.available, balance.currency) : '---'}
              </p>
            </div>

            {/* Error/Success Messages */}
            {orderError && (
              <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-sm text-red-700">
                {orderError}
              </div>
            )}

            {orderSuccess && (
              <div className="bg-green-50 border border-green-200 rounded-lg p-4 text-sm text-green-700">
                {orderSuccess}
                <Link to="/orders" className="block mt-2 text-green-600 hover:text-green-700 font-medium">
                  View Orders →
                </Link>
              </div>
            )}

            {/* Submit Button */}
            <button
              type="submit"
              disabled={!amount || placeOrderMutation.isPending}
              className={`w-full py-3 px-4 rounded-lg font-medium text-white transition-colors ${
                side === 'buy'
                  ? 'bg-green-500 hover:bg-green-600 disabled:bg-green-300'
                  : 'bg-red-500 hover:bg-red-600 disabled:bg-red-300'
              }`}
            >
              {placeOrderMutation.isPending
                ? 'Placing Order...'
                : `${side === 'buy' ? 'Buy' : 'Sell'} ${symbol}`}
            </button>
          </form>
        </div>
      </div>

      {/* Quick Links */}
      <div className="mt-8 flex gap-4 text-sm">
        <Link to="/portfolio" className="text-primary-600 hover:text-primary-700">
          View Portfolio →
        </Link>
        <Link to="/orders" className="text-primary-600 hover:text-primary-700">
          Order History →
        </Link>
      </div>
    </div>
  );
}
