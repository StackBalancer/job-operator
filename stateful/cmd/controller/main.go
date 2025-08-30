package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	dbv1 "k8s-job-operator/stateful/api/v1"

	"go.uber.org/zap/zapcore"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
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
	crlog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(dbv1.AddToScheme(scheme))
}

type DatabaseReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	kubeClient *kubernetes.Clientset
}

func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := crlog.FromContext(ctx).WithValues("NamespacedName", req.NamespacedName)
	log.Info("Reconciling Database", "name", req.Name, "namespace", req.Namespace)

	db := &dbv1.Database{}
	if err := r.Get(ctx, req.NamespacedName, db); err != nil {
		if k8serrors.IsNotFound(err) {
			// resource deleted -> ensure StatefulSet/service removed
			log.Info("Database CR deleted; nothing more to do")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Determine a stable name for resources
	name := db.Spec.DatabaseName
	if name == "" {
		name = db.Name // fallback to CR name
	}

	// clients for core operations
	stsClient := r.kubeClient.AppsV1().StatefulSets(req.Namespace)
	svcClient := r.kubeClient.CoreV1().Services(req.Namespace)
	//pvcClient := r.kubeClient.CoreV1().PersistentVolumeClaims(req.Namespace)

	// ensure headless service exists (for stable DNS)
	_, err := svcClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			svc := makeHeadlessService(name)
			if _, err := svcClient.Create(ctx, svc, metav1.CreateOptions{}); err != nil {
				return ctrl.Result{}, fmt.Errorf("create service: %w", err)
			}
			log.Info("Created headless service", "service", name)
		} else {
			return ctrl.Result{}, err
		}
	}

	// ensure statefulset exists
	sts, err := stsClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			stsObj := makeStatefulSet(db, name)
			if _, err := stsClient.Create(ctx, stsObj, metav1.CreateOptions{}); err != nil {
				return ctrl.Result{}, fmt.Errorf("create statefulset: %w", err)
			}
			log.Info("Created StatefulSet", "name", name)
			// Requeue so status can be observed later
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	// Sync replicas if changed
	desired := int32(db.Spec.Replicas)
	if sts.Spec.Replicas == nil || *sts.Spec.Replicas != desired {
		sts.Spec.Replicas = &desired
		if _, err := stsClient.Update(ctx, sts, metav1.UpdateOptions{}); err != nil {
			return ctrl.Result{}, fmt.Errorf("update statefulset replicas: %w", err)
		}
		log.Info("Updated StatefulSet replicas", "name", name, "replicas", desired)
		// requeue to observe readiness
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// Update status based on statefulset readiness
	var phase string
	ready := sts.Status.ReadyReplicas
	if ready == 0 {
		phase = "Pending"
	} else if ready < *sts.Spec.Replicas {
		phase = "Running"
	} else {
		phase = "Ready"
	}

	if db.Status.Phase != phase || db.Status.ReadyReplicas != ready {
		db.Status.Phase = phase
		db.Status.ReadyReplicas = ready
		if err := r.Status().Update(ctx, db); err != nil {
			return ctrl.Result{}, fmt.Errorf("update status: %w", err)
		}
		log.Info("Updated Database status", "phase", phase, "readyReplicas", ready)
	}

	// Requeue periodically to watch readiness
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func main() {
	var (
		config *rest.Config
		err    error
	)
	opts := zap.Options{Development: true, Level: zapcore.DebugLevel}

	kubeconfigFilePath := filepath.Join(homedir.HomeDir(), ".kube", "config")
	if _, err := os.Stat(kubeconfigFilePath); errors.Is(err, os.ErrNotExist) {
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

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(config, ctrl.Options{Scheme: scheme})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&dbv1.Database{}).
		Complete(&DatabaseReconciler{
			Client:     mgr.GetClient(),
			Scheme:     mgr.GetScheme(),
			kubeClient: clientset,
		}); err != nil {
		setupLog.Error(err, "unable to create controller")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "error running manager")
		os.Exit(1)
	}
}

// helpers

func makeHeadlessService(name string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: corev1.ServiceSpec{
			ClusterIP: corev1.ClusterIPNone, // headless
			Ports: []corev1.ServicePort{
				{Port: 5432, TargetPort: intstrFromInt(5432)},
			},
			Selector: map[string]string{"app": name},
		},
	}
}

func makeStatefulSet(db *dbv1.Database, name string) *appsv1.StatefulSet {
	replicas := int32(db.Spec.Replicas)
	storage := db.Spec.Storage
	if storage == "" {
		storage = "1Gi"
	}

	pvc := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "data",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(storage),
				},
			},
		},
	}

	image := db.Spec.Image
	if image == "" {
		image = "postgres:15-alpine"
	}

	pullPolicy := corev1.PullIfNotPresent
	if db.Spec.ImagePullPolicy != "" {
		pullPolicy = corev1.PullPolicy(db.Spec.ImagePullPolicy)
	}

	labels := map[string]string{"app": name}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			Selector:    &metav1.LabelSelector{MatchLabels: labels},
			ServiceName: name, // headless service
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "postgres",
							Image:           image,
							ImagePullPolicy: pullPolicy,
							Ports:           []corev1.ContainerPort{{ContainerPort: 5432}},
							Env: []corev1.EnvVar{
								{Name: "POSTGRES_PASSWORD", Value: db.Spec.Password},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "data", MountPath: "/var/lib/postgresql/data"},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{Port: intstrFromInt(5432)},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{pvc},
		},
	}
}

// small helper for intstr
func intstrFromInt(i int) intstr.IntOrString {
	return intstr.FromInt(i)
}

func int32Ptr(i int32) *int32 { return &i }
