package condition

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Conder interface {
	Condition() metav1.Condition
}

func NewConditionBuilder(conditionType fmt.Stringer) *ConditionBuilder {
	return &ConditionBuilder{
		cond: metav1.Condition{
			Type:   conditionType.String(),
			Status: metav1.ConditionUnknown,
			Reason: "Unknown",
		},
	}
}

type ConditionBuilder struct {
	cond metav1.Condition
}

func (c *ConditionBuilder) Condition() metav1.Condition {
	return *c.cond.DeepCopy()
}

func (c *ConditionBuilder) Status(status metav1.ConditionStatus) *ConditionBuilder {
	if status != "" {
		c.cond.Status = status
	}
	return c
}

func (c *ConditionBuilder) Reason(reason fmt.Stringer) *ConditionBuilder {
	if r := reason.String(); r != "" {
		c.cond.Reason = r
	}
	return c
}

func (c *ConditionBuilder) Message(msg string) *ConditionBuilder {
	c.cond.Message = msg
	return c
}

func (c *ConditionBuilder) Generation(generation int64) *ConditionBuilder {
	c.cond.ObservedGeneration = generation
	return c
}

func (c *ConditionBuilder) Clone() *ConditionBuilder {
	return &ConditionBuilder{
		cond: *c.cond.DeepCopy(),
	}
}

func GetCondition(conditionType fmt.Stringer, conditions []metav1.Condition) (metav1.Condition, bool) {
	for _, condition := range conditions {
		if condition.Type == conditionType.String() {
			return condition, true
		}
	}
	return metav1.Condition{}, false
}

func SetCondition(c Conder, conditions *[]metav1.Condition) {
	meta.SetStatusCondition(conditions, c.Condition())
}

func RemoveCondition(conditionType fmt.Stringer, conditions *[]metav1.Condition) {
	meta.RemoveStatusCondition(conditions, conditionType.String())
}
