import time
from collections import deque, defaultdict
import math
import statistics

# Represents a single request record, adapted from your .NET structure.
# Added 'status_code' which is crucial for error rate calculation.
class RequestRecord:
    def __init__(self, timestamp, method, path, query, headers, content_length, status_code=200):
        self.timestamp = timestamp
        self.method = method
        self.path = path
        self.query = query
        self.headers = headers if headers is not None else {} # Ensure headers is a dict
        self.content_length = content_length
        self.status_code = status_code

    def __repr__(self):
        return (f"RequestRecord(Path='{self.path}', Method='{self.method}', "
                f"Status={self.status_code}, Time={self.timestamp:.2f})")

class IPSScoringEngine:
    """
    Manages request history for different IPs, calculates features, and assigns a risk score.
    This version incorporates dynamic baselines for behavioral anomaly detection.
    """
    def __init__(self, time_window_seconds=60, ban_threshold=10, baseline_history_size=20):
        """
        Initializes the IPS Scoring Engine.

        Args:
            time_window_seconds (int): The duration in seconds for which raw request history is kept.
            ban_threshold (int): The score threshold at which an IP is banned.
            baseline_history_size (int): The number of past feature sets to keep for baseline calculation.
                                         A larger number makes the baseline more stable but less reactive.
        """
        # Stores raw requests for each IP address within the time window.
        # Format: {ip_address: deque of RequestRecord objects}
        self.ip_request_history = defaultdict(lambda: deque())
        self.time_window = time_window_seconds
        self.ban_threshold = ban_threshold
        self.banned_ips = set()

        # Stores historical *feature values* for each IP, used for dynamic baseline calculation.
        # Format: {ip_address: {feature_name: deque of past feature values}}
        self.ip_feature_history = defaultdict(lambda: defaultdict(deque))
        self.baseline_history_size = baseline_history_size

        # Define common sensitive paths for rule-based detection
        self.admin_paths = ["/admin", "/dashboard", "/settings/users", "/management"]
        self.login_paths = ["/login", "/signin", "/auth", "/authenticate"]

    def add_request(self, ip_address, request_record):
        """
        Adds a new request record for an IP, updates history, calculates features and score.

        Args:
            ip_address (str): The IP address of the client.
            request_record (RequestRecord): The details of the incoming request.
        """
        # Add the new request to the raw request history
        self.ip_request_history[ip_address].append(request_record)
        # Remove requests that are older than the defined time window
        self._prune_old_requests(ip_address)

        # Calculate current features for the IP
        current_features = self.get_features(ip_address)

        # Update the historical feature values for baseline calculation
        self._update_feature_history(ip_address, current_features)

        # Calculate the score for the IP based on current and historical features
        score = self.calculate_score(ip_address, current_features)
        print(f"IP: {ip_address}, Current Score: {score:.2f}")

        # Check if the IP should be banned
        if score >= self.ban_threshold and ip_address not in self.banned_ips:
            self.ban_ip(ip_address, score)

    def _prune_old_requests(self, ip_address):
        """
        Removes raw request records older than the time window for a given IP address.
        This keeps the history relevant to the defined time window.
        """
        current_time = time.time()
        while self.ip_request_history[ip_address] and \
              self.ip_request_history[ip_address][0].timestamp < current_time - self.time_window:
            self.ip_request_history[ip_address].popleft()

    def _update_feature_history(self, ip_address, current_features):
        """
        Appends current feature values to the historical feature deques for baseline calculation.
        Prunes old feature values to maintain the baseline_history_size.
        """
        # Features to track for baselines
        features_to_track = [
            "request_rate", "unique_paths_count", "error_rate", "payload_stddev", "path_entropy"
        ]

        for feature_name in features_to_track:
            if feature_name in current_features:
                self.ip_feature_history[ip_address][feature_name].append(current_features[feature_name])
                # Keep only the most recent 'baseline_history_size' feature values
                while len(self.ip_feature_history[ip_address][feature_name]) > self.baseline_history_size:
                    self.ip_feature_history[ip_address][feature_name].popleft()

    def get_features(self, ip_address):
        """
        Calculates various behavioral features for a given IP based on its request history
        within the current time window. It also calculates deviations from historical baselines.

        Args:
            ip_address (str): The IP address to calculate features for.

        Returns:
            dict: A dictionary of calculated features, including current values and deviations.
        """
        requests = list(self.ip_request_history[ip_address])
        num_requests = len(requests)

        # Return default features if no requests are present to avoid division by zero
        if num_requests == 0:
            return {
                "request_count": 0, "request_rate": 0, "unique_paths_count": 0,
                "unique_paths_ratio": 0, "header_changes": 0, "payload_stddev": 0,
                "error_rate": 0, "login_attempts": 0, "failed_login_attempts": 0,
                "admin_path_attempts": 0, "invalid_token_attempts": 0,
                "path_entropy": 0,
                # Add default deviation features
                "request_rate_deviation": 0, "unique_paths_count_deviation": 0,
                "error_rate_deviation": 0, "payload_stddev_deviation": 0,
                "path_entropy_deviation": 0
            }

        # Determine the effective duration of the requests for rate calculations
        first_timestamp = requests[0].timestamp
        last_timestamp = requests[-1].timestamp
        # Ensure duration is at least 1 second to prevent division by zero for very fast requests
        actual_window_duration = last_timestamp - first_timestamp if last_timestamp > first_timestamp else 1

        # Initialize feature accumulators
        unique_paths = set()
        header_snapshots = defaultdict(set)
        payload_lengths = []
        error_responses = 0
        login_attempts = 0
        failed_login_attempts = 0
        admin_path_attempts = 0
        invalid_token_attempts = 0
        path_counts = defaultdict(int) # For path entropy calculation

        for req in requests:
            unique_paths.add(req.path)
            payload_lengths.append(req.content_length)

            path_counts[req.path] += 1

            if 400 <= req.status_code < 600:
                error_responses += 1

            if any(login_path in req.path for login_path in self.login_paths):
                login_attempts += 1
                if req.status_code in [401, 403]:
                    failed_login_attempts += 1

            if any(admin_path in req.path for admin_path in self.admin_paths):
                admin_path_attempts += 1

            for key, value in req.headers.items():
                header_snapshots[key.lower()].add(value)

            auth_header = req.headers.get('authorization', '').lower()
            if auth_header and (not auth_header.startswith('bearer ') or len(auth_header) < 10):
                invalid_token_attempts += 1

        header_changes = sum(1 for key, values in header_snapshots.items() if len(values) > 1)

        path_entropy = 0
        total_paths_visited_for_entropy = sum(path_counts.values())
        if total_paths_visited_for_entropy > 0:
            for count in path_counts.values():
                probability = count / total_paths_visited_for_entropy
                if probability > 0:
                    path_entropy -= probability * math.log2(probability)
            max_entropy = math.log2(len(unique_paths)) if len(unique_paths) > 1 else 0
            if max_entropy > 0:
                path_entropy /= max_entropy

        current_request_rate = num_requests / actual_window_duration
        current_unique_paths_count = len(unique_paths)
        current_payload_stddev = statistics.stdev(payload_lengths) if len(payload_lengths) > 1 else 0
        current_error_rate = error_responses / num_requests if num_requests > 0 else 0


        # Compile all calculated current features
        features = {
            "request_count": num_requests,
            "request_rate": current_request_rate,
            "unique_paths_count": current_unique_paths_count,
            "unique_paths_ratio": len(unique_paths) / num_requests if num_requests > 0 else 0,
            "header_changes": header_changes,
            "payload_stddev": current_payload_stddev,
            "error_rate": current_error_rate,
            "login_attempts": login_attempts,
            "failed_login_attempts": failed_login_attempts,
            "admin_path_attempts": admin_path_attempts,
            "invalid_token_attempts": invalid_token_attempts,
            "path_entropy": path_entropy
        }

        # Calculate deviations from historical baselines
        historical_features = self.ip_feature_history[ip_address]
        for feature_name, current_value in [
            ("request_rate", current_request_rate),
            ("unique_paths_count", current_unique_paths_count),
            ("error_rate", current_error_rate),
            ("payload_stddev", current_payload_stddev),
            ("path_entropy", path_entropy)
        ]:
            history = list(historical_features[feature_name])
            if len(history) > 1: # Need at least 2 data points for stddev
                mean = statistics.mean(history)
                stdev = statistics.stdev(history)
                if stdev > 0:
                    features[f"{feature_name}_deviation"] = (current_value - mean) / stdev
                else: # If stddev is 0, all historical values are the same. Deviation is 0 unless current is different.
                    features[f"{feature_name}_deviation"] = 0 if current_value == mean else 100 # Large deviation if different
            else: # Not enough history to calculate deviation meaningfully
                features[f"{feature_name}_deviation"] = 0 # Default to no deviation

        return features

    def calculate_score(self, ip_address, features):
        """
        Calculates the overall risk score for an IP address based on its features,
        including deviations from its dynamic baseline, and predefined rules.

        Args:
            ip_address (str): The IP address to score.
            features (dict): The dictionary of calculated features for the current request window.

        Returns:
            float: The calculated risk score.
        """
        score = 0.0

        # --- Scoring Rules based on Attacker Types and Behavioral Anomalies ---

        # 1. Endpoint Exploration Attacker (checks multiple routes)
        # High unique paths count, high unique paths ratio, high path entropy, moderate to high request rate
        if features["unique_paths_count"] > 10:
            score += 0.5 # Basic scanning
        if features["unique_paths_ratio"] > 0.5 and features["request_rate"] > 5:
            score += 1.0 # More aggressive scanning
        # Stronger emphasis on deviation from normal path entropy
        if features["path_entropy_deviation"] > 2.0: # Current path entropy is 2+ stddevs above average
            score += 2.0
        elif features["path_entropy"] > 0.7: # Still score high entropy if no baseline or less deviation
            score += 1.0

        # 2. Basic DDoS Congestion Attacker (high request rate, often to few endpoints)
        # Very high request rate, low path entropy (hitting same endpoint repeatedly)
        if features["request_rate"] > 20:
            score += 2.0 # High volume
        if features["request_rate"] > 50:
            score += 4.0 # Very aggressive DDoS
        # Significant deviation in request rate is a strong indicator
        if features["request_rate_deviation"] > 3.0:
            score += 3.0 # Current rate is 3+ stddevs above average
        if features["path_entropy"] < 0.2 and features["request_rate"] > 10: # Low entropy + high rate
            score += 1.5 # Concentrated attack

        # 3. Slower/Faster Attacker (variable request rate, bursty traffic)
        # Detected by significant positive deviation in request rate.
        if features["request_rate_deviation"] > 2.5 and features["request_count"] > 30:
            score += 2.5 # Detects significant bursts of activity
        elif features["request_rate"] > 15 and features["request_count"] > 50:
            score += 1.0 # Basic burst detection if deviation isn't strong yet

        # 4. Multiple Routes Attacker (does actual app workflow to avoid suspicion)
        # This is still challenging. We rely on other specific indicators.
        # If they make mistakes (e.g., access forbidden routes, use invalid tokens), other rules will catch them.
        # A low error rate might indicate sophistication, but isn't directly a malicious indicator.
        # However, if their path entropy is *unusually low* for their typical behavior (e.g., suddenly hitting only one path)
        # or if they suddenly start hitting admin paths, it will be caught.

        # 5. High Error Attacker (will always do an input that triggers an exception)
        # High error rate and deviation in error rate
        if features["error_rate"] > 0.3:
            score += 1.5 # Significant errors
        if features["error_rate"] > 0.6:
            score += 2.5 # Very high error rate
        if features["error_rate_deviation"] > 2.0: # Current error rate is 2+ stddevs above average
            score += 2.0

        # 6. Header Spoofer Attacker (will change headers trying to access admin routes)
        # High header changes
        if features["header_changes"] > 3:
            score += 1.5 # Frequent header changes are suspicious
        if features["header_changes"] > 5:
            score += 2.5 # Highly suspicious header manipulation
        # Combine with admin path attempts for higher score
        if features["header_changes"] > 2 and features["admin_path_attempts"] > 0:
            score += 2.0 # Header changes combined with admin attempts

        # 7. Session Hijacker Attacker (modifies tokens to access admin routes)
        # Invalid token attempts, unauthorized access attempts
        if features["invalid_token_attempts"] > 0:
            score += 3.0 # Even one invalid token attempt is highly suspicious
        if features["failed_login_attempts"] > 0 and features["invalid_token_attempts"] > 0:
            score += 2.0 # Combination of failed logins and token issues is worse

        # 8. Brute Force Permissions Attacker (will attack the login page until it finds an admin user)
        # High login attempts, high failed login attempts
        if features["login_attempts"] > 5:
            score += 0.5 # Moderate login attempts
        if features["failed_login_attempts"] > 3:
            score += 2.0 # Multiple failed logins is a strong indicator of brute force
        if features["failed_login_attempts"] > 5 and features["login_attempts"] > 10:
            score += 3.0 # Very aggressive brute force attempt

        # General suspicious behavior / Anomaly from baseline
        # Very uniform payloads (low stddev) that deviate significantly from normal payload stddev
        if features["payload_stddev_deviation"] < -1.5 and features["request_count"] > 10:
            score += 1.0 # Payloads are unusually uniform compared to history
        elif features["payload_stddev"] < 5 and features["request_count"] > 10:
            score += 0.5 # Uniform payloads, but not necessarily anomalous for this IP

        if features["admin_path_attempts"] > 0:
            score += 1.5 # Any attempt to access admin paths is suspicious if not expected from the user

        return score

    def ban_ip(self, ip_address, score):
        """
        Adds an IP to the banned list and prints a notification.
        In a real system, this would trigger an actual ban mechanism (e.g., firewall rule).
        """
        self.banned_ips.add(ip_address)
        print(f"\n!!! IP {ip_address} banned with score {score:.2f} !!!\n")

    def is_banned(self, ip_address):
        """
        Checks if an IP is currently banned.
        """
        return ip_address in self.banned_ips

# --- Example Usage and Simulation ---

def simulate_requests(ips_engine, scenario_name, ip, requests_data):
    """
    Helper function to simulate a series of requests for a given IP.

    Args:
        ips_engine (IPSScoringEngine): The IPS engine instance.
        scenario_name (str): A descriptive name for the simulation scenario.
        ip (str): The IP address to simulate requests from.
        requests_data (list): A list of tuples, each containing:
                              (path, method, status_code, content_length, headers, delay_after_request)
    """
    print(f"\n--- Simulating Scenario: {scenario_name} for IP: {ip} ---")
    for i, (path, method, status, content_len, headers, delay) in enumerate(requests_data):
        current_time = time.time()
        record = RequestRecord(current_time, method, path, "", headers, content_len, status)
        ips_engine.add_request(ip, record)
        print(f"  [{i+1}] {ip} - {method} {path} (Status: {status})")
        time.sleep(delay) # Simulate time passing between requests

# Initialize the IPS engine with a shorter time window and ban threshold for demonstration
# In a real system, you'd use a longer window (e.g., 300-600 seconds) and tune the threshold.
ips_engine = IPSScoringEngine(time_window_seconds=60, ban_threshold=10, baseline_history_size=10)

# --- Scenario 1: Legitimate User ---
# Low request rate, diverse paths, no errors, consistent headers. Should not be banned.
legit_ip = "192.168.1.100"
legit_requests = [
    ("/home", "GET", 200, 100, {"User-Agent": "Mozilla/5.0"}, 0.5),
    ("/products/1", "GET", 200, 150, {"User-Agent": "Mozilla/5.0"}, 1.0),
    ("/cart", "GET", 200, 80, {"User-Agent": "Mozilla/5.0"}, 0.7),
    ("/products/2", "GET", 200, 120, {"User-Agent": "Mozilla/5.0"}, 0.5),
    ("/checkout", "POST", 200, 50, {"User-Agent": "Mozilla/5.0", "Content-Type": "application/json"}, 1.2),
    ("/profile", "GET", 200, 90, {"User-Agent": "Mozilla/5.0"}, 0.8),
    ("/products/3", "GET", 200, 110, {"User-Agent": "Mozilla/5.0"}, 0.6),
]
simulate_requests(ips_engine, "Legitimate User", legit_ip, legit_requests)
print(f"Final status for {legit_ip}: Banned? {ips_engine.is_banned(legit_ip)}")

# --- Scenario 2: Basic DDoS Attacker ---
# High request rate to a single endpoint. Should be banned quickly.
ddos_ip = "10.0.0.1"
ddos_requests = [
    ("/api/data", "GET", 200, 50, {"User-Agent": "AttackerBot/1.0"}, 0.1) for _ in range(30)
]
simulate_requests(ips_engine, "Basic DDoS Attacker", ddos_ip, ddos_requests)
print(f"Final status for {ddos_ip}: Banned? {ips_engine.is_banned(ddos_ip)}")


# --- Scenario 3: Endpoint Exploration Attacker ---
# High request rate, many unique paths (scanning behavior). Should be banned.
exploration_ip = "172.16.0.5"
exploration_requests = [
    (f"/explore/path_{i}", "GET", 200, 70, {"User-Agent": "Scanner/1.0"}, 0.2) for i in range(15)
] + [
    (f"/api/v1/resource_{i}", "GET", 404, 30, {"User-Agent": "Scanner/1.0"}, 0.2) for i in range(10)
]
simulate_requests(ips_engine, "Endpoint Exploration Attacker", exploration_ip, exploration_requests)
print(f"Final status for {exploration_ip}: Banned? {ips_engine.is_banned(exploration_ip)}")


# --- Scenario 4: High Error Attacker ---
# Repeated requests causing server errors or 404s. Should be banned.
error_ip = "192.168.1.200"
error_requests = [
    ("/api/bad_input", "POST", 500, 20, {"Content-Type": "application/json"}, 0.3) for _ in range(10)
] + [
    ("/nonexistent_page", "GET", 404, 30, {}, 0.2) for _ in range(5)
]
simulate_requests(ips_engine, "High Error Attacker", error_ip, error_requests)
print(f"Final status for {error_ip}: Banned? {ips_engine.is_banned(error_ip)}")

# --- Scenario 5: Brute Force Permissions Attacker ---
# Many failed login attempts. Should be banned.
brute_force_ip = "10.0.0.50"
brute_force_requests = [
    ("/login", "POST", 401, 10, {"Content-Type": "application/x-www-form-urlencoded"}, 0.1) for _ in range(20)
] + [
    ("/login", "POST", 200, 50, {"Content-Type": "application/x-www-form-urlencoded"}, 0.5) # One successful login after many fails
]
simulate_requests(ips_engine, "Brute Force Attacker", brute_force_ip, brute_force_requests)
print(f"Final status for {brute_force_ip}: Banned? {ips_engine.is_banned(brute_force_ip)}")

# --- Scenario 6: Header Spoofer Attacker ---
# Rapidly changing headers. Should be banned.
spoofer_ip = "172.16.0.10"
spoofer_requests = [
    ("/data", "GET", 200, 100, {"User-Agent": f"Agent{i}"}, 0.3) for i in range(10)
] + [
    ("/data", "GET", 200, 100, {"X-Custom-Header": f"Value{i}"}, 0.3) for i in range(5)
]
simulate_requests(ips_engine, "Header Spoofer Attacker", spoofer_ip, spoofer_requests)
print(f"Final status for {spoofer_ip}: Banned? {ips_engine.is_banned(spoofer_ip)}")

# --- Scenario 7: Session Hijacker Attacker (simple heuristic) ---
# Attempts with invalid/malformed tokens. Should be banned.
hijacker_ip = "192.168.1.150"
hijacker_requests = [
    ("/api/protected", "GET", 401, 50, {"Authorization": "Bearer invalid.token.123"}, 0.2),
    ("/api/protected", "GET", 401, 50, {"Authorization": "Bearer malformed_token_too_short"}, 0.2),
    ("/api/protected", "GET", 401, 50, {"Authorization": "Bearer short"}, 0.2),
    ("/admin/users", "GET", 403, 50, {"Authorization": "Bearer another.invalid.token.xyz"}, 0.2),
    ("/home", "GET", 200, 100, {"User-Agent": "Mozilla/5.0"}, 0.5) # Legitimate request after attempts
]
simulate_requests(ips_engine, "Session Hijacker Attacker", hijacker_ip, hijacker_requests)
print(f"Final status for {hijacker_ip}: Banned? {ips_engine.is_banned(hijacker_ip)}")

# --- Scenario 8: Slower/Faster Attacker (Bursty Traffic) ---
# Alternating high and low request rates, demonstrating bursts.
burst_ip = "10.0.0.70"
burst_requests = []
# Fast burst
for _ in range(15):
    burst_requests.append(("/data", "GET", 200, 80, {"User-Agent": "BurstBot"}, 0.1))
# Slow period (to allow time window to shift and rates to drop)
for _ in range(5):
    burst_requests.append(("/status", "GET", 200, 60, {"User-Agent": "BurstBot"}, 1.0))
# Another fast burst
for _ in range(15):
    burst_requests.append(("/data", "GET", 200, 80, {"User-Agent": "BurstBot"}, 0.1))

simulate_requests(ips_engine, "Slower/Faster Attacker (Bursty)", burst_ip, burst_requests)
print(f"Final status for {burst_ip}: Banned? {ips_engine.is_banned(burst_ip)}")
