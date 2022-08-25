/*
Copyright 2022.

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

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	internal "github.com/neilmcgibbon/eks-iam-operator/internal"

	eksiamoperatorv1beta1 "github.com/neilmcgibbon/eks-iam-operator/api/v1beta1"
)

const RoleUpsertTag = "iamoperator.fiit.tv/v1beta1"

// RoleReconciler reconciles a Role object
type RoleReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	RolePrefix         string
	RoleSuffix         string
	InlinePolicyPrefix string
	InlinePolicySuffix string
	OIDCIssuerURL      string
	OIDCProviderARN    string
}

//+kubebuilder:rbac:groups=eks-iam-operator.neilmcgibbon.com,resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=eks-iam-operator.neilmcgibbon.com,resources=roles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=eks-iam-operator.neilmcgibbon.com,resources=roles/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Role object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *RoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var role eksiamoperatorv1beta1.Role
	if err := r.Get(ctx, req.NamespacedName, &role); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Generate role name from prefix, cluster, region and role
	fullRoleName := fmt.Sprintf("%s%s%s", r.RolePrefix, role.Name, r.RoleSuffix)
	r.Log.Info("Reconciling role", "role", fullRoleName)

	// Check if need to reconcile this
	if role.Status.ObservedGeneration == role.ObjectMeta.Generation && role.Status.State == eksiamoperatorv1beta1.SyncStateOK {
		r.Log.Info("Role already reconciled, not doing anything", "role", fullRoleName)
	}

	// Get our AWS client
	client, err := internal.NewAWSRoleClient(ctx, r.Log)
	if err != nil {
		r.statusUpdater(ctx, &role, err)
		return ctrl.Result{}, err
	}

	finalizer := "role.eks-iam-operator.neilmcgibbon.com/finalizer"

	if role.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&role, finalizer) {
			// Update the object with the finalizer and return, because the change triggers
			// another run of the reconciler
			controllerutil.AddFinalizer(&role, finalizer)
			if err := r.Update(ctx, &role); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(&role, finalizer) {
			if err := client.Delete(ctx, fullRoleName); err != nil {
				r.statusUpdater(ctx, &role, err)
				return ctrl.Result{}, err
			}

			// AWS Role is deleted, so now remove finalizer so Kubernets deletes the dead resource
			controllerutil.RemoveFinalizer(&role, finalizer)
			if err := r.Update(ctx, &role); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	trustPolicy, err := generateTrustPolicy(role.Spec.ServiceAccounts, role.Spec.Namespace, r.OIDCIssuerURL, r.OIDCProviderARN)
	if err != nil {
		r.statusUpdater(ctx, &role, err)
		return ctrl.Result{}, err
	}

	policies, err := r.generateInlinePolicies(role.Spec.Statements)
	if err != nil {
		r.statusUpdater(ctx, &role, err)
		return ctrl.Result{}, err
	}

	if err = client.Upsert(ctx, fullRoleName, trustPolicy, policies); err != nil {
		r.statusUpdater(ctx, &role, err)
		return ctrl.Result{}, err
	}

	// Set observed generation
	//role.Status.ObservedGeneration = role.ObjectMeta.Generation

	r.statusUpdater(ctx, &role, nil)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&eksiamoperatorv1beta1.Role{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

// stringOrArray takes an array and returns the value of the first element if the array has one item, otherwise
// returns the raw array
func stringOrArray(tst []string) interface{} {
	if len(tst) == 1 {
		return tst[0]
	}
	return tst
}

// generateTrustPolicy returns a string representation of an AWS Assume Role policy, formatted specifically for
// service accounts running in an EKS cluster
func generateTrustPolicy(services []string, ns, oidcIssuerURL, oidcProviderARN string) (string, error) {

	// Create our StringLike condition for service name matches
	sa := []string{}
	for _, v := range services {
		sa = append(sa, fmt.Sprintf("system:serviceaccount:%s:%s", ns, v))
	}

	doc := internal.NewAWSTrustPolicy()
	doc.Statement[0].Principal["Federated"] = oidcProviderARN
	doc.Statement[0].Condition["StringLike"] = map[string][]string{
		fmt.Sprintf("%s:sub", strings.TrimPrefix(oidcIssuerURL, "https://")): sa,
	}

	j, err := json.Marshal(doc)
	return string(j), err
}

// generateInlinePolicies returns a map of JSON string IAM policies, with the map key as the intended inline
// policy name
func (r *RoleReconciler) generateInlinePolicies(perms map[string][]eksiamoperatorv1beta1.StatementSpec) (map[string]string, error) {

	policies := map[string]string{}

	for svc, stmts := range perms {
		d := &internal.AWSPolicyDocument{Version: "2012-10-17", Statement: []internal.AWSPolicyDocumentStatement{}}
		for _, stmt := range stmts {
			d.Statement = append(d.Statement, internal.AWSPolicyDocumentStatement{
				Effect:    "Allow",
				Actions:   stringOrArray(stmt.Actions),
				Resources: stringOrArray(stmt.Resources),
			})
		}

		j, err := json.Marshal(d)
		if err != nil {
			return policies, err
		}

		policies[fmt.Sprintf("%s%s%s", r.InlinePolicyPrefix, svc, r.InlinePolicySuffix)] = string(j)
	}

	return policies, nil
}

// generateInlinePolicies returns a map of JSON string IAM policies, with the map key as the intended inline
// policy name
func (r *RoleReconciler) statusUpdater(ctx context.Context, role *eksiamoperatorv1beta1.Role, err error) {

	if err == nil {
		role.Status.Error = "<none>"
		role.Status.State = eksiamoperatorv1beta1.SyncStateOK
		role.Status.ObservedGeneration = role.ObjectMeta.Generation
	} else {
		role.Status.Error = err.Error()
		role.Status.State = eksiamoperatorv1beta1.SyncStateErr
	}

	if e := r.Status().Update(ctx, role); e != nil {
		r.Log.Error(e, "unable to update Role status")
	}
}
