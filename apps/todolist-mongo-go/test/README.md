# Todolist Application Test Suite

This directory contains a comprehensive test suite for the todolist/mongo application, providing robust testing capabilities for all aspects of the application.

## Overview

The test suite includes:

- **Unit Tests** - Test individual components and functions
- **Integration Tests** - Test complete workflows and API interactions
- **Error Handling Tests** - Test error scenarios and edge cases
- **Performance Tests** - Test performance characteristics under various loads
- **Database Tests** - Test MongoDB operations and data consistency
- **Legacy Tests** - Backward-compatible tests for existing functionality

## Test Structure

```
test/
├── test_utils.py              # Test utilities and helper functions
├── unit_tests.py              # Unit tests for individual components
├── integration_tests.py       # Integration tests for API workflows
├── error_handling_tests.py    # Error handling and edge case tests
├── performance_tests.py       # Performance and load tests
├── database_tests.py          # Database-specific tests
├── legacy_test.py            # Improved version of original test.py
├── run_tests.py              # Comprehensive test runner
├── requirements.txt           # Test dependencies
├── README.md                 # This file
└── test.py                   # Original test file (for reference)
```

## Prerequisites

1. **Python 3.7+** - Required for running the tests
2. **Application Running** - The todolist application must be running
3. **MongoDB** - Database must be accessible
4. **Dependencies** - Install test dependencies

## Installation

1. Install Python dependencies:
```bash
pip install -r requirements.txt
```

2. Ensure the todolist application is running:
```bash
# From the project root
go run todolist.go
# OR
docker-compose up -d
```

## Running Tests

### Quick Start

Run all tests with the comprehensive test runner:
```bash
python run_tests.py
```

### Specific Test Suites

Run individual test suites:
```bash
# Unit tests only
python run_tests.py --suite unit

# Integration tests only
python run_tests.py --suite integration

# Error handling tests only
python run_tests.py --suite error

# Performance tests only
python run_tests.py --suite performance

# Database tests only
python run_tests.py --suite database
```

### Quick Smoke Tests

Run quick smoke tests for basic validation:
```bash
python run_tests.py --quick
```

### Legacy Tests

Run the improved version of the original test:
```bash
# Original test logic
python legacy_test.py --base_url http://localhost:8000

# Comprehensive test with better error handling
python legacy_test.py --base_url http://localhost:8000 --comprehensive
```

### Individual Test Files

Run specific test files directly:
```bash
# Unit tests
python -m unittest unit_tests

# Integration tests
python -m unittest integration_tests

# Error handling tests
python -m unittest error_handling_tests

# Performance tests
python -m unittest performance_tests

# Database tests
python -m unittest database_tests
```

## Test Configuration

### Environment Variables

- `TODOLIST_BASE_URL` - Base URL for the application (default: http://localhost:8000)
- `TODOLIST_TIMEOUT` - Request timeout in seconds (default: 30)
- `TODOLIST_MAX_RETRIES` - Maximum retry attempts (default: 3)

### Command Line Options

```bash
python run_tests.py --help
```

Available options:
- `--base-url` - Base URL of the todolist application
- `--suite` - Run specific test suite only
- `--quick` - Run quick smoke tests only
- `--verbose` - Verbose output
- `--parallel` - Run tests in parallel (experimental)

## Test Categories

### 1. Unit Tests (`unit_tests.py`)

Test individual components and functions:
- API client functionality
- Test data management
- Custom assertions
- Performance metrics tracking

### 2. Integration Tests (`integration_tests.py`)

Test complete workflows:
- CRUD operations
- API consistency
- Concurrent operations
- Performance characteristics

### 3. Error Handling Tests (`error_handling_tests.py`)

Test error scenarios:
- Invalid input handling
- Network error handling
- Concurrent error handling
- Edge cases and boundary conditions
- Resource limits

### 4. Performance Tests (`performance_tests.py`)

Test performance characteristics:
- Basic performance metrics
- Concurrent performance
- Load testing (light, medium, heavy)
- Resource usage monitoring
- Stress testing

### 5. Database Tests (`database_tests.py`)

Test database operations:
- Data consistency
- Transaction handling
- Concurrent database operations
- Performance characteristics
- Constraints and limitations
- Recovery scenarios

## Test Features

### Comprehensive Coverage

- **API Endpoints** - All REST endpoints tested
- **Data Validation** - Input validation and sanitization
- **Error Handling** - Graceful error handling
- **Performance** - Load and stress testing
- **Concurrency** - Multi-threaded operations
- **Database** - MongoDB operations and consistency

### Advanced Testing Capabilities

- **Test Data Management** - Automatic cleanup and isolation
- **Performance Metrics** - Response time tracking
- **Resource Monitoring** - Memory and CPU usage
- **Concurrent Testing** - Multi-threaded test execution
- **Error Simulation** - Network and database failure scenarios

### Reporting

- **Detailed Reports** - JSON reports with timestamps
- **Performance Metrics** - Response times and throughput
- **Error Tracking** - Comprehensive error reporting
- **Test Results** - Pass/fail status for all test suites

## Continuous Integration

### GitHub Actions Example

```yaml
name: Test Suite
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
    - name: Set up Python
      uses: actions/setup-python@v2
      with:
        python-version: 3.9
    - name: Install dependencies
      run: |
        pip install -r test/requirements.txt
    - name: Start application
      run: |
        # Start your application here
    - name: Run tests
      run: |
        python test/run_tests.py --base-url http://localhost:8000
```

### Docker Testing

```dockerfile
FROM python:3.9-slim
WORKDIR /app
COPY test/requirements.txt .
RUN pip install -r requirements.txt
COPY test/ .
CMD ["python", "run_tests.py"]
```

## Troubleshooting

### Common Issues

1. **API Not Ready**
   - Ensure the application is running
   - Check the base URL configuration
   - Verify network connectivity

2. **Database Connection Issues**
   - Ensure MongoDB is running
   - Check connection credentials
   - Verify database accessibility

3. **Test Failures**
   - Check application logs
   - Verify test data cleanup
   - Review error messages

4. **Performance Issues**
   - Monitor system resources
   - Check database performance
   - Review test configuration

### Debug Mode

Run tests with verbose output:
```bash
python run_tests.py --verbose
```

### Test Isolation

Each test suite runs in isolation with automatic cleanup to prevent test interference.

## Contributing

When adding new tests:

1. Follow the existing test structure
2. Use the test utilities provided
3. Ensure proper cleanup
4. Add appropriate documentation
5. Test error scenarios
6. Consider performance implications

## Best Practices

1. **Test Isolation** - Each test should be independent
2. **Cleanup** - Always clean up test data
3. **Error Handling** - Test both success and failure scenarios
4. **Performance** - Monitor test execution time
5. **Documentation** - Document test purpose and expected behavior
6. **Maintenance** - Keep tests up to date with application changes

## Support

For issues or questions about the test suite:

1. Check the application logs
2. Review test output and error messages
3. Verify application and database status
4. Check test configuration and dependencies

## License

This test suite is part of the todolist application and follows the same license terms.

