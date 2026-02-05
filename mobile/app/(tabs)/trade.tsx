import { useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TextInput,
  ScrollView,
  TouchableOpacity,
  FlatList,
  ActivityIndicator,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useQuery } from '@tanstack/react-query';
import { router } from 'expo-router';
import apiClient from '../../src/api/client';
import { tradingApi } from '../../src/api/trading';
import { useDebouncedCallback } from '../../src/hooks/useDebounce';
import type { Asset, Quote } from '../../src/types';

export default function TradeScreen() {
  const [searchQuery, setSearchQuery] = useState('');
  const [isSearching, setIsSearching] = useState(false);
  const [searchResults, setSearchResults] = useState<Asset[]>([]);

  // Featured stocks
  const { data: featuredData } = useQuery({
    queryKey: ['featured-stocks'],
    queryFn: () =>
      apiClient.get<{ quotes: Quote[] }>('/trading/quotes', {
        symbols: 'AAPL,GOOGL,MSFT,AMZN,TSLA,NVDA',
      }),
  });

  const handleSearch = useDebouncedCallback(async (query: string) => {
    if (query.length < 2) {
      setSearchResults([]);
      setIsSearching(false);
      return;
    }

    setIsSearching(true);
    try {
      const results = await tradingApi.searchAssets(query);
      setSearchResults(results.assets || []);
    } catch (error) {
      console.error('Search error:', error);
      setSearchResults([]);
    } finally {
      setIsSearching(false);
    }
  }, 300);

  const onSearchChange = (text: string) => {
    setSearchQuery(text);
    handleSearch(text);
  };

  const navigateToStock = (symbol: string) => {
    router.push({
      pathname: '/stock/[symbol]',
      params: { symbol },
    });
  };

  const featuredStocks = featuredData?.quotes || [];

  return (
    <SafeAreaView style={styles.container} edges={['top']}>
      <View style={styles.header}>
        <Text style={styles.title}>Trade</Text>
      </View>

      {/* Search Bar */}
      <View style={styles.searchContainer}>
        <View style={styles.searchBar}>
          <Text style={styles.searchIcon}>🔍</Text>
          <TextInput
            style={styles.searchInput}
            value={searchQuery}
            onChangeText={onSearchChange}
            placeholder="Search stocks..."
            placeholderTextColor="#9CA3AF"
            autoCapitalize="characters"
            autoCorrect={false}
          />
          {isSearching && <ActivityIndicator size="small" color="#10B981" />}
          {searchQuery.length > 0 && !isSearching && (
            <TouchableOpacity onPress={() => onSearchChange('')}>
              <Text style={styles.clearButton}>✕</Text>
            </TouchableOpacity>
          )}
        </View>
      </View>

      {/* Search Results */}
      {searchQuery.length > 0 ? (
        <FlatList
          data={searchResults}
          keyExtractor={(item) => item.id}
          style={styles.searchResults}
          ListEmptyComponent={
            !isSearching ? (
              <View style={styles.emptySearch}>
                <Text style={styles.emptySearchText}>
                  {searchQuery.length < 2
                    ? 'Enter at least 2 characters'
                    : 'No results found'}
                </Text>
              </View>
            ) : null
          }
          renderItem={({ item }: { item: Asset }) => (
            <TouchableOpacity
              style={styles.searchResultItem}
              onPress={() => navigateToStock(item.symbol)}
            >
              <View>
                <Text style={styles.resultSymbol}>{item.symbol}</Text>
                <Text style={styles.resultName} numberOfLines={1}>
                  {item.name}
                </Text>
              </View>
              <View style={styles.resultMeta}>
                <Text style={styles.resultExchange}>{item.exchange}</Text>
                {item.fractionable && (
                  <Text style={styles.fractionableTag}>Fractional</Text>
                )}
              </View>
            </TouchableOpacity>
          )}
        />
      ) : (
        <ScrollView style={styles.content}>
          {/* Featured Stocks */}
          <View style={styles.section}>
            <Text style={styles.sectionTitle}>Popular Stocks</Text>
            <View style={styles.stockGrid}>
              {featuredStocks.map((stock) => (
                <TouchableOpacity
                  key={stock.symbol}
                  style={styles.stockCard}
                  onPress={() => navigateToStock(stock.symbol)}
                >
                  <View style={styles.stockHeader}>
                    <Text style={styles.stockSymbol}>{stock.symbol}</Text>
                    <Text
                      style={[
                        styles.stockChange,
                        stock.change >= 0 ? styles.positive : styles.negative,
                      ]}
                    >
                      {stock.change >= 0 ? '+' : ''}
                      {stock.change_percent.toFixed(2)}%
                    </Text>
                  </View>
                  <Text style={styles.stockName} numberOfLines={1}>
                    {stock.name}
                  </Text>
                  <Text style={styles.stockPrice}>${stock.price.toFixed(2)}</Text>
                </TouchableOpacity>
              ))}
            </View>
          </View>

          {/* Categories */}
          <View style={styles.section}>
            <Text style={styles.sectionTitle}>Browse by Category</Text>
            <View style={styles.categoryGrid}>
              {[
                { icon: '💻', name: 'Technology', query: 'tech' },
                { icon: '🏥', name: 'Healthcare', query: 'health' },
                { icon: '💰', name: 'Finance', query: 'finance' },
                { icon: '⚡', name: 'Energy', query: 'energy' },
                { icon: '🛒', name: 'Consumer', query: 'consumer' },
                { icon: '🏭', name: 'Industrial', query: 'industrial' },
              ].map((category) => (
                <TouchableOpacity
                  key={category.name}
                  style={styles.categoryCard}
                  onPress={() => onSearchChange(category.query)}
                >
                  <Text style={styles.categoryIcon}>{category.icon}</Text>
                  <Text style={styles.categoryName}>{category.name}</Text>
                </TouchableOpacity>
              ))}
            </View>
          </View>
        </ScrollView>
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
    padding: 20,
    backgroundColor: '#fff',
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#111827',
  },
  searchContainer: {
    padding: 16,
    backgroundColor: '#fff',
    borderBottomWidth: 1,
    borderBottomColor: '#E5E7EB',
  },
  searchBar: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#F3F4F6',
    borderRadius: 12,
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  searchIcon: {
    fontSize: 16,
    marginRight: 12,
  },
  searchInput: {
    flex: 1,
    fontSize: 16,
    color: '#111827',
  },
  clearButton: {
    fontSize: 16,
    color: '#9CA3AF',
    padding: 4,
  },
  content: {
    flex: 1,
  },
  searchResults: {
    flex: 1,
    backgroundColor: '#fff',
  },
  searchResultItem: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#F3F4F6',
  },
  resultSymbol: {
    fontSize: 16,
    fontWeight: '600',
    color: '#111827',
  },
  resultName: {
    fontSize: 13,
    color: '#6B7280',
    marginTop: 2,
    maxWidth: 200,
  },
  resultMeta: {
    alignItems: 'flex-end',
    gap: 4,
  },
  resultExchange: {
    fontSize: 12,
    color: '#9CA3AF',
    backgroundColor: '#F3F4F6',
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 4,
  },
  fractionableTag: {
    fontSize: 10,
    color: '#10B981',
    backgroundColor: '#ECFDF5',
    paddingHorizontal: 6,
    paddingVertical: 2,
    borderRadius: 4,
  },
  emptySearch: {
    padding: 40,
    alignItems: 'center',
  },
  emptySearchText: {
    color: '#6B7280',
    fontSize: 14,
  },
  section: {
    padding: 20,
  },
  sectionTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#111827',
    marginBottom: 16,
  },
  stockGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 12,
  },
  stockCard: {
    backgroundColor: '#fff',
    borderRadius: 16,
    padding: 16,
    width: '48%',
  },
  stockHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  stockSymbol: {
    fontSize: 16,
    fontWeight: '700',
    color: '#111827',
  },
  stockChange: {
    fontSize: 12,
    fontWeight: '600',
    paddingHorizontal: 6,
    paddingVertical: 2,
    borderRadius: 4,
  },
  positive: {
    color: '#10B981',
    backgroundColor: '#ECFDF5',
  },
  negative: {
    color: '#EF4444',
    backgroundColor: '#FEF2F2',
  },
  stockName: {
    fontSize: 12,
    color: '#6B7280',
    marginBottom: 8,
  },
  stockPrice: {
    fontSize: 18,
    fontWeight: '600',
    color: '#111827',
  },
  categoryGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 12,
  },
  categoryCard: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    width: '31%',
    alignItems: 'center',
  },
  categoryIcon: {
    fontSize: 24,
    marginBottom: 8,
  },
  categoryName: {
    fontSize: 12,
    fontWeight: '500',
    color: '#374151',
    textAlign: 'center',
  },
});
