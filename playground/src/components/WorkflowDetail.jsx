import { useRef, useEffect, useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { RefreshCw, ArrowLeft, Clock, CheckCircle, XCircle, AlertTriangle, Trash2 } from 'lucide-react';

import useApi from '../hooks/useApi';
import useWebSocket from '../hooks/useWebSocket';
import { workflowsApi } from '../services/api';
import { formatDate, getStatusColor } from '../utils/format';

const statusColors = {
  pending: '#777',
  completed: '#22c55e',
  failed: '#ef4444',
  running: '#eab308',
  processing: '#eab308',
  skipped: '#6b7280',
};

const getStatusIcon = (status, size = 18) => {
  switch (status) {
    case 'completed':
      return <CheckCircle size={size} className="text-green-400" />;
    case 'failed':
      return <XCircle size={size} className="text-red-400" />;
    case 'running':
    case 'processing':
      return <Clock size={size} className="animate-pulse text-yellow-400" />;
    case 'skipped':
      return <AlertTriangle size={size} className="text-gray-400" />;
    default:
      return <Clock size={size} className="text-gray-500" />;
  }
};

const WorkflowDetail = () => {
  const { workflowId } = useParams();
  const navigate = useNavigate();
  const canvasRef = useRef(null);
  const [deleting, setDeleting] = useState(false);

  const { data, loading, error, execute } = useApi(
    () => workflowsApi.getWorkflow(workflowId),
    [workflowId],
    true
  );

  const { connected } = useWebSocket(`/jobs/${workflowId}`);

  // Auto-refresh every 3s while workflow is running
  useEffect(() => {
    const workflow = data?.data;
    if (!workflow || workflow.status !== 'running') return;
    const interval = setInterval(execute, 3000);
    return () => clearInterval(interval);
  }, [data, execute]);

  // Redraw canvas whenever data changes
  useEffect(() => {
    if (data?.data?.steps && canvasRef.current) {
      renderWorkflowGraph(data.data, canvasRef.current);
    }
  }, [data]);

  const handleDelete = async () => {
    if (!window.confirm('Delete this workflow?')) return;
    setDeleting(true);
    try {
      await workflowsApi.deleteWorkflow(workflowId);
      navigate('/workflows');
    } catch {
      setDeleting(false);
    }
  };

  if (loading && !data) {
    return (
      <div className="flex justify-center items-center p-8">
        <RefreshCw size={32} className="animate-spin text-yellow-400" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-8 bg-red-900/20 border-2 border-red-700 rounded-lg text-center">
        <AlertTriangle size={32} className="text-red-400 mx-auto mb-4" />
        <h3 className="text-red-400 font-bold mb-2">Error Loading Workflow</h3>
        <p className="text-white mb-4">{error}</p>
        <button onClick={execute} className="text-yellow-400 flex items-center gap-1 mx-auto">
          <RefreshCw size={16} /> Try Again
        </button>
      </div>
    );
  }

  if (!data?.data) {
    return (
      <div className="p-8 border-2 border-gray-700 rounded-lg text-center">
        <h3 className="text-xl mb-4">Workflow Not Found</h3>
        <Link to="/workflows" className="text-yellow-400 flex items-center gap-1 max-w-fit mx-auto">
          <ArrowLeft size={16} /> Back to Workflows
        </Link>
      </div>
    );
  }

  const workflow = data.data;
  const stepOrder = workflow.step_order || Object.keys(workflow.steps || {});

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <div className="flex items-center gap-3">
          <Link to="/workflows" className="text-gray-400 hover:text-white">
            <ArrowLeft size={20} />
          </Link>
          <h2 className="text-2xl">{workflow.name}</h2>
          <span className={`px-2 py-1 rounded text-sm ${getStatusColor(workflow.status)}`}>
            {workflow.status}
          </span>
        </div>

        <div className="flex gap-4">
          <button onClick={execute} className="text-yellow-400 flex items-center gap-1">
            <RefreshCw size={16} /> Refresh
          </button>
          <button
            onClick={handleDelete}
            disabled={deleting}
            className="text-red-400 flex items-center gap-1 hover:text-red-300 disabled:opacity-50"
          >
            <Trash2 size={16} /> {deleting ? 'Deleting...' : 'Delete'}
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Main content */}
        <div className="md:col-span-2">
          <div className="border-2 border-gray-700 rounded-lg p-6 mb-6">
            <h3 className="text-xl mb-4">Workflow Visualization</h3>
            <div className="bg-gray-900 rounded-lg p-2 overflow-hidden">
              <canvas ref={canvasRef} width={800} height={400} className="w-full h-auto" />
            </div>
            <div className="flex flex-wrap justify-center gap-4 mt-4 text-sm">
              {[['#777','Pending'],['#eab308','Running'],['#22c55e','Completed'],['#ef4444','Failed'],['#6b7280','Skipped']].map(([color, label]) => (
                <div key={label} className="flex items-center gap-2">
                  <div className="w-3 h-3 rounded-full" style={{ background: color }}></div>
                  <span>{label}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Step Details */}
          <div className="border-2 border-gray-700 rounded-lg p-6">
            <h3 className="text-xl mb-4">Steps</h3>
            <div className="space-y-3">
              {stepOrder.map((stepId, i) => {
                const step = workflow.steps[stepId];
                if (!step) return null;
                return (
                  <div key={stepId} className="border border-gray-700 rounded p-3 hover:border-gray-500">
                    <div className="flex justify-between items-center mb-1">
                      <div className="flex items-center gap-2">
                        <span className="text-gray-500 text-xs">#{i + 1}</span>
                        <h4 className="font-medium">{step.job_type}</h4>
                      </div>
                      {getStatusIcon(step.status)}
                    </div>
                    <p className="text-xs text-gray-500 font-mono mb-1">{stepId}</p>
                    {step.depends_on?.length > 0 && (
                      <p className="text-xs text-gray-500">Depends on: {step.depends_on.length} step(s)</p>
                    )}
                    {step.error_message && (
                      <div className="mt-2 text-sm text-red-400 bg-red-900/20 p-2 rounded">
                        {step.error_message}
                      </div>
                    )}
                    {step.started_at && (
                      <p className="text-xs text-gray-500 mt-1">Started: {formatDate(step.started_at)}</p>
                    )}
                    {step.completed_at && (
                      <p className="text-xs text-gray-500">Completed: {formatDate(step.completed_at)}</p>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        </div>

        {/* Sidebar */}
        <div>
          <div className="border-2 border-gray-700 rounded-lg p-6 mb-6">
            <h3 className="text-xl mb-4">Info</h3>
            <div className="space-y-4">
              <div>
                <h4 className="text-gray-400 text-sm">ID</h4>
                <p className="font-mono text-sm break-all">{workflow.id}</p>
              </div>
              <div>
                <h4 className="text-gray-400 text-sm">Created</h4>
                <p>{formatDate(workflow.created_at)}</p>
              </div>
              {workflow.started_at && (
                <div>
                  <h4 className="text-gray-400 text-sm">Started</h4>
                  <p>{formatDate(workflow.started_at)}</p>
                </div>
              )}
              {workflow.finished_at && (
                <div>
                  <h4 className="text-gray-400 text-sm">Finished</h4>
                  <p>{formatDate(workflow.finished_at)}</p>
                </div>
              )}
              <div>
                <h4 className="text-gray-400 text-sm">Steps</h4>
                <p>{stepOrder.length}</p>
              </div>
              <div className={`flex items-center text-xs ${connected ? 'text-green-400' : 'text-red-400'}`}>
                <span className={`inline-block w-2 h-2 rounded-full mr-2 ${connected ? 'bg-green-400' : 'bg-red-400'}`}></span>
                {connected ? 'Live updates' : 'Offline'}
              </div>
            </div>
          </div>

          {workflow.metadata && Object.keys(workflow.metadata).length > 0 && (
            <div className="border-2 border-gray-700 rounded-lg p-6">
              <h3 className="text-xl mb-4">Metadata</h3>
              <pre className="text-xs text-gray-300 overflow-auto">
                {JSON.stringify(workflow.metadata, null, 2)}
              </pre>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

const renderWorkflowGraph = (workflow, canvas) => {
  const ctx = canvas.getContext('2d');
  const width = canvas.width;
  const height = canvas.height;

  ctx.clearRect(0, 0, width, height);
  ctx.fillStyle = '#111';
  ctx.fillRect(0, 0, width, height);

  const steps = workflow.steps;
  if (!steps || Object.keys(steps).length === 0) return;

  const nodeRadius = 30;
  const nodeGap = 150;
  const nodePositions = {};

  const rootIds = [];
  const isDep = new Set();
  Object.values(steps).forEach(s => s.depends_on?.forEach(d => isDep.add(d)));
  Object.keys(steps).forEach(id => { if (!isDep.has(id)) rootIds.push(id); });

  const positionNode = (id, x, y, visited = new Set()) => {
    if (visited.has(id)) return;
    visited.add(id);
    const node = steps[id];
    if (!node) return;
    nodePositions[id] = { x, y, node };
    const children = Object.entries(steps)
      .filter(([, s]) => s.depends_on?.includes(id))
      .map(([cid]) => cid);
    if (children.length > 0) {
      const totalW = nodeGap * (children.length - 1);
      children.forEach((cid, i) =>
        positionNode(cid, x - totalW / 2 + i * nodeGap, y + nodeGap, visited)
      );
    }
  };

  const totalRootW = nodeGap * (rootIds.length - 1);
  rootIds.forEach((id, i) =>
    positionNode(id, width / 2 - totalRootW / 2 + i * nodeGap, 60)
  );

  // Draw edges
  ctx.strokeStyle = '#444';
  ctx.lineWidth = 2;
  Object.entries(steps).forEach(([id, step]) => {
    if (!step.depends_on?.length) return;
    const target = nodePositions[id];
    if (!target) return;
    step.depends_on.forEach(srcId => {
      const src = nodePositions[srcId];
      if (!src) return;
      const dx = target.x - src.x;
      const dy = target.y - src.y;
      const angle = Math.atan2(dy, dx);
      const sx = src.x + nodeRadius * Math.cos(angle);
      const sy = src.y + nodeRadius * Math.sin(angle);
      const ex = target.x - nodeRadius * Math.cos(angle);
      const ey = target.y - nodeRadius * Math.sin(angle);
      ctx.beginPath();
      ctx.moveTo(sx, sy);
      ctx.lineTo(ex, ey);
      ctx.stroke();
      const arr = Math.PI / 6;
      ctx.beginPath();
      ctx.moveTo(ex, ey);
      ctx.lineTo(ex - 10 * Math.cos(angle - arr), ey - 10 * Math.sin(angle - arr));
      ctx.lineTo(ex - 10 * Math.cos(angle + arr), ey - 10 * Math.sin(angle + arr));
      ctx.closePath();
      ctx.fillStyle = '#444';
      ctx.fill();
    });
  });

  // Draw nodes
  Object.entries(nodePositions).forEach(([, { x, y, node }]) => {
    ctx.beginPath();
    ctx.arc(x, y, nodeRadius, 0, Math.PI * 2);
    ctx.fillStyle = statusColors[node.status] || statusColors.pending;
    ctx.fill();
    ctx.strokeStyle = '#fff';
    ctx.lineWidth = 2;
    ctx.stroke();
    ctx.fillStyle = '#fff';
    ctx.font = '10px Arial';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    const label = node.job_type.length > 10 ? node.job_type.slice(0, 9) + '…' : node.job_type;
    ctx.fillText(label, x, y);
  });
};

export default WorkflowDetail;
