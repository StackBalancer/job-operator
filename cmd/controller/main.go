package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	hpcv1 "hpc-operator/api/v1"

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
	utilruntime.Must(hpcv1.AddToScheme(scheme))
}

// HPCJobReconciler reconciles HPCJob resources
type HPCJobReconciler struct {
	client.Client
	scheme     *runtime.Scheme
	kubeClient *kubernetes.Clientset
}

// Reconcile handles changes to HPCJob resources
func (r *HPCJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("NamespacedName", req.NamespacedName)
	log.Info("Reconciling HPCJob", "Namespace", req.Namespace, "Name", req.Name)

	hpcJob := &hpcv1.HPCJob{}
	// create deployment if not exists
	deploymentsClient := r.kubeClient.AppsV1().Deployments(req.Namespace)
	svClient := r.kubeClient.CoreV1().Services(req.Namespace)

	// Define the deployment name based on the HPCJob name
	hpcJobName := hpcJob.Name + req.Name

	// Fetch the HPCJob custom resource
	err := r.Client.Get(ctx, req.NamespacedName, hpcJob)
	if err != nil {
		if k8serrors.IsNotFound(err) { // hpcjob not found, we can delete the resources
			// handle deletion of HPCJob-related resources (e.g., HPC job resources in the HPC cluster)
			log.Info("HPCJob resource not found. Ignoring since object must be deleted", "namespace", req.NamespacedName, "name", req.Name)
			err = deploymentsClient.Delete(ctx, hpcJobName, metav1.DeleteOptions{})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("couldn't delete deployment: %s", err)
			}
			err = svClient.Delete(ctx, hpcJobName, metav1.DeleteOptions{})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("couldn't delete service: %s", err)
			}
			return ctrl.Result{}, nil

		}
		log.Error(err, "Failed to fetch HPCJob resource", "namespace", req.NamespacedName, "name", req.Name)
		return ctrl.Result{}, err
	}
	log.Info("Fetched HPCJob resource", "state", hpcJob.Status.State, "jobName", hpcJob.Spec.JobName)

	// Check if Deployment exists
	deployment, err := deploymentsClient.Get(ctx, hpcJobName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Create Deployment
			deploymentObj := getDeploymentObject(hpcJob)
			_, err := deploymentsClient.Create(ctx, deploymentObj, metav1.CreateOptions{})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("couldn't create deployment: %s", err)
			}
			// Create Service
			serviceObj := getServiceObject(hpcJob)
			_, err = svClient.Create(ctx, serviceObj, metav1.CreateOptions{})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("couldn't create service: %s", err)
			}

			log.Info("Created Deployment and Service for HPCJob", "HPCJob", hpcJobName)
			return ctrl.Result{}, nil
		} else {
			return ctrl.Result{}, fmt.Errorf("couldn't get object: %s", err)
		}
	}

	// Check if the current replica count differs from the desired count
	if int(*deployment.Spec.Replicas) != hpcJob.Spec.Replicas {
		// Update the deployment replica count to match the HPCJob specification
		deployment.Spec.Replicas = int32Ptr(int32(hpcJob.Spec.Replicas))

		// Apply the update to the cluster
		_, err := deploymentsClient.Update(ctx, deployment, metav1.UpdateOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't update deployment: %s", err)
		}

		// Log the update
		log.Info("Updated Deployment replicas for HPCJob", "HPCJob", hpcJobName)
		return ctrl.Result{}, nil
	}

	// Update the Job Status
	log.Info("Updating HPCJob status", "currentState", hpcJob.Status.State)
	if err := r.updateJobStatus(ctx, hpcJob); err != nil {
		log.Error(err, "Failed to update HPCJob status", "namespace", req.Namespace, "name", req.Name)
		return ctrl.Result{}, fmt.Errorf("failed to update job status: %v", err)
	}

	// Simulate job completion and update the status
	if hpcJob.Status.State != "Completed" {
		log.Info("Simulating job completion for HPCJob", "HPCJob", hpcJobName)
		time.Sleep(5 * time.Second) // Simulate job running for 5 seconds
		hpcJob.Status.State = "Completed"
		hpcJob.Status.CompletionTime = &metav1.Time{}
		if err := r.Status().Update(ctx, hpcJob); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update HPCJob status: %v", err)
		}
		log.Info("HPCJob completed successfully", "HPCJob", hpcJobName)
	}

	log.Info("HPCJob deployment is up-to-date", "name", hpcJobName)
	return ctrl.Result{}, nil
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

	// Register HPCJob controller
	err = ctrl.NewControllerManagedBy(mgr).
		For(&hpcv1.HPCJob{}).
		Complete(&HPCJobReconciler{
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

func getDeploymentObject(hpcJob *hpcv1.HPCJob) *appsv1.Deployment {
	var pullPolicy corev1.PullPolicy

	if hpcJob.Spec.ImagePullPolicy != "" {
		pullPolicy = corev1.PullPolicy(hpcJob.Spec.ImagePullPolicy)
	} else {
		pullPolicy = corev1.PullIfNotPresent
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: hpcJob.Name,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(int32(hpcJob.Spec.Replicas)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": hpcJob.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": hpcJob.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            hpcJob.Name,
							Image:           hpcJob.Spec.Image,
							ImagePullPolicy: pullPolicy,
							Ports:           []corev1.ContainerPort{{ContainerPort: 8080}},
							Env: []corev1.EnvVar{
								{Name: "JOB_NAME", Value: hpcJob.Spec.JobName},
								{Name: "JOB_PARAMS", Value: fmt.Sprintf("%v", hpcJob.Spec.JobParams)}, // Pass job params as env vars
							},
						},
					},
				},
			},
		},
	}
}

func getServiceObject(hpcJob *hpcv1.HPCJob) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: hpcJob.Name,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": hpcJob.Name},
			Ports: []corev1.ServicePort{
				{
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}
}

// Update the status of the job (i.e., whether it is still running or completed)
func (r *HPCJobReconciler) updateJobStatus(ctx context.Context, hpcJob *hpcv1.HPCJob) error {
	log := log.FromContext(ctx)

	log.Info("Updating job status", "currentState", hpcJob.Status.State, "jobName", hpcJob.Spec.JobName)

	log.Info("Attempting to update HPCJob status",
		"name", hpcJob.Name,
		"namespace", hpcJob.Namespace,
		"state", hpcJob.Status.State,
		"resourceVersion", hpcJob.ObjectMeta.ResourceVersion,
	)

	if hpcJob.Status.State == "" {
		log.Info("Initial state is empty, setting state to 'Pending'")
		hpcJob.Status.State = "Pending"
	}
	if err := r.Status().Update(ctx, hpcJob); err != nil {
		log.Error(err, "Failed to update HPCJob status",
			"name", hpcJob.Name,
			"namespace", hpcJob.Namespace,
			"state", hpcJob.Status.State,
			"resourceVersion", hpcJob.ObjectMeta.ResourceVersion,
		)
		return fmt.Errorf("failed to update HPCJob status: %v", err)
	}
	log.Info("Successfully updated HPCJob status",
		"name", hpcJob.Name,
		"namespace", hpcJob.Namespace,
		"newState", hpcJob.Status.State,
	)
	return nil
}

func int32Ptr(i int32) *int32 {
	return &i
}
