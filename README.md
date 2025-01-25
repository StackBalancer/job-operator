# HPCJob Kubernetes Operator

This project defines a Kubernetes Operator for managing HPC (High-Performance Computing) jobs. The operator creates and manages `HPCJob` custom resources (CRs), automatically handling job execution using a mock service that simulates job processing. The mock service listens on port 8080 and simulates job completion after a delay.

## Features
- **HPCJob CRD**: A custom resource definition (CRD) that defines the structure for HPC jobs.
- **Controller**: A Kubernetes controller that watches for `HPCJob` resources and triggers the job execution using the mock service.
- **Mock Job Service**: A simple HTTP service that simulates job completion after a delay.
- **Deployment Management**: Automatically creates and manages deployments, services, and jobs based on `HPCJob` resources.

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

### 2. **Build hpc-controller**

Build hpc-controller and load image into minikube:

```bash
docker build -t hpc-controller .
# Following commands can be replaced with (if your docker build is compatible with minikube): minikube image load staticpage-controller
docker image save -o hpc-controller.tar hpc-controller
minikube image load hpc-controller.tar
rm hpc-controller.tar 
```

### 3. **Build mock-hpc-job-service**

Build mock-hpc-job-service and load image into minikube:

```bash
docker build -t mock-hpc-job ./mock-hpc-job-service
docker image save -o mock-hpc-job.tar mock-hpc-job
minikube image load mock-hpc-job.tar
rm mock-hpc-job.tar 
```

### 4. **Deploy manifests to minikube**

```bash
kubectl apply -f k8s/crd.yaml # CRDs
kubectl apply -f k8s/controller-deployment.yaml # deploy the hpc-controller (and roles)
kubectl apply -f k8s/hpcjob.yaml # hpc job 
```

### 5. **Cleanup**
```bash
minikube delete
```

### Summary of Key Sections:
- **Building Docker Images**: Instructions for building both the controller and mock job service images.
- **Deploying to Minikube**: Steps to deploy the CRD, controller, and `HPCJob` resource.
- **Testing**: How to verify that your job is running properly.
- **Cleanup**: Commands for cleaning up the deployed resources and Minikube cluster.

This README should help set up and run the Kubernetes operator on Minikube, ensuring everything is deployed and works as expected. Let me know if you need any additional information!
