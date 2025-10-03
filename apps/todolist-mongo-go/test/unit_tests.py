#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
Unit tests for the todolist application
These tests focus on testing individual components and functions
"""

import unittest
import json
import requests
import time
from unittest.mock import Mock, patch, MagicMock
import sys
import os

# Add the parent directory to the path to import test utilities
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from test_utils import APIClient, TestDataManager, TestAssertions, TestConfig, wait_for_api_ready, PerformanceMetrics

class TestAPIClient(unittest.TestCase):
    """Test the API client functionality"""
    
    def setUp(self):
        """Set up test fixtures"""
        self.client = APIClient(TestConfig.DEFAULT_BASE_URL)
        self.test_data = TestDataManager(TestConfig.DEFAULT_BASE_URL)
    
    def tearDown(self):
        """Clean up after tests"""
        self.test_data.cleanup_all_items()
    
    def test_health_check_success(self):
        """Test successful health check"""
        with patch('requests.Session.get') as mock_get:
            mock_response = Mock()
            mock_response.status_code = 200
            mock_get.return_value = mock_response
            
            result = self.client.health_check()
            self.assertTrue(result)
    
    def test_health_check_failure(self):
        """Test health check failure"""
        with patch('requests.Session.get') as mock_get:
            mock_get.side_effect = requests.RequestException("Connection failed")
            
            result = self.client.health_check()
            self.assertFalse(result)
    
    def test_create_todo_success(self):
        """Test successful todo creation"""
        with patch('requests.Session.post') as mock_post:
            mock_response = Mock()
            mock_response.status_code = 201
            mock_response.json.return_value = {
                "Id": "507f1f77bcf86cd799439011",
                "Description": "Test todo",
                "Completed": False
            }
            mock_post.return_value = mock_response
            
            result = self.client.create_todo("Test todo", False)
            self.assertEqual(result["Description"], "Test todo")
            self.assertFalse(result["Completed"])
    
    def test_create_todo_http_error(self):
        """Test todo creation with HTTP error"""
        with patch('requests.Session.post') as mock_post:
            mock_response = Mock()
            mock_response.status_code = 500
            mock_response.raise_for_status.side_effect = requests.HTTPError("Server error")
            mock_post.return_value = mock_response
            
            with self.assertRaises(requests.HTTPError):
                self.client.create_todo("Test todo", False)
    
    def test_update_todo_success(self):
        """Test successful todo update"""
        with patch('requests.Session.post') as mock_post:
            mock_response = Mock()
            mock_response.status_code = 200
            mock_response.json.return_value = {"updated": True}
            mock_post.return_value = mock_response
            
            result = self.client.update_todo("507f1f77bcf86cd799439011", True)
            self.assertTrue(result["updated"])
    
    def test_delete_todo_success(self):
        """Test successful todo deletion"""
        with patch('requests.Session.delete') as mock_delete:
            mock_response = Mock()
            mock_response.status_code = 200
            mock_response.json.return_value = {"deleted": True}
            mock_delete.return_value = mock_response
            
            result = self.client.delete_todo("507f1f77bcf86cd799439011")
            self.assertTrue(result["deleted"])
    
    def test_get_completed_todos_success(self):
        """Test successful retrieval of completed todos"""
        with patch('requests.Session.get') as mock_get:
            mock_response = Mock()
            mock_response.status_code = 200
            mock_response.json.return_value = [
                {
                    "Id": "507f1f77bcf86cd799439011",
                    "Description": "Completed todo",
                    "Completed": True
                }
            ]
            mock_get.return_value = mock_response
            
            result = self.client.get_completed_todos()
            self.assertEqual(len(result), 1)
            self.assertTrue(result[0]["Completed"])
    
    def test_get_incomplete_todos_success(self):
        """Test successful retrieval of incomplete todos"""
        with patch('requests.Session.get') as mock_get:
            mock_response = Mock()
            mock_response.status_code = 200
            mock_response.json.return_value = [
                {
                    "Id": "507f1f77bcf86cd799439012",
                    "Description": "Incomplete todo",
                    "Completed": False
                }
            ]
            mock_get.return_value = mock_response
            
            result = self.client.get_incomplete_todos()
            self.assertEqual(len(result), 1)
            self.assertFalse(result[0]["Completed"])

class TestTestDataManager(unittest.TestCase):
    """Test the test data manager functionality"""
    
    def setUp(self):
        """Set up test fixtures"""
        self.test_data = TestDataManager(TestConfig.DEFAULT_BASE_URL)
    
    def tearDown(self):
        """Clean up after tests"""
        self.test_data.cleanup_all_items()
    
    def test_generate_test_description(self):
        """Test test description generation"""
        desc1 = self.test_data.generate_test_description("test")
        desc2 = self.test_data.generate_test_description("test")
        
        self.assertTrue(desc1.startswith("test_"))
        self.assertTrue(desc2.startswith("test_"))
        self.assertNotEqual(desc1, desc2)
    
    def test_generate_test_description_with_prefix(self):
        """Test test description generation with custom prefix"""
        desc = self.test_data.generate_test_description("custom")
        self.assertTrue(desc.startswith("custom_"))
    
    @patch('requests.Session.post')
    def test_create_test_item_success(self, mock_post):
        """Test successful test item creation"""
        mock_response = Mock()
        mock_response.status_code = 201
        mock_response.json.return_value = {
            "Id": "507f1f77bcf86cd799439011",
            "Description": "Test item",
            "Completed": False
        }
        mock_post.return_value = mock_response
        
        item = self.test_data.create_test_item("Test item", False)
        
        self.assertEqual(item["Description"], "Test item")
        self.assertFalse(item["Completed"])
        self.assertEqual(len(self.test_data.created_items), 1)
    
    @patch('requests.Session.post')
    def test_create_test_item_with_default_description(self, mock_post):
        """Test test item creation with default description"""
        mock_response = Mock()
        mock_response.status_code = 201
        mock_response.json.return_value = {
            "Id": "507f1f77bcf86cd799439011",
            "Description": "test_20231201_120000_abcd",
            "Completed": False
        }
        mock_post.return_value = mock_response
        
        item = self.test_data.create_test_item()
        
        self.assertTrue(item["Description"].startswith("test_"))
        self.assertEqual(len(self.test_data.created_items), 1)
    
    @patch('requests.Session.post')
    def test_create_test_item_failure(self, mock_post):
        """Test test item creation failure"""
        mock_post.side_effect = requests.RequestException("Connection failed")
        
        with self.assertRaises(requests.RequestException):
            self.test_data.create_test_item("Test item", False)
    
    @patch('requests.Session.delete')
    def test_cleanup_all_items(self, mock_delete):
        """Test cleanup of all test items"""
        # Add some mock items
        self.test_data.created_items = [
            {"Id": "507f1f77bcf86cd799439011"},
            {"Id": "507f1f77bcf86cd799439012"}
        ]
        
        mock_response = Mock()
        mock_response.status_code = 200
        mock_delete.return_value = mock_response
        
        self.test_data.cleanup_all_items()
        
        self.assertEqual(mock_delete.call_count, 2)
        self.assertEqual(len(self.test_data.created_items), 0)
    
    def test_context_manager(self):
        """Test TestDataManager as context manager"""
        with TestDataManager() as manager:
            self.assertIsInstance(manager, TestDataManager)
        # Cleanup should be called automatically

class TestTestAssertions(unittest.TestCase):
    """Test the custom assertions"""
    
    def test_assert_response_success(self):
        """Test successful response assertion"""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.text = "OK"
        
        # Should not raise an exception
        TestAssertions.assert_response_success(mock_response, 200)
    
    def test_assert_response_success_wrong_status(self):
        """Test response assertion with wrong status code"""
        mock_response = Mock()
        mock_response.status_code = 404
        mock_response.text = "Not Found"
        
        with self.assertRaises(AssertionError):
            TestAssertions.assert_response_success(mock_response, 200)
    
    def test_assert_todo_item_structure_valid(self):
        """Test todo item structure assertion with valid item"""
        valid_item = {
            "Id": "507f1f77bcf86cd799439011",
            "Description": "Test todo",
            "Completed": True
        }
        
        # Should not raise an exception
        TestAssertions.assert_todo_item_structure(valid_item)
    
    def test_assert_todo_item_structure_missing_id(self):
        """Test todo item structure assertion with missing ID"""
        invalid_item = {
            "Description": "Test todo",
            "Completed": True
        }
        
        with self.assertRaises(AssertionError):
            TestAssertions.assert_todo_item_structure(invalid_item)
    
    def test_assert_todo_item_structure_wrong_completed_type(self):
        """Test todo item structure assertion with wrong completed type"""
        invalid_item = {
            "Id": "507f1f77bcf86cd799439011",
            "Description": "Test todo",
            "Completed": "true"  # Should be boolean
        }
        
        with self.assertRaises(AssertionError):
            TestAssertions.assert_todo_item_structure(invalid_item)
    
    def test_assert_todo_in_list_present(self):
        """Test todo in list assertion when todo is present"""
        todo = {"Id": "507f1f77bcf86cd799439011"}
        todo_list = [
            {"Id": "507f1f77bcf86cd799439011", "Description": "Test 1"},
            {"Id": "507f1f77bcf86cd799439012", "Description": "Test 2"}
        ]
        
        result = TestAssertions.assert_todo_in_list(todo, todo_list)
        self.assertTrue(result)
    
    def test_assert_todo_in_list_absent(self):
        """Test todo in list assertion when todo is absent"""
        todo = {"Id": "507f1f77bcf86cd799439013"}
        todo_list = [
            {"Id": "507f1f77bcf86cd799439011", "Description": "Test 1"},
            {"Id": "507f1f77bcf86cd799439012", "Description": "Test 2"}
        ]
        
        result = TestAssertions.assert_todo_in_list(todo, todo_list)
        self.assertFalse(result)
    
    def test_assert_todo_not_in_list_present(self):
        """Test todo not in list assertion when todo is present"""
        todo = {"Id": "507f1f77bcf86cd799439011"}
        todo_list = [
            {"Id": "507f1f77bcf86cd799439011", "Description": "Test 1"},
            {"Id": "507f1f77bcf86cd799439012", "Description": "Test 2"}
        ]
        
        result = TestAssertions.assert_todo_not_in_list(todo, todo_list)
        self.assertFalse(result)
    
    def test_assert_todo_not_in_list_absent(self):
        """Test todo not in list assertion when todo is absent"""
        todo = {"Id": "507f1f77bcf86cd799439013"}
        todo_list = [
            {"Id": "507f1f77bcf86cd799439011", "Description": "Test 1"},
            {"Id": "507f1f77bcf86cd799439012", "Description": "Test 2"}
        ]
        
        result = TestAssertions.assert_todo_not_in_list(todo, todo_list)
        self.assertTrue(result)

class TestPerformanceMetrics(unittest.TestCase):
    """Test performance metrics tracking"""
    
    def setUp(self):
        """Set up test fixtures"""
        self.metrics = PerformanceMetrics()
    
    def test_timer_operations(self):
        """Test timer start and end operations"""
        self.metrics.start_timer()
        time.sleep(0.01)  # Small delay
        response_time = self.metrics.end_timer()
        
        self.assertIsNotNone(response_time)
        self.assertGreater(response_time, 0)
        self.assertEqual(len(self.metrics.response_times), 1)
    
    def test_multiple_timings(self):
        """Test multiple timing operations"""
        for _ in range(3):
            self.metrics.start_timer()
            time.sleep(0.001)
            self.metrics.end_timer()
        
        self.assertEqual(len(self.metrics.response_times), 3)
    
    def test_average_response_time(self):
        """Test average response time calculation"""
        self.metrics.response_times = [0.1, 0.2, 0.3]
        average = self.metrics.get_average_response_time()
        self.assertAlmostEqual(average, 0.2, places=1)
    
    def test_max_response_time(self):
        """Test maximum response time calculation"""
        self.metrics.response_times = [0.1, 0.3, 0.2]
        max_time = self.metrics.get_max_response_time()
        self.assertEqual(max_time, 0.3)
    
    def test_min_response_time(self):
        """Test minimum response time calculation"""
        self.metrics.response_times = [0.3, 0.1, 0.2]
        min_time = self.metrics.get_min_response_time()
        self.assertEqual(min_time, 0.1)
    
    def test_empty_metrics(self):
        """Test metrics with no data"""
        self.assertEqual(self.metrics.get_average_response_time(), 0.0)
        self.assertEqual(self.metrics.get_max_response_time(), 0.0)
        self.assertEqual(self.metrics.get_min_response_time(), 0.0)

if __name__ == '__main__':
    # Import time for the sleep test
    import time
    from test_utils import PerformanceMetrics
    
    unittest.main()
