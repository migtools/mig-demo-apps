#!/usr/bin/env python3
"""
Database cleanup script for todolist application
This script clears all todo items from the database using the API
"""

import requests
import sys
import argparse
import time
from datetime import datetime

def cleanup_database(base_url="http://localhost:8000", timeout=30):
    """Clean up all todo items from the database"""
    print(f"🧹 Cleaning database at {base_url}...")
    
    session = requests.Session()
    session.verify = False
    
    try:
        # Check if API is healthy
        health_response = session.get(f"{base_url}/healthz", timeout=timeout)
        if health_response.status_code != 200:
            print("❌ API health check failed")
            return False
        
        # Get all incomplete todos
        incomplete_response = session.get(f"{base_url}/todo-incomplete", timeout=timeout)
        if incomplete_response.status_code == 200:
            incomplete_todos = incomplete_response.json()
            print(f"Found {len(incomplete_todos)} incomplete todos")
            
            # Delete each incomplete todo
            for todo in incomplete_todos:
                try:
                    delete_response = session.delete(f"{base_url}/todo/{todo['Id']}", timeout=timeout)
                    if delete_response.status_code in [200, 201]:
                        print(f"✅ Deleted todo: {todo['Id']}")
                    else:
                        print(f"⚠️  Failed to delete todo {todo['Id']}: {delete_response.status_code}")
                except Exception as e:
                    print(f"⚠️  Error deleting todo {todo['Id']}: {e}")
        
        # Get all completed todos
        completed_response = session.get(f"{base_url}/todo-completed", timeout=timeout)
        if completed_response.status_code == 200:
            completed_todos = completed_response.json()
            print(f"Found {len(completed_todos)} completed todos")
            
            # Delete each completed todo
            for todo in completed_todos:
                try:
                    delete_response = session.delete(f"{base_url}/todo/{todo['Id']}", timeout=timeout)
                    if delete_response.status_code in [200, 201]:
                        print(f"✅ Deleted todo: {todo['Id']}")
                    else:
                        print(f"⚠️  Failed to delete todo {todo['Id']}: {delete_response.status_code}")
                except Exception as e:
                    print(f"⚠️  Error deleting todo {todo['Id']}: {e}")
        
        print("✅ Database cleanup completed")
        return True
        
    except Exception as e:
        print(f"❌ Error during cleanup: {e}")
        return False

def main():
    parser = argparse.ArgumentParser(description='Clean up todolist database')
    parser.add_argument('--base-url', default='http://localhost:8000',
                       help='Base URL of the todolist application')
    parser.add_argument('--timeout', type=int, default=30,
                       help='Request timeout in seconds')
    parser.add_argument('--quiet', '-q', action='store_true',
                       help='Quiet mode - minimal output')
    
    args = parser.parse_args()
    
    if args.quiet:
        # Redirect output to suppress messages
        import os
        with open(os.devnull, 'w') as devnull:
            sys.stdout = devnull
            sys.stderr = devnull
    
    success = cleanup_database(args.base_url, args.timeout)
    sys.exit(0 if success else 1)

if __name__ == '__main__':
    main()

