package internal

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/go-logr/logr"
)

const roleOwnerTag = "eks-iam-operator.neilmcgibbon.com"

type AWSRoleClient struct {
	cfg aws.Config
	log logr.Logger
}

func NewAWSRoleClient(ctx context.Context, l logr.Logger) (*AWSRoleClient, error) {
	cl := &AWSRoleClient{log: l}
	c, err := config.LoadDefaultConfig(ctx, config.WithRegion("eu-west-1"))

	if err != nil {
		return cl, err
	}

	cl.cfg = c
	return cl, nil
}

// Upsert creates or updates a role, using the provided assume role policy and map of inline policies
func (c *AWSRoleClient) Upsert(ctx context.Context, name string, trustPolicy string, inlinePolicies map[string]string) error {

	existing, err := c.getRole(ctx, name)
	if err != nil {
		return err
	}

	// Holder for existing policies, to determine deletions
	inlinePoliciesToDelete := []string{}

	// Create role (or check we can edit role if it exists)
	if existing != nil {
		// IAM role exists, lets check we can modify it
		if roleHasTag(existing, roleOwnerTag) == false {
			return errors.New("Not upserting as role does not have the operator owner tag")
		}
		existingInlinePolicies, err := c.getRoleInlinePolicies(ctx, name)
		if err != nil {
			return err
		}

		inlinePoliciesToDelete = getInlinePoliciesToDelete(existingInlinePolicies, inlinePolicies)

	} else {
		// IAM role does not exist, create it
		if err := c.createRole(ctx, name, trustPolicy); err != nil {
			return err
		}
	}

	// @todo only do this if we need to
	if err = c.updateRoleTrustPolicy(ctx, name, trustPolicy); err != nil {
		return err
	}

	// Update role inline policies
	if err = c.upsertRoleInlinePolicies(ctx, name, inlinePolicies); err != nil {
		return err
	}

	// Delete role inline policies
	if err = c.deleteRoleInlinePolicies(ctx, name, inlinePoliciesToDelete); err != nil {
		return err
	}

	return nil
}

// Delete deletes a role and its associated inline policies.
func (c *AWSRoleClient) Delete(ctx context.Context, name string) error {
	client := iam.NewFromConfig(c.cfg)

	existingInlinePolicies, err := c.getRoleInlinePolicies(ctx, name)
	if err != nil {
		return err
	}

	for _, p := range existingInlinePolicies {
		c.log.Info("Deleting AWS role policy", "role", name, "policy", p)
		if _, err = client.DeleteRolePolicy(ctx, &iam.DeleteRolePolicyInput{
			RoleName:   aws.String(name),
			PolicyName: aws.String(p),
		}); err != nil {
			return err
		}
	}

	c.log.Info("Deleting AWS role", "role", name)
	_, err = client.DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: aws.String(name),
	})
	return err
}

// createRole calls the AWS IAM API to create a new role, using the provided assume role policy
func (c *AWSRoleClient) createRole(ctx context.Context, name string, trustPolicy string) error {
	client := iam.NewFromConfig(c.cfg)

	c.log.Info("Creating IAM role", "role", name)
	_, err := client.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String(name),
		AssumeRolePolicyDocument: aws.String(trustPolicy),
		Tags: []types.Tag{{
			Key:   aws.String(roleOwnerTag),
			Value: aws.String("true"),
		}},
	})

	return err
}

// getRole calls the AWS IAM API to return the an AWS IAM role instance
func (c *AWSRoleClient) getRole(ctx context.Context, name string) (*types.Role, error) {
	client := iam.NewFromConfig(c.cfg)
	entity, err := client.GetRole(ctx, &iam.GetRoleInput{RoleName: aws.String(name)})

	var noSuchEntityException *types.NoSuchEntityException

	// AWS returns an error of specific type if not found
	if err != nil && errors.As(err, &noSuchEntityException) {
		return nil, nil
	}

	// If the error was not a NotFound error, then there's actually an error with the API
	if err != nil {
		return nil, err
	}

	return entity.Role, nil
}

// getRoleInlinePolicies calls the AWS IAM API to return a string array of currently applied inline policies
func (c *AWSRoleClient) getRoleInlinePolicies(ctx context.Context, role string) ([]string, error) {
	client := iam.NewFromConfig(c.cfg)

	c.log.Info("Retrieving list of current role policies", "role", role)
	existingPolicies, err := client.ListRolePolicies(ctx, &iam.ListRolePoliciesInput{RoleName: aws.String(role)})
	if err != nil {
		return []string{}, err
	}

	return existingPolicies.PolicyNames, nil
}

// updateRoleTrustPolicy calls the AWS IAM API to overwite the existing assume role policy on the role
func (c *AWSRoleClient) updateRoleTrustPolicy(ctx context.Context, role string, trustPolicy string) error {
	client := iam.NewFromConfig(c.cfg)

	c.log.Info("Updating role trust policy document", "role", role)
	if _, err := client.UpdateAssumeRolePolicy(ctx, &iam.UpdateAssumeRolePolicyInput{
		RoleName:       aws.String(role),
		PolicyDocument: aws.String(trustPolicy),
	}); err != nil {
		return err
	}

	return nil
}

// upsertRoleInlinePolicies iterates over a string array of inline policies and calls the AWS IAM API to
// add (or overwrite)
func (c *AWSRoleClient) upsertRoleInlinePolicies(ctx context.Context, role string, inlinePolicies map[string]string) error {
	client := iam.NewFromConfig(c.cfg)

	for policy, doc := range inlinePolicies {
		c.log.Info("Upserting inline policy", "role", role, "policy", policy)
		if _, err := client.PutRolePolicy(ctx, &iam.PutRolePolicyInput{
			RoleName:       aws.String(role),
			PolicyName:     aws.String(policy),
			PolicyDocument: aws.String(doc),
		}); err != nil {
			return err
		}
	}
	return nil
}

// deleteRoleInlinePolicies iterates over a string array of inline policies and calls the AWS IAM API to
// delete.
func (c *AWSRoleClient) deleteRoleInlinePolicies(ctx context.Context, role string, inlinePolicies []string) error {
	client := iam.NewFromConfig(c.cfg)

	for _, policy := range inlinePolicies {
		c.log.Info("Deleting inline role policy", "role", role, "policy", policy)
		if _, err := client.DeleteRolePolicy(ctx, &iam.DeleteRolePolicyInput{
			RoleName:   aws.String(role),
			PolicyName: aws.String(policy),
		}); err != nil {
			return err
		}
	}

	return nil
}

// roleHasTag iterates over IAM role object tags and returns true if passed tag exists
func roleHasTag(role *types.Role, tag string) bool {
	for _, v := range role.Tags {
		if *v.Key == tag {
			return true
		}
	}
	return false
}

// roleHasTag iterates over a string array of existing inline policy names, and compares it to map keys
// in the new inline policies to add. If there is no match, the inline policy is added to a the return
// value (to be deleted)
func getInlinePoliciesToDelete(existing []string, new map[string]string) []string {
	delete := []string{}
	for _, v := range existing {
		for k, _ := range new {
			keep := true
			if v == k {
				keep = false
				break
			}
			if keep == false {
				delete = append(delete, v)
			}
		}
	}
	return delete
}
