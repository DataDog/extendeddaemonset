// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package extendeddaemonsetsetting

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	datadoghqv1alpha1 "github.com/DataDog/extendeddaemonset/api/v1alpha1"
	"github.com/DataDog/extendeddaemonset/api/v1alpha1/test"
	commontest "github.com/DataDog/extendeddaemonset/pkg/controller/test"
)

func Test_searchPossibleConflict(t *testing.T) {
	now := time.Now()
	commonLabels := map[string]string{
		"test": "bigmemory",
	}
	edsOptions1 := &test.NewExtendedDaemonsetSettingOptions{
		CreationTime: now,
		Selector:     commonLabels,
	}
	edsNode1 := test.NewExtendedDaemonsetSetting("bar", "foo", "app", edsOptions1)
	edsOptions2 := &test.NewExtendedDaemonsetSettingOptions{
		CreationTime: now.Add(time.Minute),
		Selector:     commonLabels,
	}
	edsNode2 := test.NewExtendedDaemonsetSetting("bar", "foo2", "app", edsOptions2)
	nodeOptions := &commontest.NewNodeOptions{
		Labels: commonLabels,
		Conditions: []corev1.NodeCondition{
			{
				Type:   corev1.NodeReady,
				Status: corev1.ConditionTrue,
			},
		},
	}
	node1 := commontest.NewNode("node1", nodeOptions)
	type args struct {
		instance    *datadoghqv1alpha1.ExtendedDaemonsetSetting
		nodeList    *corev1.NodeList
		edsNodeList *datadoghqv1alpha1.ExtendedDaemonsetSettingList
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "No Node",
			args: args{
				instance: edsNode1,
				nodeList: &corev1.NodeList{},
				edsNodeList: &datadoghqv1alpha1.ExtendedDaemonsetSettingList{
					Items: []datadoghqv1alpha1.ExtendedDaemonsetSetting{*edsNode1},
				},
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "1 ExtendedDaemonsetSetting, no conflict",
			args: args{
				instance: edsNode1,
				nodeList: &corev1.NodeList{
					Items: []corev1.Node{*node1},
				},
				edsNodeList: &datadoghqv1alpha1.ExtendedDaemonsetSettingList{
					Items: []datadoghqv1alpha1.ExtendedDaemonsetSetting{*edsNode1},
				},
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "1 ExtendedDaemonsetSetting, conflict between 2 ExtendedDaemonsetSetting",
			args: args{
				instance: edsNode1,
				nodeList: &corev1.NodeList{
					Items: []corev1.Node{*node1},
				},
				edsNodeList: &datadoghqv1alpha1.ExtendedDaemonsetSettingList{
					Items: []datadoghqv1alpha1.ExtendedDaemonsetSetting{*edsNode1, *edsNode2},
				},
			},
			want:    "foo2",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := searchPossibleConflict(tt.args.instance, tt.args.nodeList, tt.args.edsNodeList)
			if (err != nil) != tt.wantErr {
				t.Errorf("searchPossibleConflict() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if got != tt.want {
				t.Errorf("searchPossibleConflict() = %v, want %v", got, tt.want)
			}
		})
	}
}
