/**
 * API Service for BoltQ
 * Provides functions to interact with the BoltQ backend API
 */

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api';
const API_VERSION = '/api/v1';
const API_URL = `${API_BASE_URL}${API_VERSION}`;

/**
 * Generic request function with error handling
 */
const request = async (endpoint, options = {}) => {
  try {
    const response = await fetch(`${API_URL}${endpoint}`, {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
      throw new Error(errorData.error || `Request failed with status ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`API request error (${endpoint}):`, error);
    throw error;
  }
};

// Job-related API functions
export const jobsApi = {
  // Get list of jobs
  getJobs: () => request('/jobs'),
  
  // Get a specific job by ID
  getJob: (jobId) => request(`/jobs/${jobId}`),
  
  // Submit a new job
  submitJob: (jobData) => request('/jobs', {
    method: 'POST',
    body: JSON.stringify(jobData),
  }),
  
  // Cancel a job
  cancelJob: (jobId) => request(`/jobs/${jobId}/cancel`, {
    method: 'POST',
  }),
};

// Queue-related API functions
export const queuesApi = {
  // Get queue stats
  getQueueStats: () => request('/queues/stats'),
};

// Workflow-related API functions
export const workflowsApi = {
  // Get list of workflows
  getWorkflows: () => request('/workflows'),
  
  // Get a specific workflow by ID
  getWorkflow: (workflowId) => request(`/workflows/${workflowId}`),
  
  // Create a new workflow
  createWorkflow: (workflowData) => request('/workflows', {
    method: 'POST',
    body: JSON.stringify(workflowData),
  }),
  
  // Delete a workflow
  deleteWorkflow: (workflowId) => request(`/workflows/${workflowId}`, {
    method: 'DELETE',
  }),
  
  // Run a workflow
  runWorkflow: (workflowId) => request(`/workflows/${workflowId}/run`, {
    method: 'POST',
  }),
};

// Health-related API functions
export const healthApi = {
  // Check the health of the API
  checkHealth: () => request('/health'),
};

export default {
  jobs: jobsApi,
  queues: queuesApi,
  workflows: workflowsApi,
  health: healthApi,
};