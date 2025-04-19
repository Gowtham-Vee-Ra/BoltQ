import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { RefreshCw, AlertTriangle, Clock, CheckCircle, XCircle } from 'lucide-react';

import useApi from '../hooks/useApi';
import { jobsApi } from '../services/api';
import { formatDate, getStatusColor } from '../utils/format';

// Sample data for demonstration
const SAMPLE_JOBS = [
  { 
    id: 'j8f72h3f', 
    type: 'echo', 
    status: 'completed', 
    created_at: '2025-03-22T10:30:00Z', 
    priority: 1,
    data: { message: "Hello World" }
  },
  { 
    id: 'k39d7h2s', 
    type: 'email', 
    status: 'running', 
    created_at: '2025-03-22T11:45:00Z', 
    priority: 0,
    data: { to: "user@example.com", subject: "Important notification" }
  },
  { 
    id: 'l12e9p4q', 
    type: 'process-image', 
    status: 'failed', 
    created_at: '2025-03-22T12:15:00Z', 
    priority: 2,
    error: "Invalid image format"
  },
  { 
    id: 'm67b3v2r', 
    type: 'generate-report', 
    status: 'pending', 
    created_at: '2025-03-23T09:00:00Z', 
    priority: 1,
    data: { reportType: "monthly", month: "March", year: 2025 }
  },
  { 
    id: 'n45k9d3e', 
    type: 'sleep', 
    status: 'scheduled', 
    created_at: '2025-03-23T09:30:00Z', 
    scheduled_at: '2025-03-23T10:30:00Z', 
    priority: 2,
    data: { seconds: 30 }
  },
  { 
    id: 'p78s5g2h', 
    type: 'echo', 
    status: 'completed', 
    created_at: '2025-03-23T08:15:00Z', 
    priority: 1,
    data: { message: "Another test message" }
  },
  { 
    id: 'q91t4j6k', 
    type: 'email', 
    status: 'retrying', 
    created_at: '2025-03-22T16:45:00Z', 
    priority: 0,
    attempts: 2,
    data: { to: "admin@example.com", subject: "System alert" }
  },
  { 
    id: 'r23w8n5z', 
    type: 'notification', 
    status: 'completed', 
    created_at: '2025-03-22T17:30:00Z', 
    priority: 1,
    data: { channel: "slack", message: "Deployment complete" }
  }
];

const JobList = () => {
  const [refreshInterval, setRefreshInterval] = useState(null);
  const { data, loading, error, execute, setData } = useApi(jobsApi.getJobs, [], false);

  // Use sample data for demonstration purposes
  useEffect(() => {
    // Set initial sample data
    setData({ data: SAMPLE_JOBS });
    
    // Simulate data refresh
    const interval = setInterval(() => {
      // Randomly update status of one job to simulate changes
      const updatedJobs = [...SAMPLE_JOBS];
      const randomIndex = Math.floor(Math.random() * updatedJobs.length);
      const job = updatedJobs[randomIndex];
      
      // Simulate status changes
      if (job.status === 'pending') {
        job.status = 'running';
      } else if (job.status === 'running') {
        job.status = Math.random() > 0.2 ? 'completed' : 'failed';
      } else if (job.status === 'scheduled') {
        if (new Date(job.scheduled_at) < new Date()) {
          job.status = 'pending';
        }
      } else if (job.status === 'retrying') {
        job.status = Math.random() > 0.5 ? 'running' : 'failed';
        job.attempts = (job.attempts || 0) + 1;
      }
      
      setData({ data: updatedJobs });
    }, 5000);
    
    setRefreshInterval(interval);
    
    return () => {
      if (refreshInterval) {
        clearInterval(refreshInterval);
      }
    };
  }, []);

  // Get status icon based on job status
  const getStatusIcon = (status, size = 16) => {
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

  if (loading && (!data || !data.data)) {
    return (
      <div className="flex items-center justify-center h-64">
        <RefreshCw size={32} className="animate-spin text-yellow-400" />
      </div>
    );
  }

  const jobs = data?.data || [];

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl">Recent Jobs</h2>
        
        <div className="flex gap-4">
          <button 
            onClick={execute} 
            className="text-yellow-400 flex items-center gap-1"
          >
            <RefreshCw size={16} />
            Refresh
          </button>
          
          <Link 
            to="/jobs/new" 
            className="bg-gradient-to-r from-yellow-400 to-orange-500 text-black font-bold py-2 px-4 rounded hover:opacity-90"
          >
            New Job
          </Link>
        </div>
      </div>

      {error && (
        <div className="bg-red-900 text-white p-4 rounded mb-4">
          <AlertTriangle className="inline-block mr-2" size={20} />
          {error}
        </div>
      )}

      {jobs.length === 0 ? (
        <div className="text-center py-10 border-2 border-gray-700 rounded-lg">
          <p className="mb-4">No jobs found</p>
          <Link 
            to="/jobs/new" 
            className="bg-gradient-to-r from-yellow-400 to-orange-500 text-black font-bold py-2 px-6 rounded hover:opacity-90"
          >
            Submit a New Job
          </Link>
        </div>
      ) : (
        <div className="border-2 border-gray-700 rounded-lg overflow-hidden">
          <table className="w-full text-left">
            <thead className="bg-gray-900">
              <tr>
                <th className="p-3">ID</th>
                <th className="p-3">Type</th>
                <th className="p-3">Status</th>
                <th className="p-3">Created</th>
                <th className="p-3">Priority</th>
                <th className="p-3"></th>
              </tr>
            </thead>
            
            <tbody className="divide-y divide-gray-800">
              {jobs.map(job => (
                <tr key={job.id} className="hover:bg-gray-900">
                  <td className="p-3 font-mono">{job.id.slice(0, 8)}...</td>
                  <td className="p-3">{job.type}</td>
                  <td className="p-3">
                    <div className="flex items-center gap-1">
                      {getStatusIcon(job.status)}
                      <span className={getStatusColor(job.status)}>
                        {job.status}
                      </span>
                    </div>
                  </td>
                  <td className="p-3">{formatDate(job.created_at)}</td>
                  <td className="p-3">
                    {job.priority === 0 ? 'High' : 
                     job.priority === 1 ? 'Normal' : 
                     job.priority === 2 ? 'Low' : job.priority}
                  </td>
                  <td className="p-3 text-right">
                    <Link to={`/jobs/${job.id}`} className="text-cyan-400 hover:underline">
                      View
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      
      {/* Filter and pagination controls */}
      <div className="mt-6 flex flex-col md:flex-row justify-between items-center">
        <div className="flex items-center gap-4 mb-4 md:mb-0">
          <div>
            <select 
              className="bg-gray-900 border border-gray-700 rounded p-2"
              defaultValue="all"
            >
              <option value="all">All Types</option>
              <option value="echo">Echo</option>
              <option value="email">Email</option>
              <option value="sleep">Sleep</option>
              <option value="process-image">Process Image</option>
              <option value="generate-report">Generate Report</option>
            </select>
          </div>
          
          <div>
            <select 
              className="bg-gray-900 border border-gray-700 rounded p-2"
              defaultValue="all"
            >
              <option value="all">All Statuses</option>
              <option value="pending">Pending</option>
              <option value="running">Running</option>
              <option value="completed">Completed</option>
              <option value="failed">Failed</option>
              <option value="scheduled">Scheduled</option>
            </select>
          </div>
        </div>
        
        <div className="flex items-center gap-2">
          <button className="px-3 py-1 bg-gray-800 rounded hover:bg-gray-700 disabled:opacity-50">
            Previous
          </button>
          <span className="px-4">Page 1 of 1</span>
          <button className="px-3 py-1 bg-gray-800 rounded hover:bg-gray-700 disabled:opacity-50">
            Next
          </button>
        </div>
      </div>
    </div>
  );
};

export default JobList;