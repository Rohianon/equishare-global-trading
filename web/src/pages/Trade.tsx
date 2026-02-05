import { useState, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link, useNavigate } from 'react-router-dom';
import apiClient from '../api/client';

const POPULAR_STOCKS = ['AAPL', 'GOOGL', 'MSFT', 'AMZN', 'TSLA', 'NVDA', 'META', 'NFLX'];

export default function Trade() {
  const navigate = useNavigate();
  const [searchQuery, setSearchQuery] = useState('');
  const [showSearch, setShowSearch] = useState(false);
  const [debouncedQuery, setDebouncedQuery] = useState('');

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(searchQuery);
    }, 300);
    return () => clearTimeout(timer);
  }, [searchQuery]);

  const { data: searchResults, isLoading: searchLoading } = useQuery({
    queryKey: ['assetSearch', debouncedQuery],
    queryFn: () => apiClient.searchAssets(debouncedQuery),
    enabled: debouncedQuery.length >= 1,
  });

  const { data: balance } = useQuery({
    queryKey: ['walletBalance'],
    queryFn: () => apiClient.getWalletBalance(),
  });

  const handleSelectAsset = (symbol: string) => {
    setShowSearch(false);
    setSearchQuery('');
    navigate(`/stock/${symbol}`);
  };

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

      <div className="grid md:grid-cols-3 gap-6">
        {/* Search Section */}
        <div className="md:col-span-2 space-y-6">
          {/* Stock Search */}
          <div className="card">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Search Stocks
            </label>
            <div className="relative">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <svg className="h-5 w-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
              </div>
              <input
                type="text"
                className="input pl-10"
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
                <div className="absolute z-10 w-full mt-1 bg-white border border-gray-200 rounded-lg shadow-lg max-h-80 overflow-y-auto">
                  {searchLoading ? (
                    <div className="p-4 text-center text-gray-500">
                      <svg className="animate-spin h-5 w-5 mx-auto mb-2" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                      </svg>
                      Searching...
                    </div>
                  ) : searchResults?.assets && searchResults.assets.length > 0 ? (
                    searchResults.assets.map((asset) => (
                      <button
                        key={asset.id}
                        type="button"
                        className="w-full px-4 py-3 text-left hover:bg-gray-50 flex justify-between items-center border-b border-gray-100 last:border-0 transition-colors"
                        onClick={() => handleSelectAsset(asset.symbol)}
                      >
                        <div>
                          <span className="font-semibold text-gray-900">{asset.symbol}</span>
                          <span className="ml-2 text-sm text-gray-500">{asset.name}</span>
                        </div>
                        <div className="flex items-center gap-2">
                          {asset.fractionable && (
                            <span className="text-xs bg-green-100 text-green-700 px-2 py-0.5 rounded">
                              Fractional
                            </span>
                          )}
                          <span className="text-xs text-gray-400">{asset.exchange}</span>
                        </div>
                      </button>
                    ))
                  ) : debouncedQuery.length >= 1 ? (
                    <div className="p-4 text-center text-gray-500">No results found</div>
                  ) : null}
                </div>
              )}
            </div>
          </div>

          {/* Popular Stocks */}
          <div className="card">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Popular Stocks</h3>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
              {POPULAR_STOCKS.map((symbol) => (
                <button
                  key={symbol}
                  onClick={() => handleSelectAsset(symbol)}
                  className="p-4 border border-gray-200 rounded-lg hover:border-primary-500 hover:bg-primary-50 transition-colors text-center"
                >
                  <span className="font-semibold text-gray-900">{symbol}</span>
                </button>
              ))}
            </div>
          </div>

          {/* Categories */}
          <div className="card">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Browse by Sector</h3>
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
              {[
                { icon: '💻', name: 'Technology', stocks: ['AAPL', 'MSFT', 'GOOGL'] },
                { icon: '🏥', name: 'Healthcare', stocks: ['JNJ', 'PFE', 'UNH'] },
                { icon: '💰', name: 'Finance', stocks: ['JPM', 'BAC', 'V'] },
                { icon: '⚡', name: 'Energy', stocks: ['XOM', 'CVX', 'COP'] },
                { icon: '🛒', name: 'Consumer', stocks: ['WMT', 'COST', 'HD'] },
                { icon: '🚗', name: 'Auto', stocks: ['TSLA', 'F', 'GM'] },
              ].map((category) => (
                <div key={category.name} className="p-4 border border-gray-200 rounded-lg">
                  <div className="text-2xl mb-2">{category.icon}</div>
                  <h4 className="font-medium text-gray-900 mb-2">{category.name}</h4>
                  <div className="flex flex-wrap gap-1">
                    {category.stocks.map((stock) => (
                      <button
                        key={stock}
                        onClick={() => handleSelectAsset(stock)}
                        className="text-xs text-primary-600 hover:text-primary-700 hover:underline"
                      >
                        {stock}
                      </button>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Wallet Balance */}
          <div className="card bg-gradient-to-br from-primary-500 to-primary-600 text-white">
            <h4 className="text-sm font-medium opacity-90 mb-2">Available Balance</h4>
            <p className="text-3xl font-bold">
              {balance ? formatCurrency(balance.available, balance.currency) : '---'}
            </p>
            <Link
              to="/deposit"
              className="mt-4 inline-block text-sm bg-white/20 hover:bg-white/30 px-4 py-2 rounded-lg transition-colors"
            >
              Add Funds →
            </Link>
          </div>

          {/* Quick Links */}
          <div className="card">
            <h4 className="font-semibold text-gray-900 mb-4">Quick Links</h4>
            <div className="space-y-2">
              <Link
                to="/portfolio"
                className="flex items-center justify-between p-3 rounded-lg hover:bg-gray-50 transition-colors"
              >
                <span className="text-gray-700">Portfolio</span>
                <svg className="w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
              </Link>
              <Link
                to="/orders"
                className="flex items-center justify-between p-3 rounded-lg hover:bg-gray-50 transition-colors"
              >
                <span className="text-gray-700">Order History</span>
                <svg className="w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
              </Link>
              <Link
                to="/transactions"
                className="flex items-center justify-between p-3 rounded-lg hover:bg-gray-50 transition-colors"
              >
                <span className="text-gray-700">Transactions</span>
                <svg className="w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
              </Link>
            </div>
          </div>

          {/* Market Status */}
          <div className="card">
            <h4 className="font-semibold text-gray-900 mb-2">Market Status</h4>
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
              <span className="text-sm text-gray-600">US Markets Open</span>
            </div>
            <p className="text-xs text-gray-400 mt-2">
              Trading hours: 9:30 AM - 4:00 PM ET
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
