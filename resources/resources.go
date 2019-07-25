package resources

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	FinalizerAlreadyPresent AddFinalizerResult    = false
	FinalizerAdded          AddFinalizerResult    = true
	FinalizerRemoved        RemoveFinalizerResult = true
	FinalizerNotFound       RemoveFinalizerResult = false
)

// CopyMap makes a copy of the map.
func CopyMap(a map[string]string) map[string]string {
	ret := make(map[string]string, len(a))
	for k, v := range a {
		ret[k] = v
	}
	return ret
}

// UnionMaps returns a map constructed from the union of `a` and `b`,
// where value from `b` wins.
func UnionMaps(a, b map[string]string) map[string]string {
	out := make(map[string]string, len(a)+len(b))

	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

// FilterMap creates a copy of the provided map, filtering out the elements
// that match `filter`.
// nil `filter` is accepted.
func FilterMap(in map[string]string, filter func(string) bool) map[string]string {
	ret := make(map[string]string, len(in))
	for k, v := range in {
		if filter != nil && filter(k) {
			continue
		}
		ret[k] = v
	}
	return ret
}

// AddFinalizerResult is used to indicate whether a finalizer was added or already present.
type AddFinalizerResult bool

// RemoveFinalizerResult is used to indicate whether a finalizer was found and removed (FinalizerRemoved), or finalizer not found (FinalizerNotFound).
type RemoveFinalizerResult bool

// AddFinalizer adds finalizerName to the Object.
func AddFinalizer(o metav1.Object, finalizerName string) AddFinalizerResult {
	finalizers := sets.NewString(o.GetFinalizers()...)
	if finalizers.Has(finalizerName) {
		return FinalizerAlreadyPresent
	}
	finalizers.Insert(finalizerName)
	o.SetFinalizers(finalizers.List())
	return FinalizerAdded
}

// RemoveFinalizer removes the finalizer(finalizerName) from the object(o) if the finalizer is present.
// Returns: - FinalizerRemoved, if the finalizer was found and removed.
//          - FinalizerNotFound, if the finalizer was not found.
func RemoveFinalizer(o metav1.Object, finalizerName string) RemoveFinalizerResult {
	finalizers := sets.NewString(o.GetFinalizers()...)
	if finalizers.Has(finalizerName) {
		finalizers.Delete(finalizerName)
		o.SetFinalizers(finalizers.List())
		return FinalizerRemoved
	}
	return FinalizerNotFound
}
