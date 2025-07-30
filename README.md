# Rinha de Backend 2025 - Go Submission

This repository contains my submission for the Rinha de Backend 2025, implemented in Go.

## Architecture Overview

- **API Layer**:  
  Two stateless API instances (`payment-api-1` and `payment-api-2`) receive incoming HTTP requests for `/payments`, `/payments-summary`, and `/purge-payments`.  
  These endpoints are load-balanced using HAProxy with the `roundrobin` strategy.

- **Worker**:  
  A dedicated worker service consumes messages from the queue, processes payments, and stores data **in memory** for maximum performance and minimum latency.  
  No external database is used to keep the solution as fast as possible.

- **Messaging**:  
  Communication between the APIs and the worker is handled by [NATS JetStream](https://docs.nats.io/nats-concepts/jetstream), which ensures reliable, high-performance message delivery.

- **Persistence (Optional)**:  
  There is a persistence worker component prepared (not currently active) that can be integrated if database persistence is desired.  
  This worker could asynchronously save processed payments to a database in the background after the main worker processes the request.

## Components

- `payment-api-1` / `payment-api-2`: Go HTTP APIs for requests
- `haproxy`: Load balancer using roundrobin to both API instances
- `payment-worker`: Go service that processes payment messages from the queue and keeps data in memory
- `nats`: NATS JetStream server for messaging
- `payment-persistence` (optional): Worker for persisting data to a database (currently not enabled)

## Running the Project

All components are provided as **public Docker images** and orchestrated via `docker-compose`.

The endpoints will be available at **localhost:9999**

```sh
docker-compose up --build
