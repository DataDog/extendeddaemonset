// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

// Package limits contains function to calculate pod create/deletion limits.
package limits

// Parameters use to provide the parameters to the Calculation function.
type Parameters struct {
	NbNodes              int
	NbPods               int
	NbAvailablesPod      int
	NbOldAvailablesPod   int
	NbCreatedPod         int
	NbUnresponsiveNodes  int
	NbOldUnavailablePods int

	MaxPodCreation      int
	MaxUnavailablePod   int
	MaxUnschedulablePod int
}

// CalculatePodToCreateAndDelete from the parameters return:
// * nbCreation: the number of pods to create
// * nbDeletion: the number of pods to delete.
func CalculatePodToCreateAndDelete(params Parameters) (nbCreation, nbDeletion int) {
	nbCreation = min(params.NbNodes-params.NbPods, params.MaxPodCreation)
	// Prevent negative number of pods to create
	nbCreation = max(nbCreation)

	effectiveUnresponsive := min(params.NbUnresponsiveNodes, params.MaxUnschedulablePod)

	nbDeletion = min(params.MaxUnavailablePod-(params.NbNodes-effectiveUnresponsive-params.NbAvailablesPod-params.NbOldAvailablesPod)+params.NbOldUnavailablePods, params.MaxUnavailablePod)
	// Prevent negative number of pods to delete
	nbDeletion = max(nbDeletion, 0)

	return nbCreation, nbDeletion
}
