// playground/src/components/JobStatus.jsx
import React, { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeftIcon, RefreshIcon, ClockIcon, CheckCircleIcon, XCircleIcon, AlertCircleIcon } from 'lucide-react';

const JobStatus = () => {
  const { jobId } = useParams();
  const navigate = useNavigate();
  const [job, setJob] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [lastUpdated, setLastUpdated] = useState(null);
  const wsRef = useRef(null);
  const [wsConnected, setWsConnected] = useState(false);

  const fetchJobStatus = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`${import.meta.env.VITE_API_URL}/api/v1/jobs/${jobId}`);
      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || 'Failed to fetch job status');
      }

      setJob(data.data);
      setLastUpdated(new Date());
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const setupWebSocket = () => {
    // Close existing connection if any
    if (wsRef.current) {
      wsRef.current.close();
    }

    // Create new WebSocket connection
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = import.meta.env.VITE_API_URL.replace(/^https?:\/\//, '');
    const wsUrl = `${protocol}//${host}/ws/jobs`;
    
    wsRef.current = new WebSocket(wsUrl);
    
    wsRef.current.onopen = () => {
      console.log('WebSocket connected');
      setWsConnected(true);
    };
    
    wsRef.current.onclose = () => {
      console.log('WebSocket disconnected');
      setWsConnected(false);
      
      // Try to reconnect after a delay
      setTimeout(() => {
        if (document.visibilityState !== 'hidden') {
          setupWebSocket();
        }
      }, 5000);
    };
    
    wsRef.current.onmessage = (event) => {
      try {
        const update = JSON.parse(event.data);
        
        // Check if this update is for our job
        if (update.job_id === jobId) {
          // Refresh job status
          fetchJobStatus();
        }
      } catch (err) {
        console.error('Error processing WebSocket message:', err);
      }
    };
    
    wsRef.current.onerror = (error) => {
      console.error('WebSocket error:', error);
      setWsConnected(false);
    };
  };

  // Initial fetch and WebSocket setup
  useEffect(() => {
    fetchJobStatus();
    setupWebSocket();
    
    // Cleanup WebSocket on unmount
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [jobId]);

  const handleBack = () => {
    navigate('/');
  };

  const handleRefresh = () => {
    fetchJobStatus();
  };

  // Format timestamp
  const formatTime = (timestamp) => {
    if (!timestamp) return 'N/A';
    return new Date(timestamp).toLocaleTimeString();
  };

  // Get status badge
  const getStatusBadge = () => {
    if (!job) return null;

    switch (job.status) {
      case 'pending':
        return (
          <span className="bg-blue-800 text-blue-200 px-3 py-1 rounded-full flex items-center">
            <ClockIcon size={16} className="mr-1" />
            Pending
          </span>
        );
      case 'running':
        return (
          <span className="bg-yellow-800 text-yellow-200 px-3 py-1 rounded-full flex items-center">
            <RefreshIcon size={16} className="mr-1 animate-spin" />
            Running
          </span>
        );
      case 'completed':
        return (
          <span className="bg-green-800 text-green-200 px-3 py-1 rounded-full flex items-center">
            <CheckCircleIcon size={16} className="mr-1" />
            Completed
          </span>
        );
      case 'failed':
        return (
          <span className="bg-red-800 text-red-200 px-3 py-1 rounded-full flex items-center">
            <XCircleIcon size={16} className="mr-1" />
            Failed
          </span>
        );
      case 'cancelled':
        return (
          <span className="bg-gray-800 text-gray-200 px-3 py-1 rounded-full flex items-center">
            <AlertCircleIcon size={16} className="mr-1" />
            Cancelled
          </span>
        );
      default:
        return (
          <span className="bg-purple-800 text-purple-200 px-3 py-1 rounded-full flex items-center">
            {job.status}
          </span>
        );
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
          Job Status
        </h1>

        <div className="border-4 border-white p-6 rounded-xl">
          <div className="flex justify-between items-center mb-6">
            <h2 className="text-xl">Job ID: <span className="font-mono text-cyan-400">{jobId}</span></h2>
            <div className="flex items-center space-x-2">
              <button 
                onClick={handleRefresh}
                className="bg-gray-800 hover:bg-gray-700 text-white p-1 rounded"
                title="Refresh"
              >
                <RefreshIcon size={20} />
              </button>
              {wsConnected ? (
                <span className="text-green-400 text-xs">● Live</span>
              ) : (
                <span className="text-red-400 text-xs">● Offline</span>
              )}
            </div>
          </div>

          {loading && !job && (
            <div className="flex justify-center items-center p-8">
              <RefreshIcon size={32} className="animate-spin text-yellow-400" />
            </div>
          )}

          {error && (
            <div className="p-4 bg-red-900 border-2 border-red-700 rounded">
              <h3 className="text-red-400 font-bold mb-2">Error</h3>
              <p className="text-white">{error}</p>
            </div>
          )}

          {job && (
            <div className="space-y-6">
              <div className="flex items-center justify-between">
                <div className="text-lg">Status: </div>
                {getStatusBadge()}
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="text-yellow-400 mb-1">Type</div>
                  <div className="font-mono bg-gray-900 p-2 rounded">{job.type}</div>
                </div>
                <div>
                  <div className="text-yellow-400 mb-1">Priority</div>
                  <div className="font-mono bg-gray-900 p-2 rounded">
                    {job.priority === 0 ? 'High' : job.priority === 1 ? 'Normal' : 'Low'}
                  </div>
                </div>
                <div>
                  <div className="text-yellow-400 mb-1">Created At</div>
                  <div className="font-mono bg-gray-900 p-2 rounded">{new Date(job.created_at).toLocaleString()}</div>
                </div>
                <div>
                  <div className="text-yellow-400 mb-1">Attempts</div>
                  <div className="font-mono bg-gray-900 p-2 rounded">{job.attempts || 0}</div>
                </div>
              </div>

              {job.scheduled_at && (
                <div>
                  <div className="text-yellow-400 mb-1">Scheduled For</div>
                  <div className="font-mono bg-gray-900 p-2 rounded">{new Date(job.scheduled_at).toLocaleString()}</div>
                </div>
              )}

              {job.last_error && (
                <div>
                  <div className="text-red-400 mb-1">Last Error</div>
                  <div className="font-mono bg-gray-900 text-red-300 p-2 rounded overflow-auto">{job.last_error}</div>
                </div>
              )}

              <div>
                <div className="text-yellow-400 mb-1">Data</div>
                <pre className="font-mono bg-gray-900 p-3 rounded overflow-auto text-sm">
                  {JSON.stringify(job.data, null, 2)}
                </pre>
              </div>

              {lastUpdated && (
                <div className="text-gray-500 text-sm text-right">
                  Last updated: {lastUpdated.toLocaleTimeString()}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </main>
  );
};

export default JobStatus;