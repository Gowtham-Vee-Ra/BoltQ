import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { 
  RefreshCw, 
  AlertTriangle, 
  Activity, 
  ClipboardList, 
  Layers, 
  Award, 
  Server, 
  Users 
} from 'lucide-react';

import useApi from '../hooks/useApi';
import { queuesApi } from '../services/api';
import { formatDate, formatPercentage } from '../utils/format';

// Sample data for demonstration
const SAMPLE_DATA = {
  // Queue stats
  'task_queue:0': 15,   // High priority queue
  'task_queue:1': 28,   // Normal priority queue
  'task_queue:2': 7,    // Low priority queue
  'delayed_tasks': 12,
  'dead_letter_queue': 5,
  
  // Job stats
  jobsProcessed: 1286,
  jobsProcessedToday: 78,
  successRate: 97.8,
  failureRate: 2.2,
  activeWorkers: 4,
  maxWorkers: 8,
  processPerMinute: 18.2,
  averageProcessingTime: 156.4,
  maxProcessingTime: 982.5,
  
  // System stats
  uptime: '2d 7h 15m',
  cpuUsage: 34.5,
  memoryUsage: 42.8,
  redisHealth: 100,
  
  // Recent job types
  jobTypeDistribution: {
    'echo': 532,
    'email': 358,
    'sleep': 214,
    'report': 146,
    'process-image': 36
  },
  
  // Recent jobs
  recentJobs: [
    { id: 'j8f72h3f', type: 'echo', status: 'completed', createdAt: '2025-03-22T10:30:00Z', priority: 1 },
    { id: 'k39d7h2s', type: 'email', status: 'processing', createdAt: '2025-03-22T11:45:00Z', priority: 0 },
    { id: 'l12e9p4q', type: 'process-image', status: 'failed', createdAt: '2025-03-22T12:15:00Z', priority: 2 },
    { id: 'm67b3v2r', type: 'generate-report', status: 'queued', createdAt: '2025-03-23T09:00:00Z', priority: 1 }
  ],
  
  // Recent workflows
  recentWorkflows: [
    {
      id: 'wf-123456',
      name: 'Data Processing Pipeline',
      status: 'running',
      createdAt: '2025-03-22T08:15:00Z',
      stepCount: 3
    },
    {
      id: 'wf-789012',
      name: 'Email Notification Workflow',
      status: 'completed',
      createdAt: '2025-03-21T14:30:00Z',
      stepCount: 2
    }
  ]
};

const Dashboard = () => {
  const [refreshInterval, setRefreshInterval] = useState(null);
  const { data, loading, error, execute, setData } = useApi(queuesApi.getQueueStats, [], false);

  // Use sample data instead of API call for demonstration
  useEffect(() => {
    // Set sample data
    setData({ data: SAMPLE_DATA });
    
    // Simulate periodic updates
    const interval = setInterval(() => {
      // Add some randomness to the data to simulate real-time changes
      const randomizedData = { ...SAMPLE_DATA };
      randomizedData['task_queue:0'] = Math.max(0, SAMPLE_DATA['task_queue:0'] + Math.floor(Math.random() * 5) - 2);
      randomizedData['task_queue:1'] = Math.max(0, SAMPLE_DATA['task_queue:1'] + Math.floor(Math.random() * 7) - 3);
      randomizedData['task_queue:2'] = Math.max(0, SAMPLE_DATA['task_queue:2'] + Math.floor(Math.random() * 3) - 1);
      randomizedData.activeWorkers = Math.min(randomizedData.maxWorkers, Math.max(1, SAMPLE_DATA.activeWorkers + Math.floor(Math.random() * 3) - 1));
      
      setData({ data: randomizedData });
    }, 5000);

    setRefreshInterval(interval);

    return () => {
      if (refreshInterval) {
        clearInterval(refreshInterval);
      }
    };
  }, []);

  if (loading && !data) {
    return (
      <div className="flex items-center justify-center h-64">
        <RefreshCw size={32} className="animate-spin text-yellow-400" />
      </div>
    );
  }

  const stats = data?.data || {};
  
  // Calculate some derived stats
  const queueDepth = 
    (stats['task_queue:0'] || 0) + 
    (stats['task_queue:1'] || 0) + 
    (stats['task_queue:2'] || 0);
  
  const delayedJobs = stats['delayed_tasks'] || 0;
  const deadLetterJobs = stats['dead_letter_queue'] || 0;
  
  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl">System Dashboard</h2>
        
        <button 
          onClick={() => execute()}
          className="text-yellow-400 flex items-center gap-1"
        >
          <RefreshCw size={16} />
          Refresh
        </button>
      </div>

      {error && (
        <div className="bg-red-900 text-white p-4 rounded mb-4">
          <AlertTriangle className="inline-block mr-2" size={20} />
          {error}
        </div>
      )}

      {/* Overview Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <div className="border-2 border-gray-700 rounded-lg p-6 bg-gray-900">
          <div className="flex justify-between items-start">
            <h3 className="text-gray-400 text-sm mb-1">Jobs Processed</h3>
            <ClipboardList className="text-yellow-400" size={20} />
          </div>
          <p className="text-3xl font-bold">{stats.jobsProcessed?.toLocaleString()}</p>
          <div className="mt-2 text-sm text-gray-400">Today: {stats.jobsProcessedToday}</div>
        </div>
        
        <div className="border-2 border-gray-700 rounded-lg p-6 bg-gray-900">
          <div className="flex justify-between items-start">
            <h3 className="text-gray-400 text-sm mb-1">Queue Depth</h3>
            <Layers className="text-blue-400" size={20} />
          </div>
          <p className="text-3xl font-bold">{queueDepth}</p>
          <div className="mt-2 text-sm text-gray-400">Delayed: {delayedJobs}</div>
        </div>
        
        <div className="border-2 border-gray-700 rounded-lg p-6 bg-gray-900">
          <div className="flex justify-between items-start">
            <h3 className="text-gray-400 text-sm mb-1">Success Rate</h3>
            <Award className="text-green-400" size={20} />
          </div>
          <p className="text-3xl font-bold">{formatPercentage(stats.successRate)}</p>
          <div className="mt-2 text-sm text-gray-400">Failed: {deadLetterJobs}</div>
        </div>
        
        <div className="border-2 border-gray-700 rounded-lg p-6 bg-gray-900">
          <div className="flex justify-between items-start">
            <h3 className="text-gray-400 text-sm mb-1">Workers</h3>
            <Users className="text-cyan-400" size={20} />
          </div>
          <p className="text-3xl font-bold">{stats.activeWorkers} <span className="text-sm text-gray-400">/ {stats.maxWorkers}</span></p>
          <div className="mt-2 text-sm text-gray-400">Tasks per minute: {stats.processPerMinute}</div>
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
                <th className="p-3 text-left">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-800">
              {/* High Priority Queue */}
              <tr className="hover:bg-gray-900">
                <td className="p-3">Queue 0</td>
                <td className="p-3">{stats['task_queue:0'] || 0}</td>
                <td className="p-3 text-red-400">High</td>
                <td className="p-3">
                  <Link 
                    to="/jobs" 
                    className="text-cyan-400 hover:underline"
                  >
                    View Jobs
                  </Link>
                </td>
              </tr>
              
              {/* Normal Priority Queue */}
              <tr className="hover:bg-gray-900">
                <td className="p-3">Queue 1</td>
                <td className="p-3">{stats['task_queue:1'] || 0}</td>
                <td className="p-3 text-blue-400">Normal</td>
                <td className="p-3">
                  <Link 
                    to="/jobs" 
                    className="text-cyan-400 hover:underline"
                  >
                    View Jobs
                  </Link>
                </td>
              </tr>
              
              {/* Low Priority Queue */}
              <tr className="hover:bg-gray-900">
                <td className="p-3">Queue 2</td>
                <td className="p-3">{stats['task_queue:2'] || 0}</td>
                <td className="p-3 text-green-400">Low</td>
                <td className="p-3">
                  <Link 
                    to="/jobs" 
                    className="text-cyan-400 hover:underline"
                  >
                    View Jobs
                  </Link>
                </td>
              </tr>
              
              {/* Delayed Tasks */}
              <tr className="hover:bg-gray-900">
                <td className="p-3">Delayed Tasks</td>
                <td className="p-3">{stats['delayed_tasks'] || 0}</td>
                <td className="p-3 text-cyan-400">Scheduled</td>
                <td className="p-3">
                  <Link 
                    to="/jobs" 
                    className="text-cyan-400 hover:underline"
                  >
                    View Jobs
                  </Link>
                </td>
              </tr>
              
              {/* Dead Letter Queue */}
              <tr className="hover:bg-gray-900">
                <td className="p-3">Dead Letter Queue</td>
                <td className="p-3">{stats['dead_letter_queue'] || 0}</td>
                <td className="p-3 text-red-400">Failed</td>
                <td className="p-3">
                  <Link 
                    to="/jobs" 
                    className="text-cyan-400 hover:underline"
                  >
                    View Jobs
                  </Link>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      {/* Job Types Distribution and System Health */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
        <div className="border-2 border-gray-700 rounded-lg p-6">
          <h3 className="text-xl mb-4 flex items-center gap-2">
            <Activity size={20} className="text-yellow-400" />
            Job Type Distribution
          </h3>
          
          <div className="space-y-3">
            {stats.jobTypeDistribution && Object.entries(stats.jobTypeDistribution).map(([type, count]) => (
              <div key={type} className="flex items-center">
                <div className="w-full bg-gray-700 rounded-full h-4 mr-2">
                  <div
                    className="bg-gradient-to-r from-yellow-400 to-orange-500 h-4 rounded-full"
                    style={{ width: `${(count / stats.jobsProcessed) * 100}%` }}
                  ></div>
                </div>
                <div className="text-sm whitespace-nowrap">{type}: {count}</div>
              </div>
            ))}
          </div>
        </div>
        
        <div className="border-2 border-gray-700 rounded-lg p-6">
          <h3 className="text-xl mb-4 flex items-center gap-2">
            <Server size={20} className="text-cyan-400" />
            System Health
          </h3>
          
          <div className="space-y-4">
            <div>
              <h4 className="text-gray-400 text-sm mb-1">Average Processing Time</h4>
              <p>{stats.averageProcessingTime} ms</p>
            </div>
            
            <div>
              <h4 className="text-gray-400 text-sm mb-1">Success Rate</h4>
              <div className="w-full bg-gray-700 rounded-full h-4 mb-2">
                <div
                  className="bg-green-500 h-4 rounded-full"
                  style={{ width: `${stats.successRate}%` }}
                ></div>
              </div>
              <p className="text-sm text-right">{formatPercentage(stats.successRate)}</p>
            </div>
            
            <div>
              <h4 className="text-gray-400 text-sm mb-1">CPU Usage</h4>
              <div className="w-full bg-gray-700 rounded-full h-4 mb-2">
                <div
                  className="bg-blue-500 h-4 rounded-full"
                  style={{ width: `${stats.cpuUsage}%` }}
                ></div>
              </div>
              <p className="text-sm text-right">{stats.cpuUsage}%</p>
            </div>
            
            <div>
              <h4 className="text-gray-400 text-sm mb-1">Memory Usage</h4>
              <div className="w-full bg-gray-700 rounded-full h-4 mb-2">
                <div
                  className="bg-purple-500 h-4 rounded-full"
                  style={{ width: `${stats.memoryUsage}%` }}
                ></div>
              </div>
              <p className="text-sm text-right">{stats.memoryUsage}%</p>
            </div>
          </div>
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
          
          <div className="space-y-3">
            {stats.recentJobs && stats.recentJobs.map(job => (
              <Link key={job.id} to={`/jobs/${job.id}`} className="block">
                <div className="border border-gray-700 rounded p-3 hover:border-gray-500">
                  <div className="flex justify-between">
                    <span className="font-mono">{job.id}</span>
                    <span className={
                      job.status === 'completed' ? 'text-green-400' :
                      job.status === 'failed' ? 'text-red-400' :
                      job.status === 'processing' ? 'text-yellow-400' : 'text-blue-400'
                    }>
                      {job.status}
                    </span>
                  </div>
                  <div className="text-sm text-gray-400 mt-1">
                    {job.type} - {new Date(job.createdAt).toLocaleString()}
                  </div>
                </div>
              </Link>
            ))}
          </div>
        </div>
        
        <div className="border-2 border-gray-700 rounded-lg p-6">
          <div className="flex justify-between mb-4">
            <h3 className="text-xl flex items-center gap-2">
              <Activity size={20} className="text-cyan-400" />
              Recent Workflows
            </h3>
            <Link to="/workflows" className="text-cyan-400 hover:underline text-sm">View All</Link>
          </div>
          
          <div className="space-y-3">
            {stats.recentWorkflows && stats.recentWorkflows.map(workflow => (
              <Link key={workflow.id} to={`/workflows/${workflow.id}`} className="block">
                <div className="border border-gray-700 rounded p-3 hover:border-gray-500">
                  <div className="flex justify-between">
                    <span className="font-medium">{workflow.name}</span>
                    <span className={
                      workflow.status === 'completed' ? 'text-green-400' :
                      workflow.status === 'failed' ? 'text-red-400' :
                      workflow.status === 'running' ? 'text-yellow-400' : 'text-blue-400'
                    }>
                      {workflow.status}
                    </span>
                  </div>
                  <div className="text-sm text-gray-400 mt-1">
                    {workflow.stepCount} steps - {new Date(workflow.createdAt).toLocaleString()}
                  </div>
                </div>
              </Link>
            ))}
          </div>
        </div>
      </div>
      
      {/* System Uptime and Stats */}
      <div className="border-2 border-gray-700 rounded-lg p-6 mb-8">
        <h3 className="text-xl mb-4 flex items-center gap-2">
          <Server size={20} />
          System Status
        </h3>
        
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <div className="text-center">
            <h4 className="text-gray-400 text-sm mb-2">Uptime</h4>
            <p className="text-xl">{stats.uptime}</p>
          </div>
          
          <div className="text-center">
            <h4 className="text-gray-400 text-sm mb-2">Active Workers</h4>
            <p className="text-xl">{stats.activeWorkers} / {stats.maxWorkers}</p>
          </div>
          
          <div className="text-center">
            <h4 className="text-gray-400 text-sm mb-2">Queue Depth</h4>
            <p className="text-xl">{queueDepth}</p>
          </div>
          
          <div className="text-center">
            <h4 className="text-gray-400 text-sm mb-2">Last Updated</h4>
            <p className="text-xl">{formatDate(new Date())}</p>
          </div>
        </div>
      </div>
      
      {/* Links to create jobs or workflows */}
      <div className="mt-8 flex justify-center gap-6">
        <Link
          to="/jobs/new"
          className="bg-gradient-to-r from-yellow-400 to-orange-500 text-black font-bold py-3 px-6 rounded-lg hover:opacity-90"
        >
          Submit a New Job
        </Link>
        
        <Link
          to="/workflows"
          className="bg-gradient-to-r from-cyan-500 to-blue-500 text-black font-bold py-3 px-6 rounded-lg hover:opacity-90"
        >
          Manage Workflows
        </Link>
      </div>
    </div>
  );
};

export default Dashboard;