package api

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	v1helper "k8s.io/kubernetes/pkg/apis/core/v1/helper"
	"math"
	"volcano.sh/volcano/pkg/scheduler/util/assert"
)

const (
	GPUResourceName = "nvidia.com/gpu"
)

type ResourceInfo struct {
	CPUNodes        int
	ScalarResources map[v1.ResourceName]float64
}

func EmptyResource() *ResourceInfo {
	return &ResourceInfo{}
}

func (r *ResourceInfo) Clone() *ResourceInfo {
	clone := &ResourceInfo{
		CPUNodes: r.CPUNodes,
	}

	if r.ScalarResources != nil {
		clone.ScalarResources = make(map[v1.ResourceName]float64)
		for k, v := range r.ScalarResources {
			clone.ScalarResources[k] = v
		}
	}

	return clone
}

var minMilliScalarResources float64 = 10

func NewResource(rl v1.ResourceList) *ResourceInfo {
	r := EmptyResource()
	r.CPUNodes = 1
	for rName, rQuant := range rl {
		//NOTE: When converting this back to k8s resource, we need record the format as well as / 1000
		if v1helper.IsScalarResourceName(rName) {
			r.AddScalar(rName, float64(rQuant.MilliValue()))
		}
	}
	return r
}

// IsEmpty returns bool after checking any of resource is less than min possible value
func (r *ResourceInfo) IsEmpty() bool {
	if r.CPUNodes != 0 {
		return false
	}

	for _, rQuant := range r.ScalarResources {
		if rQuant >= minMilliScalarResources {
			return false
		}
	}

	return true
}

// Add is used to add the two resources
func (r *ResourceInfo) Add(rr *ResourceInfo) *ResourceInfo {
	r.CPUNodes += rr.CPUNodes

	for rName, rQuant := range rr.ScalarResources {
		if r.ScalarResources == nil {
			r.ScalarResources = map[v1.ResourceName]float64{}
		}
		r.ScalarResources[rName] += rQuant
	}

	return r
}

//Sub subtracts two Resource objects.
func (r *ResourceInfo) Sub(rr *ResourceInfo) *ResourceInfo {
	assert.Assertf(rr.LessEqual(r), "resource is not sufficient to do operation: <%v> sub <%v>", r, rr)

	r.CPUNodes -= rr.CPUNodes

	for rrName, rrQuant := range rr.ScalarResources {
		if r.ScalarResources == nil {
			return r
		}
		r.ScalarResources[rrName] -= rrQuant
	}

	return r
}

// SetMaxResource compares with ResourceList and takes max value for each Resource.
func (r *ResourceInfo) SetMaxResource(rr *ResourceInfo) {
	if r == nil || rr == nil {
		return
	}

	if rr.CPUNodes > r.CPUNodes {
		r.CPUNodes = rr.CPUNodes
	}

	for rrName, rrQuant := range rr.ScalarResources {
		if r.ScalarResources == nil {
			r.ScalarResources = make(map[v1.ResourceName]float64)
			for k, v := range rr.ScalarResources {
				r.ScalarResources[k] = v
			}
			return
		}

		if rrQuant > r.ScalarResources[rrName] {
			r.ScalarResources[rrName] = rrQuant
		}
	}
}

//FitDelta Computes the delta between a resource oject representing available
//resources an operand representing resources being requested.  Any
//field that is less than 0 after the operation represents an
//insufficient resource.
func (r *ResourceInfo) FitDelta(rr *ResourceInfo) *ResourceInfo {
	if rr.CPUNodes > 0 {
		r.CPUNodes -= rr.CPUNodes
	}

	for rrName, rrQuant := range rr.ScalarResources {
		if r.ScalarResources == nil {
			r.ScalarResources = map[v1.ResourceName]float64{}
		}

		if rrQuant > 0 {
			r.ScalarResources[rrName] -= rrQuant + minMilliScalarResources
		}
	}

	return r
}

// Less checks whether a resource is less than other
func (r *ResourceInfo) Less(rr *ResourceInfo) bool {
	lessFunc := func(l, r float64) bool {
		return l < r
	}

	if !(r.CPUNodes < rr.CPUNodes) {
		return false
	}

	if r.ScalarResources == nil {
		if rr.ScalarResources != nil {
			for _, rrQuant := range rr.ScalarResources {
				if rrQuant <= minMilliScalarResources {
					return false
				}
			}
		}
		return true
	}

	if rr.ScalarResources == nil {
		return false
	}

	for rName, rQuant := range r.ScalarResources {
		rrQuant := rr.ScalarResources[rName]
		if !lessFunc(rQuant, rrQuant) {
			return false
		}
	}

	return true
}

// LessEqualStrict checks whether a resource is less or equal than other
func (r *ResourceInfo) LessEqualStrict(rr *ResourceInfo) bool {
	lessFunc := func(l, r float64) bool {
		return l <= r
	}

	if !(r.CPUNodes < rr.CPUNodes) {
		return false
	}

	for rName, rQuant := range r.ScalarResources {
		if !lessFunc(rQuant, rr.ScalarResources[rName]) {
			return false
		}
	}

	return true
}

// LessEqual checks whether a resource is less than other resource
func (r *ResourceInfo) LessEqual(rr *ResourceInfo) bool {
	lessEqualFunc := func(l, r, diff float64) bool {
		if l < r || math.Abs(l-r) < diff {
			return true
		}
		return false
	}

	if !(r.CPUNodes <= rr.CPUNodes) {
		return false
	}

	if r.ScalarResources == nil {
		return true
	}

	for rName, rQuant := range r.ScalarResources {
		if rQuant <= minMilliScalarResources {
			continue
		}
		if rr.ScalarResources == nil {
			return false
		}

		rrQuant := rr.ScalarResources[rName]
		if !lessEqualFunc(rQuant, rrQuant, minMilliScalarResources) {
			return false
		}
	}

	return true
}

// Diff calculate the difference between two resource
func (r *ResourceInfo) Diff(rr *ResourceInfo) (*ResourceInfo, *ResourceInfo) {
	increasedVal := EmptyResource()
	decreasedVal := EmptyResource()
	if r.CPUNodes > rr.CPUNodes {
		increasedVal.CPUNodes += r.CPUNodes - rr.CPUNodes
	} else {
		decreasedVal.CPUNodes += rr.CPUNodes - r.CPUNodes
	}

	for rName, rQuant := range r.ScalarResources {
		rrQuant := rr.ScalarResources[rName]

		if rQuant > rrQuant {
			if increasedVal.ScalarResources == nil {
				increasedVal.ScalarResources = map[v1.ResourceName]float64{}
			}
			increasedVal.ScalarResources[rName] += rQuant - rrQuant
		} else {
			if decreasedVal.ScalarResources == nil {
				decreasedVal.ScalarResources = map[v1.ResourceName]float64{}
			}
			decreasedVal.ScalarResources[rName] += rrQuant - rQuant
		}
	}

	return increasedVal, decreasedVal
}

// String returns resource details in string format
func (r *ResourceInfo) String() string {
	str := fmt.Sprintf("cpu %v", r.CPUNodes)
	for rName, rQuant := range r.ScalarResources {
		str = fmt.Sprintf("%s, %s %0.2f", str, rName, rQuant)
	}
	return str
}

// Get returns the resource value for that particular resource type
func (r *ResourceInfo) Get(rn v1.ResourceName) float64 {
	if r.ScalarResources == nil {
		return 0
	}
	return r.ScalarResources[rn]
}

// ResourceNames returns all resource types
func (r *ResourceInfo) ResourceNames() []v1.ResourceName {
	resNames := []v1.ResourceName{v1.ResourceCPU, v1.ResourceMemory}

	for rName := range r.ScalarResources {
		resNames = append(resNames, rName)
	}

	return resNames
}

// AddScalar adds a resource by a scalar value of this resource.
func (r *ResourceInfo) AddScalar(name v1.ResourceName, quantity float64) {
	r.SetScalar(name, r.ScalarResources[name]+quantity)
}

// SetScalar sets a resource by a scalar value of this resource.
func (r *ResourceInfo) SetScalar(name v1.ResourceName, quantity float64) {
	// Lazily allocate scalar resource map.
	if r.ScalarResources == nil {
		r.ScalarResources = map[v1.ResourceName]float64{}
	}
	r.ScalarResources[name] = quantity
}
