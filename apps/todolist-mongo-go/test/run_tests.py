#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
Comprehensive test runner for the todolist application
This script runs all test suites and provides detailed reporting
"""

import unittest
import sys
import os
import time
import argparse
import json
import subprocess
from datetime import datetime
from io import StringIO

# Add the current directory to the path
sys.path.append(os.path.dirname(os.path.abspath(__file__)))

from test_utils import wait_for_api_ready, TestConfig

class TestRunner:
    """Comprehensive test runner with reporting capabilities"""
    
    def __init__(self, base_url=None, verbose=False, parallel=False):
        self.base_url = base_url or TestConfig.DEFAULT_BASE_URL
        self.verbose = verbose
        self.parallel = parallel
        self.results = {}
        self.start_time = None
        self.end_time = None
        
    def run_all_tests(self):
        """Run all test suites"""
        print("=" * 80)
        print("TODOLIST APPLICATION - COMPREHENSIVE TEST SUITE")
        print("=" * 80)
        print(f"Base URL: {self.base_url}")
        print(f"Start Time: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
        print("=" * 80)
        
        self.start_time = time.time()
        
        # Clean database before running tests
        print("\n🧹 Cleaning database...")
        self._cleanup_database()
        
        # Check if API is ready
        if not self._check_api_ready():
            print("❌ API is not ready. Please start the application first.")
            return False
        
        # Run test suites
        test_suites = [
            ("Unit Tests", self._run_unit_tests),
            ("Integration Tests", self._run_integration_tests),
        ]
        
        all_passed = True
        
        for suite_name, test_function in test_suites:
            print(f"\n🧪 Running {suite_name}...")
            print("-" * 60)
            
            suite_passed = test_function()
            self.results[suite_name] = {
                "passed": suite_passed,
                "timestamp": datetime.now().isoformat()
            }
            
            if suite_passed:
                print(f"{suite_name}: ✅ PASSED")
            else:
                print(f"{suite_name}: ❌ FAILED")
                all_passed = False
        
        self.end_time = time.time()
        
        # Generate and display report
        self._generate_report()
        
        # Save report to file
        self._save_report_to_file()
        
        if all_passed:
            print(f"\n🎉 ALL TEST SUITES PASSED!")
            return True
        else:
            total_suites = len(self.results)
            passed_suites = sum(1 for result in self.results.values() if result["passed"])
            print(f"\n⚠️  {total_suites - passed_suites} TEST SUITE(S) FAILED")
            return False
    
    def _cleanup_database(self):
        """Clean up the database before running tests"""
        try:
            result = subprocess.run(['python', 'cleanup_database.py'], 
                                   capture_output=True, text=True, cwd='.')
            if result.returncode == 0:
                print("✅ Database cleaned up successfully")
            else:
                print(f"⚠️  Database cleanup had issues: {result.stderr}")
        except Exception as e:
            print(f"⚠️  Could not clean database: {e}")

    def _check_api_ready(self):
        """Check if the API is ready"""
        print("🔍 Checking API readiness...")
        return wait_for_api_ready(self.base_url, max_wait=30)
    
    def _run_unit_tests(self):
        """Run unit tests"""
        try:
            import unit_tests
            # Set the base URL for the unit tests
            unit_tests.TestConfig.DEFAULT_BASE_URL = self.base_url
            loader = unittest.TestLoader()
            suite = loader.loadTestsFromModule(unit_tests)
            
            # Use detailed output to show individual test results
            if self.verbose:
                print("=" * 60)
                print("RUNNING UNIT TESTS")
                print("=" * 60)
            
            runner = unittest.TextTestRunner(verbosity=2, stream=sys.stdout)
            result = runner.run(suite)
            
            if self.verbose:
                print("=" * 60)
                print(f"UNIT TESTS SUMMARY:")
                print(f"Tests run: {result.testsRun}")
                print(f"Failures: {len(result.failures)}")
                print(f"Errors: {len(result.errors)}")
                if result.failures:
                    print("FAILURES:")
                    for test, traceback in result.failures:
                        print(f"  - {test}: {traceback.splitlines()[-1]}")
                if result.errors:
                    print("ERRORS:")
                    for test, traceback in result.errors:
                        print(f"  - {test}: {traceback.splitlines()[-1]}")
                print("=" * 60)
            
            return result.wasSuccessful()
        except Exception as e:
            print(f"❌ Error running unit tests: {e}")
            return False
    
    def _run_integration_tests(self):
        """Run integration tests"""
        try:
            import integration_tests
            # Set the base URL for the integration tests
            integration_tests.TestConfig.DEFAULT_BASE_URL = self.base_url
            loader = unittest.TestLoader()
            suite = loader.loadTestsFromModule(integration_tests)
            
            # Use detailed output to show individual test results
            if self.verbose:
                print("=" * 60)
                print("RUNNING INTEGRATION TESTS")
                print("=" * 60)
            
            runner = unittest.TextTestRunner(verbosity=2, stream=sys.stdout)
            result = runner.run(suite)
            
            if self.verbose:
                print("=" * 60)
                print(f"INTEGRATION TESTS SUMMARY:")
                print(f"Tests run: {result.testsRun}")
                print(f"Failures: {len(result.failures)}")
                print(f"Errors: {len(result.errors)}")
                if result.failures:
                    print("FAILURES:")
                    for test, traceback in result.failures:
                        print(f"  - {test}: {traceback.splitlines()[-1]}")
                if result.errors:
                    print("ERRORS:")
                    for test, traceback in result.errors:
                        print(f"  - {test}: {traceback.splitlines()[-1]}")
                print("=" * 60)
            
            return result.wasSuccessful()
        except Exception as e:
            print(f"❌ Error running integration tests: {e}")
            return False
    
    def _run_error_handling_tests(self):
        """Run error handling tests"""
        try:
            import error_handling_tests
            # Set the base URL for the error handling tests
            error_handling_tests.TestConfig.DEFAULT_BASE_URL = self.base_url
            loader = unittest.TestLoader()
            suite = loader.loadTestsFromModule(error_handling_tests)
            
            # Use detailed output to show individual test results
            if self.verbose:
                print("=" * 60)
                print("RUNNING ERROR HANDLING TESTS")
                print("=" * 60)
            
            runner = unittest.TextTestRunner(verbosity=2, stream=sys.stdout)
            result = runner.run(suite)
            
            if self.verbose:
                print("=" * 60)
                print(f"ERROR HANDLING TESTS SUMMARY:")
                print(f"Tests run: {result.testsRun}")
                print(f"Failures: {len(result.failures)}")
                print(f"Errors: {len(result.errors)}")
                if result.failures:
                    print("FAILURES:")
                    for test, traceback in result.failures:
                        print(f"  - {test}: {traceback.splitlines()[-1]}")
                if result.errors:
                    print("ERRORS:")
                    for test, traceback in result.errors:
                        print(f"  - {test}: {traceback.splitlines()[-1]}")
                print("=" * 60)
            
            return result.wasSuccessful()
        except Exception as e:
            print(f"❌ Error running error handling tests: {e}")
            return False
    
    def _run_database_tests(self):
        """Run database tests"""
        try:
            import database_tests
            # Set the base URL for the database tests
            database_tests.TestConfig.DEFAULT_BASE_URL = self.base_url
            loader = unittest.TestLoader()
            suite = loader.loadTestsFromModule(database_tests)
            
            # Use detailed output to show individual test results
            if self.verbose:
                print("=" * 60)
                print("RUNNING DATABASE TESTS")
                print("=" * 60)
            
            runner = unittest.TextTestRunner(verbosity=2, stream=sys.stdout)
            result = runner.run(suite)
            
            if self.verbose:
                print("=" * 60)
                print(f"DATABASE TESTS SUMMARY:")
                print(f"Tests run: {result.testsRun}")
                print(f"Failures: {len(result.failures)}")
                print(f"Errors: {len(result.errors)}")
                if result.failures:
                    print("FAILURES:")
                    for test, traceback in result.failures:
                        print(f"  - {test}: {traceback.splitlines()[-1]}")
                if result.errors:
                    print("ERRORS:")
                    for test, traceback in result.errors:
                        print(f"  - {test}: {traceback.splitlines()[-1]}")
                print("=" * 60)
            
            return result.wasSuccessful()
        except Exception as e:
            print(f"❌ Error running database tests: {e}")
            return False
    
    def _generate_report(self):
        """Generate test report"""
        print("\n" + "=" * 80)
        print("TEST REPORT")
        print("=" * 80)
        
        total_suites = len(self.results)
        passed_suites = sum(1 for result in self.results.values() if result["passed"])
        failed_suites = total_suites - passed_suites
        success_rate = (passed_suites / total_suites * 100) if total_suites > 0 else 0
        
        print(f"Total Test Suites: {total_suites}")
        print(f"Passed Suites: {passed_suites}")
        print(f"Failed Suites: {failed_suites}")
        print(f"Success Rate: {success_rate:.1f}%")
        print(f"Total Execution Time: {self.end_time - self.start_time:.2f} seconds")
        
        print("\nDetailed Results:")
        print("-" * 40)
        for suite_name, result in self.results.items():
            status = "✅ PASSED" if result["passed"] else "❌ FAILED"
            print(f"{suite_name}: {status}")
    
    def _save_report_to_file(self):
        """Save test report to file"""
        report_data = {
            "timestamp": datetime.now().isoformat(),
            "base_url": self.base_url,
            "execution_time": self.end_time - self.start_time,
            "results": self.results
        }
        
        # Create test_report directory if it doesn't exist
        test_report_dir = "../test_report"
        os.makedirs(test_report_dir, exist_ok=True)
        
        report_file = os.path.join(test_report_dir, f"test_report_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json")
        
        try:
            with open(report_file, 'w') as f:
                json.dump(report_data, f, indent=2)
            print(f"\n📄 Report saved to: {report_file}")
        except Exception as e:
            print(f"⚠️  Could not save report: {e}")

def run_specific_test_suite(suite_name, base_url=None, verbose=False):
    """Run a specific test suite"""
    runner = TestRunner(base_url, verbose)
    
    if not runner._check_api_ready():
        print("❌ API is not ready. Please start the application first.")
        return False
    
    suite_functions = {
        "unit": runner._run_unit_tests,
        "integration": runner._run_integration_tests,
        "error": runner._run_error_handling_tests,
        "database": runner._run_database_tests
    }
    
    if suite_name not in suite_functions:
        print(f"❌ Unknown test suite: {suite_name}")
        print(f"Available suites: {', '.join(suite_functions.keys())}")
        return False
    
    print(f"🧪 Running {suite_name} tests...")
    return suite_functions[suite_name]()

def run_quick_tests(base_url=None):
    """Run quick smoke tests"""
    print("🚀 Running quick smoke tests...")
    
    # Run just the essential tests
    integration_success = run_specific_test_suite("integration", base_url)
    unit_success = run_specific_test_suite("unit", base_url)
    
    return integration_success and unit_success

def main():
    """Main entry point"""
    parser = argparse.ArgumentParser(description='Comprehensive test runner for todolist application')
    parser.add_argument('--base-url', default='http://localhost:8000',
                       help='Base URL of the todolist application')
    parser.add_argument('--suite', choices=['unit', 'integration', 'error', 'database'],
                       help='Run specific test suite only')
    parser.add_argument('--quick', action='store_true',
                       help='Run quick smoke tests only')
    parser.add_argument('--verbose', '-v', action='store_true',
                       help='Verbose output')
    parser.add_argument('--parallel', action='store_true',
                       help='Run tests in parallel (experimental)')
    
    args = parser.parse_args()
    
    if args.quick:
        success = run_quick_tests(args.base_url)
    elif args.suite:
        success = run_specific_test_suite(args.suite, args.base_url, args.verbose)
    else:
        runner = TestRunner(args.base_url, args.verbose, args.parallel)
        success = runner.run_all_tests()
    
    sys.exit(0 if success else 1)

if __name__ == '__main__':
    main()