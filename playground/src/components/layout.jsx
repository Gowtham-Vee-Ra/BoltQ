import React from 'react';
import { Link, Outlet, useLocation } from 'react-router-dom';
import { BoltIcon } from 'lucide-react';

const Layout = () => {
  const location = useLocation();
  
  // Helper to check if a nav item is active
  const isActive = (path) => location.pathname.startsWith(path);
  
  return (
    <div className="min-h-screen bg-black text-white flex flex-col font-arcade">
      {/* Header */}
      <header className="p-4 border-b border-gray-800">
        <div className="container mx-auto flex justify-between items-center">
          <Link to="/" className="flex items-center gap-2">
            <BoltIcon className="text-yellow-400" size={32} />
            <h1 className="text-2xl uppercase tracking-widest font-extrabold text-transparent bg-gradient-to-r from-yellow-400 via-orange-500 to-pink-600 bg-clip-text">
              Bolt<span className="text-cyan-400">Q</span>
            </h1>
          </Link>
          
          <nav>
            <ul className="flex space-x-6">
              <li>
                <Link 
                  to="/jobs" 
                  className={isActive('/jobs') ? "text-yellow-400" : "hover:text-yellow-400"}
                >
                  Jobs
                </Link>
              </li>
              <li>
                <Link 
                  to="/workflows" 
                  className={isActive('/workflows') ? "text-yellow-400" : "hover:text-yellow-400"}
                >
                  Workflows
                </Link>
              </li>
              <li>
                <Link 
                  to="/dashboard" 
                  className={isActive('/dashboard') ? "text-yellow-400" : "hover:text-yellow-400"}
                >
                  Dashboard
                </Link>
              </li>
            </ul>
          </nav>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 container mx-auto p-4">
        <Outlet />
      </main>

      {/* Footer */}
      <footer className="p-4 border-t border-gray-800 text-center text-gray-500 text-xs">
        © 2025 BoltQ. All rights reserved.
      </footer>
    </div>
  );
};

export default Layout;