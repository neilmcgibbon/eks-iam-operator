package internal

type AWSPolicyDocument struct {
	Statement []AWSPolicyDocumentStatement `json:"Statement"`
	Version   string                       `json:"Version"`
}

type AWSPolicyDocumentStatement struct {
	Sid       string                         `json:"Sid,omitempty"`
	Effect    string                         `json:"Effect"`
	Resources interface{}                    `json:"Resource,omitempty"`
	Principal map[string]string              `json:"Principal,omitempty"`
	Actions   interface{}                    `json:"Action,omitempty"`
	Condition map[string]map[string][]string `json:"Condition,omitempty"`
}

func NewAWSTrustPolicy() *AWSPolicyDocument {
	return &AWSPolicyDocument{
		Statement: []AWSPolicyDocumentStatement{{
			Effect:    "Allow",
			Principal: map[string]string{},
			Condition: map[string]map[string][]string{},
			Actions:   "sts:AssumeRoleWithWebIdentity",
		}},
		Version: "2012-10-17",
	}
}
