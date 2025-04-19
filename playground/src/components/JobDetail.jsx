import React from 'react';
import { useParams, Link } from 'react-router-dom';
import { RefreshCw, ArrowLeft, Clock, CheckCircle, XCircle, AlertTriangle } from 'lucide-react';

import useApi from '../hooks/useApi';
import useWebSocket from '../hooks/useWebSocket';
import { jobsApi } from '../services/api';
import { formatDate, getStatusColor, getStatusBgColor, formatJSON, formatPriority } from '../utils/format';

const JobDetail = () => {
  const { jobId } = useParams();
  const { data, loading, error, execute, setData } = useApi(
    () => jobsApi.getJob(jobId),
    [jobId]
  );

  // Connect to WebSocket for real-time updates
  const { connected: wsConnected, message: wsMessage } = useWebSocket(`/jobs/${jobId}`);

  // Update data when receiving WebSocket message
  React.useEffect(() => {
    if (wsMessage) {
      // Check if the message is a job update for this job
      if (wsMessage.type === 'job_update' && wsMessage.job_id === jobId) {
        execute();
      }
    }
  }, [wsMessage, jobId, execute]);

  // Get status icon based on job status
  const getStatusIcon = (status, size = 18) => {
    switch (status?.toLowerCase()) {
      case 'completed':
        return <CheckCircle size={size} className="text-green-400" />;
      case 'failed':
        return <XCircle size={size} className="text-red-400" />;
      case 'running':
      case 'processing':
        return <Clock size={size} className="text-yellow-400 animate-pulse" />;
      case 'retrying':
        return <RefreshCw size={size} className="text-purple-400 animate-pulse" />;
      case 'cancelled':
        return <XCircle size={size} className="text-gray-400" />;
      default:
        return <Clock size={size} className="text-blue-400" />;
    }
  };

  if (loading && !data) {
    return (
      <div className="flex items-center justify-center h-64">
        <RefreshCw size={32} className="animate-spin text-yellow-400" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center py-10">
        <div className="bg-red-900 text-white p-4 rounded mb-4">
          <AlertTriangle className="inline-block mr-2" size={20} />
          {error}
        </div>
        <button onClick={execute} className="text-yellow-400 flex items-center gap-2 mx-auto">
          <RefreshCw size={16} />
          Try Again
        </button>
      </div>
    );
  }

  const job = data?.data;

  if (!job) {
    return (
      <div className="text-center py-10">
        <p className="mb-4">Job not found</p>
        <Link to="/jobs" className="text-yellow-400">
          ← Back to Jobs
        </Link>
      </div>
    );
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <div className="flex items-center gap-3">
          <Link to="/jobs" className="text-gray-400 hover:text-white">
            <ArrowLeft size={20} />
          </Link>
          <h2 className="text-2xl">Job Details</h2>
          <span 
            className={`font-mono text-sm px-2 py-1 rounded ${getStatusColor(job.status)} ${getStatusBgColor(job.status)}`}
          >
            {getStatusIcon(job.status)} {job.status}
          </span>
        </div>
        
        <div className="flex items-center gap-4">
          <span className={`text-xs ${wsConnected ? 'text-green-400' : 'text-red-400'}`}>
            {wsConnected ? '● Live' : '● Offline'}
          </span>
          <button 
            onClick={execute} 
            className="text-yellow-400 flex items-center gap-1"
          >
            <RefreshCw size={16} />
            Refresh
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Main info */}
        <div className="md:col-span-2 border-2 border-gray-700 rounded-lg p-6">
          <div className="mb-4">
            <h3 className="text-gray-400 text-sm mb-1">Job ID</h3>
            <p className="font-mono">{job.id}</p>
          </div>
          
          <div className="mb-4">
            <h3 className="text-gray-400 text-sm mb-1">Type</h3>
            <p>{job.type}</p>
          </div>
          
          <div className="mb-4">
            <h3 className="text-gray-400 text-sm mb-1">Data</h3>
            <pre className="bg-gray-900 p-4 rounded overflow-x-auto text-sm">
              {formatJSON(job.data)}
            </pre>
          </div>
          
          {job.result && (
            <div className="mb-4">
              <h3 className="text-gray-400 text-sm mb-1">Result</h3>
              <pre className="bg-gray-900 p-4 rounded overflow-x-auto text-sm">
                {formatJSON(job.result)}
              </pre>
            </div>
          )}
          
          {job.last_error && (
            <div className="mb-4">
              <h3 className="text-gray-400 text-sm mb-1">Error</h3>
              <div className="bg-red-900 bg-opacity-50 p-4 rounded text-red-200">
                {job.last_error}
              </div>
            </div>
          )}
        </div>

        {/* Sidebar info */}
        <div className="border-2 border-gray-700 rounded-lg p-6">
          <div className="mb-4">
            <h3 className="text-gray-400 text-sm mb-1">Created</h3>
            <p>{formatDate(job.created_at)}</p>
          </div>
          
          {job.started_at && (
            <div className="mb-4">
              <h3 className="text-gray-400 text-sm mb-1">Started</h3>
              <p>{formatDate(job.started_at)}</p>
            </div>
          )}
          
          {job.finished_at && (
            <div className="mb-4">
              <h3 className="text-gray-400 text-sm mb-1">Completed</h3>
              <p>{formatDate(job.finished_at)}</p>
            </div>
          )}
          
          <div className="mb-4">
            <h3 className="text-gray-400 text-sm mb-1">Priority</h3>
            <p>{formatPriority(job.priority)}</p>
          </div>
          
          {job.scheduled_at && (
            <div className="mb-4">
              <h3 className="text-gray-400 text-sm mb-1">Scheduled For</h3>
              <p>{formatDate(job.scheduled_at)}</p>
            </div>
          )}
          
          <div className="mb-4">
            <h3 className="text-gray-400 text-sm mb-1">Attempts</h3>
            <p>{job.attempts || 0}</p>
          </div>
          
          {job.workflow_id && (
            <div className="mb-4">
              <h3 className="text-gray-400 text-sm mb-1">Part of Workflow</h3>
              <Link 
                to={`/workflows/${job.workflow_id}`} 
                className="text-cyan-400 hover:underline"
              >
                View Workflow
              </Link>
            </div>
          )}
          
          {job.status === 'pending' && (
            <button 
              onClick={() => {
                if (confirm('Are you sure you want to cancel this job?')) {
                  jobsApi.cancelJob(job.id).then(() => {
                    execute(); // Refresh job status
                  }).catch(err => {
                    console.error('Error cancelling job:', err);
                  });
                }
              }}
              className="mt-4 w-full py-2 px-4 bg-red-800 text-white rounded hover:bg-red-700"
            >
              Cancel Job
            </button>
          )}
        </div>
      </div>
    </div>
  );
};

export default JobDetail;