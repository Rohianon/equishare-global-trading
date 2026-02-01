import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import type { AxiosError } from 'axios';
import apiClient, { type ApiError } from '../api/client';
import { useAuth } from '../contexts/AuthContext';

export default function Deposit() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const [amount, setAmount] = useState('');
  const [phone, setPhone] = useState(user?.phone || '');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const quickAmounts = [100, 500, 1000, 5000];

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    const numAmount = parseFloat(amount);
    if (isNaN(numAmount) || numAmount < 10) {
      setError('Minimum deposit is KES 10');
      return;
    }

    if (numAmount > 150000) {
      setError('Maximum deposit is KES 150,000');
      return;
    }

    setIsLoading(true);

    try {
      const response = await apiClient.initiateDeposit(numAmount, phone);
      setSuccess(`STK Push sent! ${response.message || 'Check your phone to complete the payment.'}`);
      setAmount('');
    } catch (err) {
      const axiosError = err as AxiosError<ApiError>;
      setError(axiosError.response?.data?.error || 'Failed to initiate deposit. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="max-w-lg mx-auto">
      <button
        onClick={() => navigate('/dashboard')}
        className="text-gray-600 hover:text-gray-900 mb-4 flex items-center"
      >
        <span className="mr-1">&larr;</span> Back to Dashboard
      </button>

      <div className="card">
        <h2 className="text-2xl font-bold mb-2">Deposit via M-Pesa</h2>
        <p className="text-gray-600 mb-6">
          An STK push will be sent to your phone. Enter your M-Pesa PIN to complete.
        </p>

        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg mb-4">
            {error}
          </div>
        )}

        {success && (
          <div className="bg-green-50 border border-green-200 text-green-700 px-4 py-3 rounded-lg mb-4">
            {success}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="amount" className="block text-sm font-medium text-gray-700 mb-1">
              Amount (KES)
            </label>
            <input
              type="number"
              id="amount"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              placeholder="Enter amount"
              min="10"
              max="150000"
              className="input"
              required
            />
            <div className="flex gap-2 mt-2">
              {quickAmounts.map((amt) => (
                <button
                  key={amt}
                  type="button"
                  onClick={() => setAmount(amt.toString())}
                  className="px-3 py-1 text-sm bg-gray-100 hover:bg-gray-200 rounded-full text-gray-700"
                >
                  {amt.toLocaleString()}
                </button>
              ))}
            </div>
          </div>

          <div>
            <label htmlFor="phone" className="block text-sm font-medium text-gray-700 mb-1">
              M-Pesa Phone Number
            </label>
            <input
              type="tel"
              id="phone"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              placeholder="+254712345678"
              className="input"
              required
            />
            <p className="text-xs text-gray-500 mt-1">
              Number registered with M-Pesa
            </p>
          </div>

          <button
            type="submit"
            disabled={isLoading}
            className="btn btn-success w-full text-lg py-3"
          >
            {isLoading ? 'Sending STK Push...' : `Deposit KES ${amount || '0'}`}
          </button>
        </form>

        <div className="mt-6 p-4 bg-gray-50 rounded-lg">
          <h3 className="font-medium text-gray-900 mb-2">How it works</h3>
          <ol className="text-sm text-gray-600 space-y-1 list-decimal list-inside">
            <li>Enter amount and your M-Pesa phone number</li>
            <li>Click Deposit - you'll receive an STK push</li>
            <li>Enter your M-Pesa PIN on your phone</li>
            <li>Funds will be credited to your wallet</li>
          </ol>
        </div>
      </div>
    </div>
  );
}
