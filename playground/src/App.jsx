import React from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import Layout from './components/layout';
import Home from './components/Home';
import JobList from './components/JobList';
import JobDetail from './components/JobDetail';
import JobForm from './components/JobForm';
import WorkflowList from './components/WorkflowList';
import WorkflowDetail from './components/WorkflowDetail';
import Dashboard from './components/Dashboard';

const App = () => {
  return (
    <Routes>
      <Route path="/" element={<Layout />}>
        <Route index element={<Home />} />
        <Route path="jobs">
          <Route index element={<JobList />} />
          <Route path="new" element={<JobForm />} />
          <Route path=":jobId" element={<JobDetail />} />
        </Route>
        <Route path="workflows">
          <Route index element={<WorkflowList />} />
          <Route path=":workflowId" element={<WorkflowDetail />} />
        </Route>
        <Route path="dashboard" element={<Dashboard />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
};

export default App;