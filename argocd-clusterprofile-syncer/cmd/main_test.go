package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterinventoryv1alpha1 "sigs.k8s.io/cluster-inventory-api/apis/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	log.SetLogger(zap.New(zap.UseDevMode(true)))
}

func TestReconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(clusterinventoryv1alpha1.AddToScheme(scheme))

	testCases := []struct {
		name       string
		objects    []client.Object
		wantErr    bool
		wantErrMsg string
		wantResult ctrl.Result
		wantSecret *corev1.Secret
	}{
		{
			name: "delete_managed_secret_without_cluster_profile",
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-namespace.test-name",
						Namespace: argoCDNamespace,
						Annotations: map[string]string{
							managedByAnnotation:  "true",
							clusterProfileOrigin: "test-namespace/test-name",
						},
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"name":   []byte("test-namespace.test-name"),
						"server": []byte("https://test-server"),
					},
				},
			},
			wantResult: ctrl.Result{},
		},
		{
			name: "unmanaged_secret_should_not_be_deleted",
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-namespace.test-name",
						Namespace: argoCDNamespace,
						Annotations: map[string]string{
							clusterProfileOrigin: "test-namespace/test-name",
						},
					},
					Type: corev1.SecretTypeOpaque,
				},
			},
			wantResult: ctrl.Result{},
			wantSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-namespace.test-name",
					Namespace: argoCDNamespace,
					Annotations: map[string]string{
						clusterProfileOrigin: "test-namespace/test-name",
					},
				},
				Type: corev1.SecretTypeOpaque,
			},
		},
		{
			name: "missing_endpoint_annotation",
			objects: []client.Object{
				&clusterinventoryv1alpha1.ClusterProfile{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-name",
						Namespace: "test-namespace",
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "cluster endpoint annotation",
			wantResult: ctrl.Result{RequeueAfter: time.Minute},
		},
		{
			name: "should_create_secret",
			objects: []client.Object{
				&clusterinventoryv1alpha1.ClusterProfile{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-name",
						Namespace: "test-namespace",
						Annotations: map[string]string{
							gkeEndpointAnnotation: "https://test-server",
						},
					},
				},
			},
			wantResult: ctrl.Result{},
			wantSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-namespace.test-name",
					Namespace: argoCDNamespace,
					Labels: map[string]string{
						argoCDSecretType: "cluster",
					},
					Annotations: map[string]string{
						managedByAnnotation:  "true",
						clusterProfileOrigin: "test-namespace/test-name",
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"name":   []byte("test-namespace.test-name"),
					"server": []byte("https://test-server"),
					"config": []byte(secretConfig),
				},
			},
		},
		{
			name: "should_update_secret",
			objects: []client.Object{
				&clusterinventoryv1alpha1.ClusterProfile{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-name",
						Namespace: "test-namespace",
						Annotations: map[string]string{
							gkeEndpointAnnotation: "https://test-server",
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-namespace.test-name",
						Namespace: argoCDNamespace,
						Annotations: map[string]string{
							managedByAnnotation:  "true",
							clusterProfileOrigin: "test-namespace/test-name",
						},
						Labels: map[string]string{
							argoCDSecretType: "cluster",
						},
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"name":   []byte("test-namespace.test-name"),
						"server": []byte("https://old-server"),
						"config": []byte(`{"old": "config"}`),
					},
				},
			},
			wantResult: ctrl.Result{},
			wantSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-namespace.test-name",
					Namespace: argoCDNamespace,
					Labels: map[string]string{
						argoCDSecretType: "cluster",
					},
					Annotations: map[string]string{
						managedByAnnotation:  "true",
						clusterProfileOrigin: "test-namespace/test-name",
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"name":   []byte("test-namespace.test-name"),
					"server": []byte("https://test-server"),
					"config": []byte(secretConfig),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tc.objects...).
				Build()
			r := &ClusterProfileReconciler{
				Client: client,
				scheme: scheme,
			}
			ctx := context.Background()
			request := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "test-namespace",
					Name:      "test-name",
				},
			}
			result, err := r.Reconcile(ctx, request)

			if tc.wantErr {
				if err == nil {
					t.Errorf("Reconcile() expected error: %v", err)
				} else if !strings.Contains(err.Error(), tc.wantErrMsg) {
					t.Errorf("Reconcile() expected error with messagee %q, got %q", err.Error(), tc.wantErrMsg)
				}
				return
			} else if err != nil {
				t.Fatalf("Reconcile() failed: %v", err)
			}

			if diff := cmp.Diff(tc.wantResult, result); diff != "" {
				t.Errorf("Reconcile() returned unexpected result (-want +got):\n%s", diff)
			}

			var gotSecret corev1.Secret
			err = client.Get(ctx, types.NamespacedName{
				Namespace: argoCDNamespace,
				Name:      fmt.Sprintf("%s.%s", request.Namespace, request.Name),
			}, &gotSecret)
			if err != nil {
				if !apierrors.IsNotFound(err) {
					t.Fatalf("Reconcile() failed to get secret: %v", err)
				}
				if tc.wantSecret != nil {
					t.Fatalf("Reconcile() expected secret %v but not found", tc.wantSecret)
				}
				return
			}

			if diff := cmp.Diff(tc.wantSecret, &gotSecret, cmpopts.EquateEmpty(), cmpopts.IgnoreFields(metav1.ObjectMeta{}, "ResourceVersion")); diff != "" {
				t.Errorf("Reconcile() unexpected secret (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsSecretManaged(t *testing.T) {
	testCases := []struct {
		name     string
		secret   *corev1.Secret
		cpOrigin string
		want     bool
	}{
		{
			name: "managed_secret",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						managedByAnnotation:  "true",
						clusterProfileOrigin: "test-namespace/test-name",
					},
				},
			},
			cpOrigin: "test-namespace/test-name",
			want:     true,
		},
		{
			name: "missing_managed_annotation",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						clusterProfileOrigin: "test-namespace/test-name",
					},
				},
			},
			cpOrigin: "test-namespace/test-name",
		},
		{
			name: "managed_annotation_false",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						managedByAnnotation:  "false",
						clusterProfileOrigin: "test-namespace/test-name",
					},
				},
			},
			cpOrigin: "test-namespace/test-name",
		},
		{
			name: "different_cluster_profile_origin",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						managedByAnnotation:  "true",
						clusterProfileOrigin: "different-namespace/different-name",
					},
				},
			},
			cpOrigin: "test-namespace/test-name",
		},
		{
			name: "no_annotations",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{},
			},
			cpOrigin: "test-namespace/test-name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isSecretManaged(tc.secret, tc.cpOrigin)
			if got != tc.want {
				t.Errorf("isSecretManaged() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestDeleteClusterSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(clusterinventoryv1alpha1.AddToScheme(scheme))

	testCases := []struct {
		name              string
		client            client.Client
		wantErr           bool
		wantErrMsg        string
		wantSecretDeleted bool
	}{
		{
			name: "missing_secret_ok",
			client: func() client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					Build()
			}(),
			wantSecretDeleted: true,
		},
		{
			name: "failed_to_get_secret",
			client: func() client.Client {
				return &errorClient{
					error: fmt.Errorf("test error"),
				}
			}(),
			wantErr:    true,
			wantErrMsg: "failed to get secret",
		},
		{
			name: "unmanaged_secret_is_not_deleted",
			client: func() client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-namespace.test-name",
							Namespace: argoCDNamespace,
						},
					}).
					Build()
			}(),
		},
		{
			name: "managed_secret_with_matching_origin_is_deleted",
			client: func() client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-namespace.test-name",
							Namespace: argoCDNamespace,
							Annotations: map[string]string{
								managedByAnnotation:  "true",
								clusterProfileOrigin: "test-namespace/test-name",
							},
						},
					}).
					Build()
			}(),
			wantSecretDeleted: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &ClusterProfileReconciler{
				Client: tc.client,
				scheme: scheme,
			}

			ctx := context.Background()
			request := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "test-namespace",
					Name:      "test-name",
				},
			}
			err := r.deleteClusterSecret(ctx, request)

			if tc.wantErr {
				if err == nil {
					t.Errorf("deleteClusterSecret() returned nil, want %q", tc.wantErrMsg)
				} else if !strings.Contains(err.Error(), tc.wantErrMsg) {
					t.Errorf("deleteClusterSecret() returned error %q, want %q", err.Error(), tc.wantErrMsg)
				}
				return
			} else if err != nil {
				t.Errorf("deleteClusterSecret() unexpected error: %v", err)
			}

			secretName := types.NamespacedName{
				Namespace: argoCDNamespace,
				Name:      fmt.Sprintf("%s.%s", request.Namespace, request.Name),
			}
			err = tc.client.Get(ctx, secretName, &corev1.Secret{})
			if err != nil {
				if !apierrors.IsNotFound(err) {
					t.Fatalf("deleteClusterSecret() failed to get secret: %v", err)
				} else if !tc.wantSecretDeleted {
					t.Errorf("deleteClusterSecret() expected secret to not be deleted but it is")
				}
				return
			}
			if tc.wantSecretDeleted {
				t.Errorf("deleteClusterSecret() expected secret to be deleted but it's not")
			}
		})
	}
}

func TestCreateOrUpdateClusterSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(clusterinventoryv1alpha1.AddToScheme(scheme))

	testCases := []struct {
		name           string
		clusterProfile *clusterinventoryv1alpha1.ClusterProfile
		client         client.Client
		wantErr        bool
		wantErrMsg     string
		wantSecret     *corev1.Secret
	}{
		{
			name: "missing_endpoint_annotation",
			clusterProfile: &clusterinventoryv1alpha1.ClusterProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-name",
					Namespace: "test-namespace",
				},
			},
			client: func() client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					Build()
			}(),
			wantErr:    true,
			wantErrMsg: "cluster endpoint annotation",
		},
		{
			name: "failed_to_create_secret",
			clusterProfile: &clusterinventoryv1alpha1.ClusterProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-name",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						gkeEndpointAnnotation: "https://test-server",
					},
				},
			},
			client: func() client.Client {
				return &errorOnCreateClient{
					Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
					error:  fmt.Errorf("test error"),
				}
			}(),
			wantErr:    true,
			wantErrMsg: "failed to create/update secret",
		},
		{
			name: "create_secret_ok",
			clusterProfile: &clusterinventoryv1alpha1.ClusterProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-name",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						gkeEndpointAnnotation: "https://test-server",
					},
				},
			},
			client: func() client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					Build()
			}(),
			wantSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-namespace.test-name",
					Namespace: argoCDNamespace,
					Labels: map[string]string{
						argoCDSecretType: "cluster",
					},
					Annotations: map[string]string{
						managedByAnnotation:  "true",
						clusterProfileOrigin: "test-namespace/test-name",
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"name":   []byte("test-namespace.test-name"),
					"server": []byte("https://test-server"),
					"config": []byte(secretConfig),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &ClusterProfileReconciler{
				Client: tc.client,
				scheme: scheme,
			}

			ctx := context.Background()
			err := r.createOrUpdateClusterSecret(ctx, tc.clusterProfile)

			if tc.wantErr {
				if err == nil {
					t.Errorf("createOrUpdateClusterSecret() returned nil, want %q", tc.wantErrMsg)
				} else if !strings.Contains(err.Error(), tc.wantErrMsg) {
					t.Errorf("createOrUpdateClusterSecret() returned error %q, want %q", err.Error(), tc.wantErrMsg)
				}
				return
			} else if err != nil {
				t.Errorf("createOrUpdateClusterSecret() unexpected error: %v", err)
			}

			secretName := types.NamespacedName{
				Namespace: argoCDNamespace,
				Name:      "test-namespace.test-name",
			}
			var gotSecret corev1.Secret
			if err := tc.client.Get(ctx, secretName, &gotSecret); err != nil {
				t.Fatalf("createOrUpdateClusterSecret() failed to get secret: %v", err)
			}

			if diff := cmp.Diff(tc.wantSecret, &gotSecret, cmpopts.EquateEmpty(), cmpopts.IgnoreFields(metav1.ObjectMeta{}, "ResourceVersion")); diff != "" {
				t.Errorf("createOrUpdateClusterSecret() unexpected secret (-want +got):\n%s", diff)
			}
		})
	}
}

// errorClient is a mock client that returns the specified error on Get
type errorClient struct {
	client.Client
	error error
}

func (c *errorClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return c.error
}

// errorOnCreateClient is a mock client that returns the specified error on Create
type errorOnCreateClient struct {
	client.Client
	error error
}

func (c *errorOnCreateClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return c.error
}
