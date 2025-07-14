# IPS
This project implements a lightweight Intrusion Prevention System (IPS) using YARP (Yet Another Reverse Proxy) to mediate traffic between multiple attackers and a victim service. The IPS inspects and filters malicious requests, effectively preventing known attack behaviors. The full environment is orchestrated using Docker Compose for easy setup and simulation.

# Key Technologies
- YARP (Yet Another Reverse Proxy): Used to route and inspect traffic.
- Docker Compose: Manages the deployment and networking of all containers.
- .NET: For developing the IPS middleware and victim application.
- GO: For developing the Attackers

# How It Works

Victim: A service that accepts HTTP requests.
Attackers: Simulated malicious clients attempting to exploit or scan the victim.
IPS:
- Intercepts all traffic directed to the victim.
- Uses custom YARP middleware to analyze requests.
- Blocks or allows traffic based on defined rules (request pattern analysis).
Docker Compose:
- Builds and runs the victim, IPS, and attacker containers.
- Provides isolated networking and simplified service discovery.
# Getting Started
```sh
docker-compose up --build
```

All services (IPS, victim, and attackers) will spin up in isolated containers and start communicating. The IPS will act as a gatekeeper between attackers and the victim