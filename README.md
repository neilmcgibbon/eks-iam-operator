# EKS IAM Operator

An opinionated IAM role management Kubernetes operator for service accounts running in AWS EKS.

## What is this?
---
Service accounts running in EKS clusters can assume IAM roles in AWS. See the [AWS documentataion](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) for more information on this process. This operator watches for CRD resources describing AWS permissions and creates a role (with EKS federated principals using OIDC values as as AssumeRole policy) and appropriate inline policies for AWS resource access. This means that that IAM roles and permissions can be defined as part of the service, rather than separately (e.g. via Terraform).

--- 
## Exmaple

```yaml
apiVersion: eks-iam-operator.neilmcgibbon.com/v1beta1
kind: Role
metadata:
  name: eks-my-service-account # This is the role name that will be created in AWS
spec:
  namespace: default
  serviceAccounts: 
  - my-service-account
  statements: 
    log:
    - actions:
      - "cloudwatch:*"
      resources: 
      - "*"
    dynamodb:
      - actions: 
        - dynamodb:GetItem
        resources:
        - arn:aws:dynamodb:eu-west-1:111111111111:table/foo
        - arn:aws:dynamodb:eu-west-1:111111111111:table/bar
      - actions: 
        - dynamodb:PutItem
        resources:
        - arn:aws:dynamodb:eu-west-1:111111111111:table/foo
```

This will create the following resources in AWS

| Resource Type | Notes |
|-|-|
| Role | IAM role name : `eks-my-service-account` |
| AssumeRole Policy | Allows trust from k8s serviceaccount `system:serviceaccount:default:my-service-account` |
| Inline Policy | policy name: `log`, Contains one statment, with the `cloudwatch:*` access | 
| Inline Policy | policy name: `dynamodb`, Contains two statment, with the `GetItem` for tables `foo` & `bar` , and one with `PutItem` for table `foo` only | 

## IAM Permissions

This controller needs a subset of AWS permissions to operate correctly. Create your role in AWS with the (minimum) requirements below, and provide the created role ARN to the controller (using the values parameter specified in the Helm chart instructions).

  - iam:GetRole
  - iam:UpdateAssumeRolePolicy
  - iam:DeleteRolePolicy
  - iam:TagRole
  - iam:CreateRole
  - iam:DeleteRole
  - iam:UpdateRole
  - iam:PutRolePolicy
  - iam:ListRolePolicies
  - iam:GetRolePolicy

Optionally - to limit the roles that the controller will manage - you may specifiy a resource prefix in this IAM role, ensuring you specifiy the same prefix in the Helm chart configuration.

## Install
---
Please see the [Helm](https://github.com/neilmcgibbon/eks-iam-operator/tree/main/helm) README.md file installation.



