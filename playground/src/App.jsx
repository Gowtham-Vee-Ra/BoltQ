// src/App.jsx
import React from 'react';
import { BoltIcon } from 'lucide-react';

export default function App() {
  return (
    <main className="min-h-screen bg-black text-white flex flex-col items-center justify-center font-arcade">
      <h1 className="text-6xl uppercase tracking-widest font-extrabold text-transparent bg-gradient-to-r from-yellow-400 via-orange-500 to-pink-600 bg-clip-text drop-shadow-[2px_2px_0_rgba(0,0,0,0.6)] mb-8">
        Bolt<span className="text-cyan-400">Q</span>
      </h1>
      <BoltIcon className="text-yellow-400 animate-pulse mb-4" size={64} />
      <div className="border-4 border-white p-8 rounded-2xl w-[90%] max-w-xl text-center">
        <p className="mb-6 text-xl">Welcome to the BoltQ Playground</p>
        <ul className="space-y-4">
          <li className="hover:text-yellow-400 cursor-pointer">▶ Submit a Job</li>
          <li className="hover:text-yellow-400 cursor-pointer">▶ Check Job Status</li>
          <li className="hover:text-yellow-400 cursor-pointer">▶ View Queue Stats</li>
          <li className="hover:text-yellow-400 cursor-pointer">▶ Exit</li>
        </ul>
      </div>
      <p className="mt-10 text-gray-500 text-xs">© 2025 BoltQ. All rights reserved.</p>
    </main>
  );
}
