package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	taskjobv1 "k8s-job-operator/stateless/api/v1"

	"go.uber.org/zap/zapcore"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Register CRD with the Scheme
var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(taskjobv1.AddToScheme(scheme))
}

// TaskJobReconciler reconciles TaskJob resources
type TaskJobReconciler struct {
	client.Client
	scheme     *runtime.Scheme
	kubeClient *kubernetes.Clientset
}

// Reconcile handles changes to TaskJob resources
func (r *TaskJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("NamespacedName", req.NamespacedName)
	log.Info("Reconciling TaskJob", "Namespace", req.Namespace, "Name", req.Name)

	taskJob := &taskjobv1.TaskJob{}
	// create deployment if not exists
	deploymentsClient := r.kubeClient.AppsV1().Deployments(req.Namespace)
	svClient := r.kubeClient.CoreV1().Services(req.Namespace)

	// Define the deployment name based on the TaskJob name
	jobName := taskJob.Spec.JobName
	if jobName == "" {
		jobName = req.Name
	}

	// Fetch the TaskJob custom resource
	err := r.Client.Get(ctx, req.NamespacedName, taskJob)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// handle deletion of TaskJob-related resource if taskJob not found
			log.Info("TaskJob resource not found. Ignoring since object must be deleted", "namespace", req.NamespacedName, "name", req.Name)
			err = deploymentsClient.Delete(ctx, jobName, metav1.DeleteOptions{})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("couldn't delete deployment: %s", err)
			}
			err = svClient.Delete(ctx, jobName, metav1.DeleteOptions{})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("couldn't delete service: %s", err)
			}
			return ctrl.Result{}, nil

		}
		log.Error(err, "Failed to fetch TaskJob resource", "namespace", req.NamespacedName, "name", req.Name)
		return ctrl.Result{}, err
	}
	log.Info("Fetched TaskJob", "spec", taskJob.Spec, "status", taskJob.Status)
	//log.Info("Fetched TaskJob resource", "state", taskJob.Status.State, "jobName", taskJob.Spec.JobName)

	// Check if Deployment exists
	deployment, err := deploymentsClient.Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Create Deployment
			deploymentObj := getDeploymentObject(taskJob)
			_, err := deploymentsClient.Create(ctx, deploymentObj, metav1.CreateOptions{})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("couldn't create deployment: %s", err)
			}
			// Create Service
			serviceObj := getServiceObject(taskJob)
			_, err = svClient.Create(ctx, serviceObj, metav1.CreateOptions{})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("couldn't create service: %s", err)
			}

			log.Info("Created Deployment and Service for TaskJob", "TaskJob", jobName)
			return ctrl.Result{}, nil
		} else {
			return ctrl.Result{}, fmt.Errorf("couldn't get object: %s", err)
		}
	}

	// Check if the current replica count differs from the desired count
	if int(*deployment.Spec.Replicas) != taskJob.Spec.Replicas {
		// Update the deployment replica count to match the TaskJob specification
		deployment.Spec.Replicas = int32Ptr(int32(taskJob.Spec.Replicas))

		// Apply the update to the cluster
		_, err := deploymentsClient.Update(ctx, deployment, metav1.UpdateOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't update deployment: %s", err)
		}

		// Log the update
		log.Info("Updated Deployment replicas for TaskJob", "TaskJob", jobName)
		return ctrl.Result{}, nil
	}

	// Update the Job Status
	log.Info("Updating TaskJob status", "currentState", taskJob.Status.State)
	if err := r.updateJobStatus(ctx, taskJob); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue to keep monitoring Pods
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func main() {
	var (
		config *rest.Config
		err    error
	)
	opts := zap.Options{
		Development: true,
		Level:       zapcore.DebugLevel,
	}

	kubeconfigFilePath := filepath.Join(homedir.HomeDir(), ".kube", "config")
	if _, err := os.Stat(kubeconfigFilePath); errors.Is(err, os.ErrNotExist) { // if kube config doesn't exist, try incluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigFilePath)
		if err != nil {
			panic(err.Error())
		}
	}

	// Kubernetes client set
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Set logger for the controller
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Create a new controller manager
	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Register TaskJob controller
	err = ctrl.NewControllerManagedBy(mgr).
		For(&taskjobv1.TaskJob{}).
		Complete(&TaskJobReconciler{
			Client:     mgr.GetClient(),
			scheme:     mgr.GetScheme(),
			kubeClient: clientset,
		})
	if err != nil {
		setupLog.Error(err, "unable to create controller")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "error running manager")
		os.Exit(1)
	}

}

func getDeploymentObject(taskJob *taskjobv1.TaskJob) *appsv1.Deployment {
	var pullPolicy corev1.PullPolicy

	if taskJob.Spec.ImagePullPolicy != "" {
		pullPolicy = corev1.PullPolicy(taskJob.Spec.ImagePullPolicy)
	} else {
		pullPolicy = corev1.PullIfNotPresent
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: taskJob.Spec.JobName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(int32(taskJob.Spec.Replicas)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": taskJob.Spec.JobName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": taskJob.Spec.JobName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            taskJob.Spec.JobName,
							Image:           taskJob.Spec.Image,
							ImagePullPolicy: pullPolicy,
							Ports:           []corev1.ContainerPort{{ContainerPort: 8080}},
							Env: []corev1.EnvVar{
								{Name: "JOB_NAME", Value: taskJob.Spec.JobName},
								{Name: "JOB_PARAMS", Value: fmt.Sprintf("%v", taskJob.Spec.JobParams)}, // Pass taskJob params as env vars
							},
						},
					},
				},
			},
		},
	}
}

func getServiceObject(taskJob *taskjobv1.TaskJob) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: taskJob.Spec.JobName,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": taskJob.Spec.JobName},
			Ports: []corev1.ServicePort{
				{
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}
}

func (r *TaskJobReconciler) updateJobStatus(ctx context.Context, taskJob *taskjobv1.TaskJob) error {
	log := log.FromContext(ctx)

	// List Pods for this TaskJob (selector must match Deployment labels)
	pods, err := r.kubeClient.CoreV1().Pods(taskJob.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", taskJob.Spec.JobName),
	})
	if err != nil {
		log.Error(err, "Failed to list pods for TaskJob", "jobName", taskJob.Spec.JobName)
		return err
	}

	// Default state
	state := "Pending"
	hasReady := false
	hasFailed := false

	for _, pod := range pods.Items {
		switch pod.Status.Phase {
		case corev1.PodRunning:
			allReady := true
			for _, cs := range pod.Status.ContainerStatuses {
				if !cs.Ready {
					allReady = false
				}
				if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
					hasFailed = true
				}
			}
			if allReady {
				hasReady = true
			}
		case corev1.PodFailed:
			hasFailed = true
		case corev1.PodSucceeded:
			state = "Completed"
		}
	}

	// Final state decision
	switch {
	case hasFailed:
		state = "Failed"
	case hasReady && state != "Completed":
		state = "Running"
	}

	// Only update if state changed
	if taskJob.Status.State != state {
		taskJob.Status.State = state
		if state == "Completed" {
			now := metav1.Now()
			taskJob.Status.CompletionTime = &now
		}

		if err := r.Status().Update(ctx, taskJob); err != nil {
			log.Error(err, "Failed to update TaskJob status")
			return err
		}
		log.Info("Updated TaskJob state", "newState", state)
	}

	return nil
}

func int32Ptr(i int32) *int32 {
	return &i
}
