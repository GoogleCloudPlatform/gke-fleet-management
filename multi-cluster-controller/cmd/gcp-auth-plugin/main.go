/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"golang.org/x/oauth2/google"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthv1beta1 "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
)

// defaultGCPScopes:
//   - cloud-platform is the base scope to authenticate to GCP.
//   - userinfo.email is used to authenticate to GKE APIs with gserviceaccount
//     email instead of numeric uniqueID.
//
// https://github.com/kubernetes/client-go/blob/be758edd136e61a1bffadf1c0235fceb8aee8e9e/plugin/pkg/client/auth/gcp/gcp.go#L59
var defaultGCPScopes = []string{
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/userinfo.email",
}

func main() {
	ctx := context.Background()
	// Preferred way to retrieve GCP credentials
	// https://github.com/golang/oauth2/blob/9780585627b5122c8cc9c6a378ac9861507e7551/google/doc.go#L54-L68
	cred, err := google.FindDefaultCredentials(ctx, defaultGCPScopes...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get default token source: %v\n", err)
		os.Exit(1)
	}

	token, err := cred.TokenSource.Token()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get token: %v\n", err)
		os.Exit(1)
	}

	expiration := metav1.NewTime(token.Expiry)
	if token.Expiry.IsZero() {
		expiration = metav1.NewTime(time.Now().Add(1 * time.Hour))
	}

	execCred := &clientauthv1beta1.ExecCredential{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "client.authentication.k8s.io/v1beta1",
			Kind:       "ExecCredential",
		},
		Status: &clientauthv1beta1.ExecCredentialStatus{
			ExpirationTimestamp: &expiration,
			Token:               token.AccessToken,
		},
	}

	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(execCred); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode exec credential: %v\n", err)
		os.Exit(1)
	}
}
