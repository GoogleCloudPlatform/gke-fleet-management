package profileclient

import (
	"context"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterinventory "sigs.k8s.io/cluster-inventory-api/apis/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestPluginResults(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clusterinventory.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add to scheme: %v", err)
	}

	testCases := []struct {
		name       string
		profiles   []clusterinventory.ClusterProfile
		namespaces []string
		selector   *metav1.LabelSelector
		want       []Result
		wantErr    bool
	}{
		{
			name:       "no profiles",
			namespaces: []string{"ns1"},
			selector:   &metav1.LabelSelector{},
			want:       nil,
		},
		{
			name: "one profile in one namespace",
			profiles: []clusterinventory.ClusterProfile{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "profile1",
						Namespace: "ns1",
					},
				},
			},
			namespaces: []string{"ns1"},
			selector:   &metav1.LabelSelector{},
			want: []Result{
				{Name: "ns1.profile1"},
			},
		},
		{
			name: "profiles in multiple namespaces",
			profiles: []clusterinventory.ClusterProfile{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "profile1",
						Namespace: "ns1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "profile2",
						Namespace: "ns2",
					},
				},
			},
			namespaces: []string{"ns1", "ns2"},
			selector:   &metav1.LabelSelector{},
			want: []Result{
				{Name: "ns1.profile1"},
				{Name: "ns2.profile2"},
			},
		},
		{
			name: "selector matches one profile",
			profiles: []clusterinventory.ClusterProfile{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "profile1",
						Namespace: "ns1",
						Labels:    map[string]string{"foo": "bar"},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "profile2",
						Namespace: "ns1",
					},
				},
			},
			namespaces: []string{"ns1"},
			selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"foo": "bar"},
			},
			want: []Result{
				{Name: "ns1.profile1"},
			},
		},
		{
			name: "selector matches no profiles",
			profiles: []clusterinventory.ClusterProfile{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "profile1",
						Namespace: "ns1",
						Labels:    map[string]string{"foo": "bar"},
					},
				},
			},
			namespaces: []string{"ns1"},
			selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"baz": "qux"},
			},
			want: nil,
		},
		{
			name: "nonexistent namespace",
			profiles: []clusterinventory.ClusterProfile{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "profile1",
						Namespace: "ns1",
					},
				},
			},
			namespaces: []string{"ns2"},
			selector:   &metav1.LabelSelector{},
			want:       nil,
		},
		{
			name:       "invalid selector",
			namespaces: []string{"ns1"},
			selector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "foo",
						Operator: "invalid",
						Values:   []string{"bar"},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var objs []runtime.Object
			for i := range tc.profiles {
				objs = append(objs, &tc.profiles[i])
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objs...).Build()
			c := &ProfileClient{
				client: fakeClient,
			}

			got, err := c.PluginResults(context.Background(), tc.namespaces, tc.selector)
			if tc.wantErr != (err != nil) {
				t.Errorf("PluginResults() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErr {
				return
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("PluginResults() returned diff (-want +got):\n%s", diff)
			}
		})
	}
}
