import React from 'react';
import { Link } from 'react-router-dom';
import { RefreshCw, AlertTriangle } from 'lucide-react';

import useApi from '../hooks/useApi';
import { workflowsApi } from '../services/api';
import { formatDate } from '../utils/format';

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

const WorkflowList = () => {
  const { data, loading, error, execute } = useApi(workflowsApi.getWorkflows, [], true);

  if (loading && !data) {
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
          <button onClick={execute} className="text-yellow-400 flex items-center gap-1">
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
              className="border-2 border-gray-700 rounded-lg p-6 hover:border-gray-500 transition-colors"
            >
              <div className="flex justify-between items-center mb-2">
                <h3 className="text-xl">{workflow.name}</h3>
                {getStatusBadge(workflow.status)}
              </div>
              <p className="text-gray-400 mb-4 text-sm font-mono truncate">{workflow.id}</p>
              <div className="flex justify-between text-xs text-gray-500">
                <span>{Object.keys(workflow.steps || {}).length} steps</span>
                <span>{formatDate(workflow.created_at)}</span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
};

export default WorkflowList;
