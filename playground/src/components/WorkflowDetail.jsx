import React, { useRef, useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { RefreshCw, ArrowLeft, Play, Clock, CheckCircle, XCircle, AlertTriangle } from 'lucide-react';

import useApi from '../hooks/useApi';
import { workflowsApi } from '../services/api';
import { formatDate, getStatusColor } from '../utils/format';

// Sample workflows data for demonstration
const SAMPLE_WORKFLOWS = {
  'wf-123456': {
    id: 'wf-123456',
    name: 'Data Processing Pipeline',
    description: 'Extract, transform, and load data pipeline for monthly analytics',
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
  'wf-654321': {
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
  'wf-789012': {
    id: 'wf-789012',
    name: 'Report Generation',
    description: 'Generate and deliver monthly reports to stakeholders',
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
  'wf-345678': {
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
  }
};

const WorkflowDetail = () => {
  const { workflowId } = useParams();
  const canvasRef = useRef(null);
  const [wsConnected, setWsConnected] = useState(false);
  const [lastUpdated, setLastUpdated] = useState(new Date());
  
  const { data, loading, error, execute, setData } = useApi(
    () => workflowsApi.getWorkflow(workflowId),
    [workflowId],
    false
  );
  
  // Use sample data for demonstration
  useEffect(() => {
    // Find the workflow in our sample data
    const sampleWorkflow = SAMPLE_WORKFLOWS[workflowId];
    
    if (sampleWorkflow) {
      setData({ data: sampleWorkflow });
      
      // Simulate WebSocket connection
      setTimeout(() => {
        setWsConnected(true);
      }, 1000);
      
      // For running workflows, simulate status updates
      if (sampleWorkflow.status === 'running') {
        const interval = setInterval(() => {
          setData(prevData => {
            if (!prevData?.data) return prevData;
            
            const updatedWorkflow = { ...prevData.data };
            const steps = { ...updatedWorkflow.steps };
            
            // Find a running step and complete it
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
                updatedWorkflow.status = 'completed';
                updatedWorkflow.finished_at = new Date().toISOString();
                clearInterval(interval);
              }
              
              updatedWorkflow.steps = steps;
              setLastUpdated(new Date());
              return { data: updatedWorkflow };
            }
            
            return prevData;
          });
        }, 5000);
        
        return () => clearInterval(interval);
      }
      
      // For pending workflows, simulate starting when run
      if (sampleWorkflow.status === 'pending') {
        // The simulation will be handled in the handleRunWorkflow function
      }
    } else {
      // If no sample workflow exists for this ID, simulate a not found response
      setTimeout(() => {
        setData(null);
      }, 500);
    }
  }, [workflowId]);
  
  // Run workflow function (simulation)
  const handleRunWorkflow = () => {
    setData(prevData => {
      if (!prevData?.data) return prevData;
      
      const updatedWorkflow = { ...prevData.data };
      
      if (updatedWorkflow.status === 'pending') {
        updatedWorkflow.status = 'running';
        updatedWorkflow.started_at = new Date().toISOString();
        
        // Start the first step(s) - those without dependencies
        const steps = { ...updatedWorkflow.steps };
        
        Object.keys(steps).forEach(key => {
          const step = steps[key];
          if (!step.depends_on || step.depends_on.length === 0) {
            step.status = 'running';
            step.job_id = 'j-' + Math.random().toString(36).substring(2, 10);
          }
        });
        
        updatedWorkflow.steps = steps;
      }
      
      setLastUpdated(new Date());
      return { data: updatedWorkflow };
    });
  };
  
  // Render workflow visualization when data changes
  useEffect(() => {
    if (data?.data?.steps && canvasRef.current) {
      renderWorkflowGraph();
    }
  }, [data, canvasRef.current]);
  
  // Function to render workflow graph on canvas
  const renderWorkflowGraph = () => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    
    const ctx = canvas.getContext('2d');
    const width = canvas.width;
    const height = canvas.height;
    
    // Clear canvas
    ctx.clearRect(0, 0, width, height);
    
    // Draw background
    ctx.fillStyle = '#111';
    ctx.fillRect(0, 0, width, height);
    
    const workflow = data?.data;
    if (!workflow || !workflow.steps || Object.keys(workflow.steps).length === 0) {
      return;
    }
    
    const steps = workflow.steps;
    const nodeRadius = 30;
    const nodeGap = 150;
    const startX = width / 2;
    const startY = 60;
    
    // Map to store node positions
    const nodePositions = {};
    
    // Define status colors
    const statusColors = {
      pending: '#777',
      completed: '#22c55e',
      failed: '#ef4444',
      running: '#eab308',
      processing: '#eab308',
      skipped: '#6b7280'
    };
    
    // Simple layout algorithm for a tree-like structure
    const layoutNodes = () => {
      // Find root nodes (steps with no dependencies)
      const rootNodeIds = [];
      const dependentNodeIds = new Set();
      
      // Collect all nodes that are dependencies of other nodes
      Object.values(steps).forEach(step => {
        if (step.depends_on && step.depends_on.length > 0) {
          step.depends_on.forEach(depId => dependentNodeIds.add(depId));
        }
      });
      
      // Find root nodes (not dependent on any other node)
      Object.entries(steps).forEach(([id, step]) => {
        if (!dependentNodeIds.has(id)) {
          rootNodeIds.push(id);
        }
      });
      
      // Function to position a node and its dependencies
      const positionNode = (nodeId, x, y, level = 0, processed = new Set()) => {
        // Avoid cycles
        if (processed.has(nodeId)) return;
        processed.add(nodeId);
        
        const node = steps[nodeId];
        if (!node) return;
        
        // Store node position
        nodePositions[nodeId] = { x, y, node };
        
        // Find children (nodes that depend on this node)
        const childrenIds = [];
        Object.entries(steps).forEach(([id, step]) => {
          if (step.depends_on && step.depends_on.includes(nodeId)) {
            childrenIds.push(id);
          }
        });
        
        // Position children
        if (childrenIds.length > 0) {
          const childWidth = nodeGap;
          const totalWidth = childWidth * (childrenIds.length - 1);
          let startPosX = x - totalWidth / 2;
          
          childrenIds.forEach((childId, index) => {
            positionNode(
              childId,
              startPosX + index * childWidth,
              y + nodeGap,
              level + 1,
              processed
            );
          });
        }
      };
      
      // Position root nodes
      if (rootNodeIds.length > 0) {
        const rootWidth = nodeGap;
        const totalWidth = rootWidth * (rootNodeIds.length - 1);
        let startPosX = startX - totalWidth / 2;
        
        rootNodeIds.forEach((nodeId, index) => {
          positionNode(
            nodeId,
            startPosX + index * rootWidth,
            startY,
            0,
            new Set()
          );
        });
      }
    };
    
    // Execute layout
    layoutNodes();
    
    // Draw edges
    ctx.strokeStyle = '#444';
    ctx.lineWidth = 2;
    
    // Draw dependencies
    Object.entries(steps).forEach(([nodeId, step]) => {
      if (step.depends_on && step.depends_on.length > 0) {
        const targetPos = nodePositions[nodeId];
        
        if (!targetPos) return;
        
        step.depends_on.forEach(sourceId => {
          const sourcePos = nodePositions[sourceId];
          
          if (!sourcePos) return;
          
          // Calculate start and end points
          const dx = targetPos.x - sourcePos.x;
          const dy = targetPos.y - sourcePos.y;
          const angle = Math.atan2(dy, dx);
          
          const startX = sourcePos.x + nodeRadius * Math.cos(angle);
          const startY = sourcePos.y + nodeRadius * Math.sin(angle);
          const endX = targetPos.x - nodeRadius * Math.cos(angle);
          const endY = targetPos.y - nodeRadius * Math.sin(angle);
          
          // Draw the line
          ctx.beginPath();
          ctx.moveTo(startX, startY);
          ctx.lineTo(endX, endY);
          ctx.stroke();
          
          // Draw arrow
          const arrowSize = 10;
          const arrowAngle = Math.PI / 6; // 30 degrees
          
          ctx.beginPath();
          ctx.moveTo(endX, endY);
          ctx.lineTo(
            endX - arrowSize * Math.cos(angle - arrowAngle),
            endY - arrowSize * Math.sin(angle - arrowAngle)
          );
          ctx.lineTo(
            endX - arrowSize * Math.cos(angle + arrowAngle),
            endY - arrowSize * Math.sin(angle + arrowAngle)
          );
          ctx.closePath();
          ctx.fillStyle = '#444';
          ctx.fill();
        });
      }
    });
    
    // Draw nodes
    Object.entries(nodePositions).forEach(([nodeId, { x, y, node }]) => {
      // Node circle
      ctx.beginPath();
      ctx.arc(x, y, nodeRadius, 0, Math.PI * 2);
      ctx.fillStyle = statusColors[node.status] || statusColors.pending;
      ctx.fill();
      
      // Node border
      ctx.strokeStyle = '#fff';
      ctx.lineWidth = 2;
      ctx.stroke();
      
      // Node label
      ctx.fillStyle = '#fff';
      ctx.font = '12px Arial';
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      ctx.fillText(node.job_type, x, y);
    });
  };
  
  // Get status icon based on status
  const getStatusIcon = (status, size = 18) => {
    switch (status) {
      case 'completed':
        return <CheckCircle size={size} />;
      case 'failed':
        return <XCircle size={size} />;
      case 'running':
      case 'processing':
        return <Clock size={size} className="animate-pulse" />;
      case 'pending':
        return <Clock size={size} />;
      case 'skipped':
        return <AlertTriangle size={size} />;
      default:
        return null;
    }
  };
  
  if (loading && !data?.data) {
    return (
      <div className="flex justify-center items-center p-8">
        <div className="animate-spin text-yellow-400">
          <RefreshCw size={32} />
        </div>
      </div>
    );
  }
  
  if (error) {
    return (
      <div className="p-8 bg-red-900/20 border-2 border-red-700 rounded-lg text-center">
        <AlertTriangle size={32} className="text-red-400 mx-auto mb-4" />
        <h3 className="text-red-400 font-bold mb-2">Error Loading Workflow</h3>
        <p className="text-white mb-4">{error.message || "Failed to load workflow"}</p>
        <button 
          onClick={execute} 
          className="text-yellow-400 flex items-center gap-1 mx-auto"
        >
          <RefreshCw size={16} />
          Try Again
        </button>
      </div>
    );
  }
  
  if (!data?.data) {
    return (
      <div className="p-8 border-2 border-gray-700 rounded-lg text-center">
        <h3 className="text-xl mb-4">Workflow Not Found</h3>
        <p className="mb-4">The workflow you are looking for does not exist or has been deleted.</p>
        <Link to="/workflows" className="text-yellow-400 flex items-center gap-1 max-w-fit mx-auto">
          <ArrowLeft size={16} />
          Back to Workflows
        </Link>
      </div>
    );
  }
  
  const workflow = data.data;
  
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
          <button 
            onClick={execute} 
            className="text-yellow-400 flex items-center gap-1"
          >
            <RefreshCw size={16} />
            Refresh
          </button>
          
          {workflow.status === 'pending' && (
            <button
              onClick={handleRunWorkflow}
              className="bg-gradient-to-r from-yellow-400 to-orange-500 text-black font-bold py-2 px-4 rounded hover:opacity-90 flex items-center gap-1"
            >
              <Play size={16} />
              Run Workflow
            </button>
          )}
        </div>
      </div>
      
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Main content */}
        <div className="md:col-span-2">
          <div className="border-2 border-gray-700 rounded-lg p-6 mb-6">
            <h3 className="text-xl mb-2">Description</h3>
            <p className="text-gray-300">{workflow.description}</p>
          </div>
          
          <div className="border-2 border-gray-700 rounded-lg p-6">
            <h3 className="text-xl mb-4">Workflow Visualization</h3>
            <div className="bg-gray-900 rounded-lg p-2 overflow-hidden">
              <canvas
                ref={canvasRef}
                width={800}
                height={400}
                className="w-full h-auto"
              />
            </div>
            
            <div className="flex flex-wrap justify-center gap-4 mt-4">
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-gray-500"></div>
                <span>Pending</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-yellow-400"></div>
                <span>Running</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-green-500"></div>
                <span>Completed</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-red-500"></div>
                <span>Failed</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-gray-600"></div>
                <span>Skipped</span>
              </div>
            </div>
          </div>
        </div>
        
        {/* Sidebar */}
        <div>
          <div className="border-2 border-gray-700 rounded-lg p-6 mb-6">
            <h3 className="text-xl mb-4">Workflow Info</h3>
            <div className="space-y-4">
              <div>
                <h4 className="text-gray-400 text-sm">ID</h4>
                <p className="font-mono">{workflow.id}</p>
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
                <p>{Object.keys(workflow.steps).length} steps</p>
              </div>
              {wsConnected ? (
                <div className="flex items-center text-green-400 text-xs">
                  <span className="inline-block w-2 h-2 rounded-full bg-green-400 mr-2"></span>
                  Live updates
                </div>
              ) : (
                <div className="flex items-center text-red-400 text-xs">
                  <span className="inline-block w-2 h-2 rounded-full bg-red-400 mr-2"></span>
                  Offline
                </div>
              )}
              {lastUpdated && (
                <div className="text-gray-500 text-xs">
                  Last updated: {lastUpdated.toLocaleTimeString()}
                </div>
              )}
            </div>
          </div>
          
          <div className="border-2 border-gray-700 rounded-lg p-6">
            <h3 className="text-xl mb-4">Step Details</h3>
            <div className="space-y-3 max-h-96 overflow-y-auto pr-2">
              {Object.entries(workflow.steps).map(([stepId, step]) => (
                <div key={stepId} className="border border-gray-700 rounded p-3 hover:border-gray-500">
                  <div className="flex justify-between items-center mb-2">
                    <h4 className="font-medium">{step.job_type}</h4>
                    <span className={getStatusColor(step.status)}>
                      {getStatusIcon(step.status, 16)}
                    </span>
                  </div>
                  <p className="text-sm text-gray-400 mb-1">ID: {stepId}</p>
                  {step.job_id && (
                    <Link to={`/jobs/${step.job_id}`} className="text-cyan-400 hover:underline text-sm block mt-2">
                      View Job
                    </Link>
                  )}
                  {step.error_message && (
                    <div className="mt-2 text-sm text-red-400 bg-red-900/20 p-2 rounded">
                      {step.error_message}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default WorkflowDetail;