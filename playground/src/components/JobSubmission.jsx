// playground/src/components/JobSubmission.jsx
import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { BoltIcon, ArrowLeftIcon } from 'lucide-react';

const JobSubmission = () => {
  const navigate = useNavigate();
  const [jobType, setJobType] = useState('echo');
  const [jobData, setJobData] = useState('{\n  "message": "Hello World"\n}');
  const [priority, setPriority] = useState(1);
  const [delay, setDelay] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState(null);
  const [result, setResult] = useState(null);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError(null);
    setResult(null);
    setSubmitting(true);

    try {
      // Parse job data
      let parsedData;
      try {
        parsedData = JSON.parse(jobData);
      } catch (err) {
        throw new Error('Invalid JSON in job data');
      }

      // Create request body
      const requestBody = {
        type: jobType,
        data: parsedData,
        priority: parseInt(priority),
        delay_seconds: parseInt(delay)
      };

      // Submit job
      const response = await fetch(`${import.meta.env.VITE_API_URL}/api/v1/jobs`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestBody),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || 'Failed to submit job');
      }

      setResult(data);
    } catch (err) {
      setError(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  const handleBack = () => {
    navigate('/');
  };

  const handleCheckStatus = () => {
    if (result && result.data && result.data.job_id) {
      navigate(`/job/${result.data.job_id}`);
    }
  };

  return (
    <main className="min-h-screen bg-black text-white flex flex-col items-center font-arcade">
      <div className="w-full max-w-4xl p-4">
        <button 
          onClick={handleBack}
          className="text-yellow-400 hover:text-yellow-300 flex items-center mb-8"
        >
          <ArrowLeftIcon size={16} className="mr-2" />
          Back to Main Menu
        </button>

        <h1 className="text-4xl uppercase tracking-wider font-extrabold text-transparent bg-gradient-to-r from-yellow-400 via-orange-500 to-pink-600 bg-clip-text mb-8 text-center">
          Submit a Job
        </h1>

        <div className="border-4 border-white p-6 rounded-xl">
          <form onSubmit={handleSubmit}>
            <div className="mb-4">
              <label className="block text-yellow-400 mb-2">Job Type</label>
              <select
                value={jobType}
                onChange={(e) => setJobType(e.target.value)}
                className="w-full bg-gray-900 text-white border-2 border-gray-700 rounded p-2"
              >
                <option value="echo">Echo</option>
                <option value="sleep">Sleep</option>
              </select>
            </div>

            <div className="mb-4">
              <label className="block text-yellow-400 mb-2">Job Data (JSON)</label>
              <textarea
                value={jobData}
                onChange={(e) => setJobData(e.target.value)}
                className="w-full bg-gray-900 text-white border-2 border-gray-700 rounded p-2 font-mono h-32"
              />
            </div>

            <div className="grid grid-cols-2 gap-4 mb-4">
              <div>
                <label className="block text-yellow-400 mb-2">Priority</label>
                <select
                  value={priority}
                  onChange={(e) => setPriority(e.target.value)}
                  className="w-full bg-gray-900 text-white border-2 border-gray-700 rounded p-2"
                >
                  <option value="0">High</option>
                  <option value="1">Normal</option>
                  <option value="2">Low</option>
                </select>
              </div>
              
              <div>
                <label className="block text-yellow-400 mb-2">Delay (seconds)</label>
                <input
                  type="number"
                  min="0"
                  value={delay}
                  onChange={(e) => setDelay(e.target.value)}
                  className="w-full bg-gray-900 text-white border-2 border-gray-700 rounded p-2"
                />
              </div>
            </div>

            <div className="flex justify-center mt-6">
              <button
                type="submit"
                disabled={submitting}
                className="bg-yellow-500 hover:bg-yellow-400 text-black font-bold py-2 px-8 rounded flex items-center disabled:opacity-50 disabled:hover:bg-yellow-500"
              >
                {submitting ? (
                  <>
                    <BoltIcon size={20} className="mr-2 animate-pulse" />
                    Submitting...
                  </>
                ) : (
                  <>
                    <BoltIcon size={20} className="mr-2" />
                    Submit Job
                  </>
                )}
              </button>
            </div>
          </form>

          {error && (
            <div className="mt-8 p-4 bg-red-900 border-2 border-red-700 rounded">
              <h3 className="text-red-400 font-bold mb-2">Error</h3>
              <p className="text-white">{error}</p>
            </div>
          )}

          {result && (
            <div className="mt-8 p-4 bg-green-900 border-2 border-green-700 rounded">
              <h3 className="text-green-400 font-bold mb-2">Success!</h3>
              <p className="text-white mb-4">
                Job ID: <span className="font-mono">{result.data.job_id}</span>
              </p>
              <button
                onClick={handleCheckStatus}
                className="bg-green-700 hover:bg-green-600 text-white py-1 px-4 rounded text-sm"
              >
                Check Job Status
              </button>
            </div>
          )}
        </div>
      </div>
    </main>
  );
};

export default JobSubmission;