import React from 'react';
import { Link } from 'react-router-dom';
import { RefreshCw, AlertTriangle, Clock, CheckCircle, XCircle } from 'lucide-react';

import useApi from '../hooks/useApi';
import { jobsApi } from '../services/api';
import { formatDate, getStatusColor } from '../utils/format';

const JobList = () => {
  const { data, loading, error, execute } = useApi(jobsApi.getJobs, [], true);

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
                    {job.priority === 2 ? 'High' :
                     job.priority === 1 ? 'Normal' :
                     job.priority === 0 ? 'Low' : job.priority}
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