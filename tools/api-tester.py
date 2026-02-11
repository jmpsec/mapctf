#!/usr/bin/env python3
"""
API Tester for mapCTF API
Tests all available routes after authentication
"""

import argparse
import json
import sys
from typing import Dict, Optional, Tuple
import requests
from requests.exceptions import RequestException


class Colors:
    """ANSI color codes for terminal output"""
    GREEN = '\033[92m'
    RED = '\033[91m'
    YELLOW = '\033[93m'
    BLUE = '\033[94m'
    CYAN = '\033[96m'
    RESET = '\033[0m'
    BOLD = '\033[1m'


class APITester:
    """Test all API routes"""

    def __init__(self, base_url: str, username: str, password: str, ent_id: int = 0):
        self.base_url = base_url.rstrip('/')
        self.username = username
        self.password = password
        self.ent_id = ent_id
        self.token: Optional[str] = None
        self.session = requests.Session()
        self.results: list = []

    def print_header(self, text: str):
        """Print a formatted header"""
        print(f"\n{Colors.BOLD}{Colors.CYAN}{'='*60}{Colors.RESET}")
        print(f"{Colors.BOLD}{Colors.CYAN}{text:^60}{Colors.RESET}")
        print(f"{Colors.BOLD}{Colors.CYAN}{'='*60}{Colors.RESET}\n")

    def print_test(self, method: str, endpoint: str, status: str = "PENDING"):
        """Print test information"""
        status_color = Colors.YELLOW if status == "PENDING" else (
            Colors.GREEN if status == "PASS" else Colors.RED
        )
        status_symbol = "✓" if status == "PASS" else ("✗" if status == "FAIL" else "○")
        print(f"{status_color}{status_symbol}{Colors.RESET} {Colors.BOLD}{method:6}{Colors.RESET} {endpoint:40} ", end="")

    def print_result(self, status_code: int, response_time: float, details: str = ""):
        """Print test result"""
        if 200 <= status_code < 300:
            print(f"{Colors.GREEN}[{status_code}]{Colors.RESET} {response_time:.3f}s {details}")
            return "PASS"
        elif 300 <= status_code < 400:
            print(f"{Colors.YELLOW}[{status_code}]{Colors.RESET} {response_time:.3f}s {details}")
            return "PASS"
        else:
            print(f"{Colors.RED}[{status_code}]{Colors.RESET} {response_time:.3f}s {details}")
            return "FAIL"

    def authenticate(self) -> bool:
        """Authenticate and get JWT token"""
        self.print_header("Authentication")
        self.print_test("POST", "/api/v1/auth/login")

        url = f"{self.base_url}/api/v1/auth/login"
        payload = {
            "username": self.username,
            "password": self.password,
            "entID": self.ent_id
        }

        try:
            response = self.session.post(url, json=payload, timeout=10)
            response_time = response.elapsed.total_seconds()

            if response.status_code == 200:
                data = response.json()
                if data.get("success") and data.get("token"):
                    self.token = data["token"]
                    # Set Authorization header for subsequent requests
                    self.session.headers.update({
                        "Authorization": f"Bearer {self.token}"
                    })
                    result = self.print_result(
                        response.status_code,
                        response_time,
                        f"Token: {self.token[:20]}..."
                    )
                    self.results.append(("POST", "/api/v1/auth/login", result))
                    return True
                else:
                    result = self.print_result(
                        response.status_code,
                        response_time,
                        "No token in response"
                    )
                    self.results.append(("POST", "/api/v1/auth/login", result))
                    return False
            else:
                error_msg = ""
                try:
                    error_data = response.json()
                    error_msg = error_data.get("error", "")
                except:
                    error_msg = response.text[:50]

                result = self.print_result(
                    response.status_code,
                    response_time,
                    error_msg
                )
                self.results.append(("POST", "/api/v1/auth/login", result))
                return False

        except RequestException as e:
            print(f"{Colors.RED}[ERROR]{Colors.RESET} {str(e)}")
            self.results.append(("POST", "/api/v1/auth/login", "FAIL"))
            return False

    def test_route(self, method: str, endpoint: str, requires_auth: bool = False,
                   json_data: Optional[Dict] = None, expected_status: Optional[int] = None) -> Tuple[str, float]:
        """Test a single route"""
        self.print_test(method, endpoint)

        url = f"{self.base_url}{endpoint}"

        # Remove auth header if route doesn't require auth
        if not requires_auth:
            auth_header = self.session.headers.pop("Authorization", None)

        try:
            if method == "GET":
                response = self.session.get(url, timeout=10)
            elif method == "POST":
                response = self.session.post(url, json=json_data, timeout=10)
            elif method == "PUT":
                response = self.session.put(url, json=json_data, timeout=10)
            elif method == "PATCH":
                response = self.session.patch(url, json=json_data, timeout=10)
            elif method == "DELETE":
                response = self.session.delete(url, timeout=10)
            else:
                print(f"{Colors.RED}[ERROR]{Colors.RESET} Unsupported method: {method}")
                return "FAIL", 0.0

            response_time = response.elapsed.total_seconds()

            # Restore auth header if it was removed
            if not requires_auth and auth_header:
                self.session.headers["Authorization"] = auth_header

            # Determine expected status
            if expected_status is None:
                expected_status = 200 if method == "GET" else 200

            # Format response details
            details = ""
            try:
                if response.headers.get("Content-Type", "").startswith("application/json"):
                    data = response.json()
                    if isinstance(data, dict):
                        if "error" in data:
                            details = f"Error: {data['error']}"
                        elif "message" in data:
                            details = f"Message: {data['message']}"
                else:
                    text = response.text[:50]
                    if text:
                        details = f"Response: {text}"
            except:
                pass

            result = self.print_result(response.status_code, response_time, details)
            self.results.append((method, endpoint, result))

            return result, response_time

        except RequestException as e:
            print(f"{Colors.RED}[ERROR]{Colors.RESET} {str(e)}")
            self.results.append((method, endpoint, "FAIL"))
            return "FAIL", 0.0

    def run_all_tests(self):
        """Run all API tests"""
        # Authenticate first
        if not self.authenticate():
            print(f"\n{Colors.RED}Authentication failed. Cannot continue with authenticated routes.{Colors.RESET}")
            return

        self.print_header("Testing Public Routes")

        # Public routes (no authentication required)
        self.test_route("GET", "/", requires_auth=False, expected_status=302)
        self.test_route("GET", "/health", requires_auth=False)
        self.test_route("GET", "/error", requires_auth=False, expected_status=500)
        self.test_route("GET", "/forbidden", requires_auth=False, expected_status=403)
        self.test_route("GET", "/api/v1/checks-no-auth", requires_auth=False)

        self.print_header("Testing Authenticated Routes")

        # Authenticated routes
        self.test_route("GET", "/api/v1/auth/logout", requires_auth=True)

        self.print_header("Test Summary")
        self.print_summary()

    def print_summary(self):
        """Print test summary"""
        total = len(self.results)
        passed = sum(1 for _, _, status in self.results if status == "PASS")
        failed = sum(1 for _, _, status in self.results if status == "FAIL")

        print(f"\n{Colors.BOLD}Total Tests:{Colors.RESET} {total}")
        print(f"{Colors.GREEN}Passed:{Colors.RESET} {passed}")
        print(f"{Colors.RED}Failed:{Colors.RESET} {failed}")

        if failed > 0:
            print(f"\n{Colors.RED}Failed Tests:{Colors.RESET}")
            for method, endpoint, status in self.results:
                if status == "FAIL":
                    print(f"  {Colors.RED}✗{Colors.RESET} {method:6} {endpoint}")

        print()


def main():
    """Main entry point"""
    parser = argparse.ArgumentParser(
        description="Test all mapCTF API routes",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s -u admin@example.com -p password123 -U http://localhost:8080
  %(prog)s --username user@test.com --password secret --url https://api.example.com
        """
    )

    parser.add_argument(
        "-u", "--username",
        required=True,
        help="Username for authentication"
    )

    parser.add_argument(
        "-p", "--password",
        required=True,
        help="Password for authentication"
    )

    parser.add_argument(
        "-e", "--ent-id",
        type=int,
        default=0,
        help="Entity ID for authentication (default: 0)"
    )

    parser.add_argument(
        "-U", "--url",
        required=True,
        help="Base URL of the API (e.g., http://localhost:8080)"
    )

    args = parser.parse_args()

    # Create tester and run tests
    tester = APITester(args.url, args.username, args.password, args.ent_id)

    try:
        tester.run_all_tests()
    except KeyboardInterrupt:
        print(f"\n\n{Colors.YELLOW}Tests interrupted by user{Colors.RESET}")
        sys.exit(1)
    except Exception as e:
        print(f"\n{Colors.RED}Unexpected error: {e}{Colors.RESET}")
        sys.exit(1)

    # Exit with error code if any tests failed
    failed_count = sum(1 for _, _, status in tester.results if status == "FAIL")
    sys.exit(1 if failed_count > 0 else 0)


if __name__ == "__main__":
    main()
