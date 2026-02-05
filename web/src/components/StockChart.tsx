import { useEffect, useRef, useState } from 'react';
import { createChart, ColorType, CandlestickSeries } from 'lightweight-charts';
import type { IChartApi, ISeriesApi, CandlestickData, Time } from 'lightweight-charts';
import { useQuery } from '@tanstack/react-query';
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

type Timeframe = '1D' | '1W' | '1M' | '3M' | '1Y' | 'ALL';

const timeframeConfig: Record<Timeframe, { timeframe: string; days: number }> = {
  '1D': { timeframe: '5Min', days: 1 },
  '1W': { timeframe: '1Hour', days: 7 },
  '1M': { timeframe: '1Day', days: 30 },
  '3M': { timeframe: '1Day', days: 90 },
  '1Y': { timeframe: '1Day', days: 365 },
  'ALL': { timeframe: '1Week', days: 365 * 5 },
};

export default function StockChart({ symbol, height = 400 }: StockChartProps) {
  const chartContainerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null);
  const [timeframe, setTimeframe] = useState<Timeframe>('1M');

  const { data: barsData, isLoading } = useQuery({
    queryKey: ['bars', symbol, timeframe],
    queryFn: async () => {
      const config = timeframeConfig[timeframe];
      const end = new Date();
      const start = new Date();
      start.setDate(start.getDate() - config.days);

      const response = await apiClient.getBars(symbol, {
        timeframe: config.timeframe,
        start: start.toISOString().split('T')[0],
        end: end.toISOString().split('T')[0],
        limit: 1000,
      });
      return response.bars;
    },
    enabled: !!symbol,
  });

  // Initialize chart
  useEffect(() => {
    if (!chartContainerRef.current) return;

    const chart = createChart(chartContainerRef.current, {
      layout: {
        background: { type: ColorType.Solid, color: '#ffffff' },
        textColor: '#333',
      },
      grid: {
        vertLines: { color: '#f0f0f0' },
        horzLines: { color: '#f0f0f0' },
      },
      width: chartContainerRef.current.clientWidth,
      height,
      timeScale: {
        borderColor: '#e5e7eb',
        timeVisible: true,
        secondsVisible: false,
      },
      rightPriceScale: {
        borderColor: '#e5e7eb',
      },
      crosshair: {
        mode: 1, // Normal mode
      },
    });

    const candlestickSeries = chart.addSeries(CandlestickSeries, {
      upColor: '#10B981',
      downColor: '#EF4444',
      borderDownColor: '#EF4444',
      borderUpColor: '#10B981',
      wickDownColor: '#EF4444',
      wickUpColor: '#10B981',
    });

    chartRef.current = chart;
    seriesRef.current = candlestickSeries;

    // Handle resize
    const handleResize = () => {
      if (chartContainerRef.current && chartRef.current) {
        chartRef.current.applyOptions({
          width: chartContainerRef.current.clientWidth,
        });
      }
    };

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      chart.remove();
    };
  }, [height]);

  // Update chart data
  useEffect(() => {
    if (!seriesRef.current || !barsData || barsData.length === 0) return;

    const chartData: CandlestickData<Time>[] = barsData.map((bar: Bar) => ({
      time: (new Date(bar.timestamp).getTime() / 1000) as Time,
      open: bar.open,
      high: bar.high,
      low: bar.low,
      close: bar.close,
    }));

    seriesRef.current.setData(chartData);
    chartRef.current?.timeScale().fitContent();
  }, [barsData]);

  const timeframeButtons: Timeframe[] = ['1D', '1W', '1M', '3M', '1Y', 'ALL'];

  return (
    <div className="bg-white rounded-xl p-4">
      {/* Timeframe selector */}
      <div className="flex gap-2 mb-4">
        {timeframeButtons.map((tf) => (
          <button
            key={tf}
            onClick={() => setTimeframe(tf)}
            className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors ${
              timeframe === tf
                ? 'bg-primary-600 text-white'
                : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
            }`}
          >
            {tf}
          </button>
        ))}
      </div>

      {/* Chart container */}
      <div className="relative">
        {isLoading && (
          <div className="absolute inset-0 flex items-center justify-center bg-white/80 z-10">
            <div className="flex items-center gap-2 text-gray-500">
              <svg className="animate-spin h-5 w-5" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              Loading chart...
            </div>
          </div>
        )}
        <div ref={chartContainerRef} />
      </div>

      {/* No data message */}
      {!isLoading && (!barsData || barsData.length === 0) && (
        <div className="flex items-center justify-center h-[300px] text-gray-500">
          No chart data available for {symbol}
        </div>
      )}
    </div>
  );
}
