#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
Error handling and edge case tests for the todolist application
These tests verify proper error handling and edge cases
"""

import unittest
import json
import time
import sys
import os
import requests

# Add the parent directory to the path to import test utilities
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from test_utils import (
    APIClient, TestDataManager, TestAssertions, TestConfig, 
    wait_for_api_ready
)

class TestInvalidInputHandling(unittest.TestCase):
    """Test handling of invalid inputs"""
    
    def setUp(self):
        """Set up test fixtures"""
        self.client = APIClient()
        self.test_data = TestDataManager()
        
        if not wait_for_api_ready():
            self.skipTest("API is not ready")
    
    def tearDown(self):
        """Clean up after tests"""
        self.test_data.cleanup_all_items()
    
    def test_create_todo_empty_description(self):
        """Test creating todo with empty description"""
        try:
            todo = self.client.create_todo("", False)
            # If it succeeds, verify the structure
            TestAssertions.assert_todo_item_structure(todo)
        except requests.RequestException as e:
            # Expected behavior - should handle empty description gracefully
            self.assertIn("400", str(e.response.status_code) if hasattr(e, 'response') else str(e))
    
    def test_create_todo_none_description(self):
        """Test creating todo with None description"""
        try:
            todo = self.client.create_todo(None, False)
            # If it succeeds, verify the structure
            TestAssertions.assert_todo_item_structure(todo)
        except (requests.RequestException, TypeError) as e:
            # Expected behavior - should handle None description gracefully
            pass
    
    def test_create_todo_very_long_description(self):
        """Test creating todo with very long description"""
        long_description = "A" * 10000  # 10KB description
        
        try:
            todo = self.client.create_todo(long_description, False)
            TestAssertions.assert_todo_item_structure(todo)
            self.assertEqual(todo["Description"], long_description)
        except requests.RequestException as e:
            # May be rejected due to length limits
            self.assertIn("400", str(e.response.status_code) if hasattr(e, 'response') else str(e))
    
    def test_create_todo_special_characters(self):
        """Test creating todo with special characters"""
        special_chars = "!@#$%^&*()_+-=[]{}|;':\",./<>?`~"
        
        todo = self.client.create_todo(special_chars, False)
        TestAssertions.assert_todo_item_structure(todo)
        self.assertEqual(todo["Description"], special_chars)
    
    def test_create_todo_unicode_characters(self):
        """Test creating todo with unicode characters"""
        unicode_description = "测试待办事项 🚀 émojis and spéciál chârs"
        
        todo = self.client.create_todo(unicode_description, False)
        TestAssertions.assert_todo_item_structure(todo)
        self.assertEqual(todo["Description"], unicode_description)
    
    def test_update_nonexistent_todo(self):
        """Test updating a todo that doesn't exist"""
        fake_id = "507f1f77bcf86cd799439999"  # Valid ObjectID format but likely doesn't exist
        
        try:
            result = self.client.update_todo(fake_id, True)
            # If it doesn't raise an exception, check the response
            if "error" in result:
                self.assertIn("Record Not Found", result["error"])
        except requests.RequestException as e:
            # Expected behavior for nonexistent todo
            pass
    
    def test_delete_nonexistent_todo(self):
        """Test deleting a todo that doesn't exist"""
        fake_id = "507f1f77bcf86cd799439999"  # Valid ObjectID format but likely doesn't exist
        
        try:
            result = self.client.delete_todo(fake_id)
            # If it doesn't raise an exception, check the response
            if "error" in result:
                self.assertIn("Record Not Found", result["error"])
        except requests.RequestException as e:
            # Expected behavior for nonexistent todo
            pass
    
    def test_invalid_object_id_format(self):
        """Test operations with invalid ObjectID format"""
        invalid_ids = [
            "invalid_id",
            "123",
            "not_a_valid_object_id",
            "",
            None
        ]
        
        for invalid_id in invalid_ids:
            if invalid_id is None or invalid_id == "":
                continue  # Skip None and empty string as they would cause different errors
            
            try:
                # Try to update with invalid ID
                self.client.update_todo(invalid_id, True)
            except (requests.RequestException, ValueError) as e:
                # Expected behavior for invalid ObjectID
                pass
            
            try:
                # Try to delete with invalid ID
                self.client.delete_todo(invalid_id)
            except (requests.RequestException, ValueError) as e:
                # Expected behavior for invalid ObjectID
                pass

class TestNetworkErrorHandling(unittest.TestCase):
    """Test handling of network errors and timeouts"""
    
    def setUp(self):
        """Set up test fixtures"""
        self.test_data = TestDataManager()
    
    def tearDown(self):
        """Clean up after tests"""
        self.test_data.cleanup_all_items()
    
    def test_connection_timeout(self):
        """Test behavior with connection timeout"""
        # Use a non-existent host to simulate connection timeout
        client = APIClient("http://192.168.1.999:8000")  # Non-routable IP
        
        # APIClient.health_check() catches exceptions and returns False
        result = client.health_check()
        self.assertFalse(result, "Health check should return False for connection timeout")
    
    def test_invalid_url(self):
        """Test behavior with invalid URL"""
        client = APIClient("http://invalid-url-that-does-not-exist.com:8000")
        
        # APIClient.health_check() catches exceptions and returns False
        result = client.health_check()
        self.assertFalse(result, "Health check should return False for invalid URL")
    
    def test_connection_refused(self):
        """Test behavior when connection is refused"""
        client = APIClient("http://localhost:9999")  # Port that's likely not in use
        
        # APIClient.health_check() catches exceptions and returns False
        result = client.health_check()
        self.assertFalse(result, "Health check should return False for connection refused")

    
    def test_concurrent_deletes_same_todo(self):
        """Test concurrent deletes of the same todo"""
        import threading
        import queue
        
        # Create a todo
        todo = self.client.create_todo("concurrent_delete_test", False)
        results = queue.Queue()
        errors = queue.Queue()
        
        def delete_worker(worker_id):
            """Worker function to delete the same todo"""
            try:
                result = self.client.delete_todo(todo["Id"])
                results.put(f"worker_{worker_id}_success")
            except Exception as e:
                errors.put(f"worker_{worker_id}: {e}")
        
        # Start multiple threads deleting the same todo
        threads = []
        for i in range(3):
            thread = threading.Thread(target=delete_worker, args=(i,))
            threads.append(thread)
            thread.start()
        
        # Wait for all threads to complete
        for thread in threads:
            thread.join()
        
        # At least one should succeed
        self.assertGreater(len(results.queue), 0)
        
        # Verify todo is deleted
        incomplete_todos = self.client.get_incomplete_todos()
        completed_todos = self.client.get_completed_todos()
        
        self.assertFalse(TestAssertions.assert_todo_in_list(todo, incomplete_todos))
        self.assertFalse(TestAssertions.assert_todo_in_list(todo, completed_todos))

class TestEdgeCases(unittest.TestCase):
    """Test edge cases and boundary conditions"""
    
    def setUp(self):
        """Set up test fixtures"""
        self.client = APIClient()
        self.test_data = TestDataManager()
        
        if not wait_for_api_ready():
            self.skipTest("API is not ready")
    
    def tearDown(self):
        """Clean up after tests"""
        self.test_data.cleanup_all_items()
    


if __name__ == '__main__':
    unittest.main()

