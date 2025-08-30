# K8s Job Operator

This project provides a Kubernetes Operator for managing task jobs and stateful databases. It includes two separate controllers:

- Stateless Task Job Controller – Manages TaskJob CRs, creating ephemeral job deployments.

- Stateful Database Controller – Manages Database CRs, creating PostgreSQL statefulsets with PVCs.

## Features

- **TaskJob CRD**: Define and execute task jobs using a stateless controller.
- **Database CRD**: Define and manage databases with persistent storage.
- **Automatic Resource Management**: Controllers handle Deployments, StatefulSets, Services, and PVCs automatically.
- **Job Simulation**: Task job service simulates work and completes jobs after a configurable delay.
- **Status Tracking**: CRs include .status subresource to monitor readiness and completion.
---

## Prerequisites

Before proceeding, make sure you have the following installed:

- [Docker](https://www.docker.com/get-started) (for building images)
- [Minikube](https://minikube.sigs.k8s.io/docs/) (for local Kubernetes clusters)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) (for interacting with Kubernetes)
- [Go](https://go.dev/dl/) (for building the operator)

---

## Folder Structure

```
k8s-job-operator/
├── k8s
│   ├── controller-db-deployment.yaml
│   ├── controller-deployment.yaml
│   ├── crd-database.yaml
│   ├── crd.yaml
│   ├── database-controller-rbac.yaml
│   ├── postgres-database.yaml
│   └── task-job.yaml
├── LICENSE
├── README.md
├── stateful
│   ├── api
│   ├── cmd
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   └── postgres-svc
├── stateless
│   ├── api
│   ├── cmd
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   └── task-job-service
```

## Steps to Build and Deploy the Project

### 1. **Start Minikube Cluster**

Start a Minikube cluster by running the following command:

```bash
minikube start
# Set image registry
eval $(minikube docker-env)
```

### 2. **Build Controllers**

#### Stateless TaskJob Controller

```bash
cd stateless
docker build -t task-job-controller .
```

#### Stateful Database Controller

```bash
cd stateful
docker build -t database-controller .
```

### 3. **Build services**

Build task-job-service (stateless):

```bash
cd stateless
docker build -t task-job ./task-job-service
```

Build postgres (stateful) service:

```bash
cd stateful
docker -t postgres:15-alpine ./postgres-svc
```

### 4. **Deploy CRDs and Controllers**

#### Stateless TaskJob Controller

```bash
kubectl apply -f k8s/crd.yaml
kubectl apply -f k8s/controller-deployment.yaml
```

#### Stateful Database Controller

```bash
kubectl apply -f k8s/crd-database.yaml
kubectl apply -f k8s/controller-db-deployment.yaml
```

### 5. **Create CRs**

#### TaskJob (Stateless)

```bash
kubectl apply -f k8s/task-job.yaml
```

#### Database (Stateful)

```bash
kubectl apply -f k8s/postgres-database.yaml
```

### 6. **Verify Resources**
```bash
kubectl get pods
kubectl get statefulsets
kubectl get pvc
kubectl get svc
kubectl describe taskjob <name>
kubectl describe database <name>
```

### 7. **Cleanup**
```bash
minikube delete
```

### Notes
- Both controllers run independently but can coexist in the same cluster.

- The database controller automatically creates headless services and PVCs for persistent storage.

- Status subresources are enabled for both TaskJob and Database CRs, allowing the controllers to update .status.phase and readiness information.
