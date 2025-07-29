# TODO

## High (MVP Focus)
- [ ] Single-process FaaS runtime with simple in-memory scheduling
    - Keep it dead simple: one function = one Docker container instance
    - Design clean interfaces for potential future pluggability, but don't over-engineer
- [ ] Docker-based compute provider as the primary (and initially only) runtime
    - Use Docker API for all container lifecycle management
    - Runtime flexibility via Docker daemon configuration (runc, kata, nvidia, etc.)
    - IPAM, networking, image management all handled by Docker/containerd
- [ ] Traefik integration for proxy/routing
    - Use Traefik API to dynamically register/deregister function endpoints
    - Let Traefik handle load balancing, SSL termination, etc.
- [ ] Flesh out what the runtime building interface should look like
    - Build from Dockerfile or import existing OCI images
    - What metadata do we need? (entrypoint, env vars, resource limits, runtime selection)
    - Consider: How do users specify they want Kata vs nvidia vs standard runtime?
- [ ] We need a solid test suite!
    - Unit tests for core logic, integration tests with Docker containers
    - Test with different Docker runtimes (runc, kata if available)
- [ ] Simple CLI tool that interfaces with the server to upload/invoke/manage functions
    - Think about authentication/authorization early
    - Consider gRPC vs REST API design
- [ ] Makefile for build/test/dev environment setup
    - Recipe for setting up demo server with Docker + Traefik
    - Maybe include a `make dev` target that sets up a full local environment
    - Document how to configure different Docker runtimes

## Medium
- [ ] How do we handler monitoring function runtime metrics?
    - Embedded Prometheus is a thing I've seen commonly but not sure how I feel about this
    - Alternative: OpenTelemetry metrics export to external collectors
    - Consider what metrics matter: execution time, memory usage, startup latency, error rates
- [ ] Reduce usage of `interface{}`/`any` to make code more understandable
    - Good for maintainability and catching errors at compile time
    - Consider using generics where appropriate (Go 1.18+)
- [ ] Chroot/sandboxing/drop privileges by default for main binary
    - Is this the correct way? Or do we expect the user to sandbox the main process if they so wish?
    - Defense in depth: even if VMs/containers provide isolation, host process should be minimal
    - Consider running as non-root user by default
- [ ] Function runtime configuration and metadata
    - How do users specify runtime requirements? (GPU, extra isolation, etc.)
    - Should this be in function metadata, CLI flags, or config file?
    - Consider: `docker run --runtime=kata` vs `docker run --runtime=nvidia`

## Future/Scale-Up Considerations
- [ ] Pluggable orchestrator interface for external schedulers
    - Design: `type Scheduler interface { Schedule(fn Function) error }`
    - Keep door open for Nomad integration without building it now
- [ ] Direct Firecracker provider (bypassing Docker)
    - Only if we need more control than Docker runtimes provide
    - Would need to reimplement IPAM, image handling, etc.
    - Consider: Is the complexity worth it vs just using kata-containers?
- [ ] WASM runtime support alongside containers
    - Ultra-fast cold starts, but limited ecosystem
    - Could be good for specific use cases (CPU-bound, no system calls)
- [ ] Additional compute providers
    - Podman support (very similar to Docker API)
    - Nomad job execution
    - Cloud provider serverless (Lambda, Cloud Functions, etc.)

## Future Ideas (Don't Build Yet)
- [ ] Multi-node clustering and state replication
    - Start with single-node SQLite, worry about this later
    - Consider rqlite/litestream when we actually need it 
- [ ] Function versioning and blue/green deployments
    - Important for production workloads, but not MVP
    - Could reuse container image tags for this
- [ ] Advanced scheduling and bin packing
    - How do we efficiently pack functions onto available compute?
    - Consider CPU/memory/network requirements
    - **Note: This is exactly the kind of thing Nomad solves - don't reinvent**
- [ ] Cold start optimization strategies
    - Pre-warmed pools? Shared base images? Snapshot/restore?
    - Profile first, optimize second
