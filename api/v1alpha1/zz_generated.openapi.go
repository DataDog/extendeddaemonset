// +build !ignore_autogenerated

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

// Code generated by openapi-gen. DO NOT EDIT.

// This file was autogenerated by openapi-gen. Do not edit it manually!

package v1alpha1

import (
	spec "github.com/go-openapi/spec"
	common "k8s.io/kube-openapi/pkg/common"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"./api/v1alpha1.ExtendedDaemonSet":                            schema__api_v1alpha1_ExtendedDaemonSet(ref),
		"./api/v1alpha1.ExtendedDaemonSetReplicaSet":                  schema__api_v1alpha1_ExtendedDaemonSetReplicaSet(ref),
		"./api/v1alpha1.ExtendedDaemonSetReplicaSetSpec":              schema__api_v1alpha1_ExtendedDaemonSetReplicaSetSpec(ref),
		"./api/v1alpha1.ExtendedDaemonSetReplicaSetSpecStrategy":      schema__api_v1alpha1_ExtendedDaemonSetReplicaSetSpecStrategy(ref),
		"./api/v1alpha1.ExtendedDaemonSetReplicaSetStatus":            schema__api_v1alpha1_ExtendedDaemonSetReplicaSetStatus(ref),
		"./api/v1alpha1.ExtendedDaemonSetSpec":                        schema__api_v1alpha1_ExtendedDaemonSetSpec(ref),
		"./api/v1alpha1.ExtendedDaemonSetSpecStrategy":                schema__api_v1alpha1_ExtendedDaemonSetSpecStrategy(ref),
		"./api/v1alpha1.ExtendedDaemonSetSpecStrategyCanary":          schema__api_v1alpha1_ExtendedDaemonSetSpecStrategyCanary(ref),
		"./api/v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoFail":  schema__api_v1alpha1_ExtendedDaemonSetSpecStrategyCanaryAutoFail(ref),
		"./api/v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoPause": schema__api_v1alpha1_ExtendedDaemonSetSpecStrategyCanaryAutoPause(ref),
		"./api/v1alpha1.ExtendedDaemonSetSpecStrategyRollingUpdate":   schema__api_v1alpha1_ExtendedDaemonSetSpecStrategyRollingUpdate(ref),
		"./api/v1alpha1.ExtendedDaemonSetStatus":                      schema__api_v1alpha1_ExtendedDaemonSetStatus(ref),
		"./api/v1alpha1.ExtendedDaemonSetStatusCanary":                schema__api_v1alpha1_ExtendedDaemonSetStatusCanary(ref),
		"./api/v1alpha1.ExtendedDaemonsetSetting":                     schema__api_v1alpha1_ExtendedDaemonsetSetting(ref),
		"./api/v1alpha1.ExtendedDaemonsetSettingContainerSpec":        schema__api_v1alpha1_ExtendedDaemonsetSettingContainerSpec(ref),
		"./api/v1alpha1.ExtendedDaemonsetSettingSpec":                 schema__api_v1alpha1_ExtendedDaemonsetSettingSpec(ref),
		"./api/v1alpha1.ExtendedDaemonsetSettingStatus":               schema__api_v1alpha1_ExtendedDaemonsetSettingStatus(ref),
	}
}

func schema__api_v1alpha1_ExtendedDaemonSet(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ExtendedDaemonSet is the Schema for the extendeddaemonsets API",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("./api/v1alpha1.ExtendedDaemonSetSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("./api/v1alpha1.ExtendedDaemonSetStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"./api/v1alpha1.ExtendedDaemonSetSpec", "./api/v1alpha1.ExtendedDaemonSetStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema__api_v1alpha1_ExtendedDaemonSetReplicaSet(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ExtendedDaemonSetReplicaSet is the Schema for the extendeddaemonsetreplicasets API",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("./api/v1alpha1.ExtendedDaemonSetReplicaSetSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("./api/v1alpha1.ExtendedDaemonSetReplicaSetStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"./api/v1alpha1.ExtendedDaemonSetReplicaSetSpec", "./api/v1alpha1.ExtendedDaemonSetReplicaSetStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema__api_v1alpha1_ExtendedDaemonSetReplicaSetSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ExtendedDaemonSetReplicaSetSpec defines the desired state of ExtendedDaemonSetReplicaSet",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"selector": {
						SchemaProps: spec.SchemaProps{
							Description: "A label query over pods that are managed by the daemon set. Must match in order to be controlled. If empty, defaulted to labels on Pod template.",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.LabelSelector"),
						},
					},
					"template": {
						SchemaProps: spec.SchemaProps{
							Description: "An object that describes the pod that will be created. The ExtendedDaemonSetReplicaSet will create exactly one copy of this pod on every node that matches the template's node selector (or on every node if no node selector is specified).",
							Ref:         ref("k8s.io/api/core/v1.PodTemplateSpec"),
						},
					},
					"templateGeneration": {
						SchemaProps: spec.SchemaProps{
							Description: "A sequence hash representing a specific generation of the template. Populated by the system. It can be set only during the creation.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
				},
				Required: []string{"template"},
			},
		},
		Dependencies: []string{
			"k8s.io/api/core/v1.PodTemplateSpec", "k8s.io/apimachinery/pkg/apis/meta/v1.LabelSelector"},
	}
}

func schema__api_v1alpha1_ExtendedDaemonSetReplicaSetSpecStrategy(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ExtendedDaemonSetReplicaSetSpecStrategy defines the desired state of ExtendedDaemonSet",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"rollingUpdate": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("./api/v1alpha1.ExtendedDaemonSetSpecStrategyRollingUpdate"),
						},
					},
					"reconcileFrequency": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.Duration"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"./api/v1alpha1.ExtendedDaemonSetSpecStrategyRollingUpdate", "k8s.io/apimachinery/pkg/apis/meta/v1.Duration"},
	}
}

func schema__api_v1alpha1_ExtendedDaemonSetReplicaSetStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ExtendedDaemonSetReplicaSetStatus defines the observed state of ExtendedDaemonSetReplicaSet",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"status": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"desired": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"integer"},
							Format: "int32",
						},
					},
					"current": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"integer"},
							Format: "int32",
						},
					},
					"ready": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"integer"},
							Format: "int32",
						},
					},
					"available": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"integer"},
							Format: "int32",
						},
					},
					"ignoredUnresponsiveNodes": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"integer"},
							Format: "int32",
						},
					},
					"conditions": {
						VendorExtensible: spec.VendorExtensible{
							Extensions: spec.Extensions{
								"x-kubernetes-list-map-keys": []interface{}{
									"type",
								},
								"x-kubernetes-list-type": "map",
							},
						},
						SchemaProps: spec.SchemaProps{
							Description: "Conditions Represents the latest available observations of a DaemonSet's current state.",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("./api/v1alpha1.ExtendedDaemonSetReplicaSetCondition"),
									},
								},
							},
						},
					},
				},
				Required: []string{"status", "desired", "current", "ready", "available", "ignoredUnresponsiveNodes"},
			},
		},
		Dependencies: []string{
			"./api/v1alpha1.ExtendedDaemonSetReplicaSetCondition"},
	}
}

func schema__api_v1alpha1_ExtendedDaemonSetSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ExtendedDaemonSetSpec defines the desired state of ExtendedDaemonSet",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"selector": {
						SchemaProps: spec.SchemaProps{
							Description: "A label query over pods that are managed by the daemon set. Must match in order to be controlled. If empty, defaulted to labels on Pod template. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.LabelSelector"),
						},
					},
					"template": {
						SchemaProps: spec.SchemaProps{
							Description: "An object that describes the pod that will be created. The ExtendedDaemonSet will create exactly one copy of this pod on every node that matches the template's node selector (or on every node if no node selector is specified). More info: https://kubernetes.io/docs/concepts/workloads/controllers/replicationcontroller#pod-template",
							Ref:         ref("k8s.io/api/core/v1.PodTemplateSpec"),
						},
					},
					"strategy": {
						SchemaProps: spec.SchemaProps{
							Description: "Daemonset deployment strategy",
							Ref:         ref("./api/v1alpha1.ExtendedDaemonSetSpecStrategy"),
						},
					},
				},
				Required: []string{"template", "strategy"},
			},
		},
		Dependencies: []string{
			"./api/v1alpha1.ExtendedDaemonSetSpecStrategy", "k8s.io/api/core/v1.PodTemplateSpec", "k8s.io/apimachinery/pkg/apis/meta/v1.LabelSelector"},
	}
}

func schema__api_v1alpha1_ExtendedDaemonSetSpecStrategy(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ExtendedDaemonSetSpecStrategy defines the deployment strategy of ExtendedDaemonSet",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"rollingUpdate": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("./api/v1alpha1.ExtendedDaemonSetSpecStrategyRollingUpdate"),
						},
					},
					"canary": {
						SchemaProps: spec.SchemaProps{
							Description: "Canary deployment configuration",
							Ref:         ref("./api/v1alpha1.ExtendedDaemonSetSpecStrategyCanary"),
						},
					},
					"reconcileFrequency": {
						SchemaProps: spec.SchemaProps{
							Description: "ReconcileFrequency use to configure how often the ExtendedDeamonset will be fully reconcile, default is 10sec",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.Duration"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"./api/v1alpha1.ExtendedDaemonSetSpecStrategyCanary", "./api/v1alpha1.ExtendedDaemonSetSpecStrategyRollingUpdate", "k8s.io/apimachinery/pkg/apis/meta/v1.Duration"},
	}
}

func schema__api_v1alpha1_ExtendedDaemonSetSpecStrategyCanary(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ExtendedDaemonSetSpecStrategyCanary defines the canary deployment strategy of ExtendedDaemonSet",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"replicas": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/util/intstr.IntOrString"),
						},
					},
					"duration": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.Duration"),
						},
					},
					"nodeSelector": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.LabelSelector"),
						},
					},
					"nodeAntiAffinityKeys": {
						VendorExtensible: spec.VendorExtensible{
							Extensions: spec.Extensions{
								"x-kubernetes-list-type": "set",
							},
						},
						SchemaProps: spec.SchemaProps{
							Type: []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Type:   []string{"string"},
										Format: "",
									},
								},
							},
						},
					},
					"autoPause": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("./api/v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoPause"),
						},
					},
					"autoFail": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("./api/v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoFail"),
						},
					},
					"noRestartsDuration": {
						SchemaProps: spec.SchemaProps{
							Description: "NoRestartsDuration defines min duration since last restart to end the canary phase",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.Duration"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"./api/v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoFail", "./api/v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoPause", "k8s.io/apimachinery/pkg/apis/meta/v1.Duration", "k8s.io/apimachinery/pkg/apis/meta/v1.LabelSelector", "k8s.io/apimachinery/pkg/util/intstr.IntOrString"},
	}
}

func schema__api_v1alpha1_ExtendedDaemonSetSpecStrategyCanaryAutoFail(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ExtendedDaemonSetSpecStrategyCanaryAutoFail defines the canary deployment AutoFail parameters of the ExtendedDaemonSet",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"enabled": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"boolean"},
							Format: "",
						},
					},
					"maxRestarts": {
						SchemaProps: spec.SchemaProps{
							Description: "MaxRestarts defines the number of tolerable Canary pod restarts after which the Canary deployment is autofailed",
							Type:        []string{"integer"},
							Format:      "int32",
						},
					},
					"maxRestartsDuration": {
						SchemaProps: spec.SchemaProps{
							Description: "MaxRestartsDuration defines the maximum duration of tolerable Canary pod restarts after which the Canary deployment is autofailed",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.Duration"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"k8s.io/apimachinery/pkg/apis/meta/v1.Duration"},
	}
}

func schema__api_v1alpha1_ExtendedDaemonSetSpecStrategyCanaryAutoPause(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ExtendedDaemonSetSpecStrategyCanaryAutoPause defines the canary deployment AutoPause parameters of the ExtendedDaemonSet",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"enabled": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"boolean"},
							Format: "",
						},
					},
					"maxRestarts": {
						SchemaProps: spec.SchemaProps{
							Description: "MaxRestarts defines the number of tolerable Canary pod restarts after which the Canary deployment is autopaused",
							Type:        []string{"integer"},
							Format:      "int32",
						},
					},
				},
			},
		},
	}
}

func schema__api_v1alpha1_ExtendedDaemonSetSpecStrategyRollingUpdate(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ExtendedDaemonSetSpecStrategyRollingUpdate defines the rolling update deployment strategy of ExtendedDaemonSet",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"maxUnavailable": {
						SchemaProps: spec.SchemaProps{
							Description: "The maximum number of DaemonSet pods that can be unavailable during the update. Value can be an absolute number (ex: 5) or a percentage of total number of DaemonSet pods at the start of the update (ex: 10%). Absolute number is calculated from percentage by rounding up. This cannot be 0. Default value is 1.",
							Ref:         ref("k8s.io/apimachinery/pkg/util/intstr.IntOrString"),
						},
					},
					"maxPodSchedulerFailure": {
						SchemaProps: spec.SchemaProps{
							Description: "MaxPodSchedulerFailure the maxinum number of not scheduled on its Node due to a scheduler failure: resource constraints. Value can be an absolute number (ex: 5) or a percentage of total number of DaemonSet pods at the start of the update (ex: 10%). Absolute",
							Ref:         ref("k8s.io/apimachinery/pkg/util/intstr.IntOrString"),
						},
					},
					"maxParallelPodCreation": {
						SchemaProps: spec.SchemaProps{
							Description: "The maxium number of pods created in parallel. Default value is 250.",
							Type:        []string{"integer"},
							Format:      "int32",
						},
					},
					"slowStartIntervalDuration": {
						SchemaProps: spec.SchemaProps{
							Description: "SlowStartIntervalDuration the duration between to 2 Default value is 1min.",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.Duration"),
						},
					},
					"slowStartAdditiveIncrease": {
						SchemaProps: spec.SchemaProps{
							Description: "SlowStartAdditiveIncrease Value can be an absolute number (ex: 5) or a percentage of total number of DaemonSet pods at the start of the update (ex: 10%). Default value is 5.",
							Ref:         ref("k8s.io/apimachinery/pkg/util/intstr.IntOrString"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"k8s.io/apimachinery/pkg/apis/meta/v1.Duration", "k8s.io/apimachinery/pkg/util/intstr.IntOrString"},
	}
}

func schema__api_v1alpha1_ExtendedDaemonSetStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ExtendedDaemonSetStatus defines the observed state of ExtendedDaemonSet",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"desired": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"integer"},
							Format: "int32",
						},
					},
					"current": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"integer"},
							Format: "int32",
						},
					},
					"ready": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"integer"},
							Format: "int32",
						},
					},
					"available": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"integer"},
							Format: "int32",
						},
					},
					"upToDate": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"integer"},
							Format: "int32",
						},
					},
					"ignoredUnresponsiveNodes": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"integer"},
							Format: "int32",
						},
					},
					"state": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"activeReplicaSet": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"canary": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("./api/v1alpha1.ExtendedDaemonSetStatusCanary"),
						},
					},
					"reason": {
						SchemaProps: spec.SchemaProps{
							Description: "Reason provides an explanation for canary deployment autopause",
							Type:        []string{"string"},
							Format:      "",
						},
					},
				},
				Required: []string{"desired", "current", "ready", "available", "upToDate", "ignoredUnresponsiveNodes", "activeReplicaSet"},
			},
		},
		Dependencies: []string{
			"./api/v1alpha1.ExtendedDaemonSetStatusCanary"},
	}
}

func schema__api_v1alpha1_ExtendedDaemonSetStatusCanary(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ExtendedDaemonSetStatusCanary defines the observed state of ExtendedDaemonSet canary deployment",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"replicaSet": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"nodes": {
						VendorExtensible: spec.VendorExtensible{
							Extensions: spec.Extensions{
								"x-kubernetes-list-type": "set",
							},
						},
						SchemaProps: spec.SchemaProps{
							Type: []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Type:   []string{"string"},
										Format: "",
									},
								},
							},
						},
					},
				},
				Required: []string{"replicaSet"},
			},
		},
	}
}

func schema__api_v1alpha1_ExtendedDaemonsetSetting(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ExtendedDaemonsetSetting is the Schema for the extendeddaemonsetsettings API",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("./api/v1alpha1.ExtendedDaemonsetSettingSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("./api/v1alpha1.ExtendedDaemonsetSettingStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"./api/v1alpha1.ExtendedDaemonsetSettingSpec", "./api/v1alpha1.ExtendedDaemonsetSettingStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema__api_v1alpha1_ExtendedDaemonsetSettingContainerSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ExtendedDaemonsetSettingContainerSpec defines the resources override for a container identified by its name",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"name": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"resources": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/api/core/v1.ResourceRequirements"),
						},
					},
				},
				Required: []string{"name", "resources"},
			},
		},
		Dependencies: []string{
			"k8s.io/api/core/v1.ResourceRequirements"},
	}
}

func schema__api_v1alpha1_ExtendedDaemonsetSettingSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ExtendedDaemonsetSettingSpec is the Schema for the extendeddaemonsetsetting API",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"reference": {
						SchemaProps: spec.SchemaProps{
							Description: "Reference contains enough information to let you identify the referred resource.",
							Ref:         ref("k8s.io/api/autoscaling/v1.CrossVersionObjectReference"),
						},
					},
					"nodeSelector": {
						SchemaProps: spec.SchemaProps{
							Description: "NodeSelector lists labels that must be present on nodes to trigger the usage of this resource.",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.LabelSelector"),
						},
					},
					"containers": {
						VendorExtensible: spec.VendorExtensible{
							Extensions: spec.Extensions{
								"x-kubernetes-list-map-keys": []interface{}{
									"name",
								},
								"x-kubernetes-list-type": "map",
							},
						},
						SchemaProps: spec.SchemaProps{
							Description: "Containers contains a list of container spec override.",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("./api/v1alpha1.ExtendedDaemonsetSettingContainerSpec"),
									},
								},
							},
						},
					},
				},
				Required: []string{"reference", "nodeSelector"},
			},
		},
		Dependencies: []string{
			"./api/v1alpha1.ExtendedDaemonsetSettingContainerSpec", "k8s.io/api/autoscaling/v1.CrossVersionObjectReference", "k8s.io/apimachinery/pkg/apis/meta/v1.LabelSelector"},
	}
}

func schema__api_v1alpha1_ExtendedDaemonsetSettingStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ExtendedDaemonsetSettingStatus defines the observed state of ExtendedDaemonsetSetting",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"status": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"error": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
				},
				Required: []string{"status"},
			},
		},
	}
}
