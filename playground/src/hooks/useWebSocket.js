import { useEffect, useRef, useState, useCallback } from 'react';

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api';

/**
 * Custom hook for managing WebSocket connections
 * @param {string} path - The WebSocket endpoint path
 * @returns {Object} { connected, message, send }
 */
const useWebSocket = (path) => {
  const [connected, setConnected] = useState(false);
  const [message, setMessage] = useState(null);
  const wsRef = useRef(null);
  const reconnectTimeoutRef = useRef(null);

  const getWebSocketUrl = useCallback(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    
    // If we're using a relative URL or external URL
    if (API_BASE_URL.startsWith('/')) {
      return `${protocol}//${window.location.host}${API_BASE_URL}/ws${path}`;
    } else {
      // Extract host from API_BASE_URL for external URLs
      const apiUrl = new URL(API_BASE_URL);
      return `${protocol}//${apiUrl.host}/ws${path}`;
    }
  }, [path]);

  const connect = useCallback(() => {
    try {
      // Close existing connection if any
      if (wsRef.current) {
        wsRef.current.close();
      }

      // Create new WebSocket connection
      wsRef.current = new WebSocket(getWebSocketUrl());

      wsRef.current.onopen = () => {
        console.log(`WebSocket connected: ${path}`);
        setConnected(true);
        // Clear any reconnect timeout if we successfully connect
        if (reconnectTimeoutRef.current) {
          clearTimeout(reconnectTimeoutRef.current);
          reconnectTimeoutRef.current = null;
        }
      };

      wsRef.current.onclose = () => {
        console.log(`WebSocket disconnected: ${path}`);
        setConnected(false);
        
        // Try to reconnect after a delay unless component is unmounted
        reconnectTimeoutRef.current = setTimeout(() => {
          if (document.visibilityState !== 'hidden') {
            connect();
          }
        }, 5000); // 5 second reconnection delay
      };

      wsRef.current.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          setMessage(data);
        } catch (err) {
          console.error('Error parsing WebSocket message:', err);
        }
      };

      wsRef.current.onerror = (error) => {
        console.error('WebSocket error:', error);
        setConnected(false);
      };
    } catch (err) {
      console.error('Error creating WebSocket connection:', err);
    }
  }, [path, getWebSocketUrl]);

  // Connect when the component mounts
  useEffect(() => {
    connect();

    // Set up reconnection on visibility change
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible' && !connected) {
        connect();
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);

    // Cleanup WebSocket on unmount
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [connect, connected]);

  // Send a message through the WebSocket
  const send = useCallback((data) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data));
      return true;
    }
    return false;
  }, []);

  return { connected, message, send };
};

export default useWebSocket;