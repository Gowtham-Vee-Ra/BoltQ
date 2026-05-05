import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { BoltIcon, ArrowLeft, AlertTriangle } from 'lucide-react';

import { jobsApi } from '../services/api';

const JobForm = () => {
  const navigate = useNavigate();
  
  // State for form inputs
  const [jobType, setJobType] = useState('echo');
  const [jsonData, setJsonData] = useState('{\n  "message": "Hello World"\n}');
  const [priority, setPriority] = useState(1);
  const [delay, setDelay] = useState(0);
  
  // State for form submission
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState(null);
  
  // Handle form submission
  const handleSubmit = async (e) => {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    
    try {
      // Parse JSON data
      let parsedData;
      try {
        parsedData = JSON.parse(jsonData);
      } catch (err) {
        throw new Error('Invalid JSON in job data');
      }
      
      // Create request body
      const requestBody = {
        type: jobType,
        data: parsedData,
        priority: parseInt(priority),
        delay_seconds: parseInt(delay)
      };
      
      // Submit the job
      const response = await jobsApi.submitJob(requestBody);
      
      // Navigate to job details page on success
      if (response.success && response.data?.job_id) {
        navigate(`/jobs/${response.data.job_id}`);
      } else {
        throw new Error('Failed to get job ID from response');
      }
      
    } catch (err) {
      setError(err.message);
      setSubmitting(false);
    }
  };
  
  return (
    <div>
      <div className="flex items-center mb-6">
        <Link to="/jobs" className="text-gray-400 hover:text-white mr-4">
          <ArrowLeft size={20} />
        </Link>
        <h2 className="text-2xl">Submit a New Job</h2>
      </div>
      
      {error && (
        <div className="bg-red-900 text-white p-4 rounded mb-6">
          <AlertTriangle className="inline-block mr-2" size={20} />
          {error}
        </div>
      )}
      
      <form onSubmit={handleSubmit} className="border-2 border-gray-700 rounded-lg p-6">
        <div className="mb-4">
          <label className="block text-yellow-400 mb-2">Job Type</label>
          <select
            value={jobType}
            onChange={(e) => setJobType(e.target.value)}
            className="w-full bg-gray-900 text-white border-2 border-gray-700 rounded p-2"
          >
            <option value="echo">Echo</option>
            <option value="sleep">Sleep</option>
            <option value="email">Email</option>
            <option value="notification">Notification</option>
            <option value="process-image">Process Image</option>
            <option value="generate-report">Generate Report</option>
          </select>
        </div>
        
        <div className="mb-4">
          <label className="block text-yellow-400 mb-2">Job Data (JSON)</label>
          <textarea
            value={jsonData}
            onChange={(e) => setJsonData(e.target.value)}
            className="w-full bg-gray-900 text-white border-2 border-gray-700 rounded p-2 font-mono h-64"
            placeholder="Enter job data as JSON"
          />
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
          <div>
            <label className="block text-yellow-400 mb-2">Priority</label>
            <select
              value={priority}
              onChange={(e) => setPriority(e.target.value)}
              className="w-full bg-gray-900 text-white border-2 border-gray-700 rounded p-2"
            >
              <option value="2">High</option>
              <option value="1">Normal</option>
              <option value="0">Low</option>
            </select>
          </div>
          
          <div>
            <label className="block text-yellow-400 mb-2">Delay (seconds)</label>
            <input
              type="number"
              min="0"
              value={delay}
              onChange={(e) => setDelay(e.target.value)}
              className="w-full bg-gray-900 text-white border-2 border-gray-700 rounded p-2"
            />
          </div>
        </div>
        
        <div className="flex justify-center">
          <button
            type="submit"
            disabled={submitting}
            className="bg-gradient-to-r from-yellow-400 to-orange-500 text-black font-bold py-2 px-8 rounded flex items-center disabled:opacity-50 disabled:hover:bg-yellow-500"
          >
            {submitting ? (
              <>
                <BoltIcon size={20} className="mr-2 animate-pulse" />
                Submitting...
              </>
            ) : (
              <>
                <BoltIcon size={20} className="mr-2" />
                Submit Job
              </>
            )}
          </button>
        </div>
      </form>
      
      {/* Quick templates */}
      <div className="mt-6 border-2 border-gray-700 rounded-lg p-6">
        <h3 className="text-xl mb-4">Quick Templates</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <button
            onClick={() => { setJobType('echo'); setJsonData(JSON.stringify({ message: "Hello from BoltQ!" }, null, 2)); }}
            className="p-3 border border-gray-700 rounded hover:border-yellow-400 hover:bg-gray-900 text-left"
          >
            <div className="font-medium mb-1">Echo</div>
            <div className="text-xs text-gray-400">Returns whatever you send it</div>
          </button>

          <button
            onClick={() => { setJobType('sleep'); setJsonData(JSON.stringify({ seconds: 5 }, null, 2)); }}
            className="p-3 border border-gray-700 rounded hover:border-yellow-400 hover:bg-gray-900 text-left"
          >
            <div className="font-medium mb-1">Sleep</div>
            <div className="text-xs text-gray-400">Waits N seconds then completes</div>
          </button>

          <button
            onClick={() => {
              setJobType('email');
              setJsonData(JSON.stringify({ to: "user@example.com", subject: "Hello from BoltQ", body: "This is a test email." }, null, 2));
            }}
            className="p-3 border border-gray-700 rounded hover:border-yellow-400 hover:bg-gray-900 text-left"
          >
            <div className="font-medium mb-1">Email</div>
            <div className="text-xs text-gray-400">Simulates sending an email</div>
          </button>

          <button
            onClick={() => {
              setJobType('notification');
              setJsonData(JSON.stringify({ message: "Your job has completed.", recipient: "Gowtham", channel: "general" }, null, 2));
            }}
            className="p-3 border border-gray-700 rounded hover:border-yellow-400 hover:bg-gray-900 text-left"
          >
            <div className="font-medium mb-1">Notification</div>
            <div className="text-xs text-gray-400">Simulates a push notification</div>
          </button>

          <button
            onClick={() => {
              setJobType('process-image');
              setJsonData(JSON.stringify({ url: "https://example.com/photo.jpg", width: 1280, height: 720, format: "webp" }, null, 2));
            }}
            className="p-3 border border-gray-700 rounded hover:border-yellow-400 hover:bg-gray-900 text-left"
          >
            <div className="font-medium mb-1">Process Image</div>
            <div className="text-xs text-gray-400">Simulates resize + format conversion</div>
          </button>

          <button
            onClick={() => {
              setJobType('generate-report');
              setJsonData(JSON.stringify({ report_type: "monthly", format: "pdf" }, null, 2));
            }}
            className="p-3 border border-gray-700 rounded hover:border-yellow-400 hover:bg-gray-900 text-left"
          >
            <div className="font-medium mb-1">Generate Report</div>
            <div className="text-xs text-gray-400">Simulates generating a PDF report</div>
          </button>
        </div>
      </div>
    </div>
  );
};

export default JobForm;