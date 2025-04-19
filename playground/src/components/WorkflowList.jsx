import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { RefreshCw, AlertTriangle } from 'lucide-react';

import useApi from '../hooks/useApi';
import { workflowsApi } from '../services/api';
import { formatDate } from '../utils/format';

// Sample workflows data for demonstration
const SAMPLE_WORKFLOWS = [
  {
    id: 'wf-123456',
    name: 'Data Processing Pipeline',
    description: 'Extract, transform, and load data pipeline',
    status: 'running',
    created_at: '2025-03-22T08:15:00Z',
    started_at: '2025-03-22T08:30:00Z',
    steps: {
      'step1': {
        id: 'step1',
        job_type: 'extract',
        status: 'completed',
        job_id: 'j-extract-123'
      },
      'step2': {
        id: 'step2',
        job_type: 'transform',
        status: 'running',
        job_id: 'j-transform-456',
        depends_on: ['step1']
      },
      'step3': {
        id: 'step3',
        job_type: 'load',
        status: 'pending',
        depends_on: ['step2']
      }
    }
  },
  {
    id: 'wf-654321',
    name: 'Email Notification Workflow',
    description: 'Send emails to users based on defined triggers',
    status: 'completed',
    created_at: '2025-03-21T14:30:00Z',
    started_at: '2025-03-21T14:35:00Z',
    finished_at: '2025-03-21T14:40:00Z',
    steps: {
      'step1': {
        id: 'step1',
        job_type: 'fetch-users',
        status: 'completed',
        job_id: 'j-fetch-789'
      },
      'step2': {
        id: 'step2',
        job_type: 'send-email',
        status: 'completed',
        job_id: 'j-email-012',
        depends_on: ['step1']
      }
    }
  },
  {
    id: 'wf-789012',
    name: 'Report Generation',
    description: 'Generate and deliver monthly reports',
    status: 'pending',
    created_at: '2025-03-23T07:00:00Z',
    steps: {
      'step1': {
        id: 'step1',
        job_type: 'query-data',
        status: 'pending'
      },
      'step2': {
        id: 'step2',
        job_type: 'generate-report',
        status: 'pending',
        depends_on: ['step1']
      },
      'step3': {
        id: 'step3',
        job_type: 'email-report',
        status: 'pending',
        depends_on: ['step2']
      }
    }
  },
  {
    id: 'wf-345678',
    name: 'Image Processing',
    description: 'Process and optimize images for the CDN',
    status: 'failed',
    created_at: '2025-03-22T09:15:00Z',
    started_at: '2025-03-22T09:20:00Z',
    finished_at: '2025-03-22T09:25:00Z',
    steps: {
      'step1': {
        id: 'step1',
        job_type: 'fetch-images',
        status: 'completed',
        job_id: 'j-fetch-img-345'
      },
      'step2': {
        id: 'step2',
        job_type: 'resize-images',
        status: 'completed',
        job_id: 'j-resize-567',
        depends_on: ['step1']
      },
      'step3': {
        id: 'step3',
        job_type: 'optimize-images',
        status: 'failed',
        job_id: 'j-optimize-789',
        depends_on: ['step2'],
        error_message: 'Out of memory while optimizing large image batch'
      },
      'step4': {
        id: 'step4',
        job_type: 'upload-to-cdn',
        status: 'skipped',
        depends_on: ['step3']
      }
    }
  },
  {
    id: 'wf-901234',
    name: 'User Onboarding',
    description: 'Automate new user onboarding process',
    status: 'completed',
    created_at: '2025-03-20T13:00:00Z',
    started_at: '2025-03-20T13:05:00Z',
    finished_at: '2025-03-20T13:15:00Z',
    steps: {
      'step1': {
        id: 'step1',
        job_type: 'create-account',
        status: 'completed',
        job_id: 'j-create-123'
      },
      'step2': {
        id: 'step2',
        job_type: 'send-welcome',
        status: 'completed',
        job_id: 'j-welcome-456',
        depends_on: ['step1']
      },
      'step3': {
        id: 'step3',
        job_type: 'setup-defaults',
        status: 'completed',
        job_id: 'j-defaults-789',
        depends_on: ['step1']
      }
    }
  }
];

const WorkflowList = () => {
  const [refreshInterval, setRefreshInterval] = useState(null);
  const { data, loading, error, execute, setData } = useApi(workflowsApi.getWorkflows, [], false);

  // Use sample data for demonstration
  useEffect(() => {
    // Set sample data
    setData({ data: SAMPLE_WORKFLOWS });
    
    // Simulate periodic updates
    const interval = setInterval(() => {
      // Update a running workflow to simulate progress
      setData(prevData => {
        if (!prevData?.data) return prevData;
        
        const updatedWorkflows = [...prevData.data];
        const runningWorkflowIndex = updatedWorkflows.findIndex(w => w.status === 'running');
        
        if (runningWorkflowIndex >= 0) {
          const workflow = updatedWorkflows[runningWorkflowIndex];
          
          // Find a running step and complete it
          const steps = { ...workflow.steps };
          const runningStepKey = Object.keys(steps).find(key => steps[key].status === 'running');
          
          if (runningStepKey) {
            steps[runningStepKey].status = 'completed';
            
            // Find the next pending step that depends on this one
            const nextStepKey = Object.keys(steps).find(key => 
              steps[key].status === 'pending' && 
              steps[key].depends_on && 
              steps[key].depends_on.includes(runningStepKey)
            );
            
            if (nextStepKey) {
              steps[nextStepKey].status = 'running';
              steps[nextStepKey].job_id = 'j-' + Math.random().toString(36).substring(2, 10);
            } else {
              // No more steps, workflow is complete
              workflow.status = 'completed';
              workflow.finished_at = new Date().toISOString();
            }
            
            workflow.steps = steps;
          }
        }
        
        return { data: updatedWorkflows };
      });
    }, 8000);
    
    setRefreshInterval(interval);
    
    return () => {
      if (refreshInterval) {
        clearInterval(refreshInterval);
      }
    };
  }, []);

  // Get status badge for workflows
  const getStatusBadge = (status) => {
    switch (status?.toLowerCase()) {
      case 'completed':
        return <span className="bg-green-900 text-green-400 px-2 py-1 rounded-full text-xs">Completed</span>;
      case 'failed':
        return <span className="bg-red-900 text-red-400 px-2 py-1 rounded-full text-xs">Failed</span>;
      case 'running':
        return <span className="bg-yellow-900 text-yellow-400 px-2 py-1 rounded-full text-xs animate-pulse">Running</span>;
      default:
        return <span className="bg-blue-900 text-blue-400 px-2 py-1 rounded-full text-xs">Pending</span>;
    }
  };

  if (loading && (!data || !data.data)) {
    return (
      <div className="flex items-center justify-center h-64">
        <RefreshCw size={32} className="animate-spin text-yellow-400" />
      </div>
    );
  }

  const workflows = data?.data || [];

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl">Workflows</h2>
        
        <div className="flex gap-4">
          <button 
            onClick={execute} 
            className="text-yellow-400 flex items-center gap-1"
          >
            <RefreshCw size={16} />
            Refresh
          </button>
          
          <Link 
            to="/workflows/new" 
            className="bg-gradient-to-r from-yellow-400 to-orange-500 text-black font-bold py-2 px-4 rounded hover:opacity-90"
          >
            New Workflow
          </Link>
        </div>
      </div>

      {error && (
        <div className="bg-red-900 text-white p-4 rounded mb-4">
          <AlertTriangle className="inline-block mr-2" size={20} />
          {error}
        </div>
      )}

      {workflows.length === 0 ? (
        <div className="text-center py-10 border-2 border-gray-700 rounded-lg">
          <p className="mb-4">No workflows found</p>
          <Link 
            to="/workflows/new" 
            className="bg-gradient-to-r from-yellow-400 to-orange-500 text-black font-bold py-2 px-6 rounded hover:opacity-90"
          >
            Create Workflow
          </Link>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {workflows.map(workflow => (
            <Link
              key={workflow.id}
              to={`/workflows/${workflow.id}`}
              className="border-2 border-gray-700 rounded-lg p-6 hover:border-gray-500 transition-colors cursor-pointer"
            >
              <div className="flex justify-between items-center mb-2">
                <h3 className="text-xl">{workflow.name}</h3>
                {getStatusBadge(workflow.status)}
              </div>
              
              <p className="text-gray-400 mb-4 text-sm">{workflow.description || 'No description'}</p>
              
              <div className="flex justify-between text-xs text-gray-500">
                <span>{workflow.steps ? Object.keys(workflow.steps).length : 0} steps</span>
                <span>{formatDate(workflow.created_at)}</span>
              </div>
            </Link>
          ))}
        </div>
      )}
      
      {/* Filter and sorting controls */}
      <div className="mt-6 flex flex-col md:flex-row justify-between items-center">
        <div className="flex items-center gap-4 mb-4 md:mb-0">
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
            </select>
          </div>
          
          <div>
            <select 
              className="bg-gray-900 border border-gray-700 rounded p-2"
              defaultValue="newest"
            >
              <option value="newest">Newest First</option>
              <option value="oldest">Oldest First</option>
            </select>
          </div>
        </div>
      </div>
    </div>
  );
};

export default WorkflowList;