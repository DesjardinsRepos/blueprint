#!/usr/bin/env python3
"""
Analyze and compare performance results from SockShop experiments
"""

import re
import sys
from pathlib import Path

def parse_results(filepath):
    """Extract final performance metrics from a result file"""
    try:
        with open(filepath) as f:
            content = f.read()
        
        # Look for results after the "=== Final Results ===" marker
        final_section = content.split('=== Final Results ===')
        if len(final_section) > 1:
            final_content = final_section[-1]  # Get the last occurrence
        else:
            final_content = content  # Fallback to full content
        
        # Look for the final results line
        match = re.search(
            r'\[(\d+\.\d+)s\] Requests: (\d+) \| Errors: (\d+) \| Throughput: ([\d.]+) req/s \| '
            r'Avg: ([\d.]+\w+) \| p50: ([\d.]+\w+) \| p95: ([\d.]+\w+) \| p99: ([\d.]+\w+)',
            final_content
        )
        
        if match:
            return {
                'duration': float(match.group(1)),
                'requests': int(match.group(2)),
                'errors': int(match.group(3)),
                'throughput': float(match.group(4)),
                'avg_latency': match.group(5),
                'p50': match.group(6),
                'p95': match.group(7),
                'p99': match.group(8),
            }
    except Exception as e:
        print(f"Error parsing {filepath}: {e}", file=sys.stderr)
    
    return None

def main():
    results_dir = Path('performance_results')
    
    if not results_dir.exists():
        print("No results directory found. Run ./run_comparison.sh first.")
        return
    
    configs = {
        'cmp_grpc_zipkin_micro': 'gRPC + Zipkin + Micro',
        'cmp_thrift_zipkin_micro': 'Thrift + Zipkin + Micro',
        'cmp_grpc_nozipkin_micro': 'gRPC + NoTrace + Micro',
        'cmp_thrift_nozipkin_micro': 'Thrift + NoTrace + Micro',
        'cmp_grpc_zipkin_mono': 'gRPC + Zipkin + Mono',
        'cmp_thrift_zipkin_mono': 'Thrift + Zipkin + Mono',
        'cmp_grpc_nozipkin_mono': 'gRPC + NoTrace + Mono',
        'cmp_thrift_nozipkin_mono': 'Thrift + NoTrace + Mono',
    }
    
    results = {}
    for config_key, config_name in configs.items():
        filepath = results_dir / f"{config_key}.txt"
        if filepath.exists():
            data = parse_results(filepath)
            if data:
                results[config_name] = data
    
    if not results:
        print("No valid results found.")
        return
    
    # Sort by throughput (descending)
    sorted_results = sorted(results.items(), key=lambda x: x[1]['throughput'], reverse=True)
    
    # Print comparison table
    print("\n=== Performance Comparison Results (Sorted by Throughput) ===\n")
    print(f"{'Rank':<6} {'Configuration':<30} {'Throughput':<15} {'Avg Latency':<15} {'p99 Latency':<15} {'Errors':<10}")
    print("-" * 96)
    
    for rank, (config_name, data) in enumerate(sorted_results, 1):
        print(f"{rank:<6} {config_name:<30} {data['throughput']:<15.2f} {data['avg_latency']:<15} {data['p99']:<15} {data['errors']:<10}")
    
    # Print analysis
    print("\n=== Analysis ===\n")
    
    # Compare gRPC vs Thrift (microservices, with zipkin)
    if 'gRPC + Zipkin + Micro' in results and 'Thrift + Zipkin + Micro' in results:
        grpc = results['gRPC + Zipkin + Micro']
        thrift = results['Thrift + Zipkin + Micro']
        diff = ((thrift['throughput'] - grpc['throughput']) / grpc['throughput']) * 100
        print(f"1. gRPC vs Thrift (microservices):")
        print(f"   Throughput difference: {diff:+.1f}%")
        print(f"   Winner: {'Thrift' if diff > 0 else 'gRPC'}\n")
    
    # Compare Zipkin on vs off (gRPC microservices)
    if 'gRPC + Zipkin + Micro' in results and 'gRPC + NoTrace + Micro' in results:
        with_zipkin = results['gRPC + Zipkin + Micro']
        without_zipkin = results['gRPC + NoTrace + Micro']
        diff = ((without_zipkin['throughput'] - with_zipkin['throughput']) / with_zipkin['throughput']) * 100
        print(f"2. Zipkin overhead (gRPC microservices):")
        print(f"   Throughput difference: {diff:+.1f}%")
        print(f"   Overhead: {abs(diff):.1f}%\n")
    
    # Compare Microservices vs Monolith (gRPC with zipkin)
    if 'gRPC + Zipkin + Micro' in results and 'gRPC + Zipkin + Mono' in results:
        micro = results['gRPC + Zipkin + Micro']
        mono = results['gRPC + Zipkin + Mono']
        diff = ((mono['throughput'] - micro['throughput']) / micro['throughput']) * 100
        print(f"3. Microservices vs Monolith (gRPC with zipkin):")
        print(f"   Throughput difference: {diff:+.1f}%")
        print(f"   Winner: {'Monolith' if diff > 0 else 'Microservices'}\n")
    
    # Print detailed results for each configuration
    print("\n=== Detailed Results ===\n")
    for rank, (config_name, data) in enumerate(sorted_results, 1):
        print(f"{rank}. {config_name}")
        print(f"   Duration:    {data['duration']:.1f}s")
        print(f"   Requests:    {data['requests']}")
        print(f"   Errors:      {data['errors']}")
        print(f"   Throughput:  {data['throughput']:.2f} req/s")
        print(f"   Avg Latency: {data['avg_latency']}")
        print(f"   p50 Latency: {data['p50']}")
        print(f"   p95 Latency: {data['p95']}")
        print(f"   p99 Latency: {data['p99']}")
        print()

if __name__ == '__main__':
    main()
