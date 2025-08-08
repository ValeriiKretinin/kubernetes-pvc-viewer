package backend

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type PVCStatus string

const (
	StatusReady        PVCStatus = "Ready"
	StatusAgentPending PVCStatus = "AgentPending"
	StatusMountBlocked PVCStatus = "MountBlocked"
	StatusReadOnly     PVCStatus = "ReadOnly"
)

type StatusService struct{ Client kubernetes.Interface }

func (s *StatusService) GetStatus(ctx context.Context, ns, pvc string) (PVCStatus, error) {
	// Simplified: if agent Pod exists and Ready -> Ready, else AgentPending
	name := AgentName(ns, pvc)
	pod, err := s.Client.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return StatusAgentPending, nil
	}
	for _, c := range pod.Status.Conditions {
		if c.Type == "Ready" && c.Status == "True" {
			return StatusReady, nil
		}
	}
	return StatusAgentPending, nil
}

