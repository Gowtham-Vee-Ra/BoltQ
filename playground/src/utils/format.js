/**
 * Format date to a human-readable string
 * @param {string|Date} date - The date to format
 * @returns {string} Formatted date string
 */
export const formatDate = (date) => {
    if (!date) return 'N/A';
    
    const dateObj = typeof date === 'string' ? new Date(date) : date;
    
    if (isNaN(dateObj.getTime())) {
      return 'Invalid date';
    }
    
    return dateObj.toLocaleString();
  };
  
  /**
   * Format time duration in milliseconds to a human-readable string
   * @param {number} ms - Duration in milliseconds
   * @returns {string} Formatted duration string
   */
  export const formatDuration = (ms) => {
    if (!ms || isNaN(ms)) return 'N/A';
    
    if (ms < 1000) {
      return `${ms}ms`;
    }
    
    if (ms < 60000) {
      return `${(ms / 1000).toFixed(2)}s`;
    }
    
    // Convert to seconds
    const seconds = Math.floor(ms / 1000);
    
    // Calculate hours, minutes, seconds
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const remainingSeconds = seconds % 60;
    
    // Format the string
    const parts = [];
    if (hours > 0) parts.push(`${hours}h`);
    if (minutes > 0) parts.push(`${minutes}m`);
    if (remainingSeconds > 0 || parts.length === 0) parts.push(`${remainingSeconds}s`);
    
    return parts.join(' ');
  };
  
  /**
   * Format a number as a percentage with specified precision
   * @param {number} value - The value to format as percentage
   * @param {number} precision - Number of decimal places
   * @returns {string} Formatted percentage string
   */
  export const formatPercentage = (value, precision = 1) => {
    if (value === null || value === undefined || isNaN(value)) {
      return 'N/A';
    }
    
    return `${value.toFixed(precision)}%`;
  };
  
  /**
   * Get color class based on job status
   * @param {string} status - The job status
   * @returns {string} Tailwind CSS color class
   */
  export const getStatusColor = (status) => {
    switch (status?.toLowerCase()) {
      case 'completed':
        return 'text-green-400';
      case 'failed':
        return 'text-red-400';
      case 'running':
      case 'processing':
        return 'text-yellow-400';
      case 'pending':
      case 'queued':
        return 'text-blue-400';
      case 'cancelled':
        return 'text-gray-400';
      case 'retrying':
        return 'text-purple-400';
      case 'scheduled':
        return 'text-cyan-400';
      default:
        return 'text-gray-400';
    }
  };
  
  /**
   * Get background color class based on job status
   * @param {string} status - The job status
   * @returns {string} Tailwind CSS color class
   */
  export const getStatusBgColor = (status) => {
    switch (status?.toLowerCase()) {
      case 'completed':
        return 'bg-green-900 bg-opacity-20';
      case 'failed':
        return 'bg-red-900 bg-opacity-20';
      case 'running':
      case 'processing':
        return 'bg-yellow-900 bg-opacity-20';
      case 'pending':
      case 'queued':
        return 'bg-blue-900 bg-opacity-20';
      case 'cancelled':
        return 'bg-gray-900 bg-opacity-20';
      case 'retrying':
        return 'bg-purple-900 bg-opacity-20';
      case 'scheduled':
        return 'bg-cyan-900 bg-opacity-20';
      default:
        return 'bg-gray-900 bg-opacity-20';
    }
  };
  
  /**
   * Truncate a string to the specified length
   * @param {string} str - The string to truncate
   * @param {number} length - The maximum length
   * @returns {string} Truncated string
   */
  export const truncate = (str, length = 30) => {
    if (!str) return '';
    
    return str.length > length 
      ? str.substring(0, length) + '...' 
      : str;
  };
  
  /**
   * Format JSON with indentation for display
   * @param {object} data - The data to format
   * @returns {string} Formatted JSON string
   */
  export const formatJSON = (data) => {
    try {
      return JSON.stringify(data, null, 2);
    } catch (err) {
      console.error('Error formatting JSON:', err);
      return String(data);
    }
  };
  
  /**
   * Parse priority number to human-readable string
   * @param {number} priority - The priority number
   * @returns {string} Human-readable priority
   */
  export const formatPriority = (priority) => {
    if (priority === undefined || priority === null) return 'Normal';
    
    switch (Number(priority)) {
      case 0:
        return 'High';
      case 1:
        return 'Normal';
      case 2:
        return 'Low';
      default:
        return `${priority}`;
    }
  };