import { useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ActivityIndicator } from 'react-native';
import { useQuery } from '@tanstack/react-query';
import { CandlestickChart } from 'react-native-wagmi-charts';
import apiClient from '../api/client';

interface Bar {
  timestamp: string;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
}

interface StockChartProps {
  symbol: string;
  height?: number;
}

type Timeframe = '1D' | '1W' | '1M' | '3M' | '1Y';

const timeframeConfig: Record<Timeframe, { timeframe: string; days: number }> = {
  '1D': { timeframe: '5Min', days: 1 },
  '1W': { timeframe: '1Hour', days: 7 },
  '1M': { timeframe: '1Day', days: 30 },
  '3M': { timeframe: '1Day', days: 90 },
  '1Y': { timeframe: '1Day', days: 365 },
};

export default function StockChart({ symbol, height = 300 }: StockChartProps) {
  const [timeframe, setTimeframe] = useState<Timeframe>('1M');

  const { data: barsData, isLoading } = useQuery({
    queryKey: ['bars', symbol, timeframe],
    queryFn: async () => {
      const config = timeframeConfig[timeframe];
      const end = new Date();
      const start = new Date();
      start.setDate(start.getDate() - config.days);

      const response = await apiClient.get<{ symbol: string; bars: Bar[] }>(
        `/market-data/bars/${symbol}`,
        {
          timeframe: config.timeframe,
          start: start.toISOString().split('T')[0],
          end: end.toISOString().split('T')[0],
          limit: 500,
        }
      );
      return response.bars;
    },
    enabled: !!symbol,
  });

  const chartData = barsData?.map((bar: Bar) => ({
    timestamp: new Date(bar.timestamp).getTime(),
    open: bar.open,
    high: bar.high,
    low: bar.low,
    close: bar.close,
  })) || [];

  const timeframeButtons: Timeframe[] = ['1D', '1W', '1M', '3M', '1Y'];

  return (
    <View style={styles.container}>
      {/* Timeframe selector */}
      <View style={styles.timeframeContainer}>
        {timeframeButtons.map((tf) => (
          <TouchableOpacity
            key={tf}
            style={[
              styles.timeframeButton,
              timeframe === tf && styles.timeframeButtonActive,
            ]}
            onPress={() => setTimeframe(tf)}
          >
            <Text
              style={[
                styles.timeframeText,
                timeframe === tf && styles.timeframeTextActive,
              ]}
            >
              {tf}
            </Text>
          </TouchableOpacity>
        ))}
      </View>

      {/* Chart */}
      <View style={[styles.chartContainer, { height }]}>
        {isLoading ? (
          <View style={styles.loadingContainer}>
            <ActivityIndicator size="large" color="#10B981" />
            <Text style={styles.loadingText}>Loading chart...</Text>
          </View>
        ) : chartData.length > 0 ? (
          <CandlestickChart.Provider data={chartData}>
            <CandlestickChart height={height}>
              <CandlestickChart.Candles
                positiveColor="#10B981"
                negativeColor="#EF4444"
              />
              <CandlestickChart.Crosshair>
                <CandlestickChart.Tooltip />
              </CandlestickChart.Crosshair>
            </CandlestickChart>
            <CandlestickChart.PriceText
              style={styles.priceText}
              precision={2}
            />
            <CandlestickChart.DatetimeText
              style={styles.dateText}
            />
          </CandlestickChart.Provider>
        ) : (
          <View style={styles.emptyContainer}>
            <Text style={styles.emptyText}>No chart data available</Text>
          </View>
        )}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    backgroundColor: '#fff',
    borderRadius: 16,
    padding: 16,
  },
  timeframeContainer: {
    flexDirection: 'row',
    gap: 8,
    marginBottom: 16,
  },
  timeframeButton: {
    paddingHorizontal: 12,
    paddingVertical: 8,
    borderRadius: 8,
    backgroundColor: '#F3F4F6',
  },
  timeframeButtonActive: {
    backgroundColor: '#10B981',
  },
  timeframeText: {
    fontSize: 13,
    fontWeight: '600',
    color: '#6B7280',
  },
  timeframeTextActive: {
    color: '#fff',
  },
  chartContainer: {
    justifyContent: 'center',
  },
  loadingContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  loadingText: {
    marginTop: 8,
    color: '#6B7280',
    fontSize: 14,
  },
  emptyContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  emptyText: {
    color: '#6B7280',
    fontSize: 14,
  },
  priceText: {
    fontSize: 14,
    fontWeight: '600',
    color: '#111827',
    textAlign: 'center',
  },
  dateText: {
    fontSize: 12,
    color: '#6B7280',
    textAlign: 'center',
  },
});
