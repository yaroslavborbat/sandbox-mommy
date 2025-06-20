package util

import (
	"fmt"
	"os"
)

const (
	namespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	tokenPath     = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	caPath        = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

func GetNamespaceFromFs() (string, error) {
	b, err := os.ReadFile(namespacePath)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func GetServiceAccountToken() (string, error) {
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", fmt.Errorf("failed to read service account token: %w", err)
	}
	return string(token), nil
}

func GetCA() ([]byte, error) {
	caCert, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %v", err)
	}

	return caCert, nil
}
