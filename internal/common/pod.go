package common

import "github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1"

const (
	NamePrefix = "sandbox-"
)

func GetFullName(sandbox *v1alpha1.Sandbox) string {
	return NamePrefix + string(sandbox.GetUID())
}
