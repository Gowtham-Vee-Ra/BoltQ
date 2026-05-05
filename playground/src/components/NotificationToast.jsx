import { useState, useCallback, useEffect } from 'react';
import { Bell, X } from 'lucide-react';
import { usePusherNotifications } from '../hooks/usePusherNotifications';

let nextId = 1;

const NotificationToast = () => {
  const [toasts, setToasts] = useState([]);

  const dismiss = useCallback((id) => {
    setToasts(prev => prev.filter(t => t.id !== id));
  }, []);

  const addToast = useCallback((data) => {
    const id = nextId++;
    setToasts(prev => [...prev.slice(-4), { id, ...data }]); // keep max 5
  }, []);

  usePusherNotifications(addToast);

  if (toasts.length === 0) return null;

  return (
    <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-3 max-w-sm">
      {toasts.map(toast => (
        <div
          key={toast.id}
          className="bg-gray-900 border-2 border-yellow-400 rounded-lg p-4 shadow-lg flex items-start gap-3 animate-pulse-once"
        >
          <Bell size={18} className="text-yellow-400 mt-0.5 shrink-0" />
          <div className="flex-1 min-w-0">
            {toast.recipient && (
              <p className="text-xs text-gray-400 mb-0.5">To: {toast.recipient}</p>
            )}
            <p className="text-sm text-white break-words">{toast.message}</p>
            <p className="text-xs text-gray-500 mt-1">
              {new Date(toast.delivered_at).toLocaleTimeString()}
            </p>
          </div>
          <button
            onClick={() => dismiss(toast.id)}
            className="text-gray-500 hover:text-white shrink-0"
          >
            <X size={14} />
          </button>
        </div>
      ))}
    </div>
  );
};

export default NotificationToast;
