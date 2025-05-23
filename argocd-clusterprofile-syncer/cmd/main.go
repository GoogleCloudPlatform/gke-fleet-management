package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterinventoryv1alpha1 "sigs.k8s.io/cluster-inventory-api/apis/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// Argo CD constants.
	argoCDNamespace = "argocd"
	// https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#clusters
	argoCDSecretType = "argocd.argoproj.io/secret-type"

	// Annotations.
	managedByAnnotation   = "multicluster.x-k8s.io/managed-by-cp-syncer"
	clusterProfileOrigin  = "clusterprofile.x-k8s.io/origin"
	gkeEndpointAnnotation = "gateway.gke.io/endpoint"

	// Reconciliation constants.
	maxConcurrentReconciles = 3
	crdName                 = "clusterprofiles.multicluster.x-k8s.io"
	secretConfig            = `{
	"execProviderConfig": {
		"command": "argocd-k8s-auth",
		"args": ["gcp"],
		"apiVersion": "client.authentication.k8s.io/v1beta1"
	},
	"tlsClientConfig": {
		"insecure": false,
		"caData": ""
	}
}`
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(clusterinventoryv1alpha1.AddToScheme(scheme))
}

// ClusterProfileReconciler reconciles ClusterProfile objects.
type ClusterProfileReconciler struct {
	client.Client
	scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=multicluster.x-k8s.io,resources=clusterprofiles,verbs=get;list;watch
// +kubebuilder:rbac:groups=multicluster.x-k8s.io,resources=clusterprofiles/status,verbs=get

// Reconcile handles the reconciliation loop for ClusterProfile resources.
func (r *ClusterProfileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("clusterprofile", req.NamespacedName)
	logger.Info("Starting reconciliation")

	clusterProfile := &clusterinventoryv1alpha1.ClusterProfile{}
	if err := r.Get(ctx, req.NamespacedName, clusterProfile); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("ClusterProfile not found, cleaning up associated secret")
			return ctrl.Result{}, r.deleteClusterSecret(ctx, req)
		}
		logger.Error(err, "Failed to get ClusterProfile")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.createOrUpdateClusterSecret(ctx, clusterProfile); err != nil {
		logger.Error(err, "Failed to reconcile secret")
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	logger.Info("Reconciliation completed successfully")
	return ctrl.Result{}, nil
}

// deleteClusterSecret removes the associated secret if it exists and is managed by this controller.
func (r *ClusterProfileReconciler) deleteClusterSecret(ctx context.Context, req ctrl.Request) error {
	logger := log.FromContext(ctx)
	secretName := types.NamespacedName{
		Namespace: argoCDNamespace,
		Name:      fmt.Sprintf("%s.%s", req.Namespace, req.Name),
	}

	secret := &corev1.Secret{}
	if err := r.Get(ctx, secretName, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get secret: %w", err)
	}

	if !isSecretManaged(secret, req.String()) {
		return nil
	}

	logger.Info("Deleting managed secret", "secret", secretName)
	if err := r.Delete(ctx, secret); err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	return nil
}

func isSecretManaged(secret *corev1.Secret, cpOrigin string) bool {
	if secret.Annotations == nil {
		return false
	}
	if secret.Annotations[managedByAnnotation] != "true" {
		return false
	}
	return secret.Annotations[clusterProfileOrigin] == cpOrigin
}

func (r *ClusterProfileReconciler) createOrUpdateClusterSecret(ctx context.Context, cp *clusterinventoryv1alpha1.ClusterProfile) error {
	logger := log.FromContext(ctx)

	serverURL, ok := cp.Annotations[gkeEndpointAnnotation]
	if !ok {
		return fmt.Errorf("cluster endpoint annotation %q not found", gkeEndpointAnnotation)
	}

	secretName := fmt.Sprintf("%s.%s", cp.Namespace, cp.Name)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: argoCDNamespace,
		},
	}

	logger.Info("Reconciling secret", "name", secretName)
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		return r.mutateSecret(secret, cp, serverURL, secretName)
	}); err != nil {
		return fmt.Errorf("failed to create/update secret: %w", err)
	}

	return nil
}

func (r *ClusterProfileReconciler) mutateSecret(secret *corev1.Secret, cp *clusterinventoryv1alpha1.ClusterProfile, serverURL, secretName string) error {
	if secret.Labels == nil {
		secret.Labels = make(map[string]string)
	}
	secret.Labels[argoCDSecretType] = "cluster"

	if secret.Annotations == nil {
		secret.Annotations = make(map[string]string)
	}
	secret.Annotations[managedByAnnotation] = "true"
	secret.Annotations[clusterProfileOrigin] = fmt.Sprintf("%s/%s", cp.Namespace, cp.Name)

	secret.Type = corev1.SecretTypeOpaque
	secret.Data = map[string][]byte{
		"name":   []byte(secretName),
		"server": []byte(serverURL),
		"config": []byte(secretConfig),
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterProfileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterinventoryv1alpha1.ClusterProfile{}).
		Owns(&corev1.Secret{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: maxConcurrentReconciles,
		}).
		Complete(r)
}

func isCRDInstalled(ctx context.Context, cfg *rest.Config, crdName string) error {
	client, err := apiextensionsclientset.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create apiextensions client: %w", err)
	}

	crd, err := client.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crdName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting CRD: %w", err)
	}

	for _, condition := range crd.Status.Conditions {
		if condition.Type == apiextensionsv1.Established &&
			condition.Status == apiextensionsv1.ConditionTrue {
			return nil
		}
	}

	return fmt.Errorf("crd %q is installed but not established", crdName)
}

func waitForClusterProfileCRD(ctx context.Context, cfg *rest.Config) {
	for {
		err := isCRDInstalled(ctx, cfg, crdName)
		if err == nil {
			return
		}
		setupLog.V(1).Info("ClusterProfile CRD not yet available, waiting...", "error", err)
		time.Sleep(time.Second * 10)
	}
}

func main() {
	opts := zap.Options{
		Development: true,
	}
	ctx := context.Background()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	cfg := ctrl.GetConfigOrDie()
	waitForClusterProfileCRD(ctx, cfg)

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Cache: cache.Options{
			ByObject: map[client.Object]cache.ByObject{
				&corev1.Secret{}: {
					Namespaces: map[string]cache.Config{
						argoCDNamespace: {},
					},
				},
			},
		},
	})
	if err != nil {
		setupLog.Error(err, "could not create manager")
		os.Exit(1)
	}

	if err := (&ClusterProfileReconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "could not create controller", "controller", "ClusterProfile")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "could not start manager")
		os.Exit(1)
	}
}
