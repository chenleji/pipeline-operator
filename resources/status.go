package resources

import (
	"github.com/knative/build/pkg/apis/build/v1alpha1"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

const (
	ReasonBuildTimeout = "BuildTimeout"
	ReasonRetry        = "BuildRetry"
	ReasonFail         = "Fail"
)

func IsStatusExpired(status *duckv1alpha1.Status) bool {
	cond := status.GetCondition(v1alpha1.BuildCancelled)
	return cond != nil && cond.Status != corev1.ConditionUnknown
}

func IsStatusDone(status *duckv1alpha1.Status) bool {
	cond := status.GetCondition(v1alpha1.BuildSucceeded)
	return cond != nil && cond.Status == corev1.ConditionTrue
}

func IsStatusFailed(status *duckv1alpha1.Status) bool {
	cond := status.GetCondition(v1alpha1.BuildSucceeded)
	return cond != nil && cond.Status == corev1.ConditionFalse
}

func IsStatusRetry(status *duckv1alpha1.Status) bool {
	cond := status.GetCondition(v1alpha1.BuildSucceeded)
	return cond != nil && cond.Status == corev1.ConditionUnknown && cond.Reason == ReasonRetry
}

func IsStatusFailedTimeout(status *duckv1alpha1.Status) bool {
	cond := status.GetCondition(v1alpha1.BuildSucceeded)
	return cond != nil && cond.Status == corev1.ConditionFalse && cond.Reason == ReasonBuildTimeout
}

func IsStatusFailedFinallyfail(status *duckv1alpha1.Status) bool {
	cond := status.GetCondition(v1alpha1.BuildSucceeded)
	return cond != nil && cond.Status == corev1.ConditionFalse && cond.Reason == ReasonFail
}
