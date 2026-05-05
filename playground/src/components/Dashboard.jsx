import React, { useEffect } from 'react';
import { Link } from 'react-router-dom';
import {
  RefreshCw,
  AlertTriangle,
  ClipboardList,
  Layers,
  Server,
  Skull,
  Clock,
} from 'lucide-react';

import useApi from '../hooks/useApi';
import { queuesApi, jobsApi, workflowsApi } from '../services/api';
import { formatDate, getStatusColor } from '../utils/format';

const Dashboard = () => {
  const { data: statsData, loading: statsLoading, error: statsError, execute: refreshStats } =
    useApi(queuesApi.getQueueStats, [], true);

  const { data: jobsData, loading: jobsLoading, execute: refreshJobs } =
    useApi(jobsApi.getJobs, [], true);

  const { data: workflowsData, loading: workflowsLoading, execute: refreshWorkflows } =
    useApi(workflowsApi.getWorkflows, [], true);

  // Auto-refresh queue stats every 10s
  useEffect(() => {
    const interval = setInterval(refreshStats, 10000);
    return () => clearInterval(interval);
  }, [refreshStats]);

  const handleRefresh = () => {
    refreshStats();
    refreshJobs();
    refreshWorkflows();
  };

  const stats = statsData?.data || {};
  const highQ  = stats['task_queue:2'] ?? 0;
  const normalQ = stats['task_queue:1'] ?? 0;
  const lowQ   = stats['task_queue:0'] ?? 0;
  const queueDepth = highQ + normalQ + lowQ;
  const delayed = stats['delayed_tasks'] ?? 0;
  const deadLetter = stats['dead_letter_queue'] ?? 0;

  const recentJobs = (jobsData?.data || []).slice(0, 5);
  const recentWorkflows = (workflowsData?.data || []).slice(0, 5);

  const isLoading = statsLoading && !statsData;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <RefreshCw size={32} className="animate-spin text-yellow-400" />
      </div>
    );
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl">System Dashboard</h2>
        <button onClick={handleRefresh} className="text-yellow-400 flex items-center gap-1">
          <RefreshCw size={16} />
          Refresh
        </button>
      </div>

      {statsError && (
        <div className="bg-red-900 text-white p-4 rounded mb-4">
          <AlertTriangle className="inline-block mr-2" size={20} />
          {statsError}
        </div>
      )}

      {/* Overview Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <div className="border-2 border-gray-700 rounded-lg p-6 bg-gray-900">
          <div className="flex justify-between items-start">
            <h3 className="text-gray-400 text-sm mb-1">Queue Depth</h3>
            <Layers className="text-blue-400" size={20} />
          </div>
          <p className="text-3xl font-bold">{queueDepth}</p>
          <div className="mt-2 text-sm text-gray-400">Pending tasks across all queues</div>
        </div>

        <div className="border-2 border-gray-700 rounded-lg p-6 bg-gray-900">
          <div className="flex justify-between items-start">
            <h3 className="text-gray-400 text-sm mb-1">High Priority</h3>
            <Server className="text-red-400" size={20} />
          </div>
          <p className="text-3xl font-bold">{highQ}</p>
          <div className="mt-2 text-sm text-gray-400">Normal: {normalQ} &nbsp;|&nbsp; Low: {lowQ}</div>
        </div>

        <div className="border-2 border-gray-700 rounded-lg p-6 bg-gray-900">
          <div className="flex justify-between items-start">
            <h3 className="text-gray-400 text-sm mb-1">Delayed</h3>
            <Clock className="text-cyan-400" size={20} />
          </div>
          <p className="text-3xl font-bold">{delayed}</p>
          <div className="mt-2 text-sm text-gray-400">Scheduled for future execution</div>
        </div>

        <div className="border-2 border-gray-700 rounded-lg p-6 bg-gray-900">
          <div className="flex justify-between items-start">
            <h3 className="text-gray-400 text-sm mb-1">Dead Letter</h3>
            <Skull className="text-red-400" size={20} />
          </div>
          <p className="text-3xl font-bold">{deadLetter}</p>
          <div className="mt-2 text-sm text-gray-400">Permanently failed tasks</div>
        </div>
      </div>

      {/* Queue Stats Table */}
      <div className="border-2 border-gray-700 rounded-lg p-6 mb-8">
        <h3 className="text-xl mb-4 flex items-center gap-2">
          <Server size={20} />
          Queue Statistics
        </h3>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-900">
              <tr>
                <th className="p-3 text-left">Queue</th>
                <th className="p-3 text-left">Size</th>
                <th className="p-3 text-left">Priority</th>
                <th className="p-3 text-left"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-800">
              <tr className="hover:bg-gray-900">
                <td className="p-3">task_queue:2</td>
                <td className="p-3 font-bold">{highQ}</td>
                <td className="p-3 text-red-400">High</td>
                <td className="p-3"><Link to="/jobs" className="text-cyan-400 hover:underline">View Jobs</Link></td>
              </tr>
              <tr className="hover:bg-gray-900">
                <td className="p-3">task_queue:1</td>
                <td className="p-3 font-bold">{normalQ}</td>
                <td className="p-3 text-blue-400">Normal</td>
                <td className="p-3"><Link to="/jobs" className="text-cyan-400 hover:underline">View Jobs</Link></td>
              </tr>
              <tr className="hover:bg-gray-900">
                <td className="p-3">task_queue:0</td>
                <td className="p-3 font-bold">{lowQ}</td>
                <td className="p-3 text-green-400">Low</td>
                <td className="p-3"><Link to="/jobs" className="text-cyan-400 hover:underline">View Jobs</Link></td>
              </tr>
              <tr className="hover:bg-gray-900">
                <td className="p-3">delayed_tasks</td>
                <td className="p-3 font-bold">{delayed}</td>
                <td className="p-3 text-cyan-400">Scheduled</td>
                <td className="p-3"><Link to="/jobs" className="text-cyan-400 hover:underline">View Jobs</Link></td>
              </tr>
              <tr className="hover:bg-gray-900">
                <td className="p-3">dead_letter_queue</td>
                <td className="p-3 font-bold">{deadLetter}</td>
                <td className="p-3 text-red-400">Failed</td>
                <td className="p-3"><Link to="/jobs" className="text-cyan-400 hover:underline">View Jobs</Link></td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      {/* Recent Jobs and Workflows */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
        <div className="border-2 border-gray-700 rounded-lg p-6">
          <div className="flex justify-between mb-4">
            <h3 className="text-xl flex items-center gap-2">
              <ClipboardList size={20} className="text-yellow-400" />
              Recent Jobs
            </h3>
            <Link to="/jobs" className="text-cyan-400 hover:underline text-sm">View All</Link>
          </div>

          {jobsLoading && !jobsData ? (
            <div className="flex justify-center py-6">
              <RefreshCw size={20} className="animate-spin text-yellow-400" />
            </div>
          ) : recentJobs.length === 0 ? (
            <p className="text-gray-500 text-sm">No jobs yet.</p>
          ) : (
            <div className="space-y-3">
              {recentJobs.map(job => (
                <Link key={job.id} to={`/jobs/${job.id}`} className="block">
                  <div className="border border-gray-700 rounded p-3 hover:border-gray-500">
                    <div className="flex justify-between">
                      <span className="font-mono text-sm">{job.id.slice(0, 8)}...</span>
                      <span className={getStatusColor(job.status)}>{job.status}</span>
                    </div>
                    <div className="text-xs text-gray-400 mt-1">
                      {job.type} &mdash; {formatDate(job.created_at)}
                    </div>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </div>

        <div className="border-2 border-gray-700 rounded-lg p-6">
          <div className="flex justify-between mb-4">
            <h3 className="text-xl flex items-center gap-2">
              <Layers size={20} className="text-cyan-400" />
              Recent Workflows
            </h3>
            <Link to="/workflows" className="text-cyan-400 hover:underline text-sm">View All</Link>
          </div>

          {workflowsLoading && !workflowsData ? (
            <div className="flex justify-center py-6">
              <RefreshCw size={20} className="animate-spin text-yellow-400" />
            </div>
          ) : recentWorkflows.length === 0 ? (
            <p className="text-gray-500 text-sm">No workflows yet.</p>
          ) : (
            <div className="space-y-3">
              {recentWorkflows.map(wf => (
                <Link key={wf.id} to={`/workflows/${wf.id}`} className="block">
                  <div className="border border-gray-700 rounded p-3 hover:border-gray-500">
                    <div className="flex justify-between">
                      <span className="font-medium">{wf.name}</span>
                      <span className={getStatusColor(wf.status)}>{wf.status}</span>
                    </div>
                    <div className="text-xs text-gray-400 mt-1">
                      {Object.keys(wf.steps || {}).length} steps &mdash; {formatDate(wf.created_at)}
                    </div>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </div>
      </div>

      <div className="mt-8 flex justify-center gap-6">
        <Link
          to="/jobs/new"
          className="bg-gradient-to-r from-yellow-400 to-orange-500 text-black font-bold py-3 px-6 rounded-lg hover:opacity-90"
        >
          Submit a New Job
        </Link>
        <Link
          to="/workflows/new"
          className="bg-gradient-to-r from-cyan-500 to-blue-500 text-black font-bold py-3 px-6 rounded-lg hover:opacity-90"
        >
          New Workflow
        </Link>
      </div>
    </div>
  );
};

export default Dashboard;
