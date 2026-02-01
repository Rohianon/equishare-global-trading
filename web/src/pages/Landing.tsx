import { Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

export default function Landing() {
  const { isAuthenticated } = useAuth();

  return (
    <div className="text-center py-16">
      <h1 className="text-4xl font-bold text-gray-900 mb-4">
        Welcome to EquiShare
      </h1>
      <p className="text-xl text-gray-600 mb-8 max-w-2xl mx-auto">
        Trade equities globally with seamless M-Pesa integration.
        Start investing in US stocks directly from Kenya.
      </p>

      <div className="flex justify-center gap-4 mb-16">
        {isAuthenticated ? (
          <Link to="/dashboard" className="btn btn-primary text-lg px-8 py-3">
            Go to Dashboard
          </Link>
        ) : (
          <>
            <Link to="/register" className="btn btn-primary text-lg px-8 py-3">
              Get Started
            </Link>
            <Link to="/login" className="btn btn-secondary text-lg px-8 py-3">
              Login
            </Link>
          </>
        )}
      </div>

      <div className="grid md:grid-cols-3 gap-8 max-w-4xl mx-auto">
        <div className="card">
          <div className="text-3xl mb-4">1</div>
          <h3 className="text-lg font-semibold mb-2">Register</h3>
          <p className="text-gray-600">
            Sign up with your phone number and verify via SMS OTP
          </p>
        </div>
        <div className="card">
          <div className="text-3xl mb-4">2</div>
          <h3 className="text-lg font-semibold mb-2">Fund Account</h3>
          <p className="text-gray-600">
            Deposit funds instantly using M-Pesa STK Push
          </p>
        </div>
        <div className="card">
          <div className="text-3xl mb-4">3</div>
          <h3 className="text-lg font-semibold mb-2">Trade</h3>
          <p className="text-gray-600">
            Buy and sell fractional shares of US equities
          </p>
        </div>
      </div>
    </div>
  );
}
