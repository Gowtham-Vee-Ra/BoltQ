#!/usr/bin/env python3
# cmd/test/performance_test.py
"""
BoltQ Performance Testing Script
-------------------------------
This script submits a large number of jobs to BoltQ API
to test performance under load.

Usage:
    python performance_test.py [--jobs=1000] [--concurrency=10]
"""

import argparse
import json
import random
import time
import uuid
from concurrent.futures import ThreadPoolExecutor
import requests
import statistics

# Configuration
DEFAULT_JOBS = 1000
DEFAULT_CONCURRENCY = 10
API_URL = "http://localhost:8080/api/v1"

# Job types for testing
JOB_TYPES = ["echo", "sleep"]

def generate_job():
    """Generate a random job for testing."""
    job_type = random.choice(JOB_TYPES)
    
    # Create job data based on type
    if job_type == "echo":
        data = {
            "message": f"Test message {uuid.uuid4()}"
        }
    elif job_type == "sleep":
        data = {
            "seconds": random.uniform(0.1, 2.0)
        }
    
    # Randomly assign priority
    priority = random.choice([0, 1, 1, 1, 2])  # Weight normal priority higher
    
    # Randomly assign delay (80% no delay, 20% with delay)
    delay = 0
    if random.random() < 0.2:
        delay = random.randint(1, 10)
    
    return {
        "type": job_type,
        "data": data,
        "priority": priority,
        "delay_seconds": delay
    }

def submit_job(job_data):
    """Submit a job to the API and return timing information."""
    start_time = time.time()
    
    try:
        response = requests.post(
            f"{API_URL}/jobs",
            json=job_data,
            headers={"Content-Type": "application/json"}
        )
        
        end_time = time.time()
        duration = end_time - start_time
        
        if response.status_code == 200:
            job_id = response.json()["data"]["job_id"]
            return {
                "success": True,
                "job_id": job_id,
                "duration": duration,
                "status_code": response.status_code
            }
        else:
            return {
                "success": False,
                "error": response.text,
                "duration": duration,
                "status_code": response.status_code
            }
    except Exception as e:
        end_time = time.time()
        return {
            "success": False,
            "error": str(e),
            "duration": end_time - start_time,
            "status_code": 0
        }

def check_queue_stats():
    """Check queue statistics."""
    try:
        response = requests.get(f"{API_URL}/queues/stats")
        if response.status_code == 200:
            return response.json()["data"]
        return None
    except Exception as e:
        print(f"Error checking queue stats: {e}")
        return None

def run_test(num_jobs, concurrency):
    """Run the performance test with the given parameters."""
    print(f"Starting performance test with {num_jobs} jobs and concurrency of {concurrency}")
    
    # Generate all jobs first
    jobs = [generate_job() for _ in range(num_jobs)]
    
    # Record start time
    overall_start = time.time()
    
    # Submit jobs concurrently
    results = []
    with ThreadPoolExecutor(max_workers=concurrency) as executor:
        results = list(executor.map(submit_job, jobs))
    
    # Record end time
    overall_end = time.time()
    
    # Calculate statistics
    success_count = sum(1 for r in results if r["success"])
    failure_count = num_jobs - success_count
    
    # Calculate timing statistics
    durations = [r["duration"] for r in results]
    avg_duration = statistics.mean(durations)
    min_duration = min(durations)
    max_duration = max(durations)
    median_duration = statistics.median(durations)
    
    # Calculate percentiles
    percentile_90 = sorted(durations)[int(len(durations) * 0.9)]
    percentile_95 = sorted(durations)[int(len(durations) * 0.95)]
    percentile_99 = sorted(durations)[int(len(durations) * 0.99)]
    
    # Calculate throughput
    total_time = overall_end - overall_start
    throughput = num_jobs / total_time
    
    # Print results
    print("\n--- Performance Test Results ---")
    print(f"Total Jobs: {num_jobs}")
    print(f"Concurrency: {concurrency}")
    print(f"Total Time: {total_time:.2f} seconds")
    print(f"Success Rate: {success_count}/{num_jobs} ({success_count/num_jobs*100:.2f}%)")
    print("\nTiming Statistics (seconds):")
    print(f"  Average: {avg_duration:.4f}")
    print(f"  Median: {median_duration:.4f}")
    print(f"  Min: {min_duration:.4f}")
    print(f"  Max: {max_duration:.4f}")
    print(f"  90th Percentile: {percentile_90:.4f}")
    print(f"  95th Percentile: {percentile_95:.4f}")
    print(f"  99th Percentile: {percentile_99:.4f}")
    print(f"\nThroughput: {throughput:.2f} jobs/second")
    
    # Check queue stats
    print("\nQueue Statistics:")
    queue_stats = check_queue_stats()
    if queue_stats:
        print(json.dumps(queue_stats, indent=2))
    
    return {
        "total_jobs": num_jobs,
        "concurrency": concurrency,
        "success_count": success_count,
        "failure_count": failure_count,
        "total_time": total_time,
        "avg_duration": avg_duration,
        "median_duration": median_duration,
        "min_duration": min_duration,
        "max_duration": max_duration,
        "percentile_90": percentile_90,
        "percentile_95": percentile_95,
        "percentile_99": percentile_99,
        "throughput": throughput
    }

def main():
    """Main entry point for the script."""
    parser = argparse.ArgumentParser(description="BoltQ Performance Testing Script")
    parser.add_argument("--jobs", type=int, default=DEFAULT_JOBS, help="Number of jobs to submit")
    parser.add_argument("--concurrency", type=int, default=DEFAULT_CONCURRENCY, help="Concurrency level")
    
    args = parser.parse_args()
    
    try:
        # Check if the API is available
        requests.get(f"{API_URL}/queues/stats")
    except Exception as e:
        print(f"Error: API not available at {API_URL}. Is BoltQ running?")
        print(f"Exception: {e}")
        return
    
    # Run the test
    run_test(args.jobs, args.concurrency)

if __name__ == "__main__":
    main()