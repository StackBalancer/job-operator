# K8s Task Job Operator

This project defines a Kubernetes Operator for managing task jobs. The operator creates and manages `TaskJob` custom resources (CRs), automatically handling job execution using a task-job service that simulates job processing. The task-job service listens on port 8080 and simulates job completion after a delay.

## Features
- **TaskJob CRD**: A custom resource definition (CRD) that defines the structure for task jobs.
- **Controller**: A Kubernetes controller that watches for `TaskJob` resources and triggers the job execution using the task-job service.
- **Task Job Service**: A simple HTTP service that simulates job completion after a delay.
- **Deployment Management**: Automatically creates and manages deployments, services, and jobs based on `TaskJob` resources.

---

## Prerequisites

Before proceeding, make sure you have the following installed:

- [Docker](https://www.docker.com/get-started) (for building images)
- [Minikube](https://minikube.sigs.k8s.io/docs/) (for local Kubernetes clusters)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) (for interacting with Kubernetes)
- [Go](https://go.dev/dl/) (for building the operator)

---

## Steps to Build and Deploy the Project

### 1. **Start Minikube Cluster**

Start a Minikube cluster by running the following command:

```bash
minikube start
```

### 2. **Build job-controller**

Build task-job-controller and load image into minikube:

```bash
docker build -t task-job-controller .
docker image save -o task-job-controller.tar task-job-controller
minikube image load task-job-controller.tar
rm task-job-controller.tar 
```

### 3. **Build task-job-service**

Build task-job-service and load image into minikube:

```bash
docker build -t task-job ./task-job-service
docker image save -o task-job.tar task-job
minikube image load task-job.tar
rm task-job.tar 
```

### 4. **Deploy manifests to minikube**

```bash
kubectl apply -f k8s/crd.yaml # CRDs
kubectl apply -f k8s/controller-deployment.yaml # deploy the task-job-controller (and roles)
kubectl apply -f k8s/task-job.yaml # job definition 
```

### 5. **Cleanup**
```bash
minikube delete
```

### Summary of Key Sections:
- **Building Docker Images**: Instructions for building both the controller and task job service images.
- **Deploying to Minikube**: Steps to deploy the CRD, controller, and `TaskJob` resource.
- **Testing**: How to verify that your job is running properly.
- **Cleanup**: Commands for cleaning up the deployed resources and Minikube cluster.
