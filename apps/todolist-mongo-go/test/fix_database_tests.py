#!/usr/bin/env python3
"""
Script to fix database tests by replacing TestAssertions.assert_todo_in_list calls
"""

import re

def fix_database_tests():
    file_path = '/home/whayutin/OPENSHIFT/git/OADP/mig-demo-apps/apps/todolist-mongo-go/test/database_tests.py'
    
    with open(file_path, 'r') as f:
        content = f.read()
    
    # Replace TestAssertions.assert_todo_in_list(todo, list) with proper logic
    def replace_assertion(match):
        todo_var = match.group(1)
        list_var = match.group(2)
        return f'any(item["Id"] == {todo_var}["Id"] for item in {list_var})'
    
    # Pattern to match TestAssertions.assert_todo_in_list(todo, list)
    pattern = r'TestAssertions\.assert_todo_in_list\(([^,]+),\s*([^)]+)\)'
    content = re.sub(pattern, replace_assertion, content)
    
    # Now we need to replace the assertTrue/assertFalse calls
    # Replace assertTrue(any(...)) with assertTrue(any(...), "message")
    content = re.sub(
        r'self\.assertTrue\(any\(item\["Id"\] == ([^"]+)\["Id"\] for item in ([^)]+)\)\)',
        r'self.assertTrue(any(item["Id"] == \1["Id"] for item in \2), f"Todo {\1[\'Id\']} not found in list")',
        content
    )
    
    # Replace assertFalse(any(...)) with assertFalse(any(...), "message")
    content = re.sub(
        r'self\.assertFalse\(any\(item\["Id"\] == ([^"]+)\["Id"\] for item in ([^)]+)\)\)',
        r'self.assertFalse(any(item["Id"] == \1["Id"] for item in \2), f"Todo {\1[\'Id\']} still found in list")',
        content
    )
    
    with open(file_path, 'w') as f:
        f.write(content)
    
    print("Database tests fixed!")

if __name__ == '__main__':
    fix_database_tests()
