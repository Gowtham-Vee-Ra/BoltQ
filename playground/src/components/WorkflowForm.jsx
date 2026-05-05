import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { ArrowLeft, Plus, Trash2, AlertTriangle } from 'lucide-react';
import { workflowsApi } from '../services/api';

const randomId = () => crypto.randomUUID();

const emptyStep = () => ({
  id: randomId(),
  job_type: 'echo',
  params: '{}',
  depends_on: [],
});

const WorkflowForm = () => {
  const navigate = useNavigate();
  const [name, setName] = useState('');
  const [steps, setSteps] = useState([emptyStep()]);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState(null);

  const addStep = () => setSteps(prev => [...prev, emptyStep()]);

  const removeStep = (id) => {
    setSteps(prev => {
      const removed = prev.filter(s => s.id !== id);
      // Clean up any dependencies that referenced the removed step
      return removed.map(s => ({ ...s, depends_on: s.depends_on.filter(d => d !== id) }));
    });
  };

  const updateStep = (id, field, value) => {
    setSteps(prev => prev.map(s => s.id === id ? { ...s, [field]: value } : s));
  };

  const toggleDep = (stepId, depId) => {
    setSteps(prev => prev.map(s => {
      if (s.id !== stepId) return s;
      const has = s.depends_on.includes(depId);
      return { ...s, depends_on: has ? s.depends_on.filter(d => d !== depId) : [...s.depends_on, depId] };
    }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError(null);

    if (!name.trim()) { setError('Workflow name is required.'); return; }
    if (steps.length === 0) { setError('Add at least one step.'); return; }

    // Validate params JSON for each step
    for (const step of steps) {
      try { JSON.parse(step.params); } catch {
        setError(`Step "${step.job_type}" has invalid JSON params.`);
        return;
      }
    }

    setSubmitting(true);
    try {
      const payload = {
        name: name.trim(),
        steps: steps.map(s => ({
          id: s.id,
          job_type: s.job_type,
          params: JSON.parse(s.params),
          depends_on: s.depends_on,
        })),
      };
      const res = await workflowsApi.createWorkflow(payload);
      navigate(`/workflows/${res.data.workflow_id}`);
    } catch (err) {
      setError(err.message || 'Failed to create workflow.');
      setSubmitting(false);
    }
  };

  return (
    <div className="max-w-3xl mx-auto">
      <div className="flex items-center gap-3 mb-6">
        <Link to="/workflows" className="text-gray-400 hover:text-white">
          <ArrowLeft size={20} />
        </Link>
        <h2 className="text-2xl">New Workflow</h2>
      </div>

      {error && (
        <div className="bg-red-900 text-white p-4 rounded mb-6 flex items-center gap-2">
          <AlertTriangle size={18} />
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit}>
        {/* Name */}
        <div className="border-2 border-gray-700 rounded-lg p-6 mb-6">
          <label className="block text-gray-400 text-sm mb-1">Workflow Name</label>
          <input
            type="text"
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder="e.g. Data Processing Pipeline"
            className="w-full bg-gray-900 border border-gray-700 rounded p-3 text-white focus:outline-none focus:border-yellow-400"
          />
        </div>

        {/* Steps */}
        <div className="mb-6">
          <div className="flex justify-between items-center mb-4">
            <h3 className="text-xl">Steps</h3>
            <button
              type="button"
              onClick={addStep}
              className="flex items-center gap-1 text-yellow-400 hover:text-yellow-300"
            >
              <Plus size={16} /> Add Step
            </button>
          </div>

          <div className="space-y-4">
            {steps.map((step, idx) => {
              const available = steps.filter(s => s.id !== step.id).slice(0, idx);
              return (
                <div key={step.id} className="border-2 border-gray-700 rounded-lg p-5">
                  <div className="flex justify-between items-center mb-4">
                    <span className="text-gray-400 text-sm">Step {idx + 1}</span>
                    {steps.length > 1 && (
                      <button
                        type="button"
                        onClick={() => removeStep(step.id)}
                        className="text-red-400 hover:text-red-300"
                      >
                        <Trash2 size={16} />
                      </button>
                    )}
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
                    <div>
                      <label className="block text-gray-400 text-sm mb-1">Job Type</label>
                      <select
                        value={step.job_type}
                        onChange={e => updateStep(step.id, 'job_type', e.target.value)}
                        className="w-full bg-gray-900 border border-gray-700 rounded p-2 text-white"
                      >
                        <option value="echo">echo</option>
                        <option value="sleep">sleep</option>
                        <option value="email">email</option>
                        <option value="notification">notification</option>
                        <option value="process-image">process-image</option>
                        <option value="generate-report">generate-report</option>
                      </select>
                    </div>

                    <div>
                      <label className="block text-gray-400 text-sm mb-1">Params (JSON)</label>
                      <input
                        type="text"
                        value={step.params}
                        onChange={e => updateStep(step.id, 'params', e.target.value)}
                        placeholder='{"key": "value"}'
                        className="w-full bg-gray-900 border border-gray-700 rounded p-2 text-white font-mono text-sm"
                      />
                    </div>
                  </div>

                  {available.length > 0 && (
                    <div>
                      <label className="block text-gray-400 text-sm mb-2">Depends on</label>
                      <div className="flex flex-wrap gap-2">
                        {available.map((dep, depIdx) => {
                          const checked = step.depends_on.includes(dep.id);
                          return (
                            <button
                              key={dep.id}
                              type="button"
                              onClick={() => toggleDep(step.id, dep.id)}
                              className={`px-3 py-1 rounded-full text-sm border transition-colors ${
                                checked
                                  ? 'bg-yellow-400 text-black border-yellow-400'
                                  : 'bg-gray-900 text-gray-400 border-gray-700 hover:border-gray-500'
                              }`}
                            >
                              Step {depIdx + 1} ({dep.job_type})
                            </button>
                          );
                        })}
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>

        <button
          type="submit"
          disabled={submitting}
          className="w-full bg-gradient-to-r from-yellow-400 to-orange-500 text-black font-bold py-3 px-6 rounded-lg hover:opacity-90 disabled:opacity-50"
        >
          {submitting ? 'Creating...' : 'Create Workflow'}
        </button>
      </form>
    </div>
  );
};

export default WorkflowForm;
