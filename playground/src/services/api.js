// Use `??` (not `||`) so an explicitly empty VITE_API_URL means "same origin":
// in production the app is served behind nginx, which proxies /api/ and injects
// the API key. Falls back to the dev API server only when the var is undefined.
const API_BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080';
const API_URL = `${API_BASE_URL}/api/v1`;

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

export const jobsApi = {
  getJobs: (limit = 20, offset = 0) => request(`/jobs?limit=${limit}&offset=${offset}`),
  getJob: (jobId) => request(`/jobs/${jobId}`),
  submitJob: (jobData) => request('/jobs', { method: 'POST', body: JSON.stringify(jobData) }),
  cancelJob: (jobId) => request(`/jobs/${jobId}/cancel`, { method: 'POST' }),
};

export const queuesApi = {
  getQueueStats: () => request('/queues/stats'),
};

export const workflowsApi = {
  getWorkflows: () => request('/workflows'),
  getWorkflow: (workflowId) => request(`/workflows/${workflowId}`),
  createWorkflow: (workflowData) => request('/workflows', { method: 'POST', body: JSON.stringify(workflowData) }),
  deleteWorkflow: (workflowId) => request(`/workflows/${workflowId}`, { method: 'DELETE' }),
};

export const healthApi = {
  checkHealth: () => request('/health'),
};

export default { jobs: jobsApi, queues: queuesApi, workflows: workflowsApi, health: healthApi };
