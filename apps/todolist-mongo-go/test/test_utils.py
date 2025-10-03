#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
Test utilities and helper functions for the todolist application
"""

import json
import requests
import time
import random
import string
from datetime import datetime
from typing import Dict, List, Optional, Tuple
import logging

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class TestConfig:
    """Configuration for tests"""
    DEFAULT_BASE_URL = "http://localhost:8000"
    TIMEOUT = 30
    MAX_RETRIES = 3
    RETRY_DELAY = 1

class TestDataManager:
    """Manages test data creation and cleanup"""
    
    def __init__(self, base_url: str = TestConfig.DEFAULT_BASE_URL):
        self.base_url = base_url
        self.created_items = []
        self.test_session = requests.Session()
        self.test_session.verify = False
        
    def generate_test_description(self, prefix: str = "test") -> str:
        """Generate a unique test description"""
        timestamp = datetime.now().strftime('%Y%m%d_%H%M%S_%f')
        random_suffix = ''.join(random.choices(string.ascii_lowercase, k=4))
        return f"{prefix}_{timestamp}_{random_suffix}"
    
    def create_test_item(self, description: str = None, completed: bool = False) -> Dict:
        """Create a test todo item and track it for cleanup"""
        if description is None:
            description = self.generate_test_description()
            
        data = {
            "description": description,
            "completed": completed
        }
        
        try:
            response = self.test_session.post(
                f"{self.base_url}/todo",
                data=data,
                timeout=TestConfig.TIMEOUT
            )
            response.raise_for_status()
            item = response.json()
            self.created_items.append(item)
            logger.info(f"Created test item: {item['Id']}")
            return item
        except requests.RequestException as e:
            logger.error(f"Failed to create test item: {e}")
            raise
    
    def cleanup_all_items(self) -> None:
        """Clean up all created test items"""
        for item in self.created_items:
            try:
                self.test_session.delete(
                    f"{self.base_url}/todo/{item['Id']}",
                    timeout=TestConfig.TIMEOUT
                )
                logger.info(f"Cleaned up test item: {item['Id']}")
            except requests.RequestException as e:
                logger.warning(f"Failed to cleanup item {item['Id']}: {e}")
        
        self.created_items.clear()
    
    def __enter__(self):
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        self.cleanup_all_items()

class APIClient:
    """Client for interacting with the todolist API"""
    
    def __init__(self, base_url: str = TestConfig.DEFAULT_BASE_URL):
        self.base_url = base_url
        self.session = requests.Session()
        self.session.verify = False
    
    def health_check(self) -> bool:
        """Check if the API is healthy"""
        try:
            response = self.session.get(
                f"{self.base_url}/healthz",
                timeout=TestConfig.TIMEOUT
            )
            return response.status_code == 200
        except requests.RequestException:
            return False
    
    def create_todo(self, description: str, completed: bool = False) -> Dict:
        """Create a new todo item"""
        data = {
            "description": description,
            "completed": completed
        }
        response = self.session.post(
            f"{self.base_url}/todo",
            data=data,
            timeout=TestConfig.TIMEOUT
        )
        response.raise_for_status()
        return response.json()
    
    def update_todo(self, todo_id: str, completed: bool) -> Dict:
        """Update a todo item"""
        data = {
            "id": todo_id,
            "completed": completed
        }
        response = self.session.post(
            f"{self.base_url}/todo/{todo_id}",
            data=data,
            timeout=TestConfig.TIMEOUT
        )
        response.raise_for_status()
        return response.json()
    
    def delete_todo(self, todo_id: str) -> Dict:
        """Delete a todo item"""
        response = self.session.delete(
            f"{self.base_url}/todo/{todo_id}",
            timeout=TestConfig.TIMEOUT
        )
        response.raise_for_status()
        return response.json()
    
    def get_completed_todos(self) -> List[Dict]:
        """Get all completed todos"""
        response = self.session.get(
            f"{self.base_url}/todo-completed",
            timeout=TestConfig.TIMEOUT
        )
        response.raise_for_status()
        return response.json()
    
    def get_incomplete_todos(self) -> List[Dict]:
        """Get all incomplete todos"""
        response = self.session.get(
            f"{self.base_url}/todo-incomplete",
            timeout=TestConfig.TIMEOUT
        )
        response.raise_for_status()
        return response.json()
    
    def get_logs(self) -> str:
        """Get application logs"""
        response = self.session.get(
            f"{self.base_url}/log",
            timeout=TestConfig.TIMEOUT
        )
        response.raise_for_status()
        return response.text

class TestAssertions:
    """Custom assertions for tests"""
    
    @staticmethod
    def assert_response_success(response: requests.Response, expected_status: int = 200):
        """Assert that a response is successful"""
        assert response.status_code == expected_status, \
            f"Expected status {expected_status}, got {response.status_code}: {response.text}"
    
    @staticmethod
    def assert_todo_item_structure(item: Dict):
        """Assert that a todo item has the expected structure"""
        assert "Id" in item, "Todo item missing 'Id' field"
        assert "Description" in item, "Todo item missing 'Description' field"
        assert "Completed" in item, "Todo item missing 'Completed' field"
        assert isinstance(item["Completed"], bool), "Completed field must be boolean"
    
    @staticmethod
    def assert_todo_in_list(todo: Dict, todo_list: List[Dict]) -> bool:
        """Assert that a todo item exists in a list"""
        for item in todo_list:
            if item["Id"] == todo["Id"]:
                return True
        return False
    
    @staticmethod
    def assert_todo_not_in_list(todo: Dict, todo_list: List[Dict]) -> bool:
        """Assert that a todo item does not exist in a list"""
        return not TestAssertions.assert_todo_in_list(todo, todo_list)

class PerformanceMetrics:
    """Track performance metrics during tests"""
    
    def __init__(self):
        self.response_times = []
        self.start_time = None
    
    def start_timer(self):
        """Start timing an operation"""
        self.start_time = time.time()
    
    def end_timer(self):
        """End timing an operation and record the result"""
        if self.start_time is not None:
            response_time = time.time() - self.start_time
            self.response_times.append(response_time)
            self.start_time = None
            return response_time
        return None
    
    def get_average_response_time(self) -> float:
        """Get average response time"""
        if not self.response_times:
            return 0.0
        return sum(self.response_times) / len(self.response_times)
    
    def get_max_response_time(self) -> float:
        """Get maximum response time"""
        return max(self.response_times) if self.response_times else 0.0
    
    def get_min_response_time(self) -> float:
        """Get minimum response time"""
        return min(self.response_times) if self.response_times else 0.0

def wait_for_api_ready(base_url: str = TestConfig.DEFAULT_BASE_URL, 
                      max_wait: int = 60) -> bool:
    """Wait for the API to be ready"""
    client = APIClient(base_url)
    start_time = time.time()
    
    while time.time() - start_time < max_wait:
        if client.health_check():
            logger.info("API is ready")
            return True
        time.sleep(1)
    
    logger.error(f"API not ready after {max_wait} seconds")
    return False

