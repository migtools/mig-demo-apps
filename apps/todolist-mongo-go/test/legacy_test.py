#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
Legacy test file - improved version of the original test.py
This maintains compatibility with the original test while adding improvements
"""

import argparse
from datetime import datetime
import json
import requests
import sys
import os

# Add the current directory to the path to import test utilities
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from test_utils import APIClient, TestDataManager, TestAssertions, TestConfig, wait_for_api_ready

def updateToDo(base_url, id, completed):
    """Update data to the todo application

    Args:
        base_url: url
        id: todo item id
        completed: bool

    Returns:
        {"updated": true/false}
    """
    client = APIClient(base_url)
    try:
        result = client.update_todo(id, completed)
        print("Task updated successfully!")
        return result.get("updated", False)
    except Exception as e:
        print(f"Error updating task: {e}")
        return False

def createToDo(base_url, description, completed):
    """Post data to the todo application

    Args:
        base_url: url
        description: todo list description
        completed: bool

    Returns:
        dict with todo item data
    """
    client = APIClient(base_url)
    try:
        result = client.create_todo(description, completed)
        print("Task created successfully!")
        return result
    except Exception as e:
        print(f"Error creating task: {e}")
        return {"Id": None, "Description": description, "Completed": completed}

def checkToDoLists(base_url, completed):
    """Get todo lists from the application

    Args:
        base_url: url
        completed: bool

    Returns:
        json list
    """
    client = APIClient(base_url)
    try:
        if completed:
            result = client.get_completed_todos()
        else:
            result = client.get_incomplete_todos()
        print("Got list of items")
        return result
    except Exception as e:
        print(f"Failed to get list of items: {e}")
        return []

def checkAppLog(base_url):
    """Get log data from the todo application
    Args:
        base_url: url

    Returns:
        bool 
    """
    client = APIClient(base_url)
    try:
        logs = client.get_logs()
        print("Got the log")
        return True
    except Exception as e:
        print(f"Failed to get the app log: {e}")
        return False

def deleteToDoItems(base_url, item):
    """Delete todo item from the application

    Args:
        base_url: url
        item: dict with todo item data

    Returns:
        bool
    """
    client = APIClient(base_url)
    try:
        result = client.delete_todo(item["Id"])
        print(f"Deleted item {item['Id']}")
        return result.get("deleted", False)
    except Exception as e:
        print(f"Failed to delete item {item['Id']}: {e}")
        return False

def run_comprehensive_test(base_url):
    """Run comprehensive test with improved error handling and reporting"""
    print("=" * 60)
    print("COMPREHENSIVE TODOLIST TEST")
    print("=" * 60)
    print(f"Base URL: {base_url}")
    print(f"Start Time: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print("=" * 60)
    
    # Check if API is ready
    if not wait_for_api_ready(base_url):
        print("❌ API is not ready. Please start the application first.")
        return False
    
    # Use TestDataManager for better test data management
    with TestDataManager(base_url) as test_data:
        try:
            # Create test items with unique descriptions
            date = datetime.today().strftime('%Y-%m-%d-%H:%M:%S')
            test1 = createToDo(base_url, f"pytest-1-{date}", False)   
            test2 = createToDo(base_url, f"pytest-2-{date}", False)
            test3 = createToDo(base_url, f"pytest-3-{date}", False)
            
            # Verify items were created
            if not test1.get("Id") or not test2.get("Id") or not test3.get("Id"):
                print("❌ FAILED: Could not create test items")
                return False
            
            # Update todo items
            success1 = updateToDo(base_url, test1["Id"], True)
            success2 = updateToDo(base_url, test2["Id"], True)
            
            if not success1 or not success2:
                print("❌ FAILED: Could not update test items")
                return False
            
            # Check todo lists
            completed = checkToDoLists(base_url, True)
            incomplete = checkToDoLists(base_url, False)
            
            print("COMPLETED ITEMS:")
            print(json.dumps(completed, indent=2))
            print("INCOMPLETE ITEMS:")
            print(json.dumps(incomplete, indent=2))
            
            # Test complete or incomplete
            found_completed = False
            for i in completed:
                if test1["Description"] == i["Description"]:
                    found_completed = True
                    break
            
            found_incomplete = False
            for i in incomplete:
                if test3["Description"] == i["Description"]:
                    found_incomplete = True
                    break
            
            if not found_completed or not found_incomplete:
                print("❌ FAILED complete / incomplete TEST")
                return False
            else:
                print("✅ SUCCESS: Complete/incomplete test passed!")
            
            # Delete items
            delete_success1 = deleteToDoItems(base_url, test1)
            delete_success3 = deleteToDoItems(base_url, test3)
            
            if not delete_success1 or not delete_success3:
                print("❌ FAILED: Could not delete test items")
                return False
            
            # Verify deletion
            completed_after_delete = checkToDoLists(base_url, True)
            incomplete_after_delete = checkToDoLists(base_url, False)
            
            # Test deleted items
            found_completed_after = False
            for i in completed_after_delete:
                if test1["Description"] == i["Description"]:
                    found_completed_after = True
                    break
            
            found_incomplete_after = False
            for i in incomplete_after_delete:
                if test3["Description"] == i["Description"]:
                    found_incomplete_after = True
                    break
            
            if found_completed_after or found_incomplete_after:
                print("❌ FAILED Delete TEST")
                return False
            else:
                print("✅ SUCCESS: Delete test passed!")
            
            # Test the app log
            if checkAppLog(base_url):
                print("✅ SUCCESS: Log test passed!")
            else:
                print("❌ FAILED: Log test failed!")
                return False
            
            print("\n🎉 FULL TEST PASSED!")
            return True
            
        except Exception as e:
            print(f"❌ TEST FAILED with exception: {e}")
            return False

def main():
    """Main function with improved argument handling"""
    parser = argparse.ArgumentParser(description='Test the todo application.')
    parser.add_argument('--base_url', default="http://localhost:8000",
                       help='The base URL of the todo application.')
    parser.add_argument('--comprehensive', action='store_true',
                       help='Run comprehensive test with improved error handling')
    
    args = parser.parse_args()
    base_url = args.base_url
    
    if args.comprehensive:
        success = run_comprehensive_test(base_url)
    else:
        # Run original test logic for backward compatibility
        print("Running original test logic...")
        
        # create date
        date = datetime.today().strftime('%Y-%m-%d-%H:%M:%S')
        
        # create todo items
        test1 = createToDo(base_url, "pytest-1-" + date, False)   
        test2 = createToDo(base_url, "pytest-2-" + date, False)
        test3 = createToDo(base_url, "pytest-1-" + date, False)
        
        # update todo items
        success = updateToDo(base_url, test1["Id"], True)
        success = updateToDo(base_url, test2["Id"], True)
        
        # check todo's
        completed = checkToDoLists(base_url, True)
        incomplete = checkToDoLists(base_url, False)
        print("COMPLETED ITEMS:")
        print(completed)
        print("INCOMPLETE ITEMS:")
        print(incomplete)
        
        # test complete or incomplete
        found_completed = False
        for i in completed:
            if test1["Description"] == i["Description"]:
                found_completed = True
                
        found_incomplete = False
        for i in incomplete:
            if test3["Description"] == i["Description"]:
                found_incomplete = True
        
        if found_completed == False or found_incomplete == False:
            print("FAILED complete / incomplete TEST")
            success = False
        else:
            print("SUCCESS!")
            success = True
        
        # Delete items
        deleteToDoItems(base_url, test1)
        deleteToDoItems(base_url, test3)
        completed = checkToDoLists(base_url, True)
        incomplete = checkToDoLists(base_url, False)
        
        # Test deleted items
        found_completed = False
        for i in completed:
            if test1["Description"] == i["Description"]:
                found_completed = True
        
        found_incomplete = False
        for i in incomplete:
            if test3["Description"] == i["Description"]:
                found_incomplete = True
        
        if found_completed == True or found_incomplete == True:
            print("FAILED Delete TEST")
            success = False
        else:
            print("SUCCESS!")
        
        # Test the app log
        if checkAppLog(base_url):
            print("LOG FOUND: SUCCESS!")
        else:
            print("FAILED!")
            success = False
        
        if success:
            print("FULL TEST PASSED!")
        else:
            print("FULL TEST FAILED!")
    
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()

