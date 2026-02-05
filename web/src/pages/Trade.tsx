import { useState, useEffect, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import apiClient, { type Asset, type PlaceOrderRequest } from '../api/client';

export default function Trade() {
  const queryClient = useQueryClient();
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedAsset, setSelectedAsset] = useState<Asset | null>(null);
  const [side, setSide] = useState<'buy' | 'sell'>('buy');
  const [amount, setAmount] = useState('');
  const [showSearch, setShowSearch] = useState(false);
  const [orderSuccess, setOrderSuccess] = useState<string | null>(null);
  const [orderError, setOrderError] = useState<string | null>(null);

  // Debounced search
  const [debouncedQuery, setDebouncedQuery] = useState('');

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(searchQuery);
    }, 300);
    return () => clearTimeout(timer);
  }, [searchQuery]);

  // Search assets
  const { data: searchResults, isLoading: searchLoading } = useQuery({
    queryKey: ['assetSearch', debouncedQuery],
    queryFn: () => apiClient.searchAssets(debouncedQuery),
    enabled: debouncedQuery.length >= 1,
  });

  // Get quote for selected asset
  const { data: quote, isLoading: quoteLoading, refetch: refetchQuote } = useQuery({
    queryKey: ['quote', selectedAsset?.symbol],
    queryFn: () => apiClient.getQuote(selectedAsset!.symbol),
    enabled: !!selectedAsset,
    refetchInterval: 10000, // Refresh every 10 seconds
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

  const handleSelectAsset = useCallback((asset: Asset) => {
    setSelectedAsset(asset);
    setSearchQuery(asset.symbol);
    setShowSearch(false);
    setOrderSuccess(null);
    setOrderError(null);
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedAsset || !amount) return;

    const amountNum = parseFloat(amount);
    if (isNaN(amountNum) || amountNum <= 0) {
      setOrderError('Please enter a valid amount');
      return;
    }

    placeOrderMutation.mutate({
      symbol: selectedAsset.symbol,
      side,
      amount: amountNum,
    });
  };

  const quickAmounts = [50, 100, 250, 500, 1000];

  const formatCurrency = (value: number, currency = 'USD') => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency,
    }).format(value);
  };

  return (
    <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">Trade Stocks</h1>
        <p className="mt-1 text-gray-600">Buy and sell US stocks with fractional shares</p>
      </div>

      <div className="grid md:grid-cols-2 gap-6">
        {/* Search and Quote Section */}
        <div className="space-y-6">
          {/* Stock Search */}
          <div className="card">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Search Stock
            </label>
            <div className="relative">
              <input
                type="text"
                className="input"
                placeholder="Search by symbol or name (e.g., AAPL, Apple)"
                value={searchQuery}
                onChange={(e) => {
                  setSearchQuery(e.target.value);
                  setShowSearch(true);
                }}
                onFocus={() => setShowSearch(true)}
              />

              {/* Search Results Dropdown */}
              {showSearch && searchQuery && (
                <div className="absolute z-10 w-full mt-1 bg-white border border-gray-200 rounded-lg shadow-lg max-h-60 overflow-y-auto">
                  {searchLoading ? (
                    <div className="p-4 text-center text-gray-500">Searching...</div>
                  ) : searchResults?.assets && searchResults.assets.length > 0 ? (
                    searchResults.assets.map((asset) => (
                      <button
                        key={asset.id}
                        type="button"
                        className="w-full px-4 py-3 text-left hover:bg-gray-50 flex justify-between items-center border-b border-gray-100 last:border-0"
                        onClick={() => handleSelectAsset(asset)}
                      >
                        <div>
                          <span className="font-semibold text-gray-900">{asset.symbol}</span>
                          <span className="ml-2 text-sm text-gray-500">{asset.name}</span>
                        </div>
                        <span className="text-xs text-gray-400">{asset.exchange}</span>
                      </button>
                    ))
                  ) : debouncedQuery.length >= 1 ? (
                    <div className="p-4 text-center text-gray-500">No results found</div>
                  ) : null}
                </div>
              )}
            </div>
          </div>

          {/* Quote Card */}
          {selectedAsset && (
            <div className="card">
              <div className="flex justify-between items-start mb-4">
                <div>
                  <h3 className="text-xl font-bold text-gray-900">{selectedAsset.symbol}</h3>
                  <p className="text-sm text-gray-500">{selectedAsset.name}</p>
                </div>
                <button
                  type="button"
                  onClick={() => refetchQuote()}
                  className="text-sm text-primary-600 hover:text-primary-700"
                >
                  Refresh
                </button>
              </div>

              {quoteLoading ? (
                <div className="animate-pulse space-y-3">
                  <div className="h-8 bg-gray-200 rounded w-1/3"></div>
                  <div className="h-4 bg-gray-200 rounded w-1/2"></div>
                </div>
              ) : quote ? (
                <div className="space-y-4">
                  <div>
                    <span className="text-3xl font-bold text-gray-900">
                      {formatCurrency(quote.mid_price)}
                    </span>
                  </div>

                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                      <span className="text-gray-500">Bid</span>
                      <p className="font-medium text-gray-900">{formatCurrency(quote.bid_price)}</p>
                    </div>
                    <div>
                      <span className="text-gray-500">Ask</span>
                      <p className="font-medium text-gray-900">{formatCurrency(quote.ask_price)}</p>
                    </div>
                    <div>
                      <span className="text-gray-500">Spread</span>
                      <p className="font-medium text-gray-900">{formatCurrency(quote.spread)}</p>
                    </div>
                    <div>
                      <span className="text-gray-500">Updated</span>
                      <p className="font-medium text-gray-900">
                        {new Date(quote.timestamp).toLocaleTimeString()}
                      </p>
                    </div>
                  </div>

                  {selectedAsset.fractionable && (
                    <div className="text-xs text-green-600 bg-green-50 px-2 py-1 rounded inline-block">
                      Fractional shares available
                    </div>
                  )}
                </div>
              ) : null}
            </div>
          )}

          {/* Wallet Balance */}
          <div className="card bg-gray-50">
            <h4 className="text-sm font-medium text-gray-700 mb-2">Available Balance</h4>
            <p className="text-2xl font-bold text-gray-900">
              {balance ? formatCurrency(balance.available, balance.currency) : '---'}
            </p>
            <Link to="/deposit" className="text-sm text-primary-600 hover:text-primary-700 mt-2 inline-block">
              Add funds →
            </Link>
          </div>
        </div>

        {/* Order Form Section */}
        <div className="card">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Place Order</h3>

          {!selectedAsset ? (
            <div className="text-center py-8 text-gray-500">
              <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
              <p className="mt-2">Search for a stock to get started</p>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-6">
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
              {amount && quote && (
                <div className="bg-gray-50 rounded-lg p-4 space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-gray-500">Estimated shares</span>
                    <span className="font-medium">
                      ~{(parseFloat(amount) / quote.mid_price).toFixed(6)} {selectedAsset.symbol}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-500">Price per share</span>
                    <span className="font-medium">{formatCurrency(quote.mid_price)}</span>
                  </div>
                  <div className="flex justify-between border-t border-gray-200 pt-2 mt-2">
                    <span className="text-gray-700 font-medium">Total</span>
                    <span className="font-bold">{formatCurrency(parseFloat(amount))}</span>
                  </div>
                </div>
              )}

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
                  : `${side === 'buy' ? 'Buy' : 'Sell'} ${selectedAsset.symbol}`}
              </button>
            </form>
          )}
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
